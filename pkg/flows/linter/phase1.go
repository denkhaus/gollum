package linter

import (
	"github.com/denkhaus/gollum/pkg/flows"
)

// SchemaChecker validates flow schema
type SchemaChecker struct{}

// Check runs schema validation
func (s *SchemaChecker) Check(flow *flows.Flow, result *flows.LinterResult) {
	// Check mandatory input section
	if flow.Input == nil {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrMissingInput,
			Message: "flow must have <input> section",
		})
	}

	// Check mandatory output section
	if flow.Output == nil {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrMissingOutput,
			Message: "flow must have <output> section",
		})
	}

	// Check exactly one initial state
	hasInitial := false
	for _, state := range flow.States {
		if state.Initial {
			if hasInitial {
				result.Errors = append(result.Errors, flows.LinterError{
					Code:    flows.ErrNoInitialState,
					Message: "flow must have exactly one initial state",
				})
			}
			hasInitial = true
		}
	}

	if !hasInitial && len(flow.States) > 0 {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrNoInitialState,
			Message: "flow must have exactly one initial state",
		})
	}
}
