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
