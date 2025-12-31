package tools

import (
	"context"
	"errors"
	"testing"
	"time"

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

// Use local package reference for AgentExecutionHelper
type toolsPkg interface{}

var _ toolsPkg = (*AgentExecutionHelper)(nil)

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
		senderID:        senderID,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentPrompt("Tester", "Test task").Return("You are a helpful assistant", nil)
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
		registry: agentRegistry,
		senderID: senderID,
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

// TestBackgroundAgent_AsyncExecution tests the complete asynchronous execution flow
func TestBackgroundAgent_AsyncExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	// Use the same registry from the injector (execution helper uses this one)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	// Setup mocks
	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)

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
		Role:         "Async Tester",
		SystemPrompt: "test",
	}).AnyTimes()

	// Create spawn tool
	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		senderID:        senderID,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentPrompt("Async Tester", "Async test task").Return("You are a helpful assistant", nil)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
		spawnedAgentID = config.ID
		return mockAgent, nil
	})

	// Expect agent execution - will run in background
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
		Texts: []string{"Async task completed"},
	}, nil).Do(func(_ context.Context, _ gollem.Input) (*gollem.ExecuteResponse, error) {
		// Simulate async work
		time.Sleep(50 * time.Millisecond)
		return &gollem.ExecuteResponse{Texts: []string{"Async task completed"}}, nil
	})

	// Create agent output tool for result retrieval
	outputTool := &AgentOutputTool{
		registry: agentRegistry,
		senderID: senderID,
	}

	// Execute spawn agent in background
	result, err := tool.Run(ctx, map[string]any{
		"role":              "Async Tester",
		"description":       "Async test task",
		"prompt":            "Do something async",
		"run_in_background": true,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, spawnedAgentID.String(), result["agent_id"].(string))
	assert.Equal(t, "running", result["status"].(string))

	// Verify initial agent result was stored with running status
	storedResult, exists := agentRegistry.GetAgentResult(spawnedAgentID)
	require.True(t, exists)
	assert.Equal(t, shared.AgentStatusRunning, storedResult.Status)

	// Test AgentOutputTool in blocking mode - wait for completion
	outputResult, err := outputTool.Run(ctx, map[string]any{
		"agent_id": spawnedAgentID.String(),
		"block":    true,
		"timeout":  5000, // 5 seconds
	})

	require.NoError(t, err)
	assert.True(t, outputResult["success"].(bool))
	assert.Equal(t, string(shared.AgentStatusCompleted), outputResult["status"].(string))
	assert.Equal(t, "Async task completed", outputResult["output"].(map[string]interface{})["response"])
}

// TestBackgroundAgent_AsyncExecutionTimeout tests timeout handling for async execution
func TestBackgroundAgent_AsyncExecutionTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	// Use the same registry from the injector (execution helper uses this one)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	// Setup mocks
	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)

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
		Role:         "Slow Agent",
		SystemPrompt: "test",
	}).AnyTimes()

	// Create spawn tool
	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		senderID:        senderID,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentPrompt("Slow Agent", "Slow task").Return("You are a helpful assistant", nil)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
		spawnedAgentID = config.ID
		return mockAgent, nil
	})

	// Agent execution will take a long time (simulate slow task)
	done := make(chan bool)
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
		Texts: []string{"Finally done"},
	}, nil).Do(func(_ context.Context, _ gollem.Input) (*gollem.ExecuteResponse, error) {
		time.Sleep(500 * time.Millisecond) // Will exceed our timeout
		done <- true
		return &gollem.ExecuteResponse{Texts: []string{"Finally done"}}, nil
	})

	// Spawn agent in background
	result, err := tool.Run(ctx, map[string]any{
		"role":              "Slow Agent",
		"description":       "Slow task",
		"prompt":            "Take your time",
		"run_in_background": true,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, spawnedAgentID.String(), result["agent_id"].(string))
	assert.Equal(t, "running", result["status"].(string))

	// Create agent output tool
	outputTool := &AgentOutputTool{
		registry: agentRegistry,
		senderID: senderID,
	}

	// Wait with short timeout - should timeout before completion
	outputResult, err := outputTool.Run(ctx, map[string]any{
		"agent_id": spawnedAgentID.String(),
		"block":    true,
		"timeout":  100, // 100ms
	})

	require.NoError(t, err)
	assert.False(t, outputResult["success"].(bool))
	assert.Contains(t, outputResult["error"].(string), "timeout")

	// Wait for background goroutine to complete
	<-done

	// Verify agent is still running in registry
	storedResult, exists := agentRegistry.GetAgentResult(spawnedAgentID)
	require.True(t, exists)
	// Status should still be running or completed (depending on timing)
	assert.Contains(t, []shared.AgentStatus{shared.AgentStatusRunning, shared.AgentStatusCompleted}, storedResult.Status)
}

