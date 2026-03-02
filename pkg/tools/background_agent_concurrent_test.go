package tools

import (
	"context"
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
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	mockConfigService := setupMockConfigService(ctrl)

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
		configService:   mockConfigService,
		hookManager:     mockHookManager,
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
		mockPromptMgr.EXPECT().GetSubagentTaskPrompt("Concurrent Agent", "Concurrent task").Return("You are a helpful assistant", nil)
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
		registry:    agentRegistry,
		senderID:    senderID,
		hookManager: mockHookManager,
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
	mockConfigService := mocks.NewMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	execHelper := do.MustInvoke[AgentExecutionHelper](injector)

	// Setup mock hook manager to pass-through hooks
	setupMockHookManagerPassThrough(mockHookManager)

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
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        rootID,
	}

	// Step 1: Root spawns child
	mockPromptMgr.EXPECT().GetSubagentTaskPrompt("Child", "Child task").Return("You are a helpful assistant", nil)
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
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        childID,
	}

	mockPromptMgr.EXPECT().GetSubagentTaskPrompt("Grandchild", "Grandchild task").Return("You are a helpful assistant", nil)
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
		registry:    agentRegistry,
		senderID:    rootID,
		hookManager: mockHookManager,
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
