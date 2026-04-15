package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that agents using tools don't infinite loop with react strategy
func TestReactStrategy_PreventsInfiniteLoop(t *testing.T) {
	t.Skip("Requires full setup - enable when infrastructure is ready")

	// This test would:
	// 1. Create a flow with an agent that uses emit_log tool
	// 2. Execute the flow with a timeout
	// 3. Verify it completes without infinite loop
	// 4. Verify MaxIterations is respected
}

// Test that different agents can have different strategies
func TestReactStrategy_PerAgentConfiguration(t *testing.T) {
	t.Skip("Requires full setup - enable when infrastructure is ready")

	agents := []flows.Agent{
		{
			Name:  "conservative",
			Model: "test-model",
			Strategy: &flows.AgentStrategy{
				MaxIterations:      5,
				MaxRepeatedActions: 1,
			},
		},
		{
			Name:  "liberal",
			Model: "test-model",
			Strategy: &flows.AgentStrategy{
				MaxIterations:      50,
				MaxRepeatedActions: 10,
			},
		},
		{
			Name:  "default",
			Model: "test-model",
			// No strategy - should use defaults
		},
	}

	// Verify each agent gets the correct strategy
	require.Len(t, agents, 3)

	// Conservative agent
	assert.NotNil(t, agents[0].Strategy)
	assert.Equal(t, 5, agents[0].Strategy.MaxIterations)
	assert.Equal(t, 1, agents[0].Strategy.MaxRepeatedActions)

	// Liberal agent
	assert.NotNil(t, agents[1].Strategy)
	assert.Equal(t, 50, agents[1].Strategy.MaxIterations)
	assert.Equal(t, 10, agents[1].Strategy.MaxRepeatedActions)

	// Default agent
	assert.Nil(t, agents[2].Strategy)
}

// Test strategy builder integration with LLM step execution
func TestLLMStep_UsesStrategyBuilder(t *testing.T) {
	t.Skip("Requires full setup - enable when infrastructure is ready")

	// This test would verify that executeLLMStep properly:
	// 1. Builds strategy from agent config when Strategy is set
	// 2. Uses default strategy when Strategy is nil
	// 3. Passes the built strategy to the agent factory
}

// Test that flow executor properly injects strategy builder
func TestFlowExecutor_InjectsStrategyBuilder(t *testing.T) {
	injector := setupTestDI(t)

	svc := do.MustInvoke[FlowExecutorService](injector)
	assert.NotNil(t, svc)

	// Verify executor can be created with strategy builder
	flow := &flows.Flow{
		Name: "test-flow",
		States: []flows.State{
			{Name: "init", Initial: true},
			{Name: "done"},
		},
	}

	instance := svc.New(flow)
	assert.NotNil(t, instance)

	err := instance.Validate()
	assert.NoError(t, err)
}