// TestBackgroundAgent_ErrorHandling tests error scenarios
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
		senderID:        senderID,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentPrompt("Failing Agent", "Failing task").Return("You are a helpful assistant", nil)
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

// TestBackgroundAgent_AsyncExecutionError tests async execution with error
func TestBackgroundAgent_AsyncExecutionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	// Use the same registry from the injector (execution helper uses this one)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	// Setup mocks
	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)

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
		Role:         "Failing Async Agent",
		SystemPrompt: "test",
	}).AnyTimes()

	// Create spawn tool
	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		senderID:        senderID,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentPrompt("Failing Async Agent", "Failing async task").Return("You are a helpful assistant", nil)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
		spawnedAgentID = config.ID
		return mockAgent, nil
	})

	// Agent execution will fail in background
	testError := errors.New("async execution failed")
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil, testError)

	// Create agent output tool
	outputTool := &AgentOutputTool{
		registry: agentRegistry,
		senderID: senderID,
	}

	// Spawn agent in background
	result, err := tool.Run(ctx, map[string]any{
		"role":              "Failing Async Agent",
		"description":       "Failing async task",
		"prompt":            "Fail async please",
		"run_in_background": true,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, spawnedAgentID.String(), result["agent_id"].(string))
	assert.Equal(t, "running", result["status"].(string))

	// Wait a bit for background execution
	time.Sleep(100 * time.Millisecond)

	// Check result via AgentOutputTool
	outputResult, err := outputTool.Run(ctx, map[string]any{
		"agent_id": spawnedAgentID.String(),
		"block":    false,
	})

	require.NoError(t, err)
	assert.True(t, outputResult["success"].(bool))
	assert.Equal(t, string(shared.AgentStatusFailed), outputResult["status"].(string))
	assert.Contains(t, outputResult["error"].(string), "async execution failed")
}

