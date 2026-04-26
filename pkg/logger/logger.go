// Package logger provides logging utilities and configurations for the brevo-auth-sender service.
package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/shared"
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
	// InfoWithFlowStep logs an info message with flow and step context
	InfoWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field)
	// ErrorWithFlowStep logs an error message with flow and step context
	ErrorWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field)
	// DebugWithFlowStep logs a debug message with flow and step context
	DebugWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field)
	// WarnWithFlowStep logs a warning message with flow and step context
	WarnWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field)
	// InfoWithContext logs an info message with session, channel, and agent context.
	InfoWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field)
	// ErrorWithContext logs an error message with session, channel, and agent context.
	ErrorWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field)
	// DebugWithContext logs a debug message with session, channel, and agent context.
	DebugWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field)
	// WarnWithContext logs a warning message with session, channel, and agent context.
	WarnWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field)
	GetLogger() *zap.Logger
	// GetLogs retrieves log entries from the session buffer
	GetLogs(filter LogFilter) []LogEntry
	// GetLogStats returns statistics about the log buffer
	GetLogStats() map[string]interface{}
	// SetTUIMode disables stdout logging when TUI is active
	SetTUIMode(enabled bool)
	// SetLogForwarder sets the log forwarder for channel-based log routing
	SetLogForwarder(forwarder shared.LogForwarder)
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
	fileLogger     *zap.Logger         // Separate logger that always writes to file
	forwarder      shared.LogForwarder // NEW: for channel log forwarding
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
		_ = s.fileLogger.Sync()           // Immediately sync to disk
	}
	s.storeInBuffer("info", msg, fields)
}

func (s *service) Infof(template string, args ...any) {
	s.logger.Sugar().Infof(template, args...)
	if s.fileLogger != nil {
		s.fileLogger.Sugar().Infof(template, args...) // Always write to file
		_ = s.fileLogger.Sync()                       // Immediately sync to disk
	}
	// Store formatted message in buffer for TUI log panel
	msg := fmt.Sprintf(template, args...)
	s.storeInBuffer("info", msg, nil)
}

func (s *service) Error(msg string, fields ...zap.Field) {
	s.logger.Error(msg, fields...)
	if s.fileLogger != nil {
		s.fileLogger.Error(msg, fields...) // Always write to file
		_ = s.fileLogger.Sync()            // Immediately sync to disk
	}
	s.storeInBuffer("error", msg, fields)
}

func (s *service) Errorf(template string, args ...any) {
	s.logger.Sugar().Errorf(template, args...)
	if s.fileLogger != nil {
		s.fileLogger.Sugar().Errorf(template, args...) // Always write to file
		_ = s.fileLogger.Sync()                        // Immediately sync to disk
	}
	// Store formatted message in buffer for TUI log panel
	msg := fmt.Sprintf(template, args...)
	s.storeInBuffer("error", msg, nil)
}

func (s *service) Debug(msg string, fields ...zap.Field) {
	s.logger.Debug(msg, fields...)
	if s.fileLogger != nil {
		s.fileLogger.Debug(msg, fields...) // Always write to file
		_ = s.fileLogger.Sync()            // Immediately sync to disk
	}
	s.storeInBuffer("debug", msg, fields)
}

func (s *service) Debugf(template string, args ...any) {
	s.logger.Sugar().Debugf(template, args...)
	if s.fileLogger != nil {
		s.fileLogger.Sugar().Debugf(template, args...) // Always write to file
		_ = s.fileLogger.Sync()                        // Immediately sync to disk
	}
	// Store formatted message in buffer for TUI log panel
	msg := fmt.Sprintf(template, args...)
	s.storeInBuffer("debug", msg, nil)
}

func (s *service) Warn(msg string, fields ...zap.Field) {
	s.logger.Warn(msg, fields...)
	if s.fileLogger != nil {
		s.fileLogger.Warn(msg, fields...) // Always write to file
		_ = s.fileLogger.Sync()           // Immediately sync to disk
	}
	s.storeInBuffer("warn", msg, fields)
}

func (s *service) Warnf(template string, args ...any) {
	s.logger.Sugar().Warnf(template, args...)
	if s.fileLogger != nil {
		s.fileLogger.Sugar().Warnf(template, args...) // Always write to file
		_ = s.fileLogger.Sync()                       // Immediately sync to disk
	}
	// Store formatted message in buffer for TUI log panel
	msg := fmt.Sprintf(template, args...)
	s.storeInBuffer("warn", msg, nil)
}

// InfoWithFlowStep logs an info message with flow and step context included as structured fields.
func (s *service) InfoWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field) {
	allFields := append([]zap.Field{
		zap.String("flow", flowName),
		zap.String("state", stateName),
		zap.String("step", stepType),
	}, fields...)
	s.Info(msg, allFields...)
}

// ErrorWithFlowStep logs an error message with flow and step context included as structured fields.
func (s *service) ErrorWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field) {
	allFields := append([]zap.Field{
		zap.String("flow", flowName),
		zap.String("state", stateName),
		zap.String("step", stepType),
	}, fields...)
	s.Error(msg, allFields...)
}

