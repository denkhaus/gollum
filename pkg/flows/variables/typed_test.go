package variables_test

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
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

func TestInputValues_GetSet(t *testing.T) {
	block := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "name", Type: "string"}},
		Ints:    []flows.FieldDef{{Name: "count", Type: "int"}},
	}

	input := variables.NewInputValues(block)

	// Set values
	err := input.SetString("name", "test")
	require.NoError(t, err)

	err = input.SetInt("count", 42)
	require.NoError(t, err)

	// Get values
	val, err := input.GetString("name")
	require.NoError(t, err)
	assert.Equal(t, "test", val)

	count, err := input.GetInt("count")
	require.NoError(t, err)
	assert.Equal(t, 42, count)
}

func TestInputValues_UnknownField(t *testing.T) {
	block := &flows.InputBlock{}
	input := variables.NewInputValues(block)

	err := input.SetString("unknown", "test")
	assert.Error(t, err)
}

func TestInputValues_TypeMismatch(t *testing.T) {
	block := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "name", Type: "string"}},
	}
	input := variables.NewInputValues(block)

	err := input.SetInt("name", 42)
	assert.Error(t, err)
}
