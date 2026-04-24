package executor

import (
	"fmt"
	"time"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/shared"
)

// executeCall executes a call step (sub-flow invocation).
func (p *flowExecutorImpl) executeCall(call *flows.Call, _ string) error {
	// Check if call has a condition
	if call.When != "" {
		// Evaluate the condition
		scope := p.ctx.buildScope()
		eval := NewEvaluator()
		result, err := eval.EvaluateExprTyped(call.When, scope)
		if err != nil {
			return fmt.Errorf("call condition evaluation failed: %w", err)
		}

		// Skip call if condition is false
		if boolVal, ok := result.(bool); !ok || !boolVal {
			// Call is skipped - this is not an error, just don't execute
			return nil
		}
	}

	// Look up the sub-flow
	subFlow, err := p.flowRegistry.GetFlow(call.Ref)
	if err != nil {
		return fmt.Errorf("flow lookup failed for %s: %w", call.Ref, err)
	}

	// Build input map from call.Input fields with bare notation resolution
	subInput := make(map[string]string)
	if call.Input != nil {
		for _, field := range call.Input.GetFields() {
			param := field.GetParam()
			if param != nil {
				// Resolve bare notation reference
				value := p.resolveAssignFrom(param.AssignFrom)
				subInput[param.Name] = value
			}
		}
	}

	// Create executor for sub-flow using the service
	subCtx := newContext(subFlow.Input, subFlow.Output, subFlow.Context, subInput)

	// Initialize computed fields for sub-flow
	if subFlow.Computed != nil {
		subCtx.SetComputedBlock(subFlow.Computed)
	}

	subExec := &flowExecutorImpl{
		flow:             subFlow,
		ctx:              subCtx,
		history:          NewExecutionHistory(),
		startTime:        time.Now(),
		bashToolProvider: p.bashToolProvider,
		extService:       p.extService,
		flowRegistry:     p.flowRegistry,
		hookManager:      p.hookManager,
		agentFactory:     p.agentFactory,
		mcpRegistry:      p.mcpRegistry,
		logService:       p.logService,
		strategyBuilder:  p.strategyBuilder,
	}

	// Execute the sub-flow
	result, err := subExec.Run()
	if err != nil {
		return fmt.Errorf("sub-flow execution failed: %w", err)
	}

	// Map output fields back using call.Output
	if call.Output != nil {
		for _, field := range call.Output.GetFields() {
			param := field.GetParam()
			if param != nil {
				// Get the value from sub-flow result
				// Call-Output assignTo specifies the parent context destination (scope.field format)
				// The source is param.Name (subflow output field), target is parsed from param.AssignTo
				sourceField := param.Name

				value, ok := result.Outputs[sourceField]
				if !ok {
					continue // Skip if field doesn't exist in sub-flow output
				}

				// Parse assignTo to get scope and field name
				targetScope, targetField, err := p.parseAssignTarget(param.AssignTo)
				if err != nil {
					return fmt.Errorf("invalid assignTo '%s': %w", param.AssignTo, err)
				}

				// Set the value in parent flow context based on scope
				switch targetScope {
				case flows.FlowVariableScopeOutput:
					if err := p.ctx.SetOutputField(targetField, shared.AnyToString(value)); err != nil {
						return fmt.Errorf("failed to set output field '%s': %w", targetField, err)
					}
				case flows.FlowVariableScopeContext:
					if err := p.ctx.SetContextField(targetField, shared.AnyToString(value)); err != nil {
						return fmt.Errorf("failed to set context field '%s': %w", targetField, err)
					}
				}
			}
		}
	}

	return nil
}
