package executor

import (
	"fmt"
	"time"

	"github.com/denkhaus/gollum/pkg/flows"
)

// FuncError represents an error during func step execution
type FuncError struct {
	Function string
	Step     string
	Err      error
}

func (e *FuncError) Error() string {
	return fmt.Sprintf("func step '%s' in step '%s': %v", e.Function, e.Step, e.Err)
}

func (e *FuncError) Unwrap() error {
	return e.Err
}

// ErrorContext holds error lifecycle information
type ErrorContext struct {
	StepName  string
	StepType  string
	Message   string
	Timestamp time.Time
}

// captureError captures error information and sets context fields
func (e *Executor) captureError(step *flows.Step, errMsg string) {
	now := time.Now()

	// Set error context fields
	e.ctx.SetContextField("error.step_name", step.Name)
	e.ctx.SetContextField("error.step_type", step.Type)
	e.ctx.SetContextField("error.message", errMsg)
	e.ctx.SetContextField("error.timestamp", now.Format(time.RFC3339))

	// Record in history
	e.history.RecordError(step.Name, step.Type, errMsg, now)
}

// handleError with on-error transition support
func (e *Executor) handleError(err error, step *flows.Step, state *flows.State) error {
	if step != nil && step.OnError != nil {
		// Capture error context
		e.captureError(step, err.Error())

		// Transition to error state
		return e.transitionTo(step.OnError.State)
	}

	// No error handler - fail
	return fmt.Errorf("unhandled error in state %s: %w", state.Name, err)
}
