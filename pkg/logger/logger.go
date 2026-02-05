// Package logger provides logging utilities and configurations for the brevo-auth-sender service.
package logger

import (
	"bytes"
	"os"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
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
}

// service implements the Service interface
type service struct {
	logger         *zap.Logger
	atomicLevel    zap.AtomicLevel
	config         zap.Config
	originalLogger *zap.Logger
	logBuffer      *logBuffer
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

	// logger, err := config.Build()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to build logger: %w", err)
	// }

	// Build encoder with raw mode line endings (\r\n instead of \n)
	// This is necessary for proper terminal output in raw terminal mode
	encoder := newRawModeConsoleEncoder(config.EncoderConfig)

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), atomicLevel.Level())
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Initialize log buffer from config
	loggingConfig := cnf.GetLoggingConfig()
	logBuffer := newLogBuffer(loggingConfig.SessionLogBufferSize, loggingConfig.SessionLogEnabled)

	return &service{
		logger:         logger,
		atomicLevel:    atomicLevel,
		config:         config,
		originalLogger: logger,
		logBuffer:      logBuffer,
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
	// Note: We don't store formatted messages in buffer to avoid duplication
	// The structured Info() method should be used for buffer storage
}

func (s *service) Error(msg string, fields ...zap.Field) {
	s.logger.Error(msg, fields...)
	s.storeInBuffer("error", msg, fields)
}

func (s *service) Errorf(template string, args ...any) {
	s.logger.Sugar().Errorf(template, args...)
	// Note: We don't store formatted messages in buffer to avoid duplication
}

func (s *service) Debug(msg string, fields ...zap.Field) {
	s.logger.Debug(msg, fields...)
	s.storeInBuffer("debug", msg, fields)
}

func (s *service) Debugf(template string, args ...any) {
	s.logger.Sugar().Debugf(template, args...)
	// Note: We don't store formatted messages in buffer to avoid duplication
}

func (s *service) Warn(msg string, fields ...zap.Field) {
	s.logger.Warn(msg, fields...)
	s.storeInBuffer("warn", msg, fields)
}

func (s *service) Warnf(template string, args ...any) {
	s.logger.Sugar().Warnf(template, args...)
	// Note: We don't store formatted messages in buffer to avoid duplication
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

// ============================================================================
// Custom Console Encoder with \r\n line endings for raw terminal mode
// ============================================================================

// rawModeConsoleEncoder is a custom console encoder that uses \r\n instead of \n
// for proper line endings in raw terminal mode.
type rawModeConsoleEncoder struct {
	zapcore.Encoder
}

// newRawModeConsoleEncoder creates a new console encoder with \r\n line endings.
func newRawModeConsoleEncoder(encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
	return &rawModeConsoleEncoder{
		Encoder: zapcore.NewConsoleEncoder(encoderConfig),
	}
}

// Clone creates a copy of the encoder.
func (e *rawModeConsoleEncoder) Clone() zapcore.Encoder {
	return &rawModeConsoleEncoder{
		Encoder: e.Encoder.Clone(),
	}
}

// EncodeEntry encodes a log entry and replaces \n with \r\n for raw terminal mode.
func (e *rawModeConsoleEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}

	// Replace all \n with \r\n for proper raw terminal mode handling
	// We need to be careful not to double-replace existing \r\n
	str := buf.String()
	var result bytes.Buffer
	result.Grow(len(str) + len(str)/10) // Pre-allocate with some extra space

	for i := 0; i < len(str); i++ {
		if str[i] == '\n' {
			// Check if this is already \r\n
			if i > 0 && str[i-1] == '\r' {
				// Already \r\n, just write the \n
				result.WriteByte('\n')
			} else {
				// Standalone \n, convert to \r\n
				result.WriteString("\r\n")
			}
		} else {
			result.WriteByte(str[i])
		}
	}

	// Create a new buffer from the pool and write our processed string to it
	newBuf := buffer.NewPool().Get()
	newBuf.WriteString(result.String())
	return newBuf, nil
}
