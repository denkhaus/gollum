package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecutionHistory_TracksStateTransitions(t *testing.T) {
	hist := NewExecutionHistory()

	hist.RecordStateEntry("init", time.Now())
	time.Sleep(10 * time.Millisecond)
	hist.RecordStateExit("init", time.Now())
	hist.RecordStateEntry("done", time.Now())

	states := hist.GetStates()
	assert.Len(t, states, 2)
	assert.Equal(t, "init", states[0].Name)
	assert.True(t, states[0].Duration > 0)
}

func TestExecutionHistory_TracksErrors(t *testing.T) {
	hist := NewExecutionHistory()

	hist.RecordError("test-step", "llm", "timeout error", time.Now())

	errors := hist.GetErrors()
	assert.Len(t, errors, 1)
	assert.Equal(t, "timeout error", errors[0].Message)
}
