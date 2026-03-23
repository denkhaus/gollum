package linter

import (
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/variables"
)

// OutputBindingsChecker validates output field bindings
type OutputBindingsChecker struct {
	PosTracker *PositionTracker
}

// NewOutputBindingsChecker creates a new output bindings checker
func NewOutputBindingsChecker() *OutputBindingsChecker {
	return &OutputBindingsChecker{}
}

// Check validates all output field bindings in the flow
func (c *OutputBindingsChecker) Check(flow *flows.Flow, result *flows.LinterResult) {
	if flow.Output == nil {
		return
	}

	for _, field := range flow.Output.GetDeclarative() {
		c.checkBinding(flow, &field, result)
	}

	// Also validate step output assign attributes
	c.checkStepOutputAssigns(flow, result)
}

// checkBinding validates a single output binding
func (c *OutputBindingsChecker) checkBinding(flow *flows.Flow, field *flows.FieldDef, result *flows.LinterResult) {
	// Get position of this output field by its 'assignFrom' attribute (more specific than name)
	line, col := 1, 1
	if c.PosTracker != nil {
		line, col = c.PosTracker.FindOutputFieldByFrom(field.AssignFrom)
		if line == 0 {
			// Fallback to searching by name
			line, col = c.PosTracker.FindOutputFieldPosition(field.Name)
		}
		if line == 0 {
			line, col = 1, 1
		}
	}

	// Check for ${} syntax - should NOT be used in assignFrom attribute
	if len(field.AssignFrom) > 2 && field.AssignFrom[0] == '$' && field.AssignFrom[1] == '{' && field.AssignFrom[len(field.AssignFrom)-1] == '}' {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:     flows.ErrTemplateNotation,
			Message:  fmt.Sprintf("invalid assignFrom syntax '%s' - use 'scope.field' without ${} (e.g., 'input.value' not '${input.value}')", field.AssignFrom),
			Line:     line,
			Column:   col,
		})
		return
	}

	// Parse the 'assignFrom' reference
	sourceScope, sourceName, err := variables.ParseFieldReference(field.AssignFrom)
	if err != nil {
		// Check if it's a missing scope prefix error (no dot separator)
		if !strings.Contains(field.AssignFrom, ".") {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:     flows.ErrMissingScope,
				Message:  fmt.Sprintf("invalid assignFrom reference '%s' - must use scope.field notation (e.g., 'output.result' not 'result')", field.AssignFrom),
				Line:     line,
				Column:   col,
			})
		} else {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:     flows.ErrInvalidRef,
				Message:  fmt.Sprintf("invalid 'assignFrom' reference: %s", field.AssignFrom),
				Line:     line,
				Column:   col,
			})
		}
		return
	}

	// Validate scope
	if err := sourceScope.Validate(); err != nil {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:     flows.ErrInvalidScope,
			Message:  fmt.Sprintf("invalid scope '%s' in 'assignFrom' attribute (allowed: input, context, computed, output)", sourceScope),
			Line:     line,
			Column:   col,
		})
		return
	}

	// Check for circular dependencies in output->output references
	if sourceScope == flows.FlowVariableScopeOutput {
		if c.hasCycle(flow, field.Name, sourceName, make(map[string]bool)) {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:     flows.ErrCircularDep,
				Message:  fmt.Sprintf("circular dependency: output.%s -> output.%s", field.Name, sourceName),
				Line:     line,
				Column:   col,
			})
			return
		}
	}

	// Check referenced field exists
	if !c.fieldExists(flow, sourceScope, sourceName) {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:     flows.ErrFieldNotFound,
			Message:  fmt.Sprintf("field '%s.%s' does not exist", sourceScope, sourceName),
			Line:     line,
			Column:   col,
		})
		return
	}

	// Check for duplicate sources (warning)
	if c.hasDuplicateSource(flow.Output, field.AssignFrom, field.Name) {
		result.Warnings = append(result.Warnings, flows.LinterError{
			Code:     "W003",
			Message:  fmt.Sprintf("multiple output fields map from '%s'", field.AssignFrom),
			Line:     line,
			Column:   col,
		})
	}
}

// fieldExists checks if a field exists in the specified scope
func (c *OutputBindingsChecker) fieldExists(flow *flows.Flow, scope flows.FlowVariableScope, name string) bool {
	switch scope {
	case flows.FlowVariableScopeInput:
		return flow.Input != nil && flow.Input.HasField(name)
	case flows.FlowVariableScopeContext:
		return flow.Context != nil && flow.Context.HasField(name)
	case flows.FlowVariableScopeComputed:
		return flow.Computed != nil && flow.Computed.HasField(name)
	case flows.FlowVariableScopeOutput:
		return flow.Output != nil && flow.Output.HasField(name)
	}
	return false
}

