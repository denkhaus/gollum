package variables_test

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/variables"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputedValues_Register(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "is_large", Type: "bool", When: "GT(context.x, 10)"},
		{Name: "doubled", Type: "int", When: "MUL(context.x, 2)"},
	}

	computed := variables.NewComputedValues(computedFields)

	assert.True(t, computed.Has("is_large"))
	assert.True(t, computed.Has("doubled"))
	assert.False(t, computed.Has("unknown"))
}

func TestComputedValues_GetBeforeEvaluate(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "is_large", Type: "bool", When: "GT(context.x, 10)"},
	}

	computed := variables.NewComputedValues(computedFields)

	// Getting before evaluating should return error
	_, err := computed.GetBool("is_large")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not evaluated")
}

func TestComputedValues_GetUnknownField(t *testing.T) {
	computed := variables.NewComputedValues(nil)

	_, err := computed.GetBool("unknown")
	assert.Error(t, err)
}

func TestComputedValues_SetValue(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "is_large", Type: "bool", When: "GT(context.x, 10)"},
	}

	computed := variables.NewComputedValues(computedFields)

	// Set value
	computed.SetValue("is_large", variables.NewBoolValue(true))

	// Should now be retrievable
	val, err := computed.GetBool("is_large")
	require.NoError(t, err)
	assert.True(t, val)
}

func TestComputedValues_DirtyTracking(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "is_large", Type: "bool", When: "GT(context.x, 10)"},
	}

	computed := variables.NewComputedValues(computedFields)

	// Initially dirty
	assert.True(t, computed.IsDirty("is_large"))

	// Set value clears dirty
	computed.SetValue("is_large", variables.NewBoolValue(true))
	assert.False(t, computed.IsDirty("is_large"))

	// Mark dirty again
	computed.MarkDirty("is_large")
	assert.True(t, computed.IsDirty("is_large"))
}

func TestComputedValues_GetDependents(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "is_large", Type: "bool", When: "GT(context.x, 10)"},
		{Name: "is_valid", Type: "bool", When: "AND(context.x, context.y)"},
		{Name: "unrelated", Type: "bool", When: "EQ(input.status, active)"},
	}

	computed := variables.NewComputedValues(computedFields)

	// Find dependents of context.x
	deps := computed.GetDependents(variables.FieldReference{Scope: "context", Name: "x"})
	assert.ElementsMatch(t, []string{"is_large", "is_valid"}, deps)

	// Find dependents of context.y
	deps = computed.GetDependents(variables.FieldReference{Scope: "context", Name: "y"})
	assert.ElementsMatch(t, []string{"is_valid"}, deps)

	// Find dependents of input.status
	deps = computed.GetDependents(variables.FieldReference{Scope: "input", Name: "status"})
	assert.ElementsMatch(t, []string{"unrelated"}, deps)
}

func TestComputedValues_GetField(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "is_large", Type: "bool", When: "GT(context.x, 10)"},
	}

	computed := variables.NewComputedValues(computedFields)

	field, err := computed.GetField("is_large")
	require.NoError(t, err)
	assert.Equal(t, "is_large", field.Name)
	assert.Equal(t, variables.TypeBool, field.Type)
	assert.Equal(t, "GT(context.x, 10)", field.Expression)
	assert.Len(t, field.Dependencies, 1) // context.x only (literal 10 is not a field reference)
}

func TestComputedValues_MultipleTypes(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "flag", Type: "bool", When: "EQ(context.status, 1)"},
		{Name: "count", Type: "int", When: "ADD(context.a, context.b)"},
		{Name: "label", Type: "string", When: "context.name"},
		{Name: "ratio", Type: "float", When: "DIV(context.total, context.count)"},
	}

	computed := variables.NewComputedValues(computedFields)

	assert.True(t, computed.Has("flag"))
	assert.True(t, computed.Has("count"))
	assert.True(t, computed.Has("label"))
	assert.True(t, computed.Has("ratio"))

	// Set and get values of different types
	computed.SetValue("flag", variables.NewBoolValue(true))
	computed.SetValue("count", variables.NewIntValue(42))
	computed.SetValue("label", variables.NewStringValue("test"))
	computed.SetValue("ratio", variables.NewFloatValue(3.14))

	val, err := computed.GetBool("flag")
	require.NoError(t, err)
	assert.True(t, val)

	intVal, err := computed.GetInt("count")
	require.NoError(t, err)
	assert.Equal(t, 42, intVal)

	strVal, err := computed.GetString("label")
	require.NoError(t, err)
	assert.Equal(t, "test", strVal)

	floatVal, err := computed.GetFloat("ratio")
	require.NoError(t, err)
	assert.InDelta(t, 3.14, floatVal, 0.001)
}

func TestComputedValues_NilInput(t *testing.T) {
	computed := variables.NewComputedValues(nil)

	assert.False(t, computed.Has("anything"))
	assert.Len(t, computed.GetAll(), 0)
}

func TestComputedValues_GetAll(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "a", Type: "bool", When: "context.x"},
		{Name: "b", Type: "int", When: "context.y"},
	}

	computed := variables.NewComputedValues(computedFields)

	all := computed.GetAll()
	assert.Len(t, all, 2)
	assert.Contains(t, all, "a")
	assert.Contains(t, all, "b")
}

func TestComputedEvaluator_ReactiveUpdate(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "is_large", Type: "bool", When: "GT(context.x, 10)"},
	}
	computed := variables.NewComputedValues(computedFields)

	contextBlock := &flows.ContextBlock{
		Ints: []flows.ContextField{{Name: "x", Type: "int"}},
	}
	context := variables.NewContextValues(contextBlock)

	evaluator := variables.NewComputedEvaluator(computed, nil, context, nil)
	context.SetEvaluator(evaluator)

	// Initial evaluation
	context.SetInt("x", 15)
	err := evaluator.ComputeDirty()
	require.NoError(t, err)

	val, err := computed.GetBool("is_large")
	require.NoError(t, err)
	assert.True(t, val) // 15 > 10

	// Change dependency
	context.SetInt("x", 5)
	err = evaluator.ComputeDirty()
	require.NoError(t, err)

	val, err = computed.GetBool("is_large")
	require.NoError(t, err)
	assert.False(t, val) // 5 < 10
}

func TestComputedEvaluator_CircularDependency(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "a", Type: "bool", When: "computed.b"},
		{Name: "b", Type: "bool", When: "computed.a"},
	}
	computed := variables.NewComputedValues(computedFields)

	evaluator := variables.NewComputedEvaluator(computed, nil, nil, nil)

	err := evaluator.ComputeDirty()
	assert.Error(t, err)
}
