package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAgentOutputTool_Spec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    uuid.New(),
	}

	spec := tool.Spec()

	// Verify spec structure
	assert.Equal(t, shared.ToolNameAgentOutput, spec.Name)
	assert.NotEmpty(t, spec.Description)

	// Verify required parameters

	// Verify agent_id parameter
	assert.NotNil(t, spec.Parameters["agent_id"])
	assert.Equal(t, gollem.TypeString, spec.Parameters["agent_id"].Type)

	// Verify block parameter
	assert.NotNil(t, spec.Parameters["block"])
	assert.Equal(t, gollem.TypeBoolean, spec.Parameters["block"].Type)

	// Verify timeout parameter
	assert.NotNil(t, spec.Parameters["timeout"])
	assert.Equal(t, gollem.TypeInteger, spec.Parameters["timeout"].Type)
}

func TestAgentOutputTool_Run_MissingAgentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    uuid.New(),
	}

	result, err := tool.Run(context.Background(), map[string]any{})

	// Tool returns nil error with error response
	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "agent_id is required")
}

func TestAgentOutputTool_Run_EmptyAgentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    uuid.New(),
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": "",
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "agent_id is required")
}

func TestAgentOutputTool_Run_InvalidUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    uuid.New(),
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": "not-a-uuid",
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "agent_id must be a valid UUID")
}

func TestAgentOutputTool_Run_AgentNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(true)

	mockRegistry.EXPECT().
		GetAgentResult(agentID).
		Return(nil, false)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "not found")
}

func TestAgentOutputTool_Run_NonBlockingMode_Running(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()
	startedAt := time.Now().Unix()

	taskResult := &shared.AgentResult{
		AgentID:   agentID,
		Status:    shared.AgentStatusRunning,
		StartedAt: startedAt,
	}

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(true)

	mockRegistry.EXPECT().
		GetAgentResult(agentID).
		Return(taskResult, true)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
		"block":    false,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, agentID.String(), result["agent_id"].(string))
	assert.Equal(t, string(shared.AgentStatusRunning), result["status"].(string))
	assert.Equal(t, startedAt, result["started_at"].(int64))
	// No output or error for running agents in non-blocking mode
	assert.NotContains(t, result, "output")
	assert.NotContains(t, result, "error")
}

func TestAgentOutputTool_Run_NonBlockingMode_Completed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()
	startedAt := time.Now().Add(-1 * time.Hour).Unix()
	completedAt := time.Now().Unix()

	output := map[string]interface{}{
		"response": "Task completed successfully",
		"code":     200,
	}

	taskResult := &shared.AgentResult{
		AgentID:     agentID,
		Status:      shared.AgentStatusCompleted,
		Output:      output,
		StartedAt:   startedAt,
		CompletedAt: &completedAt,
	}

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(true)

	mockRegistry.EXPECT().
		GetAgentResult(agentID).
		Return(taskResult, true)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
		"block":    false,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, agentID.String(), result["agent_id"].(string))
	assert.Equal(t, string(shared.AgentStatusCompleted), result["status"].(string))
	assert.Equal(t, startedAt, result["started_at"].(int64))
	assert.Equal(t, completedAt, result["completed_at"].(int64))
	assert.Equal(t, output, result["output"])
}

func TestAgentOutputTool_Run_NonBlockingMode_Failed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()
	startedAt := time.Now().Add(-1 * time.Hour).Unix()
	completedAt := time.Now().Unix()

	taskResult := &shared.AgentResult{
		AgentID:     agentID,
		Status:      shared.AgentStatusFailed,
		Error:       "Something went wrong",
		StartedAt:   startedAt,
		CompletedAt: &completedAt,
	}

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(true)

	mockRegistry.EXPECT().
		GetAgentResult(agentID).
		Return(taskResult, true)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
		"block":    false,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, agentID.String(), result["agent_id"].(string))
	assert.Equal(t, string(shared.AgentStatusFailed), result["status"].(string))
	assert.Equal(t, startedAt, result["started_at"].(int64))
	assert.Equal(t, completedAt, result["completed_at"].(int64))
	assert.Equal(t, "Something went wrong", result["error"])
}

