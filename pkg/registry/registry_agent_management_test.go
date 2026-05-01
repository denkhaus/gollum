package registry

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAgentRegistry_GetAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()

	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	parentID := uuid.New()
	childID := uuid.New()

	// Create mock agents
	mockParent := shared.NewMockAgent(ctrl)
	mockParent.EXPECT().GetID().Return(parentID).AnyTimes()
	mockParent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: parentID},
		Role:           "Parent",
	}).AnyTimes()

	mockChild := shared.NewMockAgent(ctrl)
	mockChild.EXPECT().GetID().Return(childID).AnyTimes()
	mockChild.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: childID},
		Role:           "Child",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	// Register parent
	parentConfig := &shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: parentID},
		Role:           "Parent",
	}
	err = registry.Register(mockParent, parentConfig)
	require.NoError(t, err)

	// Register child
	childConfig := &shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: childID},
		ParentID:       &parentID,
		Role:           "Child",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}
	err = registry.Register(mockChild, childConfig)
	require.NoError(t, err)

	t.Run("GetExistingParent", func(t *testing.T) {
		agent, found := registry.GetAgent(parentID)
		require.True(t, found)
		require.NotNil(t, agent)
		assert.Equal(t, parentID, agent.GetID())
	})

	t.Run("GetExistingChild", func(t *testing.T) {
		agent, found := registry.GetAgent(childID)
		require.True(t, found)
		require.NotNil(t, agent)
		assert.Equal(t, childID, agent.GetID())
	})

	t.Run("GetNonExistentAgent", func(t *testing.T) {
		nonExistentID := uuid.New()
		agent, found := registry.GetAgent(nonExistentID)
		require.False(t, found)
		require.Nil(t, agent)
	})
}

func TestAgentRegistry_GetChildren(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()

	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	parentID := uuid.New()
	childID1 := uuid.New()
	childID2 := uuid.New()
	unrelatedID := uuid.New()

	// Create mock agents
	mockParent := shared.NewMockAgent(ctrl)
	mockParent.EXPECT().GetID().Return(parentID).AnyTimes()
	mockParent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: parentID},
		Role:           "Parent",
	}).AnyTimes()

	mockChild1 := shared.NewMockAgent(ctrl)
	mockChild1.EXPECT().GetID().Return(childID1).AnyTimes()
	mockChild1.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: childID1},
		Role:           "Child1",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	mockChild2 := shared.NewMockAgent(ctrl)
	mockChild2.EXPECT().GetID().Return(childID2).AnyTimes()
	mockChild2.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: childID2},
		Role:           "Child2",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	mockUnrelated := shared.NewMockAgent(ctrl)
	mockUnrelated.EXPECT().GetID().Return(unrelatedID).AnyTimes()
	mockUnrelated.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: unrelatedID},
		Role:           "Unrelated",
	}).AnyTimes()

	// Register agents
	parentConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: parentID}, Role: "Parent"}
	err = registry.Register(mockParent, parentConfig)
	require.NoError(t, err)

	child1Config := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: childID1}, ParentID: &parentID, Role: "Child1", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockChild1, child1Config)
	require.NoError(t, err)

	child2Config := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: childID2}, ParentID: &parentID, Role: "Child2", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockChild2, child2Config)
	require.NoError(t, err)

	unrelatedConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: unrelatedID}, Role: "Unrelated"}
	err = registry.Register(mockUnrelated, unrelatedConfig)
	require.NoError(t, err)

	t.Run("GetChildrenOfParent", func(t *testing.T) {
		children := registry.GetChildren(parentID)
		require.Len(t, children, 2)

		ids := make([]uuid.UUID, 0, 2)
		for _, child := range children {
			ids = append(ids, child.GetID())
		}

		assert.Contains(t, ids, childID1)
		assert.Contains(t, ids, childID2)
		assert.NotContains(t, ids, unrelatedID)
	})

	t.Run("GetChildrenOfChildWithNoChildren", func(t *testing.T) {
		children := registry.GetChildren(childID1)
		require.Empty(t, children)
	})

	t.Run("GetChildrenOfNonExistentAgent", func(t *testing.T) {
		nonExistentID := uuid.New()
		children := registry.GetChildren(nonExistentID)
		require.Empty(t, children)
	})
}

