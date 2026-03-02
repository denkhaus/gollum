package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestBackgroundAgent_SyncExecution tests the complete synchronous execution flow
func TestBackgroundAgent_SyncExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	// Use the same registry from the injector (execution helper uses this one)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	// Setup mocks
	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	mockConfigService := setupMockConfigService(ctrl)

	// Use real execution helper for integration tests
	execHelper := do.MustInvoke[AgentExecutionHelper](injector)

	senderID := uuid.New()
	var spawnedAgentID uuid.UUID

	// Create mock agent - ID will be determined at spawn time
	mockAgent := mocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().DoAndReturn(func() uuid.UUID {
		return spawnedAgentID
	}).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMProvider:  shared.LLMProviderAnthropic,
		Role:         "Tester",
		SystemPrompt: "test",
	}).AnyTimes()

	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        senderID,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentTaskPrompt("Tester", "Test task").Return("You are a helpful assistant", nil)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
		spawnedAgentID = config.ID
		return mockAgent, nil
	})

	// Expect agent execution - the real execution helper will call this
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
		Texts: []string{"Task completed successfully"},
	}, nil)

	// Execute spawn agent
	result, err := tool.Run(ctx, map[string]any{
		"role":              "Tester",
		"description":       "Test task",
		"prompt":            "Do something",
		"run_in_background": false,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, spawnedAgentID.String(), result["agent_id"].(string))
	assert.Equal(t, "completed", result["status"].(string))
	assert.Equal(t, "Task completed successfully", result["response"].(string))

	// Verify agent result was stored
	storedResult, exists := agentRegistry.GetAgentResult(spawnedAgentID)
	require.True(t, exists)
	assert.Equal(t, shared.AgentStatusCompleted, storedResult.Status)
	assert.Equal(t, "Task completed successfully", storedResult.Output["response"])

	// Create agent output tool for result retrieval
	outputTool := &AgentOutputTool{
		registry:    agentRegistry,
		senderID:    senderID,
		hookManager: mockHookManager,
	}

	// Test AgentOutputTool in non-blocking mode
	outputResult, err := outputTool.Run(ctx, map[string]any{
		"agent_id": spawnedAgentID.String(),
		"block":    false,
	})

	require.NoError(t, err)
	assert.True(t, outputResult["success"].(bool), "Expected success to be true, got error: %v", outputResult["error"])
	assert.Equal(t, string(shared.AgentStatusCompleted), outputResult["status"].(string))
	assert.NotNil(t, outputResult["output"])
}

// TestBackgroundAgent_SyncExecutionError tests error scenarios in sync execution
func TestBackgroundAgent_SyncExecutionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	// Use the same registry from the injector (execution helper uses this one)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	// Setup mocks
	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	mockConfigService := setupMockConfigService(ctrl)

	// Use real execution helper for integration tests
	execHelper := do.MustInvoke[AgentExecutionHelper](injector)

	senderID := uuid.New()
	var spawnedAgentID uuid.UUID

	// Create mock agent - ID will be determined at spawn time
	mockAgent := mocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().DoAndReturn(func() uuid.UUID {
		return spawnedAgentID
	}).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMProvider:  shared.LLMProviderAnthropic,
		Role:         "Failing Agent",
		SystemPrompt: "test",
	}).AnyTimes()

	// Create spawn tool
	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        senderID,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentTaskPrompt("Failing Agent", "Failing task").Return("You are a helpful assistant", nil)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
		spawnedAgentID = config.ID
		return mockAgent, nil
	})

	// Agent execution will fail
	testError := errors.New("execution failed")
	mockAgent.EXPECT().Execute(ctx, gomock.Any()).Return(nil, testError)

	// Execute spawn agent - should handle error gracefully
	result, err := tool.Run(ctx, map[string]any{
		"role":              "Failing Agent",
		"description":       "Failing task",
		"prompt":            "Fail please",
		"run_in_background": false,
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "execution failed")

	// Verify agent result was stored with failed status
	storedResult, exists := agentRegistry.GetAgentResult(spawnedAgentID)
	require.True(t, exists)
	assert.Equal(t, shared.AgentStatusFailed, storedResult.Status)
	assert.Contains(t, storedResult.Error, "execution failed")
}
