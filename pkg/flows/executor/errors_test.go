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
	exec.captureError(&flows.Step{Type: "llm", Name: "test-step"}, "test error", 0)

	// Check error context is set via dedicated API
	errorCtx := exec.ctx.GetError()
	assert.NotNil(t, errorCtx)
	assert.Equal(t, "test-step", errorCtx.StepName)
	assert.Equal(t, "test error", errorCtx.Message)
	assert.Equal(t, "llm", errorCtx.StepType)
	assert.Equal(t, 0, errorCtx.ExitCode)
}
