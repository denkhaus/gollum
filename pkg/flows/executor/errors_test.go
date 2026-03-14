package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestExecutor_CaptureError_SetsErrorContextFields(t *testing.T) {
	flow := &flows.Flow{
		Name: "test-flow",
		States: []flows.State{
			{
				Name: "init", Initial: true,
				Steps: []flows.Step{
					{Type: "llm", Name: "test-step", OnError: &flows.OnErrorTransition{State: "error"}},
				},
			},
			{Name: "error"},
		},
	}

	exec := NewExecutor(flow)
	_ = exec.captureError(&flows.Step{Type: "llm", Name: "test-step"}, "test error")

	// Check error context fields are set
	val, err := exec.ctx.GetContextField("error.step_name")
	assert.NoError(t, err)
	assert.Equal(t, "test-step", val)

	val, err = exec.ctx.GetContextField("error.message")
	assert.NoError(t, err)
	assert.Equal(t, "test error", val)

	val, err = exec.ctx.GetContextField("error.step_type")
	assert.NoError(t, err)
	assert.Equal(t, "llm", val)
}
