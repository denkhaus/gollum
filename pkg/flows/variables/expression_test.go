package variables_test

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows/variables"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpressionParser_Parse(t *testing.T) {
	parser := variables.NewExpressionParser()

	expr, err := parser.Parse("GT(context.x, 10)")
	require.NoError(t, err)
	assert.Equal(t, "GT", expr.Operator)
	assert.Len(t, expr.Args, 2)
}

func TestExpressionParser_ParseInvalid(t *testing.T) {
	parser := variables.NewExpressionParser()

	_, err := parser.Parse("INVALID(context.x, 10)")
	assert.Error(t, err)
}

func TestExpressionParser_ExtractDependencies(t *testing.T) {
	parser := variables.NewExpressionParser()

	deps := parser.ExtractDependencies("GT(context.x, 10)")

	assert.Len(t, deps, 1)
	assert.Equal(t, "context", string(deps[0].Scope))
	assert.Equal(t, "x", deps[0].Name)
}

func TestExpressionParser_ExtractDependenciesMultiple(t *testing.T) {
	parser := variables.NewExpressionParser()

	deps := parser.ExtractDependencies("AND(input.a, context.b, output.c)")

	assert.Len(t, deps, 3)

	scopeMap := map[string]bool{}
	for _, dep := range deps {
		scopeMap[string(dep.Scope)] = true
	}

	assert.True(t, scopeMap["input"])
	assert.True(t, scopeMap["context"])
	assert.True(t, scopeMap["output"])

	// Also check names
	names := []string{}
	for _, dep := range deps {
		names = append(names, dep.Name)
	}
	assert.Contains(t, names, "a")
	assert.Contains(t, names, "b")
	assert.Contains(t, names, "c")
}

func TestExpressionEvaluator_EvaluateGT(t *testing.T) {
	evaluator := variables.NewExpressionEvaluator()
	scope := &variables.EvaluationScope{
		Context: map[string]any{"x": 15},
	}

	result, err := evaluator.Evaluate("GT(context.x, 10)", scope)
	require.NoError(t, err)
	assert.True(t, result.(bool))
}

func TestExpressionEvaluator_EvaluateAND(t *testing.T) {
	evaluator := variables.NewExpressionEvaluator()
	scope := &variables.EvaluationScope{
		Context: map[string]any{
			"a": true,
			"b": true,
		},
	}

	result, err := evaluator.Evaluate("AND(context.a, context.b)", scope)
	require.NoError(t, err)
	assert.True(t, result.(bool))
}

func TestExpressionEvaluator_EvaluateADD(t *testing.T) {
	evaluator := variables.NewExpressionEvaluator()
	scope := &variables.EvaluationScope{
		Input: map[string]any{"x": 5, "y": 3},
	}

	result, err := evaluator.Evaluate("ADD(input.x, input.y)", scope)
	require.NoError(t, err)
	assert.Equal(t, 8, result)
}
