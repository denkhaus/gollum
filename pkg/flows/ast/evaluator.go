package ast

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/shared"
)

// Evaluate evaluates an expression against a context
// This is a convenience wrapper that calls the expr's Evaluate method
func Evaluate(expr Expr, ctx map[string]any) (any, error) {
	return expr.Evaluate(ctx)
}

// evaluateCall evaluates a function call
func evaluateCall(call *CallExpr, ctx map[string]any) (any, error) {
	// Evaluate arguments
	args := make([]any, len(call.Args))
	for i, arg := range call.Args {
		val, err := arg.Evaluate(ctx)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	// Built-in functions
	switch call.Func {
	case "EQ", "eq":
		return compareEqual(args)
	case "NEQ", "neq":
		result, err := compareEqual(args)
		if err != nil {
			return nil, err
		}
		return !result, nil
	case "GT", "gt":
		return compare(args, func(a, b float64) bool { return a > b })
	case "GTE", "gte":
		return compare(args, func(a, b float64) bool { return a >= b })
	case "LT", "lt":
		return compare(args, func(a, b float64) bool { return a < b })
	case "LTE", "lte":
		return compare(args, func(a, b float64) bool { return a <= b })
	case "AND", "and":
		return logicalAnd(args)
	case "OR", "or":
		return logicalOr(args)
	case "NOT", "not":
		return logicalNot(args)
	case "ADD", "add":
		return arithmetic(args, func(a, b float64) float64 { return a + b })
	case "SUB", "sub":
		return arithmetic(args, func(a, b float64) float64 { return a - b })
	case "MUL", "mul":
		return arithmetic(args, func(a, b float64) float64 { return a * b })
	case "DIV", "div":
		return divide(args)
	default:
		return nil, fmt.Errorf("unknown function: %s", call.Func)
	}
}

// resolveFieldRef resolves a field reference from context
func resolveFieldRef(ref *FieldRef, ctx map[string]any) (any, error) {
	// Bare identifier (no prefix) - check in all scopes
	if ref.Prefix == "" {
		if len(ref.Path) != 1 {
			return nil, fmt.Errorf("bare field reference must be single identifier")
		}
		// For bare identifiers, we'll check context values directly
		// This handles cases like `is_open` in boolean expressions
		// In a real flow, this would need to be resolved against the actual scope
		// For now, return the value from context if it exists
		return ref.Path[0], nil
	}

	// Get prefix scope
	scope, ok := ctx[string(ref.Prefix)]
	if !ok {
		return nil, fmt.Errorf("prefix '%s' not found in context", ref.Prefix)
	}

	// Navigate path
	current := scope
	for i, part := range ref.Path {
		switch v := current.(type) {
		case map[string]any:
			var exists bool
			current, exists = v[part]
			if !exists {
				return nil, fmt.Errorf("field '%s' not found in path %s", part, ref.AbsolutePath())
			}
		default:
			return nil, fmt.Errorf("cannot access field '%s' on non-map type at path depth %d", part, i)
		}
	}

	return current, nil
}

// Helper functions

func compareEqual(args []any) (bool, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("EQ requires 2 arguments, got %d", len(args))
	}

	// Type coercion for EQ
	a, b := shared.AnyToString(args[0]), shared.AnyToString(args[1])
	return a == b, nil
}

func compare(args []any, cmp func(a, b float64) bool) (bool, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("comparison requires 2 arguments, got %d", len(args))
	}

	a, err := shared.ConvertToFloat(args[0])
	if err != nil {
		return false, fmt.Errorf("first argument: %w", err)
	}

	b, err := shared.ConvertToFloat(args[1])
	if err != nil {
		return false, fmt.Errorf("second argument: %w", err)
	}

	return cmp(a, b), nil
}

func logicalAnd(args []any) (bool, error) {
	if len(args) < 2 {
		return false, fmt.Errorf("AND requires at least 2 arguments, got %d", len(args))
	}

	for _, arg := range args {
		b, err := shared.ConvertToBool(arg)
		if err != nil {
			return false, err
		}
		if !b {
			return false, nil
		}
	}
	return true, nil
}

func logicalOr(args []any) (bool, error) {
	if len(args) < 2 {
		return false, fmt.Errorf("OR requires at least 2 arguments, got %d", len(args))
	}

	for _, arg := range args {
		b, err := shared.ConvertToBool(arg)
		if err != nil {
			return false, err
		}
		if b {
			return true, nil
		}
	}
	return false, nil
}

func logicalNot(args []any) (bool, error) {
	if len(args) != 1 {
		return false, fmt.Errorf("NOT requires 1 argument, got %d", len(args))
	}

	b, err := shared.ConvertToBool(args[0])
	if err != nil {
		return false, err
	}
	return !b, nil
}

// arithmetic performs arithmetic operations on two arguments
func arithmetic(args []any, op func(a, b float64) float64) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("arithmetic operation requires 2 arguments, got %d", len(args))
	}

	a, err := shared.ConvertToFloat(args[0])
	if err != nil {
		return nil, fmt.Errorf("first argument: %w", err)
	}

	b, err := shared.ConvertToFloat(args[1])
	if err != nil {
		return nil, fmt.Errorf("second argument: %w", err)
	}

	result := op(a, b)

	// Return int64 if result is a whole number and both inputs were integers
	if shared.IsWholeNumber(result) && shared.IsInt(args[0]) && shared.IsInt(args[1]) {
		return int64(result), nil
	}

	return result, nil
}

// divide performs division with zero-check
func divide(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("DIV requires 2 arguments, got %d", len(args))
	}

	a, err := shared.ConvertToFloat(args[0])
	if err != nil {
		return nil, fmt.Errorf("first argument: %w", err)
	}

	b, err := shared.ConvertToFloat(args[1])
	if err != nil {
		return nil, fmt.Errorf("second argument: %w", err)
	}

	if b == 0 {
		return nil, fmt.Errorf("division by zero")
	}

	// Integer division truncates - return int64 when both inputs are actual integer types
	if shared.IsActualIntType(args[0]) && shared.IsActualIntType(args[1]) {
		return int64(a / b), nil
	}

	result := a / b

	// Return int64 if result is a whole number
	if shared.IsWholeNumber(result) {
		return int64(result), nil
	}

	return result, nil
}
