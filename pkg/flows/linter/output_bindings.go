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
}

// checkBinding validates a single output binding
func (c *OutputBindingsChecker) checkBinding(flow *flows.Flow, field *flows.FieldDef, result *flows.LinterResult) {
	// Parse the 'from' reference
	sourceScope, sourceName, err := variables.ParseFieldReference(field.From)
	if err != nil {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:     "E005",
			Message:  fmt.Sprintf("invalid 'from' reference: %s", field.From),
			Line:     1,
			Column:   1,
		})
		return
	}

	// Validate scope
	if !variables.IsValidSourceScope(sourceScope) {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:     "E006",
			Message:  fmt.Sprintf("invalid scope '%s' in 'from' attribute (allowed: input, context, computed, output)", sourceScope),
			Line:     1,
			Column:   1,
		})
		return
	}

	// Check for circular dependencies in output->output references
	if sourceScope == "output" {
		if c.hasCycle(flow, field.Name, sourceName, make(map[string]bool)) {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:     "E008",
				Message:  fmt.Sprintf("circular dependency: output.%s -> output.%s", field.Name, sourceName),
				Line:     1,
				Column:   1,
			})
			return
		}
	}

	// Check referenced field exists
	if !c.fieldExists(flow, sourceScope, sourceName) {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:     "E007",
			Message:  fmt.Sprintf("field '%s.%s' does not exist", sourceScope, sourceName),
			Line:     1,
			Column:   1,
		})
		return
	}

	// Check for duplicate sources (warning)
	if c.hasDuplicateSource(flow.Output, field.From, field.Name) {
		result.Warnings = append(result.Warnings, flows.LinterError{
			Code:     "W003",
			Message:  fmt.Sprintf("multiple output fields map from '%s'", field.From),
			Line:     1,
			Column:   1,
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
