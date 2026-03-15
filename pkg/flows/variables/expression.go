package variables

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// FieldReference represents a reference to a field
type FieldReference struct {
	Scope string // "input", "context", "output", "computed"
	Name  string
}

// Expression represents a parsed expression
type Expression struct {
	Operator string
	Args     []string
}

// ExpressionParser parses expression strings
type ExpressionParser struct {
}

// NewExpressionParser creates a new parser
func NewExpressionParser() *ExpressionParser {
	return &ExpressionParser{}
}

// Parse parses an expression string
// Format: OPERATOR(arg1, arg2, ...)
func (ep *ExpressionParser) Parse(expr string) (*Expression, error) {
	expr = strings.TrimSpace(expr)

	// Match: OPERATOR(...)
	re := regexp.MustCompile(`^([A-Z]+)\((.*)\)$`)
	matches := re.FindStringSubmatch(expr)
	if matches == nil {
		return nil, fmt.Errorf("invalid expression format: %s", expr)
	}

	operator := matches[1]
	argsStr := matches[2]

	// Validate operator
	validOperators := map[string]bool{
		"GT": true, "LT": true, "GTE": true, "LTE": true,
		"EQ": true, "NEQ": true,
		"AND": true, "OR": true, "NOT": true,
		"ADD": true, "SUB": true, "MUL": true, "DIV": true,
	}

	if !validOperators[operator] {
		return nil, fmt.Errorf("unknown operator: %s", operator)
	}

	// Parse arguments (simple split by comma for now)
	var args []string
	if argsStr != "" {
		for _, arg := range strings.Split(argsStr, ",") {
			args = append(args, strings.TrimSpace(arg))
		}
	}

	return &Expression{
		Operator: operator,
		Args:     args,
	}, nil
}

