package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

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

	tool := &agentOutputToolImpl{
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

	tool := &agentOutputToolImpl{
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

	tool := &agentOutputToolImpl{
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

	tool := &agentOutputToolImpl{
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

	tool := &agentOutputToolImpl{
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

	tool := &agentOutputToolImpl{
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

	tool := &agentOutputToolImpl{
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
