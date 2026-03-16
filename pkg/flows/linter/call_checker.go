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
		// Call reference not found - report error
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrCallRefNotFound,
			Message: fmt.Sprintf("call reference '%s' not found: %v", call.Ref, err),
		})
		return
	}

	// Parse the called flow
	calledFlow, err := parser.Parse(resolvedPath)
	if err != nil {
		// Can't parse the called flow - report error
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrCallRefNotFound,
			Message: fmt.Sprintf("failed to parse called flow '%s' (resolved to %s): %v", call.Ref, resolvedPath, err),
		})
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
	callInputsByName := make(map[string]flows.CallInputField)
	if call.Input != nil {
		for _, f := range call.Input.GetFields() {
			if typed := f.GetTypedField(); typed != nil {
				callInputsByName[typed.Name] = f
			}
		}
	}
	callOutputsByName := make(map[string]flows.CallOutputField)
	if call.Output != nil {
		for _, f := range call.Output.GetFields() {
			if typed := f.GetTypedField(); typed != nil {
				callOutputsByName[typed.Name] = f
			}
		}
	}

	// Check required inputs are provided
	for _, inputField := range calledInputFields {
		callInput, provided := callInputsByName[inputField.Name]
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
			continue
		}

		// Check type compatibility
		callType := callInput.GetType()
		if callType != "" && callType != string(inputField.Type) {
			result.Errors = append(result.Errors, flows.LinterError{
				FlowPath: flowPath,
				Code:     flows.ErrCallInputType,
				Message:  fmt.Sprintf("call to '%s': input parameter '%s' type mismatch - call uses <%s>, but flow expects <%s>", call.Ref, inputField.Name, callType, inputField.Type),
			})
		}
	}

	// Check required outputs are captured
	for _, outputField := range calledOutputFields {
		callOutput, captured := callOutputsByName[outputField.Name]
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
			continue
		}

		// Check type compatibility
		callType := callOutput.GetType()
		if callType != "" && callType != string(outputField.Type) {
			result.Errors = append(result.Errors, flows.LinterError{
				FlowPath: flowPath,
				Code:     flows.ErrCallOutputType,
				Message:  fmt.Sprintf("call to '%s': output parameter '%s' type mismatch - call uses <%s>, but flow expects <%s>", call.Ref, outputField.Name, callType, outputField.Type),
			})
		}
	}

	// Check for extra inputs (not defined in called flow)
	if call.Input != nil {
		for _, callInput := range call.Input.GetFields() {
			typedField := callInput.GetTypedField()
			if typedField != nil {
				_, exists := calledInputsByName[typedField.Name]
				if !exists {
					result.Warnings = append(result.Warnings, flows.LinterError{
						FlowPath: flowPath,
						Code:     flows.ErrCallExtraInput,
						Message:  fmt.Sprintf("call to '%s': input parameter '%s' not defined in called flow's input block", call.Ref, typedField.Name),
					})
				}
			}
		}
	}

	// Check for extra outputs (not defined in called flow)
	if call.Output != nil {
		for _, callOutput := range call.Output.GetFields() {
			typedField := callOutput.GetTypedField()
			if typedField != nil {
				_, exists := calledOutputsByName[typedField.Name]
				if !exists {
					result.Warnings = append(result.Warnings, flows.LinterError{
						FlowPath: flowPath,
						Code:     flows.ErrCallExtraOutput,
						Message:  fmt.Sprintf("call to '%s': output parameter '%s' not defined in called flow's output block", call.Ref, typedField.Name),
					})
				}
			}
		}
	}
}
