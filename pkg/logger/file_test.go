package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// TestJSONLogFormat verifies that file logs are written in JSON format
func TestJSONLogFormat(t *testing.T) {
	// Create temp directory for logs
	tmpDir := t.TempDir()
	sessionID := uuid.New()

	// Create a minimal logger service
	logger, err := zap.NewProduction()
	require.NoError(t, err)

	// Setup gomock for config service
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfigService := config.NewMockConfigService(ctrl)
	mockConfigService.EXPECT().GetLoggingConfig().Return(&config.LoggingConfig{
		SessionLogEnabled:    false,
		SessionLogBufferSize: 1000,
		MaxSessionLogFiles:   10,
	}).AnyTimes()

	svc := &service{
		logger:        logger,
		atomicLevel:   zap.NewAtomicLevelAt(zap.InfoLevel),
		config:        zap.NewProductionConfig(),
		logBuffer:     newLogBuffer(1000, false), // disabled for test
		configService: mockConfigService,
	}

	// Enable file logging
	logErr := svc.EnableFileLogging(tmpDir, sessionID)
	require.NoError(t, logErr)
	require.NoError(t, err)

	// Log with COMPLETE context (all fields required for file logging)
	// Using InfoWithContext instead of InfoWithAgent to provide full context
	agentID := uuid.New()
	channelID := uuid.New()
	testCtx := shared.LoggingContext{
		SessionID: "test-session",
		ChannelID: channelID,
		AgentID:   agentID,
	}
	svc.InfoWithContext("Test message", testCtx, zap.String("test_field", "test_value"))

	// Flush to ensure write
	err = svc.Flush()
	require.NoError(t, err)

	// Close file logging to ensure all writes are flushed
	err = svc.CloseFileLogging()
	require.NoError(t, err)

	// Read the log file (it's in the logs/ subdirectory)
	logPath := filepath.Join(tmpDir, "logs", sessionID.String()+".log")
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	// Verify it's valid JSON
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.GreaterOrEqual(t, len(lines), 2, "Log file should have at least two lines (enabled message + test message)")

	// Lines are:
	// 1. "Session logging enabled"
	// 2. The actual test message (with complete context, so it's logged to file)
	var logEntry map[string]interface{}
	// Find the test message (skip the "Session logging enabled" line)
	testMsgIndex := -1
	for i, line := range lines {
		if strings.Contains(line, "Test message") {
			testMsgIndex = i
			break
		}
	}
	require.NotEqual(t, -1, testMsgIndex, "Should find test message in log file")
	err = json.Unmarshal([]byte(lines[testMsgIndex]), &logEntry)
	require.NoError(t, err, "Log line should be valid JSON")

	// Verify required fields
	assert.Contains(t, logEntry, "timestamp")
	assert.Contains(t, logEntry, "level")
	assert.Contains(t, logEntry, "message")
	assert.Contains(t, logEntry, "agent_id")
	assert.Contains(t, logEntry, "test_field")
	assert.Equal(t, "Test message", logEntry["message"])
	assert.Equal(t, agentID.String(), logEntry["agent_id"])
	assert.Equal(t, "test_value", logEntry["test_field"])
}
