package tools

import (
	"context"
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
	mockConfigService := mocks.NewMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	execHelper := do.MustInvoke[AgentExecutionHelper](injector)

	// Setup mock hook manager to pass-through hooks
	setupMockHookManagerPassThrough(mockHookManager)

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
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        senderID,
	}

	ctx := context.Background()

	// Step 1: Spawn the agent
	mockPromptMgr.EXPECT().GetSubagentTaskPrompt("Lifecycle Agent", "Lifecycle task").Return("You are a helpful assistant", nil)
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
		hookManager:     mockHookManager,
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
		registry:    agentRegistry,
		senderID:    senderID,
		hookManager: mockHookManager,
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
		logService:  logService,
		hookManager: mockHookManager,
		registry:    agentRegistry,
		senderID:    senderID,
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