func TestAgentRegistry_GetParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()

	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	parentID := uuid.New()
	childID := uuid.New()

	// Create mock agents
	mockParent := shared.NewMockAgent(ctrl)
	mockParent.EXPECT().GetID().Return(parentID).AnyTimes()
	mockParent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: parentID},
		Role:           "Parent",
	}).AnyTimes()

	mockChild := shared.NewMockAgent(ctrl)
	mockChild.EXPECT().GetID().Return(childID).AnyTimes()
	mockChild.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: childID},
		Role:           "Child",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	// Register agents
	parentConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: parentID}, Role: "Parent"}
	err = registry.Register(mockParent, parentConfig)
	require.NoError(t, err)

	childConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: childID}, ParentID: &parentID, Role: "Child", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockChild, childConfig)
	require.NoError(t, err)

	t.Run("GetParentOfChild", func(t *testing.T) {
		parent, found := registry.GetParent(childID)
		require.True(t, found)
		require.NotNil(t, parent)
		assert.Equal(t, parentID, parent.GetID())
	})

	t.Run("GetParentOfParentAgent", func(t *testing.T) {
		parent, found := registry.GetParent(parentID)
		require.False(t, found)
		require.Nil(t, parent)
	})

	t.Run("GetParentOfNonExistentAgent", func(t *testing.T) {
		nonExistentID := uuid.New()
		parent, found := registry.GetParent(nonExistentID)
		require.False(t, found)
		require.Nil(t, parent)
	})
}

func TestAgentRegistry_ListAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()

	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	parentID := uuid.New()
	childID1 := uuid.New()
	childID2 := uuid.New()

	// Create mock agents
	mockParent := shared.NewMockAgent(ctrl)
	mockParent.EXPECT().GetID().Return(parentID).AnyTimes()
	mockParent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: parentID},
		Role:           "Parent",
	}).AnyTimes()

	mockChild1 := shared.NewMockAgent(ctrl)
	mockChild1.EXPECT().GetID().Return(childID1).AnyTimes()
	mockChild1.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: childID1},
		Role:           "Child1",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	mockChild2 := shared.NewMockAgent(ctrl)
	mockChild2.EXPECT().GetID().Return(childID2).AnyTimes()
	mockChild2.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: childID2},
		Role:           "Child2",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	// Initially empty
	allAgents := registry.ListAll()
	assert.Empty(t, allAgents)

	// Register parent
	parentConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: parentID}, Role: "Parent"}
	err = registry.Register(mockParent, parentConfig)
	require.NoError(t, err)

	allAgents = registry.ListAll()
	require.Len(t, allAgents, 1)

	// Register children
	child1Config := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: childID1}, ParentID: &parentID, Role: "Child1", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockChild1, child1Config)
	require.NoError(t, err)

	child2Config := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: childID2}, ParentID: &parentID, Role: "Child2", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockChild2, child2Config)
	require.NoError(t, err)

	allAgents = registry.ListAll()
	require.Len(t, allAgents, 3)

	ids := make([]uuid.UUID, 0, 3)
	for _, agent := range allAgents {
		ids = append(ids, agent.GetID())
	}

	assert.Contains(t, ids, parentID)
	assert.Contains(t, ids, childID1)
	assert.Contains(t, ids, childID2)
}

func TestAgentRegistry_Unregister_LeafAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()

	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	parentID := uuid.New()
	childID := uuid.New()

	// Create mock agents
	mockParent := shared.NewMockAgent(ctrl)
	mockParent.EXPECT().GetID().Return(parentID).AnyTimes()
	mockParent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: parentID},
		Role:           "Parent",
	}).AnyTimes()

	mockChild := shared.NewMockAgent(ctrl)
	mockChild.EXPECT().GetID().Return(childID).AnyTimes()
	mockChild.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: childID},
		Role:           "Child",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	// Register agents
	parentConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: parentID}, Role: "Parent"}
	err = registry.Register(mockParent, parentConfig)
	require.NoError(t, err)

	childConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: childID}, ParentID: &parentID, Role: "Child", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockChild, childConfig)
	require.NoError(t, err)

	// Verify initial state
	assert.Equal(t, 1, registry.GetSubAgentCount(parentID))
	assert.Equal(t, 2, registry.GetTotalAgentCount())

	// Unregister leaf agent
	err = registry.Unregister(childID)
	require.NoError(t, err)

	// Verify child is removed
	_, found := registry.GetAgent(childID)
	require.False(t, found)

	// Verify parent now has no children
	assert.Equal(t, 0, registry.GetSubAgentCount(parentID))
	assert.Equal(t, 1, registry.GetTotalAgentCount())
}

