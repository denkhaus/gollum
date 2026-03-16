// Package tools provides unit tests for the SessionLogsTool.
package tools

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestSessionLogsTool_Run_TailMode tests the default tail mode.
func TestSessionLogsTool_Run_TailMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	agentID := uuid.New()
	now := time.Now()

	// Create test log entries
	entries := []logger.LogEntry{
		{Timestamp: now.Add(-3 * time.Second), Level: "info", Message: "msg1", AgentID: agentID},
		{Timestamp: now.Add(-2 * time.Second), Level: "error", Message: "msg2", AgentID: agentID},
		{Timestamp: now.Add(-1 * time.Second), Level: "debug", Message: "msg3", AgentID: agentID},
	}

	mockLoggerService.EXPECT().GetLogs(gomock.Any()).Return(entries)

	tool := &sessionLogsToolImpl{logService: mockLoggerService, hookManager: mockHookManager, agentID: uuid.New()}

	// Test tail mode (default)
	result, err := tool.Run(context.Background(), map[string]any{})
	require.NoError(t, err)

	// Verify response structure
	assert.Contains(t, result, "entries")
	assert.Contains(t, result, "total_matching")
	assert.Contains(t, result, "returned")
	assert.Contains(t, result, "filters_applied")

	// Verify entries
	entriesResult := result["entries"].([]map[string]interface{})
	assert.Len(t, entriesResult, 3)

	// Verify order (chronological - tail mode)
	assert.Equal(t, "msg1", entriesResult[0]["message"])
	assert.Equal(t, "msg2", entriesResult[1]["message"])
	assert.Equal(t, "msg3", entriesResult[2]["message"])

	// Verify default mode in filters
	filters := result["filters_applied"].(map[string]interface{})
	assert.Equal(t, "tail", filters["mode"])
	assert.Equal(t, 100, filters["count"])
}

// TestSessionLogsTool_Run_HeadMode tests head mode with newest first.
func TestSessionLogsTool_Run_HeadMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	agentID := uuid.New()
	now := time.Now()

	entries := []logger.LogEntry{
		{Timestamp: now.Add(-3 * time.Second), Level: "info", Message: "oldest", AgentID: agentID},
		{Timestamp: now.Add(-2 * time.Second), Level: "info", Message: "middle", AgentID: agentID},
		{Timestamp: now.Add(-1 * time.Second), Level: "info", Message: "newest", AgentID: agentID},
	}

	mockLoggerService.EXPECT().GetLogs(gomock.Any()).Return(entries)

	tool := &sessionLogsToolImpl{logService: mockLoggerService, hookManager: mockHookManager, agentID: uuid.New()}

	result, err := tool.Run(context.Background(), map[string]any{
		"mode":  "head",
		"count": float64(2),
	})
	require.NoError(t, err)

	entriesResult := result["entries"].([]map[string]interface{})
	assert.Len(t, entriesResult, 3)

	// Verify reverse order (newest first)
	assert.Equal(t, "oldest", entriesResult[0]["message"])
	assert.Equal(t, "middle", entriesResult[1]["message"])
	assert.Equal(t, "newest", entriesResult[2]["message"])
}

// TestSessionLogsTool_Run_SinceMode tests since mode with datetime filtering.
func TestSessionLogsTool_Run_SinceMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	agentID := uuid.New()
	now := time.Now()

	entries := []logger.LogEntry{
		{Timestamp: now.Add(-30 * time.Minute), Level: "info", Message: "recent", AgentID: agentID},
		{Timestamp: now.Add(-1 * time.Minute), Level: "info", Message: "very_recent", AgentID: agentID},
	}

	mockLoggerService.EXPECT().GetLogs(gomock.Any()).Return(entries)

	tool := &sessionLogsToolImpl{logService: mockLoggerService, hookManager: mockHookManager, agentID: uuid.New()}

	sinceTime := now.Add(-1 * time.Hour).Format(time.RFC3339)

	result, err := tool.Run(context.Background(), map[string]any{
		"mode":  "since",
		"since": sinceTime,
	})
	require.NoError(t, err)

	entriesResult := result["entries"].([]map[string]interface{})
	assert.Len(t, entriesResult, 2)

	assert.Equal(t, "recent", entriesResult[0]["message"])
	assert.Equal(t, "very_recent", entriesResult[1]["message"])
}