// ExtractDependencies extracts field references from an expression
func (ep *ExpressionParser) ExtractDependencies(expr string) []FieldReference {
	var deps []FieldReference

	// Match patterns like: scope.fieldName
	re := regexp.MustCompile(`(input|context|output|computed)\.([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := re.FindAllStringSubmatch(expr, -1)

	for _, match := range matches {
		deps = append(deps, FieldReference{
			Scope: match[1],
			Name:  match[2],
		})
	}

	return deps
}

// EvaluationScope provides values for expression evaluation
type EvaluationScope struct {
	Input    map[string]any
	Context  map[string]any
	Output   map[string]any
	Computed map[string]any
}

// ExpressionEvaluator evaluates expressions
type ExpressionEvaluator struct {
	parser *ExpressionParser
}

// NewExpressionEvaluator creates a new evaluator
func NewExpressionEvaluator() *ExpressionEvaluator {
	return &ExpressionEvaluator{
		parser: NewExpressionParser(),
	}
}

// Evaluate evaluates an expression against the given scope
func (ee *ExpressionEvaluator) Evaluate(expr string, scope *EvaluationScope) (any, error) {
	parsed, err := ee.parser.Parse(expr)
	if err != nil {
		return nil, err
	}

	// Resolve arguments to values
	args, err := ee.resolveArgs(parsed.Args, scope)
	if err != nil {
		return nil, err
	}

	// Apply operator
	return ee.applyOperator(parsed.Operator, args)
}

// resolveArgs resolves argument references to values
func (ee *ExpressionEvaluator) resolveArgs(args []string, scope *EvaluationScope) ([]any, error) {
	resolved := make([]any, len(args))

	for i, arg := range args {
		// Check if it's a field reference
		re := regexp.MustCompile(`^(input|context|output|computed)\.([a-zA-Z_][a-zA-Z0-9_]*)$`)
		matches := re.FindStringSubmatch(arg)

		if matches != nil {
			scopeName := matches[1]
			fieldName := matches[2]

			var scopeMap map[string]any
			switch scopeName {
			case "input":
				scopeMap = scope.Input
			case "context":
				scopeMap = scope.Context
			case "output":
				scopeMap = scope.Output
			case "computed":
				scopeMap = scope.Computed
			}

			val, ok := scopeMap[fieldName]
			if !ok {
				return nil, fmt.Errorf("field not found: %s.%s", scopeName, fieldName)
			}
			// For nil (unset field), use a sensible default for expression evaluation
			// This allows computed fields to reference unset context/output fields without error
			if val == nil {
				// Use zero/default values for unset fields in expressions
				// This is different from the actual field value - it's just for expression eval
				resolved[i] = 0 // Default to 0 for numbers (works for comparisons)
			} else {
				resolved[i] = val
			}
		} else {
			// It's a literal value (number, bool, or string)
			// Check for string literal (enclosed in single or double quotes)
			if strings.HasPrefix(arg, "'") && strings.HasSuffix(arg, "'") {
				resolved[i] = strings.Trim(arg, "'")
				continue
			}
			if strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"") {
				resolved[i] = strings.Trim(arg, "\"")
				continue
			}
			// Try integer
			if intVal, err := strconv.Atoi(arg); err == nil {
				resolved[i] = intVal
				continue
			}
			// Try boolean
			if boolVal, err := strconv.ParseBool(arg); err == nil {
				resolved[i] = boolVal
				continue
			}
			// Default to raw string
			resolved[i] = arg
		}
	}

	return resolved, nil
}

// applyOperator applies the operator to the arguments
func (ee *ExpressionEvaluator) applyOperator(op string, args []any) (any, error) {
	switch op {
	case "GT":
		return compare(args, ">")
	case "LT":
		return compare(args, "<")
	case "GTE":
		return compare(args, ">=")
	case "LTE":
		return compare(args, "<=")
	case "EQ":
		return compare(args, "==")
	case "NEQ":
		return compare(args, "!=")
	case "AND":
		return logicalAnd(args)
	case "OR":
		return logicalOr(args)
	case "NOT":
		return logicalNot(args)
	case "ADD":
		return arithmetic(args, "+")
	case "SUB":
		return arithmetic(args, "-")
	case "MUL":
		return arithmetic(args, "*")
	case "DIV":
		return arithmetic(args, "/")
	default:
		return nil, fmt.Errorf("unknown operator: %s", op)
	}
}

// compare performs comparison operations
func compare(args []any, op string) (bool, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("comparison requires 2 arguments")
	}

	a, b := args[0], args[1]

	// Try numeric comparison first
	aInt, aOk := toInt64(a)
	bInt, bOk := toInt64(b)
	if aOk && bOk {
		switch op {
		case ">":
			return aInt > bInt, nil
		case "<":
			return aInt < bInt, nil
		case ">=":
			return aInt >= bInt, nil
		case "<=":
			return aInt <= bInt, nil
		case "==":
			return aInt == bInt, nil
		case "!=":
			return aInt != bInt, nil
		}
	}

	// Try string comparison
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	if aOk && bOk {
		switch op {
		case ">":
			return aStr > bStr, nil
		case "<":
			return aStr < bStr, nil
		case ">=":
			return aStr >= bStr, nil
		case "<=":
			return aStr <= bStr, nil
		case "==":
			return aStr == bStr, nil
		case "!=":
			return aStr != bStr, nil
		}
	}

	return false, fmt.Errorf("cannot compare %T and %T", a, b)
}

// logicalAnd performs logical AND operation
func logicalAnd(args []any) (bool, error) {
	for _, arg := range args {
		b, ok := toBool(arg)
		if !ok {
			return false, fmt.Errorf("AND requires bool arguments")
		}
		if !b {
			return false, nil
		}
	}
	return true, nil
}

// logicalOr performs logical OR operation
func logicalOr(args []any) (bool, error) {
	for _, arg := range args {
		b, ok := toBool(arg)
		if !ok {
			return false, fmt.Errorf("OR requires bool arguments")
		}
		if b {
			return true, nil
		}
	}
	return false, nil
}

// logicalNot performs logical NOT operation
func logicalNot(args []any) (bool, error) {
	if len(args) != 1 {
		return false, fmt.Errorf("NOT requires 1 argument")
	}
	b, ok := toBool(args[0])
	if !ok {
		return false, fmt.Errorf("NOT requires bool argument")
	}
	return !b, nil
}

// arithmetic performs arithmetic operations
func arithmetic(args []any, op string) (int, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("%s requires 2 arguments", op)
	}

	a, ok := toInt64(args[0])
	if !ok {
		return 0, fmt.Errorf("arithmetic requires integer arguments")
	}
	b, ok := toInt64(args[1])
	if !ok {
		return 0, fmt.Errorf("arithmetic requires integer arguments")
	}

	switch op {
	case "+":
		return int(a + b), nil
	case "-":
		return int(a - b), nil
	case "*":
		return int(a * b), nil
	case "/":
		if b == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return int(a / b), nil
	default:
		return 0, fmt.Errorf("unknown arithmetic operator: %s", op)
	}
}

// toInt64 converts a value to int64 if possible
func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case float64:
		return int64(val), true
	default:
		return 0, false
	}
}

// toBool converts a value to bool if possible
func toBool(v any) (bool, bool) {
	b, ok := v.(bool)
	return b, ok
}