// TestBackgroundAgent_ConcurrentExecution tests multiple agents running concurrently
func TestBackgroundAgent_ConcurrentExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	// Use the same registry from the injector (execution helper uses this one)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	// Setup mocks
	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)

	// Use real execution helper for integration tests
	execHelper := do.MustInvoke[AgentExecutionHelper](injector)

	senderID := uuid.New()

	// Create spawn tool
	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		senderID:        senderID,
	}

	ctx := context.Background()

	// Number of concurrent agents (limited to 3 due to sub-agent limit)
	numAgents := 3
	spawnedAgentIDs := make([]uuid.UUID, numAgents)

	// Create mock agents and spawn them concurrently
	for i := 0; i < numAgents; i++ {
		// Create a variable to capture the spawned ID for this iteration
		spawnedIndex := i
		var spawnedID uuid.UUID

		mockAgent := mocks.NewMockAgent(ctrl)
		mockAgent.EXPECT().GetID().DoAndReturn(func() uuid.UUID {
			return spawnedID
		}).AnyTimes()
		mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
			LLMProvider:  shared.LLMProviderAnthropic,
			Role:         "Concurrent Agent",
			SystemPrompt: "test",
		}).AnyTimes()

		// Expect factory call - capture the config to get the agent ID
		mockPromptMgr.EXPECT().GetSubagentPrompt("Concurrent Agent", "Concurrent task").Return("You are a helpful assistant", nil)
		mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
			spawnedID = config.ID
			spawnedAgentIDs[spawnedIndex] = spawnedID
			return mockAgent, nil
		})

		// Agent execution in background
		mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
			Texts: []string{string(rune('A' + i))},
		}, nil).Do(func(_ context.Context, _ gollem.Input) (*gollem.ExecuteResponse, error) {
			time.Sleep(50 * time.Millisecond)
			return &gollem.ExecuteResponse{Texts: []string{string(rune('A' + i))}}, nil
		})
	}

	// Spawn all agents concurrently
	results := make([]map[string]any, numAgents)
	for i := 0; i < numAgents; i++ {
		result, err := tool.Run(ctx, map[string]any{
			"role":              "Concurrent Agent",
			"description":       "Concurrent task",
			"prompt":            "Do something",
			"run_in_background": true,
		})

		require.NoError(t, err)
		results[i] = result
		assert.True(t, results[i]["success"].(bool))
		assert.Equal(t, spawnedAgentIDs[i].String(), results[i]["agent_id"].(string))
		assert.Equal(t, "running", results[i]["status"].(string))
	}

	// Create agent output tool
	outputTool := &AgentOutputTool{
		registry: agentRegistry,
		senderID: senderID,
	}

	// Wait for all agents to complete
	for i, agentID := range spawnedAgentIDs {
		outputResult, err := outputTool.Run(ctx, map[string]any{
			"agent_id": agentID.String(),
			"block":    true,
			"timeout":  5000,
		})

		require.NoError(t, err)
		assert.True(t, outputResult["success"].(bool))
		assert.Equal(t, string(shared.AgentStatusCompleted), outputResult["status"].(string))

		// Verify each agent has unique result
		expectedResult := string(rune('A' + i))
		assert.Equal(t, expectedResult, outputResult["output"].(map[string]interface{})["response"])
	}

	// Verify all agent results are in registry
	for _, agentID := range spawnedAgentIDs {
		result, exists := agentRegistry.GetAgentResult(agentID)
		require.True(t, exists, "Agent result should be stored in registry")
		assert.Equal(t, shared.AgentStatusCompleted, result.Status)
	}
}

// TestBackgroundAgent_NonBlockingStatusChecks tests status polling with non-blocking mode
func TestBackgroundAgent_NonBlockingStatusChecks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	// Use the same registry from the injector (execution helper uses this one)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	// Setup mocks
	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)

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
		Role:         "Polled Agent",
		SystemPrompt: "test",
	}).AnyTimes()

	// Create spawn tool
	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		senderID:        senderID,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentPrompt("Polled Agent", "Polled task").Return("You are a helpful assistant", nil)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
		spawnedAgentID = config.ID
		return mockAgent, nil
	})

	// Agent will complete after delay
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
		Texts: []string{"Done after delay"},
	}, nil).Do(func(_ context.Context, _ gollem.Input) (*gollem.ExecuteResponse, error) {
		time.Sleep(200 * time.Millisecond)
		return &gollem.ExecuteResponse{Texts: []string{"Done after delay"}}, nil
	})

	// Create agent output tool
	outputTool := &AgentOutputTool{
		registry: agentRegistry,
		senderID: senderID,
	}

	// Spawn agent in background
	result, err := tool.Run(ctx, map[string]any{
		"role":              "Polled Agent",
		"description":       "Polled task",
		"prompt":            "Take time",
		"run_in_background": true,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, spawnedAgentID.String(), result["agent_id"].(string))

	// Poll status in non-blocking mode - should be running initially
	outputResult, err := outputTool.Run(ctx, map[string]any{
		"agent_id": spawnedAgentID.String(),
		"block":    false,
	})

	require.NoError(t, err)
	assert.True(t, outputResult["success"].(bool))
	assert.Equal(t, string(shared.AgentStatusRunning), outputResult["status"].(string))

	// Wait for completion
	time.Sleep(300 * time.Millisecond)

	// Poll again - should be completed now
	outputResult, err = outputTool.Run(ctx, map[string]any{
		"agent_id": spawnedAgentID.String(),
		"block":    false,
	})

	require.NoError(t, err)
	assert.True(t, outputResult["success"].(bool))
	assert.Equal(t, string(shared.AgentStatusCompleted), outputResult["status"].(string))
	assert.Equal(t, "Done after delay", outputResult["output"].(map[string]interface{})["response"])
}

