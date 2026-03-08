package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluate_Equality(t *testing.T) {
	expr, _ := ParseExpression(`EQ(context.status, 'open')`)

	ctx := map[string]any{
		"context": map[string]any{
			"status": "open",
		},
	}

	result, err := Evaluate(expr, ctx)
	assert.NoError(t, err)
	assert.True(t, result.(bool))
}

func TestEvaluate_GreaterThan(t *testing.T) {
	expr, _ := ParseExpression(`GT(context.count, 10)`)

	ctx := map[string]any{
		"context": map[string]any{
			"count": 15,
		},
	}

	result, err := Evaluate(expr, ctx)
	assert.NoError(t, err)
	assert.True(t, result.(bool))
}

func TestEvaluate_And(t *testing.T) {
	expr, _ := ParseExpression(`AND(input.is_open, input.is_large)`)

	ctx := map[string]any{
		"input": map[string]any{
			"is_open":  true,
			"is_large": false,
		},
	}

	result, err := Evaluate(expr, ctx)
	assert.NoError(t, err)
	assert.False(t, result.(bool))
}
