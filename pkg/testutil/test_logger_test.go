package testutil

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestSetupMockLoggerPassThrough(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	SetupMockLoggerPassThrough(mockLogger)

	// Should not panic when calling any method
	mockLogger.Info("test message", zap.String("key", "value"))
	mockLogger.Infof("test %s", "message")
	mockLogger.Error("error message", zap.String("key", "value"))
	mockLogger.Errorf("error %s", "message")
	mockLogger.Debug("debug message", zap.String("key", "value"))
	mockLogger.Debugf("debug %s", "message")
	mockLogger.Warn("warn message", zap.String("key", "value"))
	mockLogger.Warnf("warn %s", "message")

	assert.True(t, true) // If we get here, pass-through works
}

func TestSetupMockLoggerSilent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	SetupMockLoggerSilent(mockLogger)

	// Logger should not expect any calls
	// If a call is made, the test will fail (due to gomock)
	assert.True(t, true)
}

func TestSetupMockLoggerPassThrough_WithFlowStep(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	SetupMockLoggerPassThrough(mockLogger)

	// Should not panic when calling flow step methods
	mockLogger.InfoWithFlowStep("test", "flow", "state", "step", zap.String("key", "value"))
	mockLogger.ErrorWithFlowStep("error", "flow", "state", "step", zap.String("key", "value"))
	mockLogger.DebugWithFlowStep("debug", "flow", "state", "step", zap.String("key", "value"))
	mockLogger.WarnWithFlowStep("warn", "flow", "state", "step", zap.String("key", "value"))

	assert.True(t, true)
}

func TestSetupMockLoggerPassTerminal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	SetupMockLoggerPassThrough(mockLogger)

	// Test terminal operations
	mockLogger.GetLogger()
	mockLogger.GetLogs(logger.LogFilter{})
	mockLogger.GetLogStats()
	mockLogger.SetTUIMode(true)
	mockLogger.IsTUIMode()
	mockLogger.EnableFileLogging("/tmp", testUUID)
	mockLogger.CloseFileLogging()
	mockLogger.Flush()

	assert.True(t, true)
}
