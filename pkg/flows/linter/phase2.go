package linter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/ast"
)

var varRefRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// ExpressionChecker validates expressions in flows
type ExpressionChecker struct {
	PosTracker *PositionTracker
}

// NewExpressionChecker creates a new expression checker
func NewExpressionChecker() *ExpressionChecker {
	return &ExpressionChecker{}
}

// Check runs expression validation
func (e *ExpressionChecker) Check(flow *flows.Flow, result *flows.LinterResult) {
	// Build context map of available fields
	availableFields := e.buildFieldMap(flow)

	// Validate computed field expressions
	if flow.Computed != nil {
		for _, computed := range flow.Computed.GetAllFields() {
			expr, err := ast.ParseExpression(computed.Eval)
			if err != nil {
				result.Errors = append(result.Errors, flows.LinterError{
					Code:    flows.ErrInvalidExpr,
					Message: err.Error(),
					Context: computed.Eval,
				})
				continue
			}

			// Validate field references exist
			if err := e.validateFields(expr, availableFields); err != nil {
				result.Errors = append(result.Errors, flows.LinterError{
					Code:    flows.ErrFieldNotFound,
					Message: err.Error(),
				})
			}
		}
	}

	// Validate transition conditions
	for _, state := range flow.States {
		for _, trans := range state.Transitions {
			if trans.When != "" {
				e.checkVarRefs(trans.When, "transition", result)
				expr, err := ast.ParseExpression(trans.When)
				if err != nil {
					result.Errors = append(result.Errors, flows.LinterError{
						Code:    flows.ErrInvalidExpr,
						Message: err.Error(),
						Context: trans.When,
					})
				} else if err := e.validateFields(expr, availableFields); err != nil {
					result.Errors = append(result.Errors, flows.LinterError{
						Code:    flows.ErrFieldNotFound,
						Message: err.Error(),
					})
				}
			}
		}
	}

	// Validate call conditions
	for _, state := range flow.States {
		for _, call := range state.Calls {
			if call.When != "" {
				e.checkVarRefs(call.When, "call condition", result)
				expr, err := ast.ParseExpression(call.When)
				if err != nil {
					result.Errors = append(result.Errors, flows.LinterError{
						Code:    flows.ErrInvalidExpr,
						Message: err.Error(),
						Context: call.When,
					})
				} else if err := e.validateFields(expr, availableFields); err != nil {
					result.Errors = append(result.Errors, flows.LinterError{
						Code:    flows.ErrFieldNotFound,
						Message: err.Error(),
					})
				}
			}
		}
	}

	// Validate call input/output field references
	for _, state := range flow.States {
		for _, call := range state.Calls {
			if call.Input != nil {
				for _, field := range call.Input.GetFields() {
					if param := field.GetParam(); param != nil {
						e.checkAssignFrom(param.AssignFrom, "call input", result)
					}
				}
			}
			if call.Output != nil {
				for _, field := range call.Output.GetFields() {
					if param := field.GetParam(); param != nil {
						e.checkAssignTo(param.AssignTo, "call output", result)
					}
				}
			}
		}
	}

	// Validate step parameters and prompts
	for _, state := range flow.States {
		for _, step := range state.Steps {
			for _, param := range step.Params {
				e.checkAssignFrom(param.AssignFrom, "step param", result)
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
		if len(parts) < 2 || flows.FlowVariableScope(parts[0]).Validate() != nil {
			result.Errors = append(result.Errors, flows.LinterError{
				Code:    flows.ErrRelativePath,
				Message: fmt.Sprintf("variable reference %s in %s must use absolute path (e.g., ${context.%s} or ${output.%s})", ref, location, path, path),
				Context: text,
			})
		}
	}
}

// Valid scopes for assignFrom (data sources)
var validAssignFromScopes = map[flows.FlowVariableScope]bool{
	flows.FlowVariableScopeInput:    true,
	flows.FlowVariableScopeContext:  true,
	flows.FlowVariableScopeComputed: true,
	flows.FlowVariableScopeSys:      true,
}

// Valid scopes for assignTo (data destinations)
var validAssignToScopes = map[flows.FlowVariableScope]bool{
	flows.FlowVariableScopeOutput:  true,
	flows.FlowVariableScopeContext: true,
}

// checkAssignFrom validates assignFrom attribute
// - No ${} interpolation (bare notation only)
// - All values must be scope.field references (no literals allowed)
// - Valid scopes: input, context, computed, sys
func (e *ExpressionChecker) checkAssignFrom(ref, location string, result *flows.LinterResult) {
	// Check for ${} interpolation - not allowed in assignFrom
	if strings.Contains(ref, "${") || strings.Contains(ref, "}") {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrTemplateNotation,
			Message: fmt.Sprintf("assignFrom in %s must use bare notation (e.g., assignFrom=\"input.field\") instead of template notation (e.g., ${input.field})", location),
			Context: ref,
		})
		return
	}

	// All values must be scope.field references - check for dot
	if !strings.Contains(ref, ".") {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrMissingScope,
			Message: fmt.Sprintf("assignFrom in %s must reference a flow variable (e.g., input.field, context.field) - literal values are not allowed", location),
			Context: ref,
		})
		return
	}

	// Parse scope.field
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) < 2 {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrMissingScope,
			Message: fmt.Sprintf("assignFrom in %s must include scope prefix (e.g., input.field, context.field)", location),
			Context: ref,
		})
		return
	}

	scope := flows.FlowVariableScope(parts[0])
	// Validate scope
	if scope.Validate() != nil {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrInvalidScope,
			Message: fmt.Sprintf("invalid scope '%s' in assignFrom for %s", parts[0], location),
			Context: ref,
		})
		return
	}

	// Check if scope is valid for assignFrom
	if !validAssignFromScopes[scope] {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrInvalidAssignFromScope,
			Message: fmt.Sprintf("invalid scope '%s' in assignFrom for %s - allowed scopes are: input, context, computed, sys", parts[0], location),
			Context: ref,
		})
	}
}

