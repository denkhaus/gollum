package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseExpression_SimpleEquality(t *testing.T) {
	expr, err := ParseExpression(`EQ(context.pr.state, 'open')`)

	assert.NoError(t, err)
	call, ok := expr.(*CallExpr)
	assert.True(t, ok, "should be CallExpr")
	assert.Equal(t, "EQ", call.Func)
	assert.Len(t, call.Args, 2)
}

func TestParseExpression_NestedCalls(t *testing.T) {
	expr, err := ParseExpression(`AND(is_open, GT(changed_files, 10))`)

	assert.NoError(t, err)
	call, ok := expr.(*CallExpr)
	assert.True(t, ok)
	assert.Equal(t, "AND", call.Func)
	assert.Len(t, call.Args, 2)

	// Second arg should be another CallExpr
	inner, ok := call.Args[1].(*CallExpr)
	assert.True(t, ok, "second arg should be CallExpr")
	assert.Equal(t, "GT", inner.Func)
}