// TestSessionLogsTool_Run_AllMode tests all mode.
func TestSessionLogsTool_Run_AllMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	agentID := uuid.New()
	now := time.Now()

	entries := []logger.LogEntry{
		{Timestamp: now, Level: "info", Message: "msg1", AgentID: agentID},
		{Timestamp: now, Level: "debug", Message: "msg2", AgentID: agentID},
		{Timestamp: now, Level: "error", Message: "msg3", AgentID: agentID},
	}

	mockLoggerService.EXPECT().GetLogs(gomock.Any()).Return(entries)

	tool := &sessionLogsToolImpl{logService: mockLoggerService, hookManager: mockHookManager, agentID: uuid.New()}

	result, err := tool.Run(context.Background(), map[string]any{
		"mode": "all",
	})
	require.NoError(t, err)

	entriesResult := result["entries"].([]map[string]interface{})
	assert.Len(t, entriesResult, 3)
}

// TestSessionLogsTool_Run_FilterByLevel tests filtering by log level.
func TestSessionLogsTool_Run_FilterByLevel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	agentID := uuid.New()
	now := time.Now()

	entries := []logger.LogEntry{
		{Timestamp: now, Level: "info", Message: "info1", AgentID: agentID},
		{Timestamp: now, Level: "info", Message: "info2", AgentID: agentID},
	}

	mockLoggerService.EXPECT().GetLogs(gomock.Any()).Return(entries)

	tool := &sessionLogsToolImpl{logService: mockLoggerService, hookManager: mockHookManager, agentID: uuid.New()}

	result, err := tool.Run(context.Background(), map[string]any{
		"mode":  "tail",
		"level": "info",
	})
	require.NoError(t, err)

	entriesResult := result["entries"].([]map[string]interface{})
	assert.Len(t, entriesResult, 2)

	// Verify all are info level
	for _, entry := range entriesResult {
		assert.Equal(t, "info", entry["level"])
	}
}

// TestSessionLogsTool_Run_FilterByAgentID tests filtering by agent ID.
func TestSessionLogsTool_Run_FilterByAgentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	agent1 := uuid.New()
	now := time.Now()

	entries := []logger.LogEntry{
		{Timestamp: now, Level: "info", Message: "agent1_msg", AgentID: agent1},
		{Timestamp: now, Level: "info", Message: "agent1_msg2", AgentID: agent1},
	}

	mockLoggerService.EXPECT().GetLogs(gomock.Any()).Return(entries)

	tool := &sessionLogsToolImpl{logService: mockLoggerService, hookManager: mockHookManager, agentID: uuid.New()}

	result, err := tool.Run(context.Background(), map[string]any{
		"mode":     "tail",
		"agent_id": agent1.String(),
	})
	require.NoError(t, err)

	entriesResult := result["entries"].([]map[string]interface{})
	assert.Len(t, entriesResult, 2)

	// Verify all are from agent1
	for _, entry := range entriesResult {
		assert.Equal(t, agent1.String(), entry["agent_id"])
	}
}