// hasCycle detects cycles in output->output references using DFS
func (c *OutputBindingsChecker) hasCycle(flow *flows.Flow, current, target string, visited map[string]bool) bool {
	if current == target {
		return true
	}
	if visited[current] {
		return false
	}
	visited[current] = true

	// Find current field and check if it has 'assignFrom'
	field := flow.Output.GetField(current)
	if field == nil || field.AssignFrom == "" {
		return false
	}

	scope, name, _ := variables.ParseFieldReference(field.AssignFrom)
	if scope == flows.FlowVariableScopeOutput {
		return c.hasCycle(flow, name, target, visited)
	}

	return false
}

// hasDuplicateSource checks if multiple outputs map from the same source
func (c *OutputBindingsChecker) hasDuplicateSource(output *flows.OutputBlock, source, excludeField string) bool {
	count := 0
	for _, f := range output.GetDeclarative() {
		if f.AssignFrom == source && f.Name != excludeField {
			count++
		}
		if count > 0 {
			return true
		}
	}
	return false
}

// checkStepOutputAssigns validates result assignTo attributes in steps
func (c *OutputBindingsChecker) checkStepOutputAssigns(flow *flows.Flow, result *flows.LinterResult) {
	for _, state := range flow.States {
		for _, step := range state.Steps {
			if step.Result != nil {
				// Check the assignTo attribute on the result element itself
				if step.Result.AssignTo != "" {
					c.checkStepOutputAssign(flow, &step, &state, result, step.Result.AssignTo)
				}
				// Check assignTo attributes on child path elements
				for _, path := range step.Result.Paths {
					if path.AssignTo != "" {
						c.checkStepOutputAssign(flow, &step, &state, result, path.AssignTo)
					}
				}
			}
		}
	}
}

// checkStepOutputAssign validates a single step result assignTo attribute
func (c *OutputBindingsChecker) checkStepOutputAssign(flow *flows.Flow, step *flows.Step, state *flows.State, result *flows.LinterResult, assignValue string) {

	// Check for ${} syntax - should NOT be used in assign attributes
	if len(assignValue) > 2 && assignValue[0] == '$' && assignValue[1] == '{' && assignValue[len(assignValue)-1] == '}' {
		line, col := 1, 1
		if c.PosTracker != nil {
			line, col = c.PosTracker.FindStepOutputPosition(state.Name, step.Name)
		}

		result.Errors = append(result.Errors, flows.LinterError{
			Code:     flows.ErrTemplateNotation,
			Message:  fmt.Sprintf("invalid assign syntax '%s' - use 'scope.field' without ${} (e.g., 'output.result' not '${output.result}')", assignValue),
			Line:     line,
			Column:   col,
		})
		return
	}

	// Parse the assign reference
	sourceScope, sourceName, err := variables.ParseFieldReference(assignValue)
	if err != nil {
		line, col := 1, 1
		if c.PosTracker != nil {
			line, col = c.PosTracker.FindStepOutputPosition(state.Name, step.Name)
		}

		// Check if it's a missing scope prefix error (no dot separator)
		if !strings.Contains(assignValue, ".") {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:     flows.ErrMissingScope,
				Message:  fmt.Sprintf("invalid assign reference '%s' - must use scope.field notation (e.g., 'output.result' not 'result')", assignValue),
				Line:     line,
				Column:   col,
			})
		} else {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:     flows.ErrInvalidRef,
				Message:  fmt.Sprintf("invalid assign reference: %s", assignValue),
				Line:     line,
				Column:   col,
			})
		}
		return
	}

	// Validate scope - for step output, can be 'output' or 'context'
	if sourceScope != flows.FlowVariableScopeOutput && sourceScope != flows.FlowVariableScopeContext {
		line, col := 1, 1
		if c.PosTracker != nil {
			line, col = c.PosTracker.FindStepOutputPosition(state.Name, step.Name)
		}

		result.Errors = append(result.Errors, flows.LinterError{
			Code:     flows.ErrMissingScope,
			Message:  fmt.Sprintf("invalid scope '%s' in step output assign - must be 'output.field' or 'context.field' (e.g., 'output.result' or 'context.temp')", sourceScope),
			Line:     line,
			Column:   col,
		})
		return
	}

	// Check referenced field exists in the appropriate scope
	if sourceScope == flows.FlowVariableScopeOutput {
		if flow.Output == nil || !flow.Output.HasField(sourceName) {
			line, col := 1, 1
			if c.PosTracker != nil {
				line, col = c.PosTracker.FindStepOutputPosition(state.Name, step.Name)
			}

			result.Errors = append(result.Errors, flows.LinterError{
				Code:     flows.ErrFieldNotFound,
				Message:  fmt.Sprintf("output field '%s' does not exist", sourceName),
				Line:     line,
				Column:   col,
			})
		}
	} else if sourceScope == flows.FlowVariableScopeContext {
		if flow.Context == nil || !flow.Context.HasField(sourceName) {
			line, col := 1, 1
			if c.PosTracker != nil {
				line, col = c.PosTracker.FindStepOutputPosition(state.Name, step.Name)
			}

			result.Errors = append(result.Errors, flows.LinterError{
				Code:     flows.ErrFieldNotFound,
				Message:  fmt.Sprintf("context field '%s' does not exist", sourceName),
				Line:     line,
				Column:   col,
			})
		}
	}
}
