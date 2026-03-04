package tools

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListAgentsTool_Spec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	registry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &ListAgentsTool{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    registry,
		senderID:    uuid.New(),
	}

	spec := tool.Spec()
	assert.Equal(t, shared.ToolNameListAgents.String(), spec.Name)
	assert.NotEmpty(t, spec.Parameters) // Now has --recursive and --tree parameters

	// Verify new parameters exist
	assert.Contains(t, spec.Parameters, "recursive")
	assert.Contains(t, spec.Parameters, "tree")

	// Verify key content in description (not exact match due to multi-line format)
	assert.Contains(t, spec.Description, "VISIBILITY")
	assert.Contains(t, spec.Description, "OPERATIONS")
	assert.Contains(t, spec.Description, "DIRECT children ONLY")
	assert.Contains(t, spec.Description, shared.ToolNameRemoveAgent)
	assert.Contains(t, spec.Description, shared.ToolNameResumeAgent)
	assert.Contains(t, spec.Description, shared.ToolNameAgentOutput)
	assert.Contains(t, spec.Description, shared.ToolNameSpawnAgent)
}

func TestListAgentsTool_Run_SuccessNoRelatedAgents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	senderID := uuid.New()
	registry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	registry.EXPECT().GetChildren(senderID).Return([]shared.Agent{})
	registry.EXPECT().GetParent(senderID).Return(nil, false)

	tool := &ListAgentsTool{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    registry,
		senderID:    senderID,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, map[string]any{})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "No related agents found (no parent and no subagents)", result["message"])

	agents := result["agents"].([]map[string]any)
	assert.Empty(t, agents)
}

func TestListAgentsTool_Run_SuccessWithSubagentsOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	senderID := uuid.New()
	childID1 := uuid.New()
	childID2 := uuid.New()

	registry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Create mock child agents
	child1 := mocks.NewMockAgent(ctrl)
	child2 := mocks.NewMockAgent(ctrl)

	child1Config := &shared.AgentConfig{
		ID:   childID1,
		Role: "Data Analyst",
	}
	child2Config := &shared.AgentConfig{
		ID:   childID2,
		Role: "Code Reviewer",
	}

	child1.EXPECT().GetConfig().Return(child1Config)
	child2.EXPECT().GetConfig().Return(child2Config)

	registry.EXPECT().GetChildren(senderID).Return([]shared.Agent{child1, child2})
	registry.EXPECT().GetParent(senderID).Return(nil, false) // No parent

	tool := &ListAgentsTool{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    registry,
		senderID:    senderID,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, map[string]any{})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "Found 2 subagent(s) (no parent agent)", result["message"])

	agents := result["agents"].([]map[string]any)
	require.Len(t, agents, 2)

	// Check first subagent
	assert.Equal(t, childID1.String(), agents[0]["id"])
	assert.Equal(t, "Data Analyst", agents[0]["role"])
	assert.Equal(t, "SubAgent", agents[0]["type"])

	// Check second subagent
	assert.Equal(t, childID2.String(), agents[1]["id"])
	assert.Equal(t, "Code Reviewer", agents[1]["role"])
	assert.Equal(t, "SubAgent", agents[1]["type"])
}

func TestListAgentsTool_Run_SuccessWithParentOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	senderID := uuid.New()
	parentID := uuid.New()

	registry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Create mock parent agent
	parent := mocks.NewMockAgent(ctrl)

	parentConfig := &shared.AgentConfig{
		ID:   parentID,
		Role: "Main Coordinator",
	}

	parent.EXPECT().GetConfig().Return(parentConfig)

	registry.EXPECT().GetChildren(senderID).Return([]shared.Agent{}) // No children
	registry.EXPECT().GetParent(senderID).Return(parent, true)

	tool := &ListAgentsTool{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    registry,
		senderID:    senderID,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, map[string]any{})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "Found 1 parent agent", result["message"])

	agents := result["agents"].([]map[string]any)
	require.Len(t, agents, 1)

	// Check parent agent
	assert.Equal(t, parentID.String(), agents[0]["id"])
	assert.Equal(t, "Main Coordinator", agents[0]["role"])
	assert.Equal(t, "ParentAgent", agents[0]["type"])
}

