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

// MCPError represents an error during MCP step execution
type MCPError struct {
	Server string
	Tool   string
	Step   string
	Err    error
}

func (e *MCPError) Error() string {
	return fmt.Sprintf("MCP step '%s' failed: server=%s tool=%s: %v", e.Step, e.Server, e.Tool, e.Err)
}

func (e *MCPError) Unwrap() error {
	return e.Err
}

// captureError captures error information and sets error context
func (p *flowExecutorImpl) captureError(step *flows.Step, errMsg string, exitCode int) {
	now := time.Now()

	// Set error context
	p.ctx.SetError(&ErrorContext{
		StepName:  step.Name,
		StepType:  step.Type,
		Message:   errMsg,
		ExitCode:  exitCode,
		Timestamp: now,
	})

	// Record in history
	p.history.RecordError(step.Name, step.Type, errMsg, now)
}

// handleError with on-error transition support
func (p *flowExecutorImpl) handleErrorWithErrorTransition(err error, step *flows.Step, state *flows.State) error {
	if step != nil && step.OnError != nil {
		// Capture error context (exit code 0 for generic errors)
		p.captureError(step, err.Error(), 0)

		// Log error transition (with nil check for test scenarios)
		if p.logService != nil {
			p.logService.ErrorWithFlowStep(
				fmt.Sprintf("Error in state '%s', transitioning to error state '%s': %v", state.Name, step.OnError.State, err),
				p.flow.Name, state.Name, step.Type)
		}

		// Transition to error state
		return p.transitionTo(step.OnError.State)
	}

	// No error handler - fail
	return fmt.Errorf("unhandled error in state %s: %w", state.Name, err)
}
