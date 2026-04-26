package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/prompt/manager"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestBackgroundAgent_AsyncExecution tests the complete asynchronous execution flow
func TestBackgroundAgent_AsyncExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	// Use the same registry from the injector (execution helper uses this one)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	// Setup mocks
	mockFactory := shared.NewMockAgentFactory(ctrl)
	mockPromptMgr := manager.NewMockPromptManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	mockConfigService := setupMockConfigService(ctrl)

	// Use real execution helper for integration tests
	execHelper := do.MustInvoke[AgentExecutionHelper](injector)

	senderID := uuid.New()
	var spawnedAgentID uuid.UUID

	// Create mock sender agent
	mockSenderAgent := shared.NewMockAgent(ctrl)
	mockSenderAgent.EXPECT().GetID().Return(senderID).AnyTimes()
	mockSenderAgent.EXPECT().ToSessionContext().Return(shared.SessionContext{
		AgentID: senderID,
	}).AnyTimes()
	mockSenderAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
		Role:         "Parent",
		SystemPrompt: "parent",
	}).AnyTimes()

	// Create mock agent - ID will be determined at spawn time
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().DoAndReturn(func() uuid.UUID {
		return spawnedAgentID
	}).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
		Role:         "Async Tester",
		SystemPrompt: "test",
	}).AnyTimes()
	mockAgent.EXPECT().ToSessionContext().DoAndReturn(func() shared.SessionContext {
		return *shared.NewSessionContext(uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), spawnedAgentID, uuid.Nil, "")
	}).AnyTimes()
	mockAgent.EXPECT().GetMessageHistory(gomock.Any()).Return(nil, nil).AnyTimes()

	// Create spawn tool
	tool := &spawnAgentToolImpl{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		agent:           mockSenderAgent,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentTaskPrompt("Async Tester", "Async test task").Return("You are a helpful assistant", nil)
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
	outputTool := &agentOutputToolImpl{
		registry:    agentRegistry,
		agent:       mockSenderAgent,
		hookManager: mockHookManager,
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
	mockFactory := shared.NewMockAgentFactory(ctrl)
	mockPromptMgr := manager.NewMockPromptManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	mockConfigService := setupMockConfigService(ctrl)

	// Use real execution helper for integration tests
	execHelper := do.MustInvoke[AgentExecutionHelper](injector)

	senderID := uuid.New()
	var spawnedAgentID uuid.UUID

	// Create mock sender agent
	mockSenderAgent := shared.NewMockAgent(ctrl)
	mockSenderAgent.EXPECT().GetID().Return(senderID).AnyTimes()
	mockSenderAgent.EXPECT().ToSessionContext().Return(shared.SessionContext{
		AgentID: senderID,
	}).AnyTimes()
	mockSenderAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
		Role:         "Parent",
		SystemPrompt: "parent",
	}).AnyTimes()

	// Create mock agent - ID will be determined at spawn time
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().DoAndReturn(func() uuid.UUID {
		return spawnedAgentID
	}).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
		Role:         "Slow Agent",
		SystemPrompt: "test",
	}).AnyTimes()
	mockAgent.EXPECT().ToSessionContext().DoAndReturn(func() shared.SessionContext {
		return *shared.NewSessionContext(uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), spawnedAgentID, uuid.Nil, "")
	}).AnyTimes()
	mockAgent.EXPECT().GetMessageHistory(gomock.Any()).Return(nil, nil).AnyTimes()

	// Create spawn tool
	tool := &spawnAgentToolImpl{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		agent:           mockSenderAgent,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentTaskPrompt("Slow Agent", "Slow task").Return("You are a helpful assistant", nil)
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
	outputTool := &agentOutputToolImpl{
		registry:    agentRegistry,
		agent:       mockSenderAgent,
		hookManager: mockHookManager,
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

// TestBackgroundAgent_AsyncExecutionError tests async execution with error
func TestBackgroundAgent_AsyncExecutionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	// Use the same registry from the injector (execution helper uses this one)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	// Setup mocks
	mockFactory := shared.NewMockAgentFactory(ctrl)
	mockPromptMgr := manager.NewMockPromptManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	mockConfigService := setupMockConfigService(ctrl)

	// Use real execution helper for integration tests
	execHelper := do.MustInvoke[AgentExecutionHelper](injector)

	senderID := uuid.New()
	var spawnedAgentID uuid.UUID

	// Create mock sender agent
	mockSenderAgent := shared.NewMockAgent(ctrl)
	mockSenderAgent.EXPECT().GetID().Return(senderID).AnyTimes()
	mockSenderAgent.EXPECT().ToSessionContext().Return(shared.SessionContext{
		AgentID: senderID,
	}).AnyTimes()
	mockSenderAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
		Role:         "Parent",
		SystemPrompt: "parent",
	}).AnyTimes()

	// Create mock agent - ID will be determined at spawn time
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().DoAndReturn(func() uuid.UUID {
		return spawnedAgentID
	}).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
		Role:         "Failing Async Agent",
		SystemPrompt: "test",
	}).AnyTimes()
	mockAgent.EXPECT().ToSessionContext().DoAndReturn(func() shared.SessionContext {
		return *shared.NewSessionContext(uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), spawnedAgentID, uuid.Nil, "")
	}).AnyTimes()
	mockAgent.EXPECT().GetMessageHistory(gomock.Any()).Return(nil, nil).AnyTimes()

	// Create spawn tool
	tool := &spawnAgentToolImpl{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		agent:           mockSenderAgent,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentTaskPrompt("Failing Async Agent", "Failing async task").Return("You are a helpful assistant", nil)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
		spawnedAgentID = config.ID
		return mockAgent, nil
	})

	// Agent execution will fail in background
	testError := errors.New("async execution failed")
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil, testError)

	// Create agent output tool
	outputTool := &agentOutputToolImpl{
		registry:    agentRegistry,
		agent:       mockSenderAgent,
		hookManager: mockHookManager,
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

// TestBackgroundAgent_NonBlockingStatusChecks tests status polling with non-blocking mode
func TestBackgroundAgent_NonBlockingStatusChecks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	// Use the same registry from the injector (execution helper uses this one)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)

	// Setup mocks
	mockFactory := shared.NewMockAgentFactory(ctrl)
	mockPromptMgr := manager.NewMockPromptManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	mockConfigService := setupMockConfigService(ctrl)

	// Use real execution helper for integration tests
	execHelper := do.MustInvoke[AgentExecutionHelper](injector)

	senderID := uuid.New()
	var spawnedAgentID uuid.UUID

	// Create mock sender agent
	mockSenderAgent := shared.NewMockAgent(ctrl)
	mockSenderAgent.EXPECT().GetID().Return(senderID).AnyTimes()
	mockSenderAgent.EXPECT().ToSessionContext().Return(shared.SessionContext{
		AgentID: senderID,
	}).AnyTimes()
	mockSenderAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
		Role:         "Parent",
		SystemPrompt: "parent",
	}).AnyTimes()

	// Create mock agent - ID will be determined at spawn time
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().DoAndReturn(func() uuid.UUID {
		return spawnedAgentID
	}).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
		Role:         "Polled Agent",
		SystemPrompt: "test",
	}).AnyTimes()

	// Create spawn tool
	tool := &spawnAgentToolImpl{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        agentRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: execHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		agent:           mockSenderAgent,
	}

	ctx := context.Background()

	// Expect factory call - capture the config to get the agent ID
	mockPromptMgr.EXPECT().GetSubagentTaskPrompt("Polled Agent", "Polled task").Return("You are a helpful assistant", nil)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, config *shared.AgentConfig) (shared.Agent, error) {
		spawnedAgentID = config.ID
		return mockAgent, nil

	})
	mockAgent.EXPECT().ToSessionContext().DoAndReturn(func() shared.SessionContext {
		return *shared.NewSessionContext(uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), spawnedAgentID, uuid.Nil, "")
	}).AnyTimes()
	mockAgent.EXPECT().GetMessageHistory(gomock.Any()).Return(nil, nil).AnyTimes()
	// Agent will complete after delay
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
		Texts: []string{"Done after delay"},
	}, nil).Do(func(_ context.Context, _ gollem.Input) (*gollem.ExecuteResponse, error) {
		time.Sleep(200 * time.Millisecond)
		return &gollem.ExecuteResponse{Texts: []string{"Done after delay"}}, nil
	})

	// Create agent output tool
	outputTool := &agentOutputToolImpl{
		registry:    agentRegistry,
		agent:       mockSenderAgent,
		hookManager: mockHookManager,
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
