package ast

import (
	"fmt"
	"strings"
)

// Expr represents an expression node
type Expr interface {
	exprNode()
	String() string
	Evaluate(ctx map[string]any) (any, error)
}

// CallExpr represents a function call expression
type CallExpr struct {
	Func string
	Args []Expr
}

func (c *CallExpr) exprNode() {}
func (c *CallExpr) String() string {
	args := make([]string, len(c.Args))
	for i, arg := range c.Args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", c.Func, strings.Join(args, ", "))
}

// Evaluate evaluates a function call expression
func (c *CallExpr) Evaluate(ctx map[string]any) (any, error) {
	return evaluateCall(c, ctx)
}

// FieldRef represents a field reference like "context.pr.state"
type FieldRef struct {
	Prefix string // "input", "output", "context", "error"
	Path   []string
}

func (f *FieldRef) exprNode() {}
func (f *FieldRef) String() string {
	return f.AbsolutePath()
}

// Evaluate resolves a field reference from the context
func (f *FieldRef) Evaluate(ctx map[string]any) (any, error) {
	return resolveFieldRef(f, ctx)
}

// AbsolutePath returns the full absolute path
func (f *FieldRef) AbsolutePath() string {
	if f.Prefix == "" {
		return strings.Join(f.Path, ".")
	}
	return f.Prefix + "." + strings.Join(f.Path, ".")
}

// Literal represents a literal value
type Literal struct {
	Value any
}

func (l *Literal) exprNode() {}
func (l *Literal) String() string {
	return fmt.Sprintf("%v", l.Value)
}

// StringLiteral represents a string literal
type StringLiteral struct {
	Value string
}

func (s *StringLiteral) exprNode() {}
func (s *StringLiteral) String() string {
	return fmt.Sprintf("'%s'", s.Value)
}

// Evaluate returns the literal value
func (s *StringLiteral) Evaluate(ctx map[string]any) (any, error) {
	return s.Value, nil
}

// NumberLiteral represents a numeric literal
type NumberLiteral struct {
	Value float64
}

func (n *NumberLiteral) exprNode() {}
func (n *NumberLiteral) String() string {
	return fmt.Sprintf("%g", n.Value)
}

// Evaluate returns the literal value
func (n *NumberLiteral) Evaluate(ctx map[string]any) (any, error) {
	return n.Value, nil
}

// BoolLiteral represents a boolean literal
type BoolLiteral struct {
	Value bool
}

func (b *BoolLiteral) exprNode() {}
func (b *BoolLiteral) String() string {
	return fmt.Sprintf("%t", b.Value)
}

// Evaluate returns the literal value
func (b *BoolLiteral) Evaluate(ctx map[string]any) (any, error) {
	return b.Value, nil
}
