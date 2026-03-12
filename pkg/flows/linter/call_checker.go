package linter

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/parser"
)

// CallChecker validates that call input/output arguments match the called flow's interface
type CallChecker struct {
	Resolver *parser.Resolver
}

// NewCallChecker creates a new call checker
func NewCallChecker() *CallChecker {
	return &CallChecker{
		Resolver: parser.DefaultResolver(),
	}
}

// Check validates all call steps in the flow
func (c *CallChecker) Check(flowPath string, flow *flows.Flow, result *flows.LinterResult) {
	for _, state := range flow.States {
		for _, call := range state.Calls {
			c.checkCall(flowPath, call, result)
		}
	}
}

// checkCall validates a single call against its referenced flow
func (c *CallChecker) checkCall(flowPath string, call flows.Call, result *flows.LinterResult) {
	// Resolve the called flow
	resolvedPath, err := c.Resolver.ResolveCall(call.Ref, flowPath)
	if err != nil {
		// Call reference not found - this is caught by phase4, so just skip
		return
	}

	// Parse the called flow
	calledFlow, err := parser.Parse(resolvedPath)
	if err != nil {
		// Can't parse the called flow - will be caught elsewhere
		return
	}

	// Get called flow's input and output fields
	calledInputFields := calledFlow.Input.GetAllFields()
	calledOutputFields := calledFlow.Output.GetAllFields()

	// Build maps for quick lookup
	calledInputsByName := make(map[string]flows.FieldDef)
	for _, f := range calledInputFields {
		calledInputsByName[f.Name] = f
	}
	calledOutputsByName := make(map[string]flows.FieldDef)
	for _, f := range calledOutputFields {
		calledOutputsByName[f.Name] = f
	}

	// Build maps of call inputs/outputs
	callInputsByName := make(map[string]flows.CallField)
	for _, f := range call.Input {
		callInputsByName[f.Name] = f
	}
	callOutputsByName := make(map[string]flows.CallField)
	for _, f := range call.Output {
		callOutputsByName[f.Name] = f
	}

	// Check required inputs are provided
	for _, inputField := range calledInputFields {
		_, provided := callInputsByName[inputField.Name]
		if !provided {
			// Check if input has default value
			if inputField.Default != "" {
				// Has default, so not required in call
				continue
			}
			// Missing required input
			result.Warnings = append(result.Warnings, flows.LinterError{
				FlowPath: flowPath,
				Code:     flows.ErrCallInputMissing,
				Message:  fmt.Sprintf("call to '%s': required input parameter '%s' (type=%s) has no default value", call.Ref, inputField.Name, inputField.Type),
			})
		}
	}

	// Check required outputs are captured
	for _, outputField := range calledOutputFields {
		_, captured := callOutputsByName[outputField.Name]
		if !captured {
			// Check if output has default value
			if outputField.Default != "" {
				// Has default, so not required in call
				continue
			}
			// Missing required output
			result.Warnings = append(result.Warnings, flows.LinterError{
				FlowPath: flowPath,
				Code:     flows.ErrCallOutputMissing,
				Message:  fmt.Sprintf("call to '%s': required output parameter '%s' (type=%s) has no default value", call.Ref, outputField.Name, outputField.Type),
			})
		}
	}

	// Check for extra inputs (not defined in called flow)
	for _, callInput := range call.Input {
		_, exists := calledInputsByName[callInput.Name]
		if !exists {
			result.Warnings = append(result.Warnings, flows.LinterError{
				FlowPath: flowPath,
				Code:     flows.ErrCallExtraInput,
				Message:  fmt.Sprintf("call to '%s': input parameter '%s' not defined in called flow's input block", call.Ref, callInput.Name),
			})
		}
	}

	// Check for extra outputs (not defined in called flow)
	for _, callOutput := range call.Output {
		_, exists := calledOutputsByName[callOutput.Name]
		if !exists {
			result.Warnings = append(result.Warnings, flows.LinterError{
				FlowPath: flowPath,
				Code:     flows.ErrCallExtraOutput,
				Message:  fmt.Sprintf("call to '%s': output parameter '%s' not defined in called flow's output block", call.Ref, callOutput.Name),
			})
		}
	}
}
