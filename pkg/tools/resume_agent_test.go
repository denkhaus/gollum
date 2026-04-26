package tools

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestResumeAgentToolSpec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provider := &resumeAgentToolProvider{}
	testAgent := shared.NewMockAgent(ctrl)
	testAgent.EXPECT().GetID().Return(uuid.New()).AnyTimes()
	tool := provider.CreateTool(testAgent)

	spec := tool.Spec()

	assert.Equal(t, shared.ToolNameResumeAgent.String(), spec.Name)
	assert.Contains(t, spec.Description, "new prompt")

	// Check required parameters

	// Check all parameters exist
	require.Contains(t, spec.Parameters, "agent_id")
	require.Contains(t, spec.Parameters, "prompt")
	require.Contains(t, spec.Parameters, "run_in_background")

	// run_in_background should not be required
	param := spec.Parameters["run_in_background"]
	assert.NotNil(t, param)
}

func TestResumeAgentToolValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockExecHelper := NewMockAgentExecutionHelper(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Set up default behavior for response helper methods
	setupMockExecutionHelperWithDefaults(mockExecHelper)

	testAgent := shared.NewMockAgent(ctrl)
	testAgent.EXPECT().GetID().Return(uuid.New()).AnyTimes()

	tool := &resumeAgentToolImpl{
		logService:      logService,
		hookManager:     mockHookManager,
		registry:        mockRegistry,
		executionHelper: mockExecHelper,
		agent:           testAgent,
	}

	ctx := context.Background()

	t.Run("Missing agent_id", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"prompt": "do something",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "agent_id is required")
	})

	t.Run("Empty agent_id", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"agent_id": "",
			"prompt":   "do something",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "agent_id is required")
	})

	t.Run("Invalid agent_id format", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"agent_id": "not-a-uuid",
			"prompt":   "do something",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "invalid agent_id format")
	})

	t.Run("Missing prompt", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"agent_id": uuid.New().String(),
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "prompt is required")
	})

	t.Run("Empty prompt", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"agent_id": uuid.New().String(),
			"prompt":   "",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "prompt is required")
	})
}

func TestResumeAgentToolAgentNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockExecHelper := NewMockAgentExecutionHelper(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Set up default behavior for response helper methods
	setupMockExecutionHelperWithDefaults(mockExecHelper)

	senderID := uuid.New()
	senderAgent := shared.NewMockAgent(ctrl)
	senderAgent.EXPECT().GetID().Return(senderID).AnyTimes()
	agentID := uuid.New()

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().IsDirectParent(senderID, agentID).Return(true)

	// Mock agent not found
	mockRegistry.EXPECT().GetAgent(agentID).Return(nil, false)

	tool := &resumeAgentToolImpl{
		logService:      logService,
		hookManager:     mockHookManager,
		registry:        mockRegistry,
		executionHelper: mockExecHelper,
		agent:           senderAgent,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, map[string]any{
		"agent_id": agentID.String(),
		"prompt":   "do something",
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "not found in registry")
}

func TestResumeAgentToolSynchronousExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	senderID := uuid.New()
	senderAgent := shared.NewMockAgent(ctrl)
	senderAgent.EXPECT().GetID().Return(senderID).AnyTimes()
	agentID := uuid.New()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockExecHelper := NewMockAgentExecutionHelper(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockAgent := shared.NewMockAgent(ctrl)

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().IsDirectParent(senderID, agentID).Return(true)

	// Mock agent exists
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID: agentID,
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "test",
		},
		Role:        "Tester",
		Description: "Test Agent",
	}).AnyTimes()

	mockRegistry.EXPECT().GetAgent(agentID).Return(mockAgent, true)

	// Mock DeleteAgentResult call to clear previous result
	mockRegistry.EXPECT().DeleteAgentResult(agentID).Return(nil)

	// Mock execution helper call
	expectedResponse := map[string]any{
		"success":  true,
		"agent_id": agentID.String(),
		"response": "Task completed successfully",
		"status":   "completed",
		"message":  "Agent completed successfully",
	}
	mockExecHelper.EXPECT().ExecuteSynchronously(gomock.Any(), mockAgent, "Do something").Return(expectedResponse, nil)

	tool := &resumeAgentToolImpl{
		logService:      logService,
		hookManager:     mockHookManager,
		registry:        mockRegistry,
		executionHelper: mockExecHelper,
		agent:           senderAgent,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, map[string]any{
		"agent_id": agentID.String(),
		"prompt":   "Do something",
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "completed", result["status"].(string))
	assert.Equal(t, "Task completed successfully", result["response"].(string))
}

func TestResumeAgentToolAsynchronousExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	senderID := uuid.New()
	senderAgent := shared.NewMockAgent(ctrl)
	senderAgent.EXPECT().GetID().Return(senderID).AnyTimes()
	agentID := uuid.New()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockExecHelper := NewMockAgentExecutionHelper(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Set up default behavior for response helper methods
	setupMockExecutionHelperWithDefaults(mockExecHelper)

	mockAgent := shared.NewMockAgent(ctrl)

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().IsDirectParent(senderID, agentID).Return(true)

	// Mock agent exists
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID: agentID,
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "test",
		},
		Role:        "Tester",
		Description: "Test Agent",
	}).AnyTimes()

	mockRegistry.EXPECT().GetAgent(agentID).Return(mockAgent, true)

	// Mock DeleteAgentResult call to clear previous result
	mockRegistry.EXPECT().DeleteAgentResult(agentID).Return(nil)

	// Mock SetCancelFunc call for background execution
	mockRegistry.EXPECT().SetCancelFunc(agentID, gomock.Any()).Return(nil)

	// Mock execution helper background call
	mockExecHelper.EXPECT().ExecuteInBackground(gomock.Any(), mockAgent, "Do something async")

	tool := &resumeAgentToolImpl{
		logService:      logService,
		hookManager:     mockHookManager,
		registry:        mockRegistry,
		executionHelper: mockExecHelper,
		agent:           senderAgent,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, map[string]any{
		"agent_id":          agentID.String(),
		"prompt":            "Do something async",
		"run_in_background": true,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "running", result["status"].(string))
	assert.Equal(t, agentID.String(), result["agent_id"].(string))
	assert.Equal(t, "Tester", result["role"].(string))
	assert.Equal(t, "Test Agent", result["description"].(string))

	// Wait for async execution to complete
	time.Sleep(100 * time.Millisecond)
}

func TestResumeAgentToolProvider_CreateTool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)

	provider := &resumeAgentToolProvider{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    mockRegistry,
	}

	senderID := uuid.New()
	senderAgent := shared.NewMockAgent(ctrl)
	senderAgent.EXPECT().GetID().Return(senderID).AnyTimes()
	tool := provider.CreateTool(senderAgent)
	toolImpl := tool.(*resumeAgentToolImpl)

	require.NotNil(t, tool)
	assert.Equal(t, senderAgent, toolImpl.agent)
	assert.Equal(t, mockRegistry, toolImpl.registry)
	assert.Equal(t, logService, toolImpl.logService)
	assert.Equal(t, mockHookManager, toolImpl.hookManager)
}

// TestResumeAgentTool_PermissionDenied tests permission check when caller is not direct parent
func TestResumeAgentTool_PermissionDenied(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockExecHelper := NewMockAgentExecutionHelper(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Set up default behavior for response helper methods
	setupMockExecutionHelperWithDefaults(mockExecHelper)

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	senderID := uuid.New()
	senderAgent := shared.NewMockAgent(ctrl)
	senderAgent.EXPECT().GetID().Return(senderID).AnyTimes()
	agentID := uuid.New()

	// Permission check: sender is NOT direct parent
	mockRegistry.EXPECT().IsDirectParent(senderID, agentID).Return(false)

	tool := &resumeAgentToolImpl{
		logService:      logService,
		hookManager:     mockHookManager,
		registry:        mockRegistry,
		executionHelper: mockExecHelper,
		agent:           senderAgent,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, map[string]any{
		"agent_id": agentID.String(),
		"prompt":   "do something",
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "permission denied")
	assert.Contains(t, result["error"].(string), "direct subagents")
	assert.Contains(t, result["error"].(string), "not your direct child")
}
