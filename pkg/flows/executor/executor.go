package executor

import (
	"fmt"
	"time"

	"github.com/denkhaus/gollum/pkg/flows"
)

// Executor executes flow state machines
type Executor struct {
	flow        *flows.Flow
	ctx         *Context
	currentState string
	history     *ExecutionHistory
	startTime   time.Time
}

// NewExecutor creates a new executor
func NewExecutor(flow *flows.Flow) *Executor {
	exec := &Executor{
		flow:      flow,
		ctx:       NewContext(flow.Input, nil),
		history:   NewExecutionHistory(),
		startTime: time.Now(),
	}

	// Find and set initial state
	for _, state := range flow.States {
		if state.Initial {
			exec.currentState = state.Name
			break
		}
	}

	return exec
}

// SetInput sets input field values
func (e *Executor) SetInput(vals map[string]any) {
	e.ctx = NewContext(e.flow.Input, vals)
}

// Validate validates the flow before execution
func (e *Executor) Validate() error {
	// Check for initial state
	hasInitial := false
	for _, state := range e.flow.States {
		if state.Initial {
			hasInitial = true
			e.currentState = state.Name
			break
		}
	}

	if !hasInitial {
		return fmt.Errorf("flow %s: no initial state defined", e.flow.Name)
	}

	return nil
}

// Run executes the flow from the initial state
func (e *Executor) Run() error {
	defer e.history.Complete(time.Now())

	if err := e.Validate(); err != nil {
		return err
	}

	// Find initial state
	var initialState *flows.State
	for i := range e.flow.States {
		if e.flow.States[i].Initial {
			initialState = &e.flow.States[i]
			break
		}
	}

	if initialState == nil {
		return fmt.Errorf("no initial state found")
	}

	return e.executeState(initialState)
}

// executeState executes a single state
func (e *Executor) executeState(state *flows.State) error {
	// Record state entry
	e.history.RecordStateEntry(state.Name, time.Now())

	// Evaluate computed fields
	if e.flow.Context != nil {
		if err := e.ctx.EvaluateComputedFields(e.flow.Context); err != nil {
			return fmt.Errorf("computed field evaluation: %w", err)
		}
	}

	// Execute steps
	for _, step := range state.Steps {
		if err := e.executeStep(&step, state.Name); err != nil {
			return e.handleError(err, &step, state)
		}
	}

	// Execute calls
	for _, call := range state.Calls {
		if err := e.executeCall(&call, state.Name); err != nil {
			return e.handleError(err, nil, state)
		}
	}

	// Find and execute transition
	return e.executeTransition(state)
}

// executeTransition evaluates conditions and transitions to next state
func (e *Executor) executeTransition(state *flows.State) error {
	// Build evaluation scope
	scope := e.ctx.buildScope()

	for _, trans := range state.Transitions {
		if trans.Otherwise {
			// Fallback transition
			return e.transitionTo(trans.To)
		}

		if trans.When == "" {
			// Unconditional transition
			return e.transitionTo(trans.To)
		}

		// Evaluate condition
		eval := NewEvaluator()
		result, err := eval.EvaluateExpr(trans.When, scope)
		if err != nil {
			return fmt.Errorf("transition condition: %w", err)
		}

		if boolVal, ok := result.(bool); ok && boolVal {
			return e.transitionTo(trans.To)
		}
	}

	// No transition - terminal state
	e.history.RecordStateExit(state.Name, time.Now())
	return nil
}

// transitionTo transitions to a new state
func (e *Executor) transitionTo(stateName string) error {
	// Find target state
	var targetState *flows.State
	for i := range e.flow.States {
		if e.flow.States[i].Name == stateName {
			targetState = &e.flow.States[i]
			break
		}
	}

	if targetState == nil {
		return fmt.Errorf("state not found: %s", stateName)
	}

	e.currentState = stateName
	return e.executeState(targetState)
}

// handleError handles step execution errors
func (e *Executor) handleError(err error, step *flows.Step, state *flows.State) error {
	// TODO: Implement on-error transition handling
	return fmt.Errorf("step execution failed in state %s: %w", state.Name, err)
}

// executeStep executes a single step (placeholder)
func (e *Executor) executeStep(step *flows.Step, stateName string) error {
	// TODO: Implement step execution in next tasks
	return fmt.Errorf("step execution not implemented")
}

// executeCall executes a call step (placeholder)
func (e *Executor) executeCall(call *flows.Call, stateName string) error {
	// TODO: Implement call execution in next tasks
	return fmt.Errorf("call execution not implemented")
}