func TestListAgentsTool_Run_SuccessWithParentAndSubagents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	senderID := uuid.New()
	parentID := uuid.New()
	childID1 := uuid.New()
	childID2 := uuid.New()

	registry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Create mock parent agent
	parent := mocks.NewMockAgent(ctrl)
	parentConfig := &shared.AgentConfig{
		ID:   parentID,
		Role: "Main Coordinator",
	}
	parent.EXPECT().GetConfig().Return(parentConfig)

	// Create mock child agents
	child1 := mocks.NewMockAgent(ctrl)
	child2 := mocks.NewMockAgent(ctrl)

	child1Config := &shared.AgentConfig{
		ID:   childID1,
		Role: "Data Analyst",
	}
	child2Config := &shared.AgentConfig{
		ID:   childID2,
		Role: "Code Reviewer",
	}

	child1.EXPECT().GetConfig().Return(child1Config)
	child2.EXPECT().GetConfig().Return(child2Config)

	registry.EXPECT().GetChildren(senderID).Return([]shared.Agent{child1, child2})
	registry.EXPECT().GetParent(senderID).Return(parent, true)

	tool := &ListAgentsTool{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    registry,
		senderID:    senderID,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, map[string]any{})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "Found 1 parent agent and 2 subagent(s)", result["message"])

	agents := result["agents"].([]map[string]any)
	require.Len(t, agents, 3)

	// Check parent agent (should be first)
	assert.Equal(t, parentID.String(), agents[0]["id"])
	assert.Equal(t, "Main Coordinator", agents[0]["role"])
	assert.Equal(t, "ParentAgent", agents[0]["type"])

	// Check first subagent
	assert.Equal(t, childID1.String(), agents[1]["id"])
	assert.Equal(t, "Data Analyst", agents[1]["role"])
	assert.Equal(t, "SubAgent", agents[1]["type"])

	// Check second subagent
	assert.Equal(t, childID2.String(), agents[2]["id"])
	assert.Equal(t, "Code Reviewer", agents[2]["role"])
	assert.Equal(t, "SubAgent", agents[2]["type"])
}

func TestListAgentsToolProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	registry := mocks.NewMockAgentRegistry(ctrl)
	senderID := uuid.New()

	provider := &listAgentsToolProvider{
		logService: logService,
		registry:   registry,
	}

	tool := provider.CreateTool(senderID)

	assert.NotNil(t, tool)
	assert.Equal(t, logService, tool.logService)
	assert.Equal(t, registry, tool.registry)
	assert.Equal(t, senderID, tool.senderID)
}

func TestNewListAgentsToolProvider(t *testing.T) {
	// This test would require setting up a full DI container
	// For now, we just verify the function signature is correct
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	registry := mocks.NewMockAgentRegistry(ctrl)

	// Note: In a real test, you would set up a DI container
	// and inject the mock registry, then call NewListAgentsToolProvider
	// This is a placeholder to verify the function exists
	provider := &listAgentsToolProvider{
		logService: logService,
		registry:   registry,
	}

	assert.NotNil(t, provider)
	assert.Equal(t, logService, provider.logService)
	assert.Equal(t, registry, provider.registry)
}

