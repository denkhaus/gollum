package linter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/ast"
)

// varRefRegex matches ${variable.reference} patterns
var varRefRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// validPrefixes are the allowed variable reference prefixes
var validPrefixes = map[string]bool{
	"input":   true,
	"output":  true,
	"context": true,
	"error":   true,
}

// ExpressionChecker validates expressions
type ExpressionChecker struct{}

// Check runs expression validation
func (e *ExpressionChecker) Check(flow *flows.Flow, result *flows.LinterResult) {
	// Build context map of available fields
	availableFields := e.buildFieldMap(flow)

	// Validate computed field expressions
	if flow.Context != nil {
		for _, computed := range flow.Context.Computeds {
			expr, err := ast.ParseExpression(computed.When)
			if err != nil {
				result.Errors = append(result.Errors, flows.LinterError{
					Code:    flows.ErrInvalidExpr,
					Message: err.Error(),
					Context: computed.When,
				})
				continue
			}

			// Validate field references exist
			if err := e.validateFields(expr, availableFields); err != nil {
				result.Errors = append(result.Errors, flows.LinterError{
					Code:    flows.ErrFieldNotFound,
					Message: err.Error(),
					Context: computed.When,
				})
			}
		}
	}

	// Validate transition conditions
	for _, state := range flow.States {
		for _, trans := range state.Transitions {
			if trans.When != "" {
				expr, err := ast.ParseExpression(trans.When)
				if err != nil {
					result.Errors = append(result.Errors, flows.LinterError{
						Code:    flows.ErrInvalidExpr,
						Message: err.Error(),
						Context: trans.When,
					})
					continue
				}

				if err := e.validateFields(expr, availableFields); err != nil {
					result.Errors = append(result.Errors, flows.LinterError{
						Code:    flows.ErrFieldNotFound,
						Message: err.Error(),
						Context: trans.When,
					})
				}
			}
		}

		// Validate call input/output field references
		for _, call := range state.Calls {
			for _, field := range call.Input {
				e.checkVarRefs(field.Value, "call input", result)
			}
			for _, field := range call.Output {
				e.checkVarRefs(field.Value, "call output", result)
			}
		}

		// Validate step parameters and prompts
		for _, step := range state.Steps {
			for _, param := range step.Params {
				e.checkVarRefs(param.Value, "step param", result)
			}
			if step.Prompt != "" {
				e.checkVarRefs(step.Prompt, "step prompt", result)
			}
		}
	}
}

// checkVarRefs validates that all variable references have absolute paths
func (e *ExpressionChecker) checkVarRefs(text, location string, result *flows.LinterResult) {
	matches := varRefRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		ref := match[0]  // Full match ${...}
		path := match[1] // Content inside ${...}

		// Check if the path has a valid prefix
		parts := strings.SplitN(path, ".", 2)
		if len(parts) < 2 || !validPrefixes[parts[0]] {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:    flows.ErrRelativePath,
				Message: fmt.Sprintf("variable reference %s in %s must use absolute path (e.g., ${context.%s} or ${output.%s})", ref, location, path, path),
				Context: text,
			})
		}
	}
}

// buildFieldMap builds a map of available fields for validation
func (e *ExpressionChecker) buildFieldMap(flow *flows.Flow) map[string][]string {
	fields := make(map[string][]string)

	// Input fields
	if flow.Input != nil {
		for _, f := range flow.Input.GetAllFields() {
			fields["input"] = append(fields["input"], f.Name)
		}
	}

	// Output fields
	if flow.Output != nil {
		for _, f := range flow.Output.GetAllFields() {
			fields["output"] = append(fields["output"], f.Name)
		}
	}

	// Context fields
	if flow.Context != nil {
		for _, f := range flow.Context.Strings {
			fields["context"] = append(fields["context"], f.Name)
		}
		for _, f := range flow.Context.Ints {
			fields["context"] = append(fields["context"], f.Name)
		}
		for _, f := range flow.Context.Bools {
			fields["context"] = append(fields["context"], f.Name)
		}
		for _, f := range flow.Context.Floats {
			fields["context"] = append(fields["context"], f.Name)
		}
		// Add nested object fields
		for _, obj := range flow.Context.Objects {
			fields["context"] = append(fields["context"], obj.Name)
		}
	}

	return fields
}

// validateFields checks that all field references exist
func (e *ExpressionChecker) validateFields(expr ast.Expr, available map[string][]string) error {
	// Walk the AST and check FieldRef nodes
	return checkFieldRefs(expr, available)
}

func checkFieldRefs(expr ast.Expr, available map[string][]string) error {
	switch e := expr.(type) {
	case *ast.CallExpr:
		for _, arg := range e.Args {
			if err := checkFieldRefs(arg, available); err != nil {
				return err
			}
		}
	case *ast.FieldRef:
		// Check prefix exists
		fields, ok := available[e.Prefix]
		if !ok || e.Prefix == "" {
			return nil // Skip bare identifiers or missing prefixes for now
		}
		// For now, just check first level path exists
		// Full nested checking would require more complex logic
		if len(e.Path) > 0 {
			found := false
			for _, f := range fields {
				if f == e.Path[0] {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("field '%s' not found in %s", e.Path[0], e.Prefix)
			}
		}
	}
	return nil
}