// checkAssignTo validates assignTo attribute
// - No ${} interpolation (bare notation only)
// - Valid scopes: output, context
func (e *ExpressionChecker) checkAssignTo(ref, location string, result *flows.LinterResult) {
	// Check for ${} interpolation - not allowed in assignTo
	if strings.Contains(ref, "${") || strings.Contains(ref, "}") {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrTemplateNotation,
			Message: fmt.Sprintf("assignTo in %s must use bare notation (e.g., assignTo=\"output.field\") instead of template notation (e.g., ${output.field})", location),
			Context: ref,
		})
		return
	}

	// Check for scope prefix
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) < 2 {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrMissingScope,
			Message: fmt.Sprintf("assignTo in %s must include scope prefix (e.g., output.field, context.field)", location),
			Context: ref,
		})
		return
	}

	scope := flows.FlowVariableScope(parts[0])
	// Validate scope
	if scope.Validate() != nil {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrInvalidScope,
			Message: fmt.Sprintf("invalid scope '%s' in assignTo for %s", parts[0], location),
			Context: ref,
		})
		return
	}

	// Check if scope is valid for assignTo
	if !validAssignToScopes[scope] {
		result.Errors = append(result.Errors, flows.LinterError{
			Code:    flows.ErrInvalidAssignToScope,
			Message: fmt.Sprintf("invalid scope '%s' in assignTo for %s - allowed scopes are: output, context", parts[0], location),
			Context: ref,
		})
	}
}

// buildFieldMap builds a map of available fields for validation
func (e *ExpressionChecker) buildFieldMap(flow *flows.Flow) map[flows.FlowVariableScope][]string {
	fields := make(map[flows.FlowVariableScope][]string)

	// Input fields
	if flow.Input != nil {
		for _, f := range flow.Input.GetAllFields() {
			fields[flows.FlowVariableScopeInput] = append(fields[flows.FlowVariableScopeInput], f.Name)
		}
	}

	// Output fields
	if flow.Output != nil {
		for _, f := range flow.Output.GetAllFields() {
			fields[flows.FlowVariableScopeOutput] = append(fields[flows.FlowVariableScopeOutput], f.Name)
		}
	}

	// Context fields
	if flow.Context != nil {
		for _, f := range flow.Context.Strings {
			fields[flows.FlowVariableScopeContext] = append(fields[flows.FlowVariableScopeContext], f.Name)
		}
		for _, f := range flow.Context.Ints {
			fields[flows.FlowVariableScopeContext] = append(fields[flows.FlowVariableScopeContext], f.Name)
		}
		for _, f := range flow.Context.Bools {
			fields[flows.FlowVariableScopeContext] = append(fields[flows.FlowVariableScopeContext], f.Name)
		}
		for _, f := range flow.Context.Floats {
			fields[flows.FlowVariableScopeContext] = append(fields[flows.FlowVariableScopeContext], f.Name)
		}
		// Add nested object fields
		for _, obj := range flow.Context.Objects {
			fields[flows.FlowVariableScopeContext] = append(fields[flows.FlowVariableScopeContext], obj.Name)
		}
	}

	// Computed fields
	if flow.Computed != nil {
		for _, f := range flow.Computed.GetAllFields() {
			fields[flows.FlowVariableScopeComputed] = append(fields[flows.FlowVariableScopeComputed], f.Name)
		}
	}

	return fields
}

// validateFields checks that all field references exist
func (e *ExpressionChecker) validateFields(expr ast.Expr, available map[flows.FlowVariableScope][]string) error {
	// Walk the AST and check FieldRef nodes
	return checkFieldRefs(expr, available)
}

func checkFieldRefs(expr ast.Expr, available map[flows.FlowVariableScope][]string) error {
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

		// For nested references (e.g., context.obj.field), only check the first level
		// Full path validation would require understanding object structure
		if len(e.Path) > 0 {
			fieldName := e.Path[0]
			found := false
			for _, availableField := range fields {
				if availableField == fieldName {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("field '%s.%s' not found in %s scope", e.Prefix, fieldName, e.Prefix)
			}
		}
	}
	return nil
}
