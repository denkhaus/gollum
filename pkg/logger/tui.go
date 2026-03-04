package logger

import (
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SetTUIMode disables stdout logging and redirects all logs to the buffer only.
// This prevents duplicate log output when the TUI is running.
// File logging continues via the separate fileLogger.
func (s *service) SetTUIMode(enabled bool) {
	s.tuiMode = enabled
	if enabled {
		// Create a core that discards stdout (buffer only)
		noopSync := zapcore.AddSync(io.Discard)
		encoder := newRawModeConsoleEncoder(s.config.EncoderConfig)
		stdoutDiscardCore := zapcore.NewCore(encoder, noopSync, s.atomicLevel)

		// Replace logger with discard-only version
		// Note: file logging continues via s.fileLogger in logging methods
		s.logger = zap.New(stdoutDiscardCore, zap.AddCaller())
	} else {
		// Restore original logger with stdout output
		// Note: file logging continues via s.fileLogger in logging methods
		s.logger = s.originalLogger
	}
}

// IsTUIMode returns whether TUI mode is enabled.
func (s *service) IsTUIMode() bool {
	return s.tuiMode
}
