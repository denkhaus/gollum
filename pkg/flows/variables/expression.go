package variables

import (
	"fmt"
	"regexp"
	"strings"
)

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
