package ast

import (
	"fmt"
	"strconv"
)

// Evaluate evaluates an expression against a context
func Evaluate(expr Expr, ctx map[string]any) (any, error) {
	switch e := expr.(type) {
	case *CallExpr:
		return evaluateCall(e, ctx)
	case *FieldRef:
		return resolveFieldRef(e, ctx)
	case *StringLiteral:
		return e.Value, nil
	case *NumberLiteral:
		return e.Value, nil
	case *BoolLiteral:
		return e.Value, nil
	default:
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}
}

// evaluateCall evaluates a function call
func evaluateCall(call *CallExpr, ctx map[string]any) (any, error) {
	// Evaluate arguments
	args := make([]any, len(call.Args))
	for i, arg := range call.Args {
		val, err := Evaluate(arg, ctx)
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
	scope, ok := ctx[ref.Prefix]
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
	a, b := coerceToString(args[0]), coerceToString(args[1])
	return a == b, nil
}

func compare(args []any, cmp func(a, b float64) bool) (bool, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("comparison requires 2 arguments, got %d", len(args))
	}

	a, err := toFloat64(args[0])
	if err != nil {
		return false, fmt.Errorf("first argument: %w", err)
	}

	b, err := toFloat64(args[1])
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
		b, err := toBool(arg)
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
		b, err := toBool(arg)
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

	b, err := toBool(args[0])
	if err != nil {
		return false, err
	}
	return !b, nil
}

// Type coercion helpers

func toFloat64(v any) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func toBool(v any) (bool, error) {
	switch val := v.(type) {
	case bool:
		return val, nil
	case string:
		return strconv.ParseBool(val)
	default:
		return false, fmt.Errorf("cannot convert %T to bool", v)
	}
}

func coerceToString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
