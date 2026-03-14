package variables

import (
	"fmt"
	"regexp"
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
