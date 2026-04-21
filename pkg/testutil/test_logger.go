package testutil

import (
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// SetupMockLoggerPassThrough configures a mock LoggerService to pass through all calls.
// This is useful for tests that don't need to verify logging behavior.
func SetupMockLoggerPassThrough(mockLogger *logger.MockLoggerService) {
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warnf(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().ErrorWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().DebugWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().WarnWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().InfoWithContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().ErrorWithContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().DebugWithContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().WarnWithContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().GetLogger().Return(nil).AnyTimes()
	mockLogger.EXPECT().GetLogs(gomock.Any()).Return([]logger.LogEntry{}).AnyTimes()
	mockLogger.EXPECT().GetLogStats().Return(map[string]interface{}{}).AnyTimes()
	mockLogger.EXPECT().SetTUIMode(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().IsTUIMode().Return(false).AnyTimes()
	mockLogger.EXPECT().EnableFileLogging(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockLogger.EXPECT().CloseFileLogging().Return(nil).AnyTimes()
	mockLogger.EXPECT().Flush().Return(nil).AnyTimes()
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).AnyTimes()
}

// SetupMockLoggerSilent configures a mock LoggerService to expect NO calls.
// This is useful for tests that should not log anything.
func SetupMockLoggerSilent(mockLogger *logger.MockLoggerService) {
	// No expectations set - any call will cause test failure
}

// testUUID is a test UUID for use in tests
var testUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