func TestAgentRegistry_Unregister_WithChildren(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()

	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	parentID := uuid.New()
	childID := uuid.New()
	grandchildID := uuid.New()

	// Create mock agents
	mockParent := shared.NewMockAgent(ctrl)
	mockParent.EXPECT().GetID().Return(parentID).AnyTimes()
	mockParent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: parentID},
		Role:           "Parent",
	}).AnyTimes()

	mockChild := shared.NewMockAgent(ctrl)
	mockChild.EXPECT().GetID().Return(childID).AnyTimes()
	mockChild.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: childID},
		Role:           "Child",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	mockGrandchild := shared.NewMockAgent(ctrl)
	mockGrandchild.EXPECT().GetID().Return(grandchildID).AnyTimes()
	mockGrandchild.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: grandchildID},
		Role:           "Grandchild",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	// Register agents
	parentConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: parentID}, Role: "Parent"}
	err = registry.Register(mockParent, parentConfig)
	require.NoError(t, err)

	childConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: childID}, ParentID: &parentID, Role: "Child", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockChild, childConfig)
	require.NoError(t, err)

	grandchildConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: grandchildID}, ParentID: &childID, Role: "Grandchild", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockGrandchild, grandchildConfig)
	require.NoError(t, err)

	// Verify initial state
	assert.Equal(t, 1, registry.GetSubAgentCount(parentID))
	assert.Equal(t, 1, registry.GetSubAgentCount(childID))
	assert.Equal(t, 3, registry.GetTotalAgentCount())

	// Unregister only removes the single agent, not descendants
	err = registry.Unregister(childID)
	require.NoError(t, err)

	// Verify child is removed
	_, found := registry.GetAgent(childID)
	require.False(t, found)

	// Verify grandchild still exists (not recursively removed)
	_, found = registry.GetAgent(grandchildID)
	require.True(t, found) // Grandchild still exists

	// Verify parent now has no children
	assert.Equal(t, 0, registry.GetSubAgentCount(parentID))
	// Total count is 2 (parent + grandchild, child was removed)
	assert.Equal(t, 2, registry.GetTotalAgentCount())
}

func TestAgentRegistry_Unregister_NonExistentAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()

	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	agentID := uuid.New()

	// Create mock agent
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: agentID},
		Role:           "Agent",
	}).AnyTimes()

	// Register agent
	agentConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: agentID}, Role: "Agent"}
	err = registry.Register(mockAgent, agentConfig)
	require.NoError(t, err)

	// Try to unregister non-existent agent
	nonExistentID := uuid.New()
	err = registry.Unregister(nonExistentID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Count should remain unchanged
	assert.Equal(t, 1, registry.GetTotalAgentCount())
}

