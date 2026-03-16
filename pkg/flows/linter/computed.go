package linter

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/variables"
)

// ComputedChecker validates computed field definitions
type ComputedChecker struct {
	parser *variables.ExpressionParser
}

// NewComputedChecker creates a new computed field checker
func NewComputedChecker() *ComputedChecker {
	return &ComputedChecker{
		parser: variables.NewExpressionParser(),
	}
}

// Check runs computed field validation
func (c *ComputedChecker) Check(flow *flows.Flow, result *flows.LinterResult) {
	// Skip validation of top-level computed block if empty
	if flow.Computed == nil || len(flow.Computed.GetAllFields()) == 0 {
		return
	}

	// Build field registry
	registry := c.buildFieldRegistry(flow)

	// Check each computed field
	for _, field := range flow.Computed.GetAllFields() {
		c.checkField(field, registry, result)
	}

	// Check for circular dependencies
	c.checkCircularDependencies(flow.Computed.GetAllFields(), registry, result)
}

// checkField validates a single computed field
func (c *ComputedChecker) checkField(field flows.ComputedFieldDef, registry map[string]bool, result *flows.LinterResult) {
	// Check name is not empty
	if field.Name == "" {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrInvalidExpr,
			Message: "computed field must have a name attribute",
		})
		return
	}

	// Check type is valid
	validTypes := map[string]bool{
		"bool": true, "int": true, "string": true, "float": true,
	}
	if !validTypes[string(field.Type)] {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrInvalidExpr,
			Message: fmt.Sprintf("computed field '%s' has invalid type '%s'", field.Name, field.Type),
		})
	}

	// Check eval expression is present
	if field.Eval == "" {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrInvalidExpr,
			Message: fmt.Sprintf("computed field '%s' must have an eval expression", field.Name),
		})
		return
	}

	// Validate expression syntax
	_, err := c.parser.Parse(field.Eval)
	if err != nil {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrInvalidExpr,
			Message: fmt.Sprintf("computed field '%s' has invalid expression: %s", field.Name, err.Error()),
		})
		return
	}

	// Check field references
	deps := c.parser.ExtractDependencies(field.Eval)
	for _, dep := range deps {
		refKey := dep.Scope + "." + dep.Name
		if !registry[refKey] {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:    flows.ErrFieldNotFound,
				Message: fmt.Sprintf("computed field '%s' references undefined field '%s'", field.Name, refKey),
			})
		}
	}

	// Check for self-reference
	for _, dep := range deps {
		if dep.Scope == "computed" && dep.Name == field.Name {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:    flows.ErrCircularDeps,
				Message: fmt.Sprintf("computed field '%s' cannot reference itself", field.Name),
			})
		}
	}
}

// checkCircularDependencies detects circular dependencies in computed fields
func (c *ComputedChecker) checkCircularDependencies(fields []flows.ComputedFieldDef, registry map[string]bool, result *flows.LinterResult) {
	// Build dependency graph
	graph := make(map[string][]string)
	for _, field := range fields {
		deps := c.parser.ExtractDependencies(field.Eval)
		for _, dep := range deps {
			if dep.Scope == "computed" {
				graph[field.Name] = append(graph[field.Name], dep.Name)
			}
		}
	}

	// Detect cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for _, field := range fields {
		if !visited[field.Name] {
			if dfs(field.Name) {
				result.Errors = append(result.Errors, flows.LinterError{
					Code:    flows.ErrCircularDeps,
					Message: fmt.Sprintf("circular dependency detected involving computed field '%s'", field.Name),
				})
				return
			}
		}
	}
}

// buildFieldRegistry creates a map of all available fields
func (c *ComputedChecker) buildFieldRegistry(flow *flows.Flow) map[string]bool {
	registry := make(map[string]bool)

	// Input fields
	if flow.Input != nil {
		for _, f := range flow.Input.Strings {
			registry["input."+f.Name] = true
		}
		for _, f := range flow.Input.Ints {
			registry["input."+f.Name] = true
		}
		for _, f := range flow.Input.Bools {
			registry["input."+f.Name] = true
		}
		for _, f := range flow.Input.Floats {
			registry["input."+f.Name] = true
		}
		for _, f := range flow.Input.Arrays {
			registry["input."+f.Name] = true
		}
		for _, f := range flow.Input.Maps {
			registry["input."+f.Name] = true
		}
		for _, f := range flow.Input.Objects {
			registry["input."+f.Name] = true
		}
	}

	// Context fields
	if flow.Context != nil {
		for _, f := range flow.Context.Strings {
			registry["context."+f.Name] = true
		}
		for _, f := range flow.Context.Ints {
			registry["context."+f.Name] = true
		}
		for _, f := range flow.Context.Bools {
			registry["context."+f.Name] = true
		}
		for _, f := range flow.Context.Floats {
			registry["context."+f.Name] = true
		}
		for _, f := range flow.Context.Objects {
			registry["context."+f.Name] = true
		}
	}

	// Output fields
	if flow.Output != nil {
		for _, f := range flow.Output.Strings {
			registry["output."+f.Name] = true
		}
		for _, f := range flow.Output.Ints {
			registry["output."+f.Name] = true
		}
		for _, f := range flow.Output.Bools {
			registry["output."+f.Name] = true
		}
		for _, f := range flow.Output.Floats {
			registry["output."+f.Name] = true
		}
		for _, f := range flow.Output.Objects {
			registry["output."+f.Name] = true
		}
	}

	// Computed fields
	if flow.Computed != nil {
		for _, f := range flow.Computed.GetAllFields() {
			registry["computed."+f.Name] = true
		}
	}

	return registry
}