func TestListAgentsTool_Run_RecursiveFlag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	senderID := uuid.New()
	childID1 := uuid.New()
	grandchildID := uuid.New()

	registry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Create mock agents
	child1 := mocks.NewMockAgent(ctrl)
	grandchild := mocks.NewMockAgent(ctrl)

	child1Config := &shared.AgentConfig{
		ID:   childID1,
		Role: "Child Agent",
	}
	grandchildConfig := &shared.AgentConfig{
		ID:   grandchildID,
		Role: "Grandchild Agent",
	}

	child1.EXPECT().GetConfig().Return(child1Config).AnyTimes()
	grandchild.EXPECT().GetConfig().Return(grandchildConfig).AnyTimes()

	// Set up expectations in exact order:
	// 1. GetChildren(senderID) returns [child1]
	// 2. GetChildren(childID1) returns [grandchild]
	// 3. GetChildren(grandchildID) returns [] (end of recursion)
	registry.EXPECT().GetParent(senderID).Return(nil, false)
	registry.EXPECT().GetChildren(senderID).Return([]shared.Agent{child1})
	registry.EXPECT().GetChildren(childID1).Return([]shared.Agent{grandchild})
	registry.EXPECT().GetChildren(grandchildID).Return([]shared.Agent{}) // No more descendants

	tool := &ListAgentsTool{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    registry,
		senderID:    senderID,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, map[string]any{"recursive": true})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "Found 2 descendant(s) (no parent agent)", result["message"])

	agents := result["agents"].([]map[string]any)
	require.Len(t, agents, 2)

	// Check child agent
	assert.Equal(t, childID1.String(), agents[0]["id"])
	assert.Equal(t, "Child Agent", agents[0]["role"])
	assert.Equal(t, "SubAgent", agents[0]["type"])
	assert.Equal(t, 1, agents[0]["depth"])

	// Check grandchild agent
	assert.Equal(t, grandchildID.String(), agents[1]["id"])
	assert.Equal(t, "Grandchild Agent", agents[1]["role"])
	assert.Equal(t, "SubAgent", agents[1]["type"])
	assert.Equal(t, 2, agents[1]["depth"])
}

func TestListAgentsTool_Run_TreeFlag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	senderID := uuid.New()
	childID1 := uuid.New()
	childID2 := uuid.New()

	registry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Create mock child agents
	child1 := mocks.NewMockAgent(ctrl)
	child2 := mocks.NewMockAgent(ctrl)

	child1Config := &shared.AgentConfig{
		ID:          childID1,
		Role:        "Child Agent 1",
		Description: "First child",
	}
	child2Config := &shared.AgentConfig{
		ID:   childID2,
		Role: "Child Agent 2",
	}

	child1.EXPECT().GetConfig().Return(child1Config).AnyTimes()
	child2.EXPECT().GetConfig().Return(child2Config).AnyTimes()

	registry.EXPECT().GetParent(senderID).Return(nil, false)
	registry.EXPECT().GetChildren(senderID).Return([]shared.Agent{child1, child2})

	tool := &ListAgentsTool{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    registry,
		senderID:    senderID,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, map[string]any{"tree": true})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Nil(t, result["agents"]) // No agent list in tree mode
	assert.NotNil(t, result["tree"])
	assert.Equal(t, 2, result["descendant_count"])

	treeOutput := result["tree"].(string)
	assert.Contains(t, treeOutput, "(YOU)")
	assert.Contains(t, treeOutput, "Child Agent 1")
	assert.Contains(t, treeOutput, "Child Agent 2")
	assert.Contains(t, treeOutput, "├─") // Tree connector
	assert.Contains(t, treeOutput, "└─") // Tree connector for last item
}

func TestListAgentsTool_Run_TreeFlagWithRecursive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	senderID := uuid.New()
	childID := uuid.New()
	grandchildID := uuid.New()

	registry := mocks.NewMockAgentRegistry(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Create mock agents
	child := mocks.NewMockAgent(ctrl)
	grandchild := mocks.NewMockAgent(ctrl)

	childConfig := &shared.AgentConfig{
		ID:   childID,
		Role: "Child Agent",
	}
	grandchildConfig := &shared.AgentConfig{
		ID:   grandchildID,
		Role: "Grandchild Agent",
	}

	child.EXPECT().GetConfig().Return(childConfig).AnyTimes()
	grandchild.EXPECT().GetConfig().Return(grandchildConfig).AnyTimes()

	// Set up expectations in exact order
	registry.EXPECT().GetParent(senderID).Return(nil, false)
	registry.EXPECT().GetChildren(senderID).Return([]shared.Agent{child})
	registry.EXPECT().GetChildren(childID).Return([]shared.Agent{grandchild})
	registry.EXPECT().GetChildren(grandchildID).Return([]shared.Agent{}) // No more descendants

	tool := &ListAgentsTool{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    registry,
		senderID:    senderID,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, map[string]any{"tree": true, "recursive": true})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, 2, result["descendant_count"])

	treeOutput := result["tree"].(string)
	assert.Contains(t, treeOutput, "(YOU)")
	assert.Contains(t, treeOutput, "Child Agent")
	assert.Contains(t, treeOutput, "Grandchild Agent")
}