// TestBackgroundAgent_FullLifecycle tests the complete agent lifecycle:
// spawn → resume → output → remove
func TestBackgroundAgent_FullLifecycle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
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
		Role:         "Lifecycle Agent",
		SystemPrompt: "test",
	}).AnyTimes()

	spawnTool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		senderID:        senderID,
	}

	ctx := context.Background()

	// Step 1: Spawn the agent
	mockPromptMgr.EXPECT().GetSubagentPrompt("Lifecycle Agent", "Lifecycle task").Return("You are a helpful assistant", nil)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
		spawnedAgentID = config.ID
		return mockAgent, nil
	})

	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
		Texts: []string{"Initial task completed"},
	}, nil)

	spawnResult, err := spawnTool.Run(ctx, map[string]any{
		"role":              "Lifecycle Agent",
		"description":       "Lifecycle task",
		"prompt":            "Do initial task",
		"run_in_background": false,
	})

	require.NoError(t, err)
	assert.True(t, spawnResult["success"].(bool))
	assert.Equal(t, "completed", spawnResult["status"].(string))

	// Verify agent is in registry
	agent, exists := agentRegistry.GetAgent(spawnedAgentID)
	require.True(t, exists)
	assert.Equal(t, spawnedAgentID, agent.GetID())

	// Step 2: Resume the agent with a new task
	resumeTool := &ResumeAgentTool{
		logService:      logService,
		registry:        agentRegistry,
		executionHelper: execHelper,
		senderID:        senderID,
	}

	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
		Texts: []string{"Resumed task completed"},
	}, nil)

	resumeResult, err := resumeTool.Run(ctx, map[string]any{
		"agent_id": spawnedAgentID.String(),
		"prompt":   "Do another task",
	})

	require.NoError(t, err)
	assert.True(t, resumeResult["success"].(bool))
	assert.Equal(t, "completed", resumeResult["status"].(string))
	assert.Equal(t, "Resumed task completed", resumeResult["response"].(string))

	// Step 3: Get output from the agent
	outputTool := &AgentOutputTool{
		registry: agentRegistry,
		senderID: senderID,
	}

	outputResult, err := outputTool.Run(ctx, map[string]any{
		"agent_id": spawnedAgentID.String(),
		"block":    false,
	})

	require.NoError(t, err)
	assert.True(t, outputResult["success"].(bool))
	assert.Equal(t, string(shared.AgentStatusCompleted), outputResult["status"].(string))
	assert.Equal(t, "Resumed task completed", outputResult["output"].(map[string]interface{})["response"])

	// Step 4: Remove the agent
	removeTool := &RemoveAgentTool{
		logService: logService,
		registry:   agentRegistry,
		senderID:   senderID,
	}

	removeResult, err := removeTool.Run(ctx, map[string]any{
		"agent_id": spawnedAgentID.String(),
		"force":    true,
	})

	require.NoError(t, err)
	assert.True(t, removeResult["success"].(bool))
	assert.Equal(t, spawnedAgentID.String(), removeResult["removed_agent"].(string))

	// Verify agent is removed from registry
	_, exists = agentRegistry.GetAgent(spawnedAgentID)
	assert.False(t, exists, "Agent should be removed from registry")

	// Verify agent result is also removed
	_, exists = agentRegistry.GetAgentResult(spawnedAgentID)
	assert.False(t, exists, "Agent result should be removed from registry")
}

