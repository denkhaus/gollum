package registry

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAgentRegistry_Register_TotalAgentLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjectorWithLimits(ctrl, &config.AgentLimitsConfig{
		MaxTotalAgents:        2, // Small limit for testing
		MaxSubAgentsPerParent: 3,
	})

	// Create registry with mocked config
	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	// Create mock agents
	agent1 := mocks.NewMockAgent(ctrl)
	agent2 := mocks.NewMockAgent(ctrl)
	agent3 := mocks.NewMockAgent(ctrl) // This one should exceed the limit

	// Create configs
	config1 := &shared.AgentConfig{
		ID:           uuid.New(),
		SystemPrompt: "Agent 1",
		Role:         "Role 1",
	}
	config2 := &shared.AgentConfig{
		ID:           uuid.New(),
		SystemPrompt: "Agent 2",
		Role:         "Role 2",
	}
	config3 := &shared.AgentConfig{
		ID:           uuid.New(),
		SystemPrompt: "Agent 3",
		Role:         "Role 3",
	}

	// Register first two agents - should succeed
	err = registry.Register(agent1, config1)
	assert.NoError(t, err)

	err = registry.Register(agent2, config2)
	assert.NoError(t, err)

	// Try to register third agent - should fail due to total limit
	err = registry.Register(agent3, config3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum agent limit reached")
}

func TestAgentRegistry_Register_SubAgentLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjectorWithLimits(ctrl, &config.AgentLimitsConfig{
		MaxTotalAgents:        10,
		MaxSubAgentsPerParent: 2, // Small limit for testing
	})

	// Create registry with mocked config
	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	// Create parent agent
	parentAgent := mocks.NewMockAgent(ctrl)
	parentID := uuid.New()
	parentConfig := &shared.AgentConfig{
		ID:           parentID,
		SystemPrompt: "Parent Agent",
		Role:         "Parent",
	}

	// Register parent - should succeed
	err = registry.Register(parentAgent, parentConfig)
	assert.NoError(t, err)

	// Create subagents
	subAgent1 := mocks.NewMockAgent(ctrl)
	subAgent2 := mocks.NewMockAgent(ctrl)
	subAgent3 := mocks.NewMockAgent(ctrl) // This one should exceed the limit

	subConfig1 := &shared.AgentConfig{
		ID:           uuid.New(),
		ParentID:     &parentID,
		SystemPrompt: "Sub Agent 1",
		Role:         "Sub Role 1",
	}
	subConfig2 := &shared.AgentConfig{
		ID:           uuid.New(),
		ParentID:     &parentID,
		SystemPrompt: "Sub Agent 2",
		Role:         "Sub Role 2",
	}
	subConfig3 := &shared.AgentConfig{
		ID:           uuid.New(),
		ParentID:     &parentID,
		SystemPrompt: "Sub Agent 3",
		Role:         "Sub Role 3",
	}

	// Register first two subagents - should succeed
	err = registry.Register(subAgent1, subConfig1)
	assert.NoError(t, err)

	err = registry.Register(subAgent2, subConfig2)
	assert.NoError(t, err)

	// Try to register third subagent - should fail due to sub-agent limit
	err = registry.Register(subAgent3, subConfig3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum sub-agent limit reached")
}

func TestAgentRegistry_GetSubAgentCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjectorWithLimits(ctrl, &config.AgentLimitsConfig{
		MaxTotalAgents:        10,
		MaxSubAgentsPerParent: 3,
	})

	// Create registry with mocked config
	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	// Create parent agent
	parentAgent := mocks.NewMockAgent(ctrl)
	parentID := uuid.New()
	parentConfig := &shared.AgentConfig{
		ID:           parentID,
		SystemPrompt: "Parent Agent",
		Role:         "Parent",
	}

	// Create another parent agent
	otherParentAgent := mocks.NewMockAgent(ctrl)
	otherParentID := uuid.New()
	otherParentConfig := &shared.AgentConfig{
		ID:           otherParentID,
		SystemPrompt: "Other Parent Agent",
		Role:         "Other Parent",
	}

	// Register parents
	err = registry.Register(parentAgent, parentConfig)
	assert.NoError(t, err)

	err = registry.Register(otherParentAgent, otherParentConfig)
	assert.NoError(t, err)

	// Initially, no subagents
	assert.Equal(t, 0, registry.GetSubAgentCount(parentID))
	assert.Equal(t, 0, registry.GetSubAgentCount(otherParentID))

	// Create and register some subagents
	for i := 0; i < 3; i++ {
		subAgent := mocks.NewMockAgent(ctrl)
		subConfig := &shared.AgentConfig{
			ID:           uuid.New(),
			ParentID:     &parentID,
			SystemPrompt: "Sub Agent",
			Role:         "Sub Role",
		}
		err = registry.Register(subAgent, subConfig)
		assert.NoError(t, err)
	}

	// Check subagent counts
	assert.Equal(t, 3, registry.GetSubAgentCount(parentID))
	assert.Equal(t, 0, registry.GetSubAgentCount(otherParentID))

	// Add a subagent to the other parent
	otherSubAgent := mocks.NewMockAgent(ctrl)
	otherSubConfig := &shared.AgentConfig{
		ID:           uuid.New(),
		ParentID:     &otherParentID,
		SystemPrompt: "Other Sub Agent",
		Role:         "Other Sub Role",
	}
	err = registry.Register(otherSubAgent, otherSubConfig)
	assert.NoError(t, err)

	// Check counts again
	assert.Equal(t, 3, registry.GetSubAgentCount(parentID))
	assert.Equal(t, 1, registry.GetSubAgentCount(otherParentID))
}

func TestAgentRegistry_GetTotalAgentCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjectorWithLimits(ctrl, &config.AgentLimitsConfig{
		MaxTotalAgents:        10,
		MaxSubAgentsPerParent: 3,
	})

	// Create registry with mocked config
	registry, err := NewAgentRegistry(injector)
	require.NoError(t, err)

	// Initially no agents
	assert.Equal(t, 0, registry.GetTotalAgentCount())

	// Register some agents
	for i := 0; i < 5; i++ {
		agent := mocks.NewMockAgent(ctrl)
		agentConfig := &shared.AgentConfig{
			ID:           uuid.New(),
			SystemPrompt: "Agent",
			Role:         "Role",
		}
		err = registry.Register(agent, agentConfig)
		assert.NoError(t, err)
	}

	// Check total count
	assert.Equal(t, 5, registry.GetTotalAgentCount())
}