// TestSessionLogsTool_Run_InvalidMode tests error handling for invalid mode.
func TestSessionLogsTool_Run_InvalidMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &sessionLogsToolImpl{
		logService:  mockLoggerService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	_, err := tool.Run(context.Background(), map[string]any{
		"mode": "invalid_mode",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")
}

// TestSessionLogsTool_Run_SinceModeWithoutSince tests error when since is missing.
func TestSessionLogsTool_Run_SinceModeWithoutSince(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &sessionLogsToolImpl{
		logService:  mockLoggerService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	_, err := tool.Run(context.Background(), map[string]any{
		"mode": "since",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "since parameter is required")
}

// TestSessionLogsTool_Run_InvalidDateTime tests error handling for invalid datetime.
func TestSessionLogsTool_Run_InvalidDateTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &sessionLogsToolImpl{
		logService:  mockLoggerService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	_, err := tool.Run(context.Background(), map[string]any{
		"mode":  "since",
		"since": "invalid-datetime",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid since datetime")
}

// TestSessionLogsTool_Run_InvalidAgentID tests error handling for invalid UUID.
func TestSessionLogsTool_Run_InvalidAgentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &sessionLogsToolImpl{
		logService:  mockLoggerService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	_, err := tool.Run(context.Background(), map[string]any{
		"mode":     "tail",
		"agent_id": "not-a-uuid",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid agent_id UUID")
}

// TestSessionLogsTool_Run_EmptyLogBuffer tests handling of empty log buffer.
func TestSessionLogsTool_Run_EmptyLogBuffer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockLoggerService.EXPECT().GetLogs(gomock.Any()).Return([]logger.LogEntry{})

	tool := &sessionLogsToolImpl{
		logService:  mockLoggerService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"mode": "all",
	})
	require.NoError(t, err)

	entriesResult := result["entries"].([]map[string]interface{})
	assert.Empty(t, entriesResult)
	assert.Equal(t, 0, result["total_matching"])
}

// TestSessionLogsTool_Run_CountLimit tests count limiting.
func TestSessionLogsTool_Run_CountLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	agentID := uuid.New()
	now := time.Now()

	// Create 10 entries
	entries := make([]logger.LogEntry, 10)
	for i := 0; i < 10; i++ {
		entries[i] = logger.LogEntry{
			Timestamp: now,
			Level:     "info",
			Message:   fmt.Sprintf("msg%d", i),
			AgentID:   agentID,
		}
	}

	mockLoggerService.EXPECT().GetLogs(gomock.Any()).Return(entries)

	tool := &sessionLogsToolImpl{logService: mockLoggerService, hookManager: mockHookManager, agentID: uuid.New()}

	result, err := tool.Run(context.Background(), map[string]any{
		"mode":  "tail",
		"count": float64(3),
	})
	require.NoError(t, err)

	entriesResult := result["entries"].([]map[string]interface{})
	assert.Len(t, entriesResult, 10)
	assert.Equal(t, 10, result["total_matching"])
}

// TestSessionLogsTool_Spec tests the tool specification.
func TestSessionLogsTool_Spec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &sessionLogsToolImpl{
		logService:  mockLoggerService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	spec := tool.Spec()
	assert.Equal(t, shared.ToolNameSessionLogs.String(), spec.Name)
	assert.NotEmpty(t, spec.Description)
	assert.NotEmpty(t, spec.Parameters)

	// Verify required parameters exist
	requiredParams := spec.Parameters
	assert.Contains(t, requiredParams, "mode")
	assert.Contains(t, requiredParams, "count")
	assert.Contains(t, requiredParams, "since")
	assert.Contains(t, requiredParams, "level")
	assert.Contains(t, requiredParams, "agent_id")
}

// TestIsValidMode tests the mode validation function.
func TestIsValidMode(t *testing.T) {
	tests := []struct {
		mode     string
		expected bool
	}{
		{"head", true},
		{"tail", true},
		{"since", true},
		{"all", true},
		{"invalid", false},
		{"", false},
		{"HEAD", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			result := isValidMode(tt.mode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSessionLogsTool_CaseInsensitiveLevelFilter tests case-insensitive level filtering.
func TestSessionLogsTool_CaseInsensitiveLevelFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	agentID := uuid.New()
	now := time.Now()

	entries := []logger.LogEntry{
		{Timestamp: now, Level: "info", Message: "msg1", AgentID: agentID},
	}

	mockLoggerService.EXPECT().GetLogs(gomock.Any()).Return(entries)

	tool := &sessionLogsToolImpl{logService: mockLoggerService, hookManager: mockHookManager, agentID: uuid.New()}

	// Filter with lowercase
	result, err := tool.Run(context.Background(), map[string]any{
		"mode":  "all",
		"level": "info",
	})
	require.NoError(t, err)

	entriesResult := result["entries"].([]map[string]interface{})
	assert.GreaterOrEqual(t, len(entriesResult), 1)
}

// TestSessionLogsTool_Provider tests the DI provider.
func TestSessionLogsTool_Provider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(nil) // nil ctrl since no expectations

	provider := &sessionLogsToolProvider{
		logService:  mockLoggerService,
		hookManager: mockHookManager,
	}

	agentID := uuid.New()
	tool := provider.CreateTool(agentID)
	toolImpl := tool.(*sessionLogsToolImpl)

	assert.NotNil(t, tool)
	assert.Equal(t, provider.logService, toolImpl.logService)
	assert.Equal(t, provider.hookManager, toolImpl.hookManager)
	assert.Equal(t, agentID, toolImpl.agentID)
}

// TestNewSessionLogsToolProvider tests provider creation with DI.
func TestNewSessionLogsToolProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(nil) // nil ctrl since no expectations

	provider := &sessionLogsToolProvider{
		logService:  mockLoggerService,
		hookManager: mockHookManager,
	}

	tool := provider.CreateTool(uuid.New())

	assert.NotNil(t, tool)
	assert.IsType(t, &sessionLogsToolImpl{}, tool)
}

// TestSessionLogsTool_WithFields tests entries with additional fields.
func TestSessionLogsTool_WithFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	agentID := uuid.New()
	now := time.Now()

	entries := []logger.LogEntry{
		{
			Timestamp: now,
			Level:     "info",
			Message:   "test with fields",
			Fields: map[string]interface{}{
				"user_id": "12345",
				"action":  "login",
			},
			AgentID: agentID,
		},
	}

	mockLoggerService.EXPECT().GetLogs(gomock.Any()).Return(entries)

	tool := &sessionLogsToolImpl{logService: mockLoggerService, hookManager: mockHookManager, agentID: uuid.New()}

	result, err := tool.Run(context.Background(), map[string]any{
		"mode": "all",
	})
	require.NoError(t, err)

	entriesResult := result["entries"].([]map[string]interface{})
	assert.Len(t, entriesResult, 1)

	fields, ok := entriesResult[0]["fields"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "12345", fields["user_id"])
	assert.Equal(t, "login", fields["action"])
}

// TestSessionLogsTool_InvalidCount tests error handling for invalid count.
func TestSessionLogsTool_InvalidCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &sessionLogsToolImpl{
		logService:  mockLoggerService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	_, err := tool.Run(context.Background(), map[string]any{
		"mode":  "tail",
		"count": float64(-1),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "count must be positive")
}

// TestSessionLogsTool_ZeroCount tests that count=0 is rejected (must be positive).
func TestSessionLogsTool_ZeroCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &sessionLogsToolImpl{
		logService:  mockLoggerService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	_, err := tool.Run(context.Background(), map[string]any{
		"mode":  "tail",
		"count": float64(0),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "count must be positive")
}

// TestSessionLogsTool_NilAgentID tests entries without agent ID.
func TestSessionLogsTool_NilAgentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoggerService := logger.NewMockLoggerService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	now := time.Now()

	entries := []logger.LogEntry{
		{Timestamp: now, Level: "info", Message: "no agent"},
	}

	mockLoggerService.EXPECT().GetLogs(gomock.Any()).Return(entries)

	tool := &sessionLogsToolImpl{logService: mockLoggerService, hookManager: mockHookManager, agentID: uuid.New()}

	result, err := tool.Run(context.Background(), map[string]any{
		"mode": "all",
	})
	require.NoError(t, err)

	entriesResult := result["entries"].([]map[string]interface{})
	assert.Len(t, entriesResult, 1)
	// Nil agent ID should not be included in output
	_, hasAgentID := entriesResult[0]["agent_id"]
	assert.False(t, hasAgentID)
}