func TestAgentOutputTool_Run_BlockingMode_Completed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()
	startedAt := time.Now().Add(-1 * time.Hour).Unix()
	completedAt := time.Now().Unix()

	output := map[string]interface{}{
		"result": "All done",
	}

	taskResult := &shared.AgentResult{
		AgentID:     agentID,
		Status:      shared.AgentStatusCompleted,
		Output:      output,
		StartedAt:   startedAt,
		CompletedAt: &completedAt,
	}

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(true)

	mockRegistry.EXPECT().
		GetAgentResult(agentID).
		Return(taskResult, true)

	mockRegistry.EXPECT().
		WaitForAgent(gomock.Any(), agentID, 30000*time.Millisecond).
		Return(taskResult, nil)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
		"block":    true,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, agentID.String(), result["agent_id"].(string))
	assert.Equal(t, string(shared.AgentStatusCompleted), result["status"].(string))
	assert.Equal(t, startedAt, result["started_at"].(int64))
	assert.Equal(t, completedAt, result["completed_at"].(int64))
	assert.Equal(t, output, result["output"])
}

func TestAgentOutputTool_Run_BlockingMode_Timeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()
	startedAt := time.Now().Unix()

	taskResult := &shared.AgentResult{
		AgentID:   agentID,
		Status:    shared.AgentStatusRunning,
		StartedAt: startedAt,
	}

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(true)

	mockRegistry.EXPECT().
		GetAgentResult(agentID).
		Return(taskResult, true)

	mockRegistry.EXPECT().
		WaitForAgent(gomock.Any(), agentID, 30000*time.Millisecond).
		Return(taskResult, errors.New("timeout waiting for agent"))

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
		"block":    true,
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Equal(t, agentID.String(), result["agent_id"].(string))
	assert.Equal(t, string(shared.AgentStatusRunning), result["status"].(string))
	assert.Equal(t, startedAt, result["started_at"].(int64))
	assert.Contains(t, result["error"].(string), "timeout")
}

func TestAgentOutputTool_Run_BlockingMode_Failed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()
	startedAt := time.Now().Add(-1 * time.Hour).Unix()
	completedAt := time.Now().Unix()

	taskResult := &shared.AgentResult{
		AgentID:     agentID,
		Status:      shared.AgentStatusFailed,
		Error:       "Execution failed",
		StartedAt:   startedAt,
		CompletedAt: &completedAt,
	}

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(true)

	mockRegistry.EXPECT().
		GetAgentResult(agentID).
		Return(taskResult, true)

	mockRegistry.EXPECT().
		WaitForAgent(gomock.Any(), agentID, 30000*time.Millisecond).
		Return(taskResult, nil)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
		"block":    true,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, agentID.String(), result["agent_id"].(string))
	assert.Equal(t, string(shared.AgentStatusFailed), result["status"].(string))
	assert.Equal(t, startedAt, result["started_at"].(int64))
	assert.Equal(t, completedAt, result["completed_at"].(int64))
	assert.Equal(t, "Execution failed", result["error"])
}

func TestAgentOutputTool_Run_CustomTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()
	startedAt := time.Now().Unix()

	taskResult := &shared.AgentResult{
		AgentID:   agentID,
		Status:    shared.AgentStatusRunning,
		StartedAt: startedAt,
	}

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(true)

	mockRegistry.EXPECT().
		GetAgentResult(agentID).
		Return(taskResult, true)

	mockRegistry.EXPECT().
		WaitForAgent(gomock.Any(), agentID, 5000*time.Millisecond).
		Return(taskResult, nil)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
		"block":    true,
		"timeout":  5000,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
}

