package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallExpr_String(t *testing.T) {
	expr := &CallExpr{
		Func: "EQ",
		Args: []Expr{
			&FieldRef{Path: []string{"context", "pr", "state"}},
			&StringLiteral{Value: "open"},
		},
	}

	// Should render as something readable
	str := expr.String()
	assert.Contains(t, str, "EQ")
}

func TestFieldRef_AbsolutePath(t *testing.T) {
	ref := &FieldRef{
		Prefix: "context",
		Path:   []string{"pr", "state"},
	}

	assert.Equal(t, "context.pr.state", ref.AbsolutePath())
}