func TestAgentRegistry_Cleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()

	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	parentID := uuid.New()
	childID1 := uuid.New()
	childID2 := uuid.New()
	grandchildID := uuid.New()

	// Create mock agents
	mockParent := shared.NewMockAgent(ctrl)
	mockParent.EXPECT().GetID().Return(parentID).AnyTimes()
	mockParent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: parentID},
		Role:           "Parent",
	}).AnyTimes()

	mockChild1 := shared.NewMockAgent(ctrl)
	mockChild1.EXPECT().GetID().Return(childID1).AnyTimes()
	mockChild1.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: childID1},
		Role:           "Child1",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	mockChild2 := shared.NewMockAgent(ctrl)
	mockChild2.EXPECT().GetID().Return(childID2).AnyTimes()
	mockChild2.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: childID2},
		Role:           "Child2",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	mockGrandchild := shared.NewMockAgent(ctrl)
	mockGrandchild.EXPECT().GetID().Return(grandchildID).AnyTimes()
	mockGrandchild.EXPECT().GetConfig().Return(&shared.AgentConfig{
		SessionContext: shared.SessionContext{AgentID: grandchildID},
		Role:           "Grandchild",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}).AnyTimes()

	// Register agents
	parentConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: parentID}, Role: "Parent"}
	err = registry.Register(mockParent, parentConfig)
	require.NoError(t, err)

	child1Config := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: childID1}, ParentID: &parentID, Role: "Child1", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockChild1, child1Config)
	require.NoError(t, err)

	child2Config := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: childID2}, ParentID: &parentID, Role: "Child2", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockChild2, child2Config)
	require.NoError(t, err)

	grandchildConfig := &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: grandchildID}, ParentID: &childID1, Role: "Grandchild", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockGrandchild, grandchildConfig)
	require.NoError(t, err)

	// Verify initial state
	assert.Equal(t, 2, registry.GetSubAgentCount(parentID))
	assert.Equal(t, 1, registry.GetSubAgentCount(childID1))
	assert.Equal(t, 4, registry.GetTotalAgentCount())

	t.Run("CleanupSingleAgent", func(t *testing.T) {
		// Cleanup child2 (no children)
		err = registry.Cleanup(childID2)
		require.NoError(t, err)

		// Verify child2 is removed
		_, found := registry.GetAgent(childID2)
		require.False(t, found)

		// Verify parent now has one child
		assert.Equal(t, 1, registry.GetSubAgentCount(parentID))
		assert.Equal(t, 3, registry.GetTotalAgentCount())
	})

	t.Run("CleanupAgentWithChildren_RemovesRecursively", func(t *testing.T) {
		// Cleanup parent - should remove parent and all descendants
		err = registry.Cleanup(parentID)
		require.NoError(t, err)

		// Verify parent is removed
		_, found := registry.GetAgent(parentID)
		require.False(t, found)

		// Verify child1 is removed
		_, found = registry.GetAgent(childID1)
		require.False(t, found)

		// Verify grandchild is removed (recursive cleanup)
		_, found = registry.GetAgent(grandchildID)
		require.False(t, found)

		// All agents should be removed
		assert.Equal(t, 0, registry.GetTotalAgentCount())
	})

	t.Run("CleanupNonExistentAgent", func(t *testing.T) {
		nonExistentID := uuid.New()
		// Should not panic
		err = registry.Cleanup(nonExistentID)
		require.NoError(t, err)

		// Count should remain 0
		assert.Equal(t, 0, registry.GetTotalAgentCount())
	})
}

func setupTestInjector() do.Injector {
	ctrl := gomock.NewController(&testing.T{})

	return setupTestInjectorWithLimits(ctrl, &config.AgentLimitsConfig{
		MaxTotalAgents:        100,
		MaxSubAgentsPerParent: 50,
	})
}

