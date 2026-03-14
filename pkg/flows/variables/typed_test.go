package variables_test

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows/variables"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStringValue(t *testing.T) {
	fv := variables.NewStringValue("test")

	assert.Equal(t, variables.TypeString, fv.Type())

	val, err := fv.String()
	require.NoError(t, err)
	assert.Equal(t, "test", val)
}

func TestNewIntValue(t *testing.T) {
	fv := variables.NewIntValue(42)

	assert.Equal(t, variables.TypeInt, fv.Type())

	val, err := fv.Int()
	require.NoError(t, err)
	assert.Equal(t, 42, val)
}

func TestNewBoolValue(t *testing.T) {
	fv := variables.NewBoolValue(true)

	assert.Equal(t, variables.TypeBool, fv.Type())

	val, err := fv.Bool()
	require.NoError(t, err)
	assert.True(t, val)
}

func TestNewFloatValue(t *testing.T) {
	fv := variables.NewFloatValue(3.14)

	assert.Equal(t, variables.TypeFloat, fv.Type())

	val, err := fv.Float()
	require.NoError(t, err)
	assert.InDelta(t, 3.14, val, 0.001)
}

func TestFieldValue_TypeMismatch(t *testing.T) {
	fv := variables.NewStringValue("test")

	_, err := fv.Int()
	assert.Error(t, err)
}