// TestBackgroundAgent_MultiLevelHierarchy tests spawning agents at multiple hierarchy levels:
// root → child → grandchild
func TestBackgroundAgent_MultiLevelHierarchy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	execHelper := do.MustInvoke[AgentExecutionHelper](injector)

	rootID := uuid.New()
	var childID uuid.UUID
	var grandchildID uuid.UUID

	// Create mock agents for each level
	rootAgent := mocks.NewMockAgent(ctrl)
	rootAgent.EXPECT().GetID().Return(rootID).AnyTimes()
	rootAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID:          rootID,
		LLMProvider: shared.LLMProviderAnthropic,
		Role:        "Root",
	}).AnyTimes()

	childAgent := mocks.NewMockAgent(ctrl)
	childAgent.EXPECT().GetID().DoAndReturn(func() uuid.UUID {
		return childID
	}).AnyTimes()
	childAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMProvider: shared.LLMProviderAnthropic,
		Role:        "Child",
	}).AnyTimes()

	grandchildAgent := mocks.NewMockAgent(ctrl)
	grandchildAgent.EXPECT().GetID().DoAndReturn(func() uuid.UUID {
		return grandchildID
	}).AnyTimes()
	grandchildAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMProvider: shared.LLMProviderAnthropic,
		Role:        "Grandchild",
	}).AnyTimes()

	ctx := context.Background()

	// Register root agent directly (no parent)
	rootConfig := &shared.AgentConfig{
		ID:          rootID,
		LLMProvider: shared.LLMProviderAnthropic,
		Role:        "Root",
	}
	err := agentRegistry.Register(rootAgent, rootConfig)
	require.NoError(t, err)

	// Create spawn tool for root
	rootSpawnTool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		senderID:        rootID,
	}

	// Step 1: Root spawns child
	mockPromptMgr.EXPECT().GetSubagentPrompt("Child", "Child task").Return("You are a helpful assistant", nil)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
		childID = config.ID
		return childAgent, nil
	})

	childAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
		Texts: []string{"Child task completed"},
	}, nil)

	childSpawnResult, err := rootSpawnTool.Run(ctx, map[string]any{
		"role":              "Child",
		"description":       "Child task",
		"prompt":            "Do child task",
		"run_in_background": false,
	})

	require.NoError(t, err)
	assert.True(t, childSpawnResult["success"].(bool))

	// Verify parent-child relationship
	assert.True(t, agentRegistry.IsDirectParent(rootID, childID), "Root should be direct parent of child")
	assert.False(t, agentRegistry.IsDirectParent(childID, rootID), "Child should not be parent of root")

	// Step 2: Child spawns grandchild
	childSpawnTool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		senderID:        childID,
	}

	mockPromptMgr.EXPECT().GetSubagentPrompt("Grandchild", "Grandchild task").Return("You are a helpful assistant", nil)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
		grandchildID = config.ID
		return grandchildAgent, nil
	})

	grandchildAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
		Texts: []string{"Grandchild task completed"},
	}, nil)

	grandchildSpawnResult, err := childSpawnTool.Run(ctx, map[string]any{
		"role":              "Grandchild",
		"description":       "Grandchild task",
		"prompt":            "Do grandchild task",
		"run_in_background": false,
	})

	require.NoError(t, err)
	assert.True(t, grandchildSpawnResult["success"].(bool))

	// Verify hierarchy relationships
	assert.True(t, agentRegistry.IsDirectParent(childID, grandchildID), "Child should be direct parent of grandchild")
	assert.True(t, agentRegistry.IsDirectParent(rootID, childID), "Root should be direct parent of child")

	// Root is NOT direct parent of grandchild (child is)
	assert.False(t, agentRegistry.IsDirectParent(rootID, grandchildID), "Root should NOT be direct parent of grandchild")

	// Verify permission checks: root can access child but not grandchild directly
	outputTool := &AgentOutputTool{
		registry: agentRegistry,
		senderID: rootID,
	}

	// Root can get child's output (direct parent)
	childOutput, err := outputTool.Run(ctx, map[string]any{
		"agent_id": childID.String(),
		"block":    false,
	})
	require.NoError(t, err)
	assert.True(t, childOutput["success"].(bool))

	// Root CANNOT get grandchild's output (not direct parent)
	grandchildOutput, err := outputTool.Run(ctx, map[string]any{
		"agent_id": grandchildID.String(),
		"block":    false,
	})
	require.NoError(t, err)
	assert.False(t, grandchildOutput["success"].(bool))
	assert.Contains(t, grandchildOutput["error"].(string), "permission denied")

	// Verify all three agents exist in registry
	_, exists := agentRegistry.GetAgent(rootID)
	assert.True(t, exists)
	_, exists = agentRegistry.GetAgent(childID)
	assert.True(t, exists)
	_, exists = agentRegistry.GetAgent(grandchildID)
	assert.True(t, exists)
}
