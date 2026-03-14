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

func TestContextValues_GetSet(t *testing.T) {
	block := &flows.ContextBlock{
		Strings: []flows.ContextField{{Name: "status", Type: "string"}},
		Ints:    []flows.ContextField{{Name: "count", Type: "int"}},
	}

	context := variables.NewContextValues(block)

	// Set values
	err := context.SetString("status", "ready")
	require.NoError(t, err)

	err = context.SetInt("count", 10)
	require.NoError(t, err)

	// Get values
	val, err := context.GetString("status")
	require.NoError(t, err)
	assert.Equal(t, "ready", val)

	count, err := context.GetInt("count")
	require.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestContextValues_DefaultValues(t *testing.T) {
	block := &flows.ContextBlock{
		Ints: []flows.ContextField{
			{Name: "timeout", Type: "int", Default: "30"},
		},
	}

	context := variables.NewContextValues(block)

	// Get default value before set
	val, err := context.GetInt("timeout")
	require.NoError(t, err)
	assert.Equal(t, 30, val)
}

func TestContextValues_BoolFloat(t *testing.T) {
	block := &flows.ContextBlock{
		Bools:  []flows.ContextField{{Name: "enabled", Type: "bool", Default: "true"}},
		Floats: []flows.ContextField{{Name: "rate", Type: "float", Default: "1.5"}},
	}

	context := variables.NewContextValues(block)

	// Get default values
	enabled, err := context.GetBool("enabled")
	require.NoError(t, err)
	assert.True(t, enabled)

	rate, err := context.GetFloat("rate")
	require.NoError(t, err)
	assert.InDelta(t, 1.5, rate, 0.001)

	// Set new values
	err = context.SetBool("enabled", false)
	require.NoError(t, err)

	err = context.SetFloat("rate", 2.5)
	require.NoError(t, err)

	// Verify new values
	enabled, err = context.GetBool("enabled")
	require.NoError(t, err)
	assert.False(t, enabled)

	rate, err = context.GetFloat("rate")
	require.NoError(t, err)
	assert.InDelta(t, 2.5, rate, 0.001)
}

func TestContextValues_UnknownField(t *testing.T) {
	block := &flows.ContextBlock{}
	context := variables.NewContextValues(block)

	err := context.SetString("unknown", "test")
	assert.Error(t, err)
}

func TestContextValues_TypeMismatch(t *testing.T) {
	block := &flows.ContextBlock{
		Strings: []flows.ContextField{{Name: "status", Type: "string"}},
	}
	context := variables.NewContextValues(block)

	err := context.SetInt("status", 42)
	assert.Error(t, err)
}
