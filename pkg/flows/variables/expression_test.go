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
	assert.Equal(t, "context", deps[0].Scope)
	assert.Equal(t, "x", deps[0].Name)
}

func TestExpressionParser_ExtractDependenciesMultiple(t *testing.T) {
	parser := variables.NewExpressionParser()

	deps := parser.ExtractDependencies("AND(input.a, context.b, output.c)")

	assert.Len(t, deps, 3)

	scopes := []string{}
	names := []string{}
	for _, dep := range deps {
		scopes = append(scopes, dep.Scope)
		names = append(names, dep.Name)
	}

	assert.Contains(t, scopes, "input")
	assert.Contains(t, scopes, "context")
	assert.Contains(t, scopes, "output")
	assert.Contains(t, names, "a")
	assert.Contains(t, names, "b")
	assert.Contains(t, names, "c")
}
