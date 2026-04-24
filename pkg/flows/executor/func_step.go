package executor

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
)

// executeFuncStep executes a function step by calling the Yaegi interpreter.
func (p *flowExecutorImpl) executeFuncStep(_ context.Context, step *flows.Step, stateName string) error {

	// Use Yaegi runner from extension service for other functions
	funcRunner := p.extService.GetFuncRunner()

	// Build args by resolving bare notation references
	args := make(map[string]any)
	for _, param := range step.Params {
		value := p.resolveAssignFrom(param.AssignFrom)
		args[param.Name] = value
	}

	// Execute via Yaegi runner
	result, err := funcRunner.ExecuteFunc(step.Function, args)
	if err != nil {
		return &FuncError{
			Function: step.Function,
			Step:     stateName,
			Err:      err,
		}
	}

	// Map result to output
	if step.Result != nil && step.Result.AssignTo != "" {
		scope, fieldName, err := p.parseAssignTarget(step.Result.AssignTo)
		if err != nil {
			return fmt.Errorf("invalid assignTo: %w", err)
		}
		if scope == flows.FlowVariableScopeContext {
			if err := p.ctx.SetContextField(fieldName, result); err != nil {
				return fmt.Errorf("failed to set context field '%s': %w", fieldName, err)
			}
		} else {
			if err := p.ctx.SetOutputField(fieldName, result); err != nil {
				return fmt.Errorf("failed to set output field '%s': %w", fieldName, err)
			}
		}
	}

	return nil
}
