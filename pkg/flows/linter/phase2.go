package linter

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/ast"
)

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
		// Note: nested objects not fully implemented yet
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
