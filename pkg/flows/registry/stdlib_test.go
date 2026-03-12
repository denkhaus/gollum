package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinRegistry_HasExpectedFunctions(t *testing.T) {
	registry := GetBuiltinRegistry()

	// Check strings functions
	sig, ok := registry.Lookup("strings.ToUpper")
	require.True(t, ok, "strings.ToUpper should be registered")
	assert.Equal(t, "ToUpper", sig.Name)
	assert.Equal(t, "string", sig.ReturnType)

	// Check fmt functions
	sig, ok = registry.Lookup("fmt.Sprintf")
	require.True(t, ok, "fmt.Sprintf should be registered")
	assert.Equal(t, "Sprintf", sig.Name)
	assert.Equal(t, "string", sig.ReturnType)
	assert.True(t, sig.Variadic, "fmt.Sprintf should be variadic")

	// Check len function
	sig, ok = registry.Lookup("len")
	require.True(t, ok, "len should be registered")
	assert.Equal(t, "len", sig.Name)
	assert.Equal(t, "int", sig.ReturnType)
}

func TestRegisterFunction_AddsNewFunction(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register("test.Func", FunctionSignature{
		Name:       "Func",
		Params:     []Param{{Name: "x", Type: "int"}},
		ReturnType: "string",
		Variadic:   false,
		Func:       func(args []any) (any, error) { return "test", nil },
	})

	require.NoError(t, err)

	sig, ok := registry.Lookup("test.Func")
	require.True(t, ok)
	assert.Equal(t, "Func", sig.Name)
}

func TestRegisterFunction_Duplicate(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register("test.Func", FunctionSignature{
		Name:       "Func",
		Params:     []Param{{Name: "x", Type: "int"}},
		ReturnType: "string",
	})
	require.NoError(t, err)

	err = registry.Register("test.Func", FunctionSignature{
		Name:       "Func",
		Params:     []Param{{Name: "y", Type: "string"}},
		ReturnType: "int",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestValidateCall_MatchingSignature(t *testing.T) {
	registry := GetBuiltinRegistry()

	// strings.ToUpper(s string) string
	args := map[string]any{"s": "hello"}
	err := registry.ValidateCall("strings.ToUpper", args)
	assert.NoError(t, err)
}

func TestValidateCall_MissingRequiredParam(t *testing.T) {
	registry := GetBuiltinRegistry()

	// strings.ToUpper requires "s" param
	args := map[string]any{}
	err := registry.ValidateCall("strings.ToUpper", args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required parameter")
}

func TestExecuteFunction_StringsToUpper(t *testing.T) {
	registry := GetBuiltinRegistry()

	result, err := registry.Execute("strings.ToUpper", map[string]any{"s": "hello"})
	require.NoError(t, err)
	assert.Equal(t, "HELLO", result)
}

func TestExecuteFunction_FmtSprintf(t *testing.T) {
	registry := GetBuiltinRegistry()

	result, err := registry.Execute("fmt.Sprintf", map[string]any{
		"format": "Hello %s %d",
		"args":   []any{"World", 42},
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello World 42", result)
}

func TestExecuteFunction_Len(t *testing.T) {
	registry := GetBuiltinRegistry()

	// Test with string
	result, err := registry.Execute("len", map[string]any{"v": "hello"})
	require.NoError(t, err)
	assert.Equal(t, 5, result)

	// Test with slice
	result, err = registry.Execute("len", map[string]any{"v": []int{1, 2, 3}})
	require.NoError(t, err)
	assert.Equal(t, 3, result)
}

func TestExecuteFunction_FunctionNotFound(t *testing.T) {
	registry := GetBuiltinRegistry()

	_, err := registry.Execute("nonexistent.Func", map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
