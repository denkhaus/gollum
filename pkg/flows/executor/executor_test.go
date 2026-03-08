package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestNewExecutor_CreatesExecutorWithFlow(t *testing.T) {
	flow := &flows.Flow{
		Name: "test-flow",
		States: []flows.State{
			{Name: "init", Initial: true},
			{Name: "done"},
		},
	}

	exec := NewExecutor(flow)

	assert.NotNil(t, exec)
	assert.Equal(t, "test-flow", exec.flow.Name)
	assert.Equal(t, "init", exec.currentState)
}

func TestExecutor_Validate_ReturnsErrorForInvalidFlow(t *testing.T) {
	flow := &flows.Flow{
		Name: "invalid-flow",
		States: []flows.State{
			{Name: "init"}, // No initial state
		},
	}

	exec := NewExecutor(flow)
	err := exec.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no initial state")
}
