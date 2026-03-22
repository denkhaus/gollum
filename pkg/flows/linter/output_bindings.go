package linter

import (
	"fmt"

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
	// Get position of this output field by its 'from' attribute (more specific than name)
	line, col := 1, 1
	if c.PosTracker != nil {
		line, col = c.PosTracker.FindOutputFieldByFrom(field.From)
		if line == 0 {
			// Fallback to searching by name
			line, col = c.PosTracker.FindOutputFieldPosition(field.Name)
		}
		if line == 0 {
			line, col = 1, 1
		}
	}

	// Parse the 'from' reference
	sourceScope, sourceName, err := variables.ParseFieldReference(field.From)
	if err != nil {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:     "E005",
			Message:  fmt.Sprintf("invalid 'from' reference: %s", field.From),
			Line:     line,
			Column:   col,
		})
		return
	}

	// Validate scope
	if !variables.IsValidSourceScope(sourceScope) {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:     "E006",
			Message:  fmt.Sprintf("invalid scope '%s' in 'from' attribute (allowed: input, context, computed, output)", sourceScope),
			Line:     line,
			Column:   col,
		})
		return
	}

	// Check for circular dependencies in output->output references
	if sourceScope == "output" {
		if c.hasCycle(flow, field.Name, sourceName, make(map[string]bool)) {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:     "E008",
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
			Code:     "E007",
			Message:  fmt.Sprintf("field '%s.%s' does not exist", sourceScope, sourceName),
			Line:     line,
			Column:   col,
		})
		return
	}

	// Check for duplicate sources (warning)
	if c.hasDuplicateSource(flow.Output, field.From, field.Name) {
		result.Warnings = append(result.Warnings, flows.LinterError{
			Code:     "W003",
			Message:  fmt.Sprintf("multiple output fields map from '%s'", field.From),
			Line:     line,
			Column:   col,
		})
	}
}

// fieldExists checks if a field exists in the specified scope
func (c *OutputBindingsChecker) fieldExists(flow *flows.Flow, scope, name string) bool {
	switch scope {
	case "input":
		return flow.Input != nil && flow.Input.HasField(name)
	case "context":
		return flow.Context != nil && flow.Context.HasField(name)
	case "computed":
		return flow.Computed != nil && flow.Computed.HasField(name)
	case "output":
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

	// Find current field and check if it has 'from'
	field := flow.Output.GetField(current)
	if field == nil || field.From == "" {
		return false
	}

	scope, name, _ := variables.ParseFieldReference(field.From)
	if scope == "output" {
		return c.hasCycle(flow, name, target, visited)
	}

	return false
}

// hasDuplicateSource checks if multiple outputs map from the same source
func (c *OutputBindingsChecker) hasDuplicateSource(output *flows.OutputBlock, source, excludeField string) bool {
	count := 0
	for _, f := range output.GetDeclarative() {
		if f.From == source && f.Name != excludeField {
			count++
		}
		if count > 0 {
			return true
		}
	}
	return false
}

// checkStepOutputAssigns validates output assign attributes in steps
func (c *OutputBindingsChecker) checkStepOutputAssigns(flow *flows.Flow, result *flows.LinterResult) {
	for _, state := range flow.States {
		for _, step := range state.Steps {
			if step.Output != nil && step.Output.Assign != "" {
				c.checkStepOutputAssign(flow, &step, &state, result)
			}
		}
	}
}

// checkStepOutputAssign validates a single step output assign attribute
func (c *OutputBindingsChecker) checkStepOutputAssign(flow *flows.Flow, step *flows.Step, state *flows.State, result *flows.LinterResult) {
	assignValue := step.Output.Assign

	// Check for ${} syntax - should NOT be used in assign attributes
	if len(assignValue) > 2 && assignValue[0] == '$' && assignValue[1] == '{' && assignValue[len(assignValue)-1] == '}' {
		line, col := 1, 1
		if c.PosTracker != nil {
			line, col = c.PosTracker.FindStepOutputPosition(state.Name, step.Name)
		}

		result.Errors = append(result.Errors, flows.LinterError{
			Code:     "E009",
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

		result.Errors = append(result.Errors, flows.LinterError{
			Code:     "E005",
			Message:  fmt.Sprintf("invalid assign reference: %s", assignValue),
			Line:     line,
			Column:   col,
		})
		return
	}

	// Validate scope - for step output, should be 'output' only
	if sourceScope != "output" {
		line, col := 1, 1
		if c.PosTracker != nil {
			line, col = c.PosTracker.FindStepOutputPosition(state.Name, step.Name)
		}

		result.Errors = append(result.Errors, flows.LinterError{
			Code:     "E010",
			Message:  fmt.Sprintf("invalid scope '%s' in step output assign - must be 'output.field' (e.g., 'output.result')", sourceScope),
			Line:     line,
			Column:   col,
		})
		return
	}

	// Check referenced output field exists
	if flow.Output != nil && !flow.Output.HasField(sourceName) {
		line, col := 1, 1
		if c.PosTracker != nil {
			line, col = c.PosTracker.FindStepOutputPosition(state.Name, step.Name)
		}

		result.Errors = append(result.Errors, flows.LinterError{
			Code:     "E007",
			Message:  fmt.Sprintf("output field '%s' does not exist", sourceName),
			Line:     line,
			Column:   col,
		})
	}
}
