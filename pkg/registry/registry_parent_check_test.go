package registry

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestAgentRegistry_IsDirectParent tests the IsDirectParent method with various scenarios
func TestAgentRegistry_IsDirectParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()

	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	// Create a hierarchy:
	//   grandparent
	//     └── parent
	//           └── child
	//     └── uncle (sibling of parent)
	//   unrelated
	grandparentID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()
	uncleID := uuid.New()
	unrelatedID := uuid.New()

	// Create mock agents
	mockGrandparent := mocks.NewMockAgent(ctrl)
	mockGrandparent.EXPECT().GetID().Return(grandparentID).AnyTimes()
	mockGrandparent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID:   grandparentID,
		Role: "Grandparent",
	}).AnyTimes()

	mockParent := mocks.NewMockAgent(ctrl)
	mockParent.EXPECT().GetID().Return(parentID).AnyTimes()
	mockParent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID:   parentID,
		Role: "Parent",
	}).AnyTimes()

	mockChild := mocks.NewMockAgent(ctrl)
	mockChild.EXPECT().GetID().Return(childID).AnyTimes()
	mockChild.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID:   childID,
		Role: "Child",
	}).AnyTimes()

	mockUncle := mocks.NewMockAgent(ctrl)
	mockUncle.EXPECT().GetID().Return(uncleID).AnyTimes()
	mockUncle.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID:   uncleID,
		Role: "Uncle",
	}).AnyTimes()

	mockUnrelated := mocks.NewMockAgent(ctrl)
	mockUnrelated.EXPECT().GetID().Return(unrelatedID).AnyTimes()
	mockUnrelated.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID:   unrelatedID,
		Role: "Unrelated",
	}).AnyTimes()

	// Register grandparent (no parent)
	grandparentConfig := &shared.AgentConfig{ID: grandparentID, Role: "Grandparent"}
	err = registry.Register(mockGrandparent, grandparentConfig)
	require.NoError(t, err)

	// Register parent (child of grandparent)
	parentConfig := &shared.AgentConfig{ID: parentID, ParentID: &grandparentID, Role: "Parent", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockParent, parentConfig)
	require.NoError(t, err)

	// Register child (child of parent)
	childConfig := &shared.AgentConfig{ID: childID, ParentID: &parentID, Role: "Child", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockChild, childConfig)
	require.NoError(t, err)

	// Register uncle (sibling of parent, also child of grandparent)
	uncleConfig := &shared.AgentConfig{ID: uncleID, ParentID: &grandparentID, Role: "Uncle", LLMClientConfig: &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}}
	err = registry.Register(mockUncle, uncleConfig)
	require.NoError(t, err)

	// Register unrelated agent (no parent)
	unrelatedConfig := &shared.AgentConfig{ID: unrelatedID, Role: "Unrelated"}
	err = registry.Register(mockUnrelated, unrelatedConfig)
	require.NoError(t, err)

	t.Run("DirectParent_ReturnsTrue", func(t *testing.T) {
		// parent is the direct parent of child
		result := registry.IsDirectParent(parentID, childID)
		assert.True(t, result, "parent should be direct parent of child")
	})

	t.Run("Grandparent_ReturnsFalse", func(t *testing.T) {
		// grandparent is NOT the direct parent of child (parent is)
		result := registry.IsDirectParent(grandparentID, childID)
		assert.False(t, result, "grandparent should NOT be direct parent of child")
	})

	t.Run("Sibling_ReturnsFalse", func(t *testing.T) {
		// uncle is NOT the direct parent of child (they are siblings)
		result := registry.IsDirectParent(uncleID, childID)
		assert.False(t, result, "uncle should NOT be direct parent of child (they are siblings)")
	})

	t.Run("Unrelated_ReturnsFalse", func(t *testing.T) {
		// unrelated agent is NOT the direct parent of child
		result := registry.IsDirectParent(unrelatedID, childID)
		assert.False(t, result, "unrelated agent should NOT be direct parent of child")
	})

	t.Run("Child_ReturnsFalseForParent", func(t *testing.T) {
		// child is NOT the direct parent of parent (reverse relationship)
		result := registry.IsDirectParent(childID, parentID)
		assert.False(t, result, "child should NOT be direct parent of parent")
	})

	t.Run("RootAgentWithNilParentID_ReturnsFalse", func(t *testing.T) {
		// grandparent has no parent (ParentID is nil), so nobody is its direct parent
		result := registry.IsDirectParent(parentID, grandparentID)
		assert.False(t, result, "nobody should be direct parent of root agent")
	})

	t.Run("NonExistentTarget_ReturnsFalse", func(t *testing.T) {
		nonExistentID := uuid.New()
		// Non-existent target should return false
		result := registry.IsDirectParent(parentID, nonExistentID)
		assert.False(t, result, "non-existent target should return false")
	})

	t.Run("NonExistentCaller_ReturnsFalse", func(t *testing.T) {
		nonExistentID := uuid.New()
		// Non-existent caller should return false
		result := registry.IsDirectParent(nonExistentID, childID)
		assert.False(t, result, "non-existent caller should return false")
	})
}

// TestAgentRegistry_IsDirectParent_SameAgent tests when caller and target are the same
func TestAgentRegistry_IsDirectParent_SameAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()

	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	agentID := uuid.New()

	mockAgent := mocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		ID:   agentID,
		Role: "Agent",
	}).AnyTimes()

	agentConfig := &shared.AgentConfig{ID: agentID, Role: "Agent"}
	err = registry.Register(mockAgent, agentConfig)
	require.NoError(t, err)

	// An agent is not the direct parent of itself
	result := registry.IsDirectParent(agentID, agentID)
	assert.False(t, result, "agent should NOT be direct parent of itself")
}
