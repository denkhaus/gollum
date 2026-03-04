// Package logger provides logging utilities and configurations for the brevo-auth-sender service.
package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerService defines the logger service interface
//
//revive:disable-next-line:exported
type LoggerService interface {
	Info(msg string, fields ...zap.Field)
	Infof(template string, args ...any)
	Error(msg string, fields ...zap.Field)
	Errorf(template string, args ...any)
	Debug(msg string, fields ...zap.Field)
	Debugf(template string, args ...any)
	Warn(msg string, fields ...zap.Field)
	Warnf(template string, args ...any)
	GetLogger() *zap.Logger
	// GetLogs retrieves log entries from the session buffer
	GetLogs(filter LogFilter) []LogEntry
	// GetLogStats returns statistics about the log buffer
	GetLogStats() map[string]interface{}
	// SetTUIMode disables stdout logging when TUI is active
	SetTUIMode(enabled bool)
	// IsTUIMode returns whether TUI mode is enabled
	IsTUIMode() bool
	// EnableFileLogging enables file logging to .gollum/logs/<sessionID>.log
	// Creates logs directory, cleans old logs, opens file for writing
	EnableFileLogging(gollumDir string, sessionID uuid.UUID) error
	// CloseFileLogging closes the current log file if open
	CloseFileLogging() error
	// Flush forces an immediate sync of the log file to disk
	Flush() error
}

// service implements the Service interface
type service struct {
	logger         *zap.Logger
	atomicLevel    zap.AtomicLevel
	config         zap.Config
	originalLogger *zap.Logger
	logBuffer      *logBuffer
	tuiMode        bool
	configService  config.ConfigService
	logFile        *os.File
	logFilePath    string
	fileLogger     *zap.Logger // Separate logger that always writes to file
}

// NewService creates a new logger service
func NewService(injector do.Injector) (LoggerService, error) {
	cnf := do.MustInvoke[config.ConfigService](injector)
	var config zap.Config

	if cnf.IsDevMode() {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		// Use console encoding even in production for better readability
		// JSON encoding breaks TUI log parsing
		config = zap.NewProductionConfig()
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	// Config customization
	config.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	config.OutputPaths = []string{"stdout"}

	atomicLevel := zap.NewAtomicLevelAt(getLogLevel(cnf.GetLogLevel()))
	config.Level = atomicLevel

	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	loggingConfig := cnf.GetLoggingConfig()
	logBuffer := newLogBuffer(loggingConfig.SessionLogBufferSize, loggingConfig.SessionLogEnabled)

	return &service{
		logger:         logger,
		atomicLevel:    atomicLevel,
		config:         config,
		originalLogger: logger,
		logBuffer:      logBuffer,
		tuiMode:        false,
		configService:  cnf,
	}, nil
}

// getLogLevel converts string log level to zapcore.Level
func getLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Implement Service interface methods

func (s *service) Info(msg string, fields ...zap.Field) {
	s.logger.Info(msg, fields...)
	if s.fileLogger != nil {
		s.fileLogger.Info(msg, fields...) // Always write to file
		s.fileLogger.Sync()               // Immediately sync to disk
	}
	s.storeInBuffer("info", msg, fields)
}

func (s *service) Infof(template string, args ...any) {
	s.logger.Sugar().Infof(template, args...)
	if s.fileLogger != nil {
		s.fileLogger.Sugar().Infof(template, args...) // Always write to file
		s.fileLogger.Sync()                           // Immediately sync to disk
	}
	// Store formatted message in buffer for TUI log panel
	msg := fmt.Sprintf(template, args...)
	s.storeInBuffer("info", msg, nil)
}

func (s *service) Error(msg string, fields ...zap.Field) {
	s.logger.Error(msg, fields...)
	if s.fileLogger != nil {
		s.fileLogger.Error(msg, fields...) // Always write to file
		s.fileLogger.Sync()                // Immediately sync to disk
	}
	s.storeInBuffer("error", msg, fields)
}

func (s *service) Errorf(template string, args ...any) {
	s.logger.Sugar().Errorf(template, args...)
	if s.fileLogger != nil {
		s.fileLogger.Sugar().Errorf(template, args...) // Always write to file
		s.fileLogger.Sync()                            // Immediately sync to disk
	}
	// Store formatted message in buffer for TUI log panel
	msg := fmt.Sprintf(template, args...)
	s.storeInBuffer("error", msg, nil)
}

func (s *service) Debug(msg string, fields ...zap.Field) {
	s.logger.Debug(msg, fields...)
	if s.fileLogger != nil {
		s.fileLogger.Debug(msg, fields...) // Always write to file
		s.fileLogger.Sync()                // Immediately sync to disk
	}
	s.storeInBuffer("debug", msg, fields)
}

func (s *service) Debugf(template string, args ...any) {
	s.logger.Sugar().Debugf(template, args...)
	if s.fileLogger != nil {
		s.fileLogger.Sugar().Debugf(template, args...) // Always write to file
		s.fileLogger.Sync()                            // Immediately sync to disk
	}
	// Store formatted message in buffer for TUI log panel
	msg := fmt.Sprintf(template, args...)
	s.storeInBuffer("debug", msg, nil)
}

func (s *service) Warn(msg string, fields ...zap.Field) {
	s.logger.Warn(msg, fields...)
	if s.fileLogger != nil {
		s.fileLogger.Warn(msg, fields...) // Always write to file
		s.fileLogger.Sync()               // Immediately sync to disk
	}
	s.storeInBuffer("warn", msg, fields)
}

func (s *service) Warnf(template string, args ...any) {
	s.logger.Sugar().Warnf(template, args...)
	if s.fileLogger != nil {
		s.fileLogger.Sugar().Warnf(template, args...) // Always write to file
		s.fileLogger.Sync()                           // Immediately sync to disk
	}
	// Store formatted message in buffer for TUI log panel
	msg := fmt.Sprintf(template, args...)
	s.storeInBuffer("warn", msg, nil)
}

// storeInBuffer stores a log entry in the session buffer.
func (s *service) storeInBuffer(level string, msg string, fields []zap.Field) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Fields:    zapFieldsToMap(fields),
		// AgentID is optional and would need to be extracted from fields if present
		// For now, we'll leave it as Nil (can be set by tools that call Info/Error/etc)
	}

	s.logBuffer.add(entry)
}

func (s *service) GetLogger() *zap.Logger {
	return s.logger
}

// GetLogs retrieves log entries from the session buffer based on the provided filter.
func (s *service) GetLogs(filter LogFilter) []LogEntry {
	return s.logBuffer.getEntries(filter)
}

// GetLogStats returns statistics about the log buffer.
func (s *service) GetLogStats() map[string]interface{} {
	return s.logBuffer.getStats()
}