// DebugWithFlowStep logs a debug message with flow and step context included as structured fields.
func (s *service) DebugWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field) {
	allFields := append([]zap.Field{
		zap.String("flow", flowName),
		zap.String("state", stateName),
		zap.String("step", stepType),
	}, fields...)
	s.Debug(msg, allFields...)
}

// WarnWithFlowStep logs a warning message with flow and step context included as structured fields.
func (s *service) WarnWithFlowStep(msg string, flowName, stateName, stepType string, fields ...zap.Field) {
	allFields := append([]zap.Field{
		zap.String("flow", flowName),
		zap.String("state", stateName),
		zap.String("step", stepType),
	}, fields...)
	s.Warn(msg, allFields...)
}

// InfoWithContext logs an info message with session, channel, and agent context.
func (s *service) InfoWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field) {
	s.logWithContext("info", msg, ctx, fields...)
}

// ErrorWithContext logs an error message with session, channel, and agent context.
func (s *service) ErrorWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field) {
	s.logWithContext("error", msg, ctx, fields...)
}

// DebugWithContext logs a debug message with session, channel, and agent context.
func (s *service) DebugWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field) {
	s.logWithContext("debug", msg, ctx, fields...)
}

// WarnWithContext logs a warning message with session, channel, and agent context.
func (s *service) WarnWithContext(msg string, ctx shared.SessionContext, fields ...zap.Field) {
	s.logWithContext("warn", msg, ctx, fields...)
}

// storeInBuffer stores a log entry in the session buffer.
func (s *service) storeInBuffer(level string, msg string, fields []zap.Field) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Fields:    zapFieldsToMap(fields),
	}

	s.logBuffer.add(entry)
}

// logWithContext is the internal implementation for context-aware logging.
// It validates the context, logs to zap, stores in buffer, and forwards to channels.
// REQUIRES: Complete SessionContext (all fields valid). Use IsValid() to validate.
func (s *service) logWithContext(level string, msg string, ctx shared.SessionContext, fields ...zap.Field) {
	// STRICT VALIDATION: Reject incomplete contexts per Task 3 spec
	if !ctx.IsValid() {
		// Log error but still record locally - just don't forward to channels
		s.logger.Error("incomplete logging context",
			zap.String("level", level),
			zap.String("message", msg),
			zap.Bool("has_session_id", ctx.SessionID != uuid.Nil),
			zap.Bool("has_channel_id", ctx.ChannelID != uuid.Nil),
			zap.Bool("has_agent_id", ctx.AgentID != uuid.Nil),
		)

		// Still log locally with whatever context we have
		allFields := append([]zap.Field{
			zap.String("agent_id", ctx.AgentID.String()),
		}, fields...)

		switch level {
		case "debug":
			s.logger.Debug(msg, allFields...)
		case "info":
			s.logger.Info(msg, allFields...)
		case "warn":
			s.logger.Warn(msg, allFields...)
		case "error":
			s.logger.Error(msg, allFields...)
		}

		// Store in buffer but don't forward
		s.storeInBuffer(level, msg, allFields)
		return
	}

	// Context is valid - proceed with full logging and forwarding
	allFields := append([]zap.Field{
		zap.String("session_id", ctx.SessionID.String()),
		zap.String("channel_id", ctx.ChannelID.String()),
		zap.String("agent_id", ctx.AgentID.String()),
	}, fields...)

	// Log to zap (stdout/file)
	switch level {
	case "debug":
		s.logger.Debug(msg, allFields...)
		if s.fileLogger != nil {
			s.fileLogger.Debug(msg, allFields...)
			_ = s.fileLogger.Sync()
		}
	case "info":
		s.logger.Info(msg, allFields...)
		if s.fileLogger != nil {
			s.fileLogger.Info(msg, allFields...)
			_ = s.fileLogger.Sync()
		}
	case "warn":
		s.logger.Warn(msg, allFields...)
		if s.fileLogger != nil {
			s.fileLogger.Warn(msg, allFields...)
			_ = s.fileLogger.Sync()
		}
	case "error":
		s.logger.Error(msg, allFields...)
		if s.fileLogger != nil {
			s.fileLogger.Error(msg, allFields...)
			_ = s.fileLogger.Sync()
		}
	}

	// Store in buffer with full context
	entry := LogEntry{
		Timestamp:      time.Now(),
		Level:          level,
		Message:        msg,
		Fields:         zapFieldsToMap(allFields),
		SessionContext: ctx,
	}
	s.logBuffer.add(entry)

	// Forward to channel facade via LogForwarder
	if s.forwarder != nil {
		channelEntry := shared.LogEntry{
			Level:          level,
			Message:        msg,
			Timestamp:      entry.Timestamp,
			Fields:         entry.Fields,
			SessionContext: ctx,
		}
		s.forwarder.ForwardLog(channelEntry)
	}
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

// SetLogForwarder sets the log forwarder for channel-based log routing.
func (s *service) SetLogForwarder(forwarder shared.LogForwarder) {
	s.forwarder = forwarder
}
