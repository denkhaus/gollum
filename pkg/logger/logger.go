// Package logger provides logging utilities and configurations for the brevo-auth-sender service.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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

	// Build encoder with raw mode line endings (\r\n instead of \n)
	// This is necessary for proper terminal output in raw terminal mode
	// encoder := newRawModeConsoleEncoder(config.EncoderConfig)

	// core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), atomicLevel.Level())
	// logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Initialize log buffer from config
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
	s.storeInBuffer("info", msg, fields)
}

func (s *service) Infof(template string, args ...any) {
	s.logger.Sugar().Infof(template, args...)
	// Store formatted message in buffer for TUI log panel
	msg := fmt.Sprintf(template, args...)
	s.storeInBuffer("info", msg, nil)
}

func (s *service) Error(msg string, fields ...zap.Field) {
	s.logger.Error(msg, fields...)
	s.storeInBuffer("error", msg, fields)
}

func (s *service) Errorf(template string, args ...any) {
	s.logger.Sugar().Errorf(template, args...)
	// Store formatted message in buffer for TUI log panel
	msg := fmt.Sprintf(template, args...)
	s.storeInBuffer("error", msg, nil)
}

func (s *service) Debug(msg string, fields ...zap.Field) {
	s.logger.Debug(msg, fields...)
	s.storeInBuffer("debug", msg, fields)
}

func (s *service) Debugf(template string, args ...any) {
	s.logger.Sugar().Debugf(template, args...)
	// Store formatted message in buffer for TUI log panel
	msg := fmt.Sprintf(template, args...)
	s.storeInBuffer("debug", msg, nil)
}

func (s *service) Warn(msg string, fields ...zap.Field) {
	s.logger.Warn(msg, fields...)
	s.storeInBuffer("warn", msg, fields)
}

func (s *service) Warnf(template string, args ...any) {
	s.logger.Sugar().Warnf(template, args...)
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

// SetTUIMode disables stdout logging and redirects all logs to the buffer only.
// This prevents duplicate log output when the TUI is running.
func (s *service) SetTUIMode(enabled bool) {
	s.tuiMode = enabled
	if enabled {
		// Replace the core with one that only writes to discard (noop output)
		// This prevents stdout logging during TUI execution
		s.logger = s.logger.WithOptions(zap.WrapCore(func(zapcore.Core) zapcore.Core {
			// Create a noop sync to discard output
			noopSync := zapcore.AddSync(io.Discard)
			// Create a new core with the same encoder but noop output
			encoder := newRawModeConsoleEncoder(s.config.EncoderConfig)
			return zapcore.NewCore(
				encoder,
				noopSync,
				s.atomicLevel,
			)
		}))
	} else {
		// Restore original logger with stdout output
		s.logger = s.originalLogger
	}
}

// IsTUIMode returns whether TUI mode is enabled.
func (s *service) IsTUIMode() bool {
	return s.tuiMode
}

// EnableFileLogging enables file logging to .gollum/logs/<sessionID>.log.
// It creates the logs directory, cleans up old logs if needed, and opens the log file.
func (s *service) EnableFileLogging(gollumDir string, sessionID uuid.UUID) error {
	// Create logs directory
	logsDir := filepath.Join(gollumDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Clean up old log files if configured
	maxFiles := s.configService.GetLoggingConfig().MaxSessionLogFiles
	if maxFiles > 0 {
		if err := s.cleanupOldLogs(logsDir, maxFiles); err != nil {
			// Log but don't fail - cleanup is best effort
			s.Warnf("failed to cleanup old logs: %v", err)
		}
	}

	// Create log file path
	logFilePath := filepath.Join(logsDir, sessionID.String()+".log")

	// Open file for writing
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	s.logFile = file
	s.logFilePath = logFilePath

	// Add file writer to logger using WrapCore
	s.logger = s.logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		// Create file encoder with the same config
		encoder := newRawModeConsoleEncoder(s.config.EncoderConfig)
		fileCore := zapcore.NewCore(encoder, zapcore.AddSync(file), s.atomicLevel)
		// Combine existing core with file core
		return zapcore.NewTee(core, fileCore)
	}))

	s.Infof("Session logging enabled: %s", logFilePath)
	return nil
}

// CloseFileLogging closes the current log file if open.
func (s *service) CloseFileLogging() error {
	if s.logFile == nil {
		return nil
	}

	// Sync and close the file
	if err := s.logFile.Sync(); err != nil {
		s.Warnf("failed to sync log file: %v", err)
	}

	if err := s.logFile.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	s.logFile = nil
	s.logFilePath = ""
	return nil
}

// cleanupOldLogs removes the oldest log files when the count exceeds maxFiles.
func (s *service) cleanupOldLogs(logsDir string, maxFiles int) error {
	// Read all log files
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return fmt.Errorf("failed to read logs directory: %w", err)
	}

	// Filter for .log files and get their info
	type logFileInfo struct {
		name    string
		modTime time.Time
	}
	var logFiles []logFileInfo

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".log" {
			info, err := entry.Info()
			if err != nil {
				continue // Skip files we can't read
			}
			logFiles = append(logFiles, logFileInfo{
				name:    entry.Name(),
				modTime: info.ModTime(),
			})
		}
	}

	// Check if cleanup is needed
	if len(logFiles) < maxFiles {
		return nil
	}

	// Sort by modification time (oldest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].modTime.Before(logFiles[j].modTime)
	})

	// Delete oldest files until we're under the limit
	filesToDelete := len(logFiles) - maxFiles + 1 // +1 because we're about to create a new one
	for i := 0; i < filesToDelete && i < len(logFiles); i++ {
		filePath := filepath.Join(logsDir, logFiles[i].name)
		if err := os.Remove(filePath); err != nil {
			s.Warnf("failed to remove old log file %s: %v", filePath, err)
		} else {
			s.Debugf("removed old log file: %s", filePath)
		}
	}

	return nil
}