func TestAgentOutputTool_Run_TimeoutClamping_Int(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()

	tests := []struct {
		name            string
		inputTimeout    int
		expectedTimeout time.Duration
	}{
		{"Too low", 0, 1 * time.Millisecond},
		{"Negative", -100, 1 * time.Millisecond},
		{"Too high", 700000, 600000 * time.Millisecond},
		{"Valid", 30000, 30000 * time.Millisecond},
		{"Min valid", 1, 1 * time.Millisecond},
		{"Max valid", 600000, 600000 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskResult := &shared.AgentResult{
				AgentID:   agentID,
				Status:    shared.AgentStatusRunning,
				StartedAt: time.Now().Unix(),
			}

			// Permission check: sender is direct parent
			mockRegistry.EXPECT().
				IsDirectParent(senderID, agentID).
				Return(true)

			mockRegistry.EXPECT().
				GetAgentResult(agentID).
				Return(taskResult, true)

			mockRegistry.EXPECT().
				WaitForAgent(gomock.Any(), agentID, tt.expectedTimeout).
				Return(taskResult, nil)

			tool := &AgentOutputTool{
				hookManager: mockHookManager,
				registry:    mockRegistry,
				senderID:    senderID,
			}

			result, err := tool.Run(context.Background(), map[string]any{
				"agent_id": agentID.String(),
				"block":    true,
				"timeout":  tt.inputTimeout,
			})

			require.NoError(t, err)
			assert.True(t, result["success"].(bool))
			ctrl.Finish()
		})
	}
}

func TestAgentOutputTool_Run_TimeoutClamping_Float(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()

	tests := []struct {
		name            string
		inputTimeout    float64
		expectedTimeout time.Duration
	}{
		{"Too low", 0.5, 1 * time.Millisecond},
		{"Valid float", 30000.0, 30000 * time.Millisecond},
		{"Too high", 700000.5, 600000 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskResult := &shared.AgentResult{
				AgentID:   agentID,
				Status:    shared.AgentStatusRunning,
				StartedAt: time.Now().Unix(),
			}

			// Permission check: sender is direct parent
			mockRegistry.EXPECT().
				IsDirectParent(senderID, agentID).
				Return(true)

			mockRegistry.EXPECT().
				GetAgentResult(agentID).
				Return(taskResult, true)

			mockRegistry.EXPECT().
				WaitForAgent(gomock.Any(), agentID, tt.expectedTimeout).
				Return(taskResult, nil)

			tool := &AgentOutputTool{
				hookManager: mockHookManager,
				registry:    mockRegistry,
				senderID:    senderID,
			}

			result, err := tool.Run(context.Background(), map[string]any{
				"agent_id": agentID.String(),
				"block":    true,
				"timeout":  tt.inputTimeout,
			})

			require.NoError(t, err)
			assert.True(t, result["success"].(bool))
			ctrl.Finish()
		})
	}
}

func TestAgentOutputTool_Run_DefaultBlockValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()
	startedAt := time.Now().Unix()

	taskResult := &shared.AgentResult{
		AgentID:   agentID,
		Status:    shared.AgentStatusRunning,
		StartedAt: startedAt,
	}

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(true)

	// When block is not specified, it should default to true (blocking mode)
	mockRegistry.EXPECT().
		GetAgentResult(agentID).
		Return(taskResult, true)

	mockRegistry.EXPECT().
		WaitForAgent(gomock.Any(), agentID, 30000*time.Millisecond).
		Return(taskResult, nil)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	// No "block" parameter provided - should default to blocking mode
	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
}

func TestAgentOutputToolProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	senderID := uuid.New()

	provider := &agentOutputToolProvider{
		hookManager: mockHookManager,
		registry:    mockRegistry,
	}

	tool := provider.CreateTool(senderID)

	assert.NotNil(t, tool)
	assert.Equal(t, mockRegistry, tool.registry)
	assert.Equal(t, mockHookManager, tool.hookManager)
	assert.Equal(t, senderID, tool.senderID)
}

// TestAgentOutputTool_Run_PermissionDenied tests permission check when caller is not direct parent
func TestAgentOutputTool_Run_PermissionDenied(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()

	// Permission check: sender is NOT direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(false)

	tool := &AgentOutputTool{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "permission denied")
	assert.Contains(t, result["error"].(string), "direct subagents")
	assert.Contains(t, result["error"].(string), "not your direct child")
}
