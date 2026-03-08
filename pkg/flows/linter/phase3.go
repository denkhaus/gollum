package linter

import (
	"github.com/denkhaus/gollum/pkg/flows"
)

// GraphChecker validates flow graph structure
type GraphChecker struct{}

// Check runs graph validation
func (g *GraphChecker) Check(flow *flows.Flow, result *flows.LinterResult) {
	if len(flow.States) == 0 {
		return
	}

	// Build state name set
	stateNames := make(map[string]bool)
	for _, state := range flow.States {
		stateNames[state.Name] = true
	}

	// Find initial state
	var initialState *flows.State
	for i := range flow.States {
		if flow.States[i].Initial {
			initialState = &flow.States[i]
			break
		}
	}

	if initialState == nil {
		// Already reported in phase 1
		return
	}

	// Find reachable states via BFS
	reachable := make(map[string]bool)
	queue := []string{initialState.Name}
	reachable[initialState.Name] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Find state and check its transitions
		for _, state := range flow.States {
			if state.Name == current {
				// Check explicit transitions
				for _, trans := range state.Transitions {
					// Validate target exists
					if !stateNames[trans.To] {
						result.Errors = append(result.Errors, flows.LinterError{
							Code:    flows.ErrInvalidTransition,
							Message: "transition to non-existent state: " + trans.To,
						})
					} else if !reachable[trans.To] {
						reachable[trans.To] = true
						queue = append(queue, trans.To)
					}
				}

				// Check on-error transitions from steps
				for _, step := range state.Steps {
					if step.OnError != nil && step.OnError.State != "" {
						// Validate target exists
						if !stateNames[step.OnError.State] {
							result.Errors = append(result.Errors, flows.LinterError{
								Code:    flows.ErrInvalidTransition,
								Message: "on-error transition to non-existent state: " + step.OnError.State,
							})
						} else if !reachable[step.OnError.State] {
							reachable[step.OnError.State] = true
							queue = append(queue, step.OnError.State)
						}
					}
				}

				// Check on-error transitions from calls
				for _, call := range state.Calls {
					if call.OnError != nil && call.OnError.State != "" {
						// Validate target exists
						if !stateNames[call.OnError.State] {
							result.Errors = append(result.Errors, flows.LinterError{
								Code:    flows.ErrInvalidTransition,
								Message: "on-error transition to non-existent state: " + call.OnError.State,
							})
						} else if !reachable[call.OnError.State] {
							reachable[call.OnError.State] = true
							queue = append(queue, call.OnError.State)
						}
					}
				}

				break
			}
		}
	}

	// Find unreachable states
	for _, state := range flow.States {
		if !reachable[state.Name] {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:    flows.ErrUnreachableState,
				Message: "state '" + state.Name + "' is unreachable",
			})
		}
	}
}
