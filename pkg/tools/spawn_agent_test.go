package tools

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// setupMockConfigService creates a mock config service with default limits
func setupMockConfigService(ctrl *gomock.Controller) *mocks.MockConfigService {
	mockConfigService := mocks.NewMockConfigService(ctrl)
	mockConfigService.EXPECT().GetAgentLimits().Return(&config.AgentLimitsConfig{
		MaxSubAgentsPerParent: 3,
		MaxTotalAgents:        50,
	}).AnyTimes()
	return mockConfigService
}

func TestSpawnAgentToolSpec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfigService := mocks.NewMockConfigService(ctrl)
	mockConfigService.EXPECT().GetAgentLimits().Return(&config.AgentLimitsConfig{
		MaxSubAgentsPerParent: 3,
		MaxTotalAgents:        50,
	})

	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	provider := &spawnAgentToolProvider{
		configService: mockConfigService,
		hookManager:   mockHookManager,
	}
	tool := provider.CreateTool(uuid.New(), nil) // nil factory is OK for Spec() test

	spec := tool.Spec()

	assert.Equal(t, shared.ToolNameSpawnAgent, spec.Name)
	assert.Contains(t, spec.Description, "immediately")
	// Verify cross-reference to resume_agent tool
	assert.Contains(t, spec.Description, shared.ToolNameResumeAgent)
	// Verify agent limit is shown
	assert.Contains(t, spec.Description, "Maximum concurrent subagents: 3")

	// Check required parameters

	// Check all parameters exist
	require.Contains(t, spec.Parameters, "role")
	require.Contains(t, spec.Parameters, "description")
	require.Contains(t, spec.Parameters, "prompt")
	require.Contains(t, spec.Parameters, "run_in_background")

	// run_in_background should not be required
	param := spec.Parameters["run_in_background"]
	assert.NotNil(t, param)
}

func TestSpawnAgentToolValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockExecHelper := mocks.NewMockAgentExecutionHelper(ctrl)
	mockConfigService := setupMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Set up default behavior for response helper methods
	setupMockExecutionHelperWithDefaults(mockExecHelper)

	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        mockRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: mockExecHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        uuid.New(),
	}

	ctx := context.Background()

	t.Run("Missing role", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"description": "test",
			"prompt":      "do something",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "role is required")
	})

	t.Run("Missing description", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"role":   "Tester",
			"prompt": "do something",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "description is required")
	})

	t.Run("Missing prompt", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"role":        "Tester",
			"description": "test task",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "prompt is required")
	})

	t.Run("Empty role", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"role":        "",
			"description": "test",
			"prompt":      "do something",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "role is required")
	})
}

func TestSpawnAgentToolSynchronousExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	senderID := uuid.New()
	taskID := uuid.New()

	// Setup mocks
	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockExecHelper := mocks.NewMockAgentExecutionHelper(ctrl)
	mockConfigService := setupMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Mock agent
	mockAgent := mocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(taskID).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID:           taskID,
		LLMProvider:  shared.LLMProviderAnthropic,
		Role:         "Tester",
		SystemPrompt: "test",
	}).AnyTimes()

	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        mockRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: mockExecHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        senderID,
	}

	ctx := context.Background()

	// Expect GetSubagentPrompt to be called
	mockPromptMgr.EXPECT().GetSubagentPrompt(gomock.Any(), gomock.Any()).Return("You are a helpful assistant", nil)

	// Expect registry calls
	mockRegistry.EXPECT().GetAgent(senderID).Return(nil, false)               // No parent
	mockRegistry.EXPECT().StoreAgentResult(gomock.Any()).Return(nil).Times(1) // Initial result
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).Do(func(_ context.Context, cfg *shared.AgentConfig) {
		// Verify system prompt was loaded from PromptManager
		assert.Equal(t, "You are a helpful assistant", cfg.SystemPrompt)
	}).Return(mockAgent, nil)

	// Expect Register call (synchronous, no cancel function)
	mockRegistry.EXPECT().Register(mockAgent, gomock.Any()).Return(nil)

	// Expect execution helper call
	expectedResponse := map[string]any{
		"success":  true,
		"agent_id": taskID.String(),
		"response": "Task completed successfully",
		"status":   "completed",
		"message":  "Agent completed successfully",
	}
	mockExecHelper.EXPECT().ExecuteSynchronously(ctx, mockAgent, "Do something").Return(expectedResponse, nil)

	result, err := tool.Run(ctx, map[string]any{
		"role":        "Tester",
		"description": "Test task",
		"prompt":      "Do something",
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "completed", result["status"].(string))
	assert.Equal(t, "Task completed successfully", result["response"].(string))
}

func TestSpawnAgentToolAsynchronousExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	senderID := uuid.New()
	taskID := uuid.New()

	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockExecHelper := mocks.NewMockAgentExecutionHelper(ctrl)
	mockConfigService := setupMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Set up default behavior for response helper methods
	setupMockExecutionHelperWithDefaults(mockExecHelper)

	mockAgent := mocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(taskID).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID:          taskID,
		LLMProvider: shared.LLMProviderAnthropic,
		Role:        "Tester",
	}).AnyTimes()

	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        mockRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: mockExecHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        senderID,
	}

	ctx := context.Background()

	mockPromptMgr.EXPECT().GetSubagentPrompt(gomock.Any(), gomock.Any()).Return("You are a helpful assistant", nil)
	mockRegistry.EXPECT().GetAgent(senderID).Return(nil, false)
	mockRegistry.EXPECT().StoreAgentResult(gomock.Any()).Return(nil).Times(1) // Initial result
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).Return(mockAgent, nil)

	// Expect Register call (background, with cancel function)
	mockRegistry.EXPECT().Register(mockAgent, gomock.Any(), gomock.Any()).Return(nil)

	// Mock execution helper background call
	mockExecHelper.EXPECT().ExecuteInBackground(gomock.Any(), mockAgent, "Do something async")

	result, err := tool.Run(ctx, map[string]any{
		"role":              "Tester",
		"description":       "Async test",
		"prompt":            "Do something async",
		"run_in_background": true,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "running", result["status"].(string))

	// Verify agent_id is returned and is valid
	agentIDStr, ok := result["agent_id"].(string)
	require.True(t, ok, "agent_id should be a string")
	_, err = uuid.Parse(agentIDStr)
	require.NoError(t, err, "agent_id should be a valid UUID")

	assert.Equal(t, "Tester", result["role"].(string))
	assert.Equal(t, "Async test", result["description"].(string))

	// Wait for async execution to complete
	time.Sleep(100 * time.Millisecond)
}

func TestSpawnAgentToolExecutionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	senderID := uuid.New()
	taskID := uuid.New()

	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockExecHelper := mocks.NewMockAgentExecutionHelper(ctrl)
	mockConfigService := setupMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Set up default behavior for response helper methods
	setupMockExecutionHelperWithDefaults(mockExecHelper)

	mockAgent := mocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(taskID).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID:          taskID,
		LLMProvider: shared.LLMProviderAnthropic,
	}).AnyTimes()

	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        mockRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: mockExecHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        senderID,
	}

	ctx := context.Background()

	mockPromptMgr.EXPECT().GetSubagentPrompt(gomock.Any(), gomock.Any()).Return("You are a helpful assistant", nil)
	mockRegistry.EXPECT().GetAgent(senderID).Return(nil, false)
	mockRegistry.EXPECT().StoreAgentResult(gomock.Any()).Return(nil).Times(1) // Initial result
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).Return(mockAgent, nil)

	// Expect Register call (synchronous, no cancel function)
	mockRegistry.EXPECT().Register(mockAgent, gomock.Any()).Return(nil)

	// Mock execution helper error
	mockExecHelper.EXPECT().ExecuteSynchronously(ctx, mockAgent, "Do something").Return(nil, assert.AnError)

	result, err := tool.Run(ctx, map[string]any{
		"role":        "Tester",
		"description": "Test",
		"prompt":      "Do something",
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "execution failed")
}

func TestSpawnAgentToolInheritsLLMProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	senderID := uuid.New()
	taskID := uuid.New()

	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockExecHelper := mocks.NewMockAgentExecutionHelper(ctrl)
	mockConfigService := setupMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockParentAgent := mocks.NewMockAgent(ctrl)
	mockParentAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMProvider: shared.LLMProviderOpenAI,
	}).AnyTimes()

	mockAgent := mocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(taskID).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID:          taskID,
		LLMProvider: shared.LLMProviderOpenAI, // Should inherit from parent
		Role:        "Tester",
	}).AnyTimes()

	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        mockRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: mockExecHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        senderID,
	}

	ctx := context.Background()

	mockPromptMgr.EXPECT().GetSubagentPrompt(gomock.Any(), gomock.Any()).Return("System prompt", nil)
	// Expect GetAgent to be called and return parent
	mockRegistry.EXPECT().GetAgent(senderID).Return(mockParentAgent, true)
	mockRegistry.EXPECT().StoreAgentResult(gomock.Any()).Return(nil).Times(1) // Initial result
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).Do(func(_ context.Context, cfg *shared.AgentConfig) {
		// Verify LLM provider was inherited
		assert.Equal(t, shared.LLMProviderOpenAI, cfg.LLMProvider)
	}).Return(mockAgent, nil)

	// Expect Register call (synchronous, no cancel function)
	mockRegistry.EXPECT().Register(mockAgent, gomock.Any()).Return(nil)

	// Mock execution helper call
	expectedResponse := map[string]any{
		"success":  true,
		"agent_id": taskID.String(),
		"response": "Done",
		"status":   "completed",
		"message":  "Agent completed successfully",
	}
	mockExecHelper.EXPECT().ExecuteSynchronously(ctx, mockAgent, "Do something").Return(expectedResponse, nil)

	result, err := tool.Run(ctx, map[string]any{
		"role":        "Tester",
		"description": "Test",
		"prompt":      "Do something",
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
}

func TestSpawnAgentToolProvider_CreateTool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockExecHelper := mocks.NewMockAgentExecutionHelper(ctrl)
	mockConfigService := setupMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)

	provider := &spawnAgentToolProvider{
		logService:      logService,
		registry:        mockRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: mockExecHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
	}

	senderID := uuid.New()
	tool := provider.CreateTool(senderID, mockFactory)

	require.NotNil(t, tool)
	assert.Equal(t, senderID, tool.senderID)
	assert.Equal(t, mockFactory, tool.agentFactory)
	assert.Equal(t, mockRegistry, tool.registry)
	assert.Equal(t, mockPromptMgr, tool.promptManager)
	assert.Equal(t, mockExecHelper, tool.executionHelper)
	assert.Equal(t, logService, tool.logService)
	assert.Equal(t, mockHookManager, tool.hookManager)
}

// TestSpawnAgentTool_WithShareContext_NoParent tests that share_context works when there is no parent agent
func TestSpawnAgentTool_WithShareContext_NoParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	senderID := uuid.New()

	// Setup mocks - no parent agent
	mockFactory := mocks.NewMockAgentFactory(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockExecHelper := mocks.NewMockAgentExecutionHelper(ctrl)
	mockConfigService := setupMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &SpawnAgentTool{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        mockRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: mockExecHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        senderID,
	}

	ctx := context.Background()

	// No parent agent exists
	mockRegistry.EXPECT().GetAgent(senderID).Return(nil, false)
	mockPromptMgr.EXPECT().GetSubagentPrompt(gomock.Any(), gomock.Any()).Return("System prompt", nil)

	// Create the mock agent that will be returned by CreateAgent
	mockSubagent := mocks.NewMockAgent(ctrl)
	mockSubagent.EXPECT().GetID().Return(uuid.New()).Times(4) // GetID is called at lines 211, 215, 251, 258

	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).Do(func(_ context.Context, cfg *shared.AgentConfig) {
		// Verify message history is nil when no parent exists
		assert.Nil(t, cfg.History)
	}).Return(mockSubagent, nil)

	mockRegistry.EXPECT().StoreAgentResult(gomock.Any()).Return(nil).Times(1)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).Return(nil)

	// Mock execution helper
	expectedResponse := map[string]any{
		"success":  true,
		"agent_id": gomock.Any().String(),
		"response": "Task completed",
		"status":   "completed",
		"message":  "Agent completed",
	}
	mockExecHelper.EXPECT().ExecuteSynchronously(ctx, gomock.Any(), gomock.Any()).Return(expectedResponse, nil)

	result, err := tool.Run(ctx, map[string]any{
		"role":          "Tester",
		"description":   "Test task",
		"prompt":        "Do something",
		"share_context": true,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "completed", result["status"].(string))
}
