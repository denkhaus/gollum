package builtin

import (
	"os"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// mockLoggerForTest creates a mock logger for testing.
// Returns a no-op logger implementation that satisfies the LoggerService interface.
func mockLoggerForTest(t *testing.T) logger.LoggerService {
	return &nopLogger{}
}

// nopLogger is a no-op implementation of LoggerService for testing.
type nopLogger struct{}

func (n *nopLogger) Info(msg string, fields ...zap.Field)              {}
func (n *nopLogger) Infof(template string, args ...any)                {}
func (n *nopLogger) Error(msg string, fields ...zap.Field)             {}
func (n *nopLogger) Errorf(template string, args ...any)               {}
func (n *nopLogger) Debug(msg string, fields ...zap.Field)             {}
func (n *nopLogger) Debugf(template string, args ...any)               {}
func (n *nopLogger) Warn(msg string, fields ...zap.Field)              {}
func (n *nopLogger) Warnf(template string, args ...any)                {}
func (n *nopLogger) GetLogger() *zap.Logger                            { return zap.NewNop() }
func (n *nopLogger) GetLogs(filter logger.LogFilter) []logger.LogEntry { return nil }
func (n *nopLogger) GetLogStats() map[string]interface{}               { return nil }
func (n *nopLogger) SetTUIMode(enabled bool)                           {}
func (n *nopLogger) IsTUIMode() bool                                   { return false }
func (n *nopLogger) EnableFileLogging(_ string, _ uuid.UUID) error     { return nil }
func (n *nopLogger) CloseFileLogging() error                           { return nil }

// testGetEnvOrDefault gets an environment variable or returns the default value.
func testGetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