func TestAgentRegistry_GetSupervisorAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()

	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	t.Run("ReturnsSupervisorWhenRegistered", func(t *testing.T) {
		supervisorID := uuid.New()

		// Create mock supervisor
		mockSupervisor := shared.NewMockAgent(ctrl)
		mockSupervisor.EXPECT().GetID().Return(supervisorID).AnyTimes()
		mockSupervisor.EXPECT().GetConfig().Return(&shared.AgentConfig{
			SessionContext: shared.SessionContext{AgentID: supervisorID},
			Type: shared.AgentTypeSupervisor,
			Role:           "Supervisor Agent",
		}).AnyTimes()

		// Register supervisor
		supervisorConfig := &shared.AgentConfig{
			SessionContext: shared.SessionContext{AgentID: supervisorID},
			Type: shared.AgentTypeSupervisor,
			Role:           "Supervisor Agent",
		}
		err = registry.Register(mockSupervisor, supervisorConfig)
		require.NoError(t, err)

		// Get supervisor
		supervisor, err := registry.GetSupervisorAgent()
		require.NoError(t, err)
		require.NotNil(t, supervisor)
		assert.Equal(t, supervisorID, supervisor.GetID())
	})

	t.Run("ReturnsErrorWhenNoSupervisorRegistered", func(t *testing.T) {
		// Create new registry with no agents
		emptyRegistry, err := NewAgentRegistry(setupTestInjector())
		require.NoError(t, err)

		// Try to get supervisor
		supervisor, err := emptyRegistry.GetSupervisorAgent()
		require.Error(t, err)
		require.Nil(t, supervisor)
		assert.Contains(t, err.Error(), "no supervisor agent registered")
	})

	t.Run("ReturnsSupervisorWhenMultipleAgentsRegistered", func(t *testing.T) {
		// Create fresh registry for this test
		testRegistry, err := NewAgentRegistry(setupTestInjector())
		require.NoError(t, err)

		supervisorID := uuid.New()
		regularID := uuid.New()

		// Create mock supervisor
		mockSupervisor := shared.NewMockAgent(ctrl)
		mockSupervisor.EXPECT().GetID().Return(supervisorID).AnyTimes()
		mockSupervisor.EXPECT().GetConfig().Return(&shared.AgentConfig{
			SessionContext: shared.SessionContext{AgentID: supervisorID},
			Type: shared.AgentTypeSupervisor,
			Role:           "Supervisor Agent",
		}).AnyTimes()

		// Create mock regular agent
		mockRegular := shared.NewMockAgent(ctrl)
		mockRegular.EXPECT().GetID().Return(regularID).AnyTimes()
		mockRegular.EXPECT().GetConfig().Return(&shared.AgentConfig{
			SessionContext: shared.SessionContext{AgentID: regularID},
			Type: shared.AgentTypeSubAgent,
			Role:           "Regular Agent",
			LLMClientConfig: &shared.LLMClientConfig{
				Model: "anthropic/claude-3-5-sonnet-20241022",
			},
		}).AnyTimes()

		// Register both agents
		supervisorConfig := &shared.AgentConfig{
			SessionContext: shared.SessionContext{AgentID: supervisorID},
			Type: shared.AgentTypeSupervisor,
			Role:           "Supervisor Agent",
		}
		err = testRegistry.Register(mockSupervisor, supervisorConfig)
		require.NoError(t, err)

		regularConfig := &shared.AgentConfig{
			SessionContext: shared.SessionContext{AgentID: regularID},
			Type: shared.AgentTypeSubAgent,
			Role:           "Regular Agent",
			LLMClientConfig: &shared.LLMClientConfig{
				Model: "anthropic/claude-3-5-sonnet-20241022",
			},
		}
		err = testRegistry.Register(mockRegular, regularConfig)
		require.NoError(t, err)

		// Get supervisor - should return supervisor, not regular agent
		supervisor, err := testRegistry.GetSupervisorAgent()
		require.NoError(t, err)
		require.NotNil(t, supervisor)
		assert.Equal(t, supervisorID, supervisor.GetID())
		assert.NotEqual(t, regularID, supervisor.GetID())
	})

	t.Run("ReturnsErrorWhenOnlyRegularAgentsRegistered", func(t *testing.T) {
		// Create new registry with only regular agents
		testRegistry, err := NewAgentRegistry(setupTestInjector())
		require.NoError(t, err)

		regularID := uuid.New()

		// Create mock regular agent
		mockRegular := shared.NewMockAgent(ctrl)
		mockRegular.EXPECT().GetID().Return(regularID).AnyTimes()
		mockRegular.EXPECT().GetConfig().Return(&shared.AgentConfig{
			SessionContext: shared.SessionContext{AgentID: regularID},
			Type: shared.AgentTypeSubAgent,
			Role:           "Regular Agent",
			LLMClientConfig: &shared.LLMClientConfig{
				Model: "anthropic/claude-3-5-sonnet-20241022",
			},
		}).AnyTimes()

		// Register regular agent
		regularConfig := &shared.AgentConfig{
			SessionContext: shared.SessionContext{AgentID: regularID},
			Type: shared.AgentTypeSubAgent,
			Role:           "Regular Agent",
			LLMClientConfig: &shared.LLMClientConfig{
				Model: "anthropic/claude-3-5-sonnet-20241022",
			},
		}
		err = testRegistry.Register(mockRegular, regularConfig)
		require.NoError(t, err)

		// Try to get supervisor - should fail
		supervisor, err := testRegistry.GetSupervisorAgent()
		require.Error(t, err)
		require.Nil(t, supervisor)
		assert.Contains(t, err.Error(), "no supervisor agent registered")
	})
}
