package variables

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComputedField_DependencyExtraction tests that dependencies are correctly extracted
func TestComputedField_DependencyExtraction(t *testing.T) {
	// Create computed values with various dependencies
	computedFields := []flows.ComputedField{
		{Name: "is_large", Type: "bool", Eval: "GT(input.value, 10)"},
		{Name: "is_small", Type: "bool", Eval: "LT(input.min, 5)"},
		{Name: "combined", Type: "bool", Eval: "AND(input.flag, GT(context.count, 0))"},
	}

	cv := NewComputedValues(computedFields)
	require.NotNil(t, cv)

	// Verify dependencies were extracted for is_large
	field, ok := cv.fields["is_large"]
	assert.True(t, ok)
	assert.Greater(t, len(field.Dependencies), 0)
	assert.Equal(t, "input", field.Dependencies[0].Scope)
	assert.Equal(t, "value", field.Dependencies[0].Name)

	// Check is_small has different dependency
	field2, ok := cv.fields["is_small"]
	assert.True(t, ok)
	assert.Greater(t, len(field2.Dependencies), 0)
	assert.Equal(t, "min", field2.Dependencies[0].Name)

	// Check combined has dependencies from multiple scopes
	field3, ok := cv.fields["combined"]
	assert.True(t, ok)
	assert.Greater(t, len(field3.Dependencies), 0)
	// Should have at least one dependency from input
	hasInputDep := false
	hasContextDep := false
	for _, dep := range field3.Dependencies {
		if dep.Scope == "input" {
			hasInputDep = true
		}
		if dep.Scope == "context" {
			hasContextDep = true
		}
	}
	assert.True(t, hasInputDep, "Should have input dependency")
	assert.True(t, hasContextDep, "Should have context dependency")
}

// TestComputedField_ComputedToComputedDependencies tests computed field referencing other computed fields
func TestComputedField_ComputedToComputedDependencies(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "base", Type: "int", Eval: "input.value"},
		{Name: "doubled", Type: "int", Eval: "MUL(computed.base, 2)"},
		{Name: "quadrupled", Type: "int", Eval: "MUL(computed.doubled, 2)"},
	}

	cv := NewComputedValues(computedFields)

	// Verify base has no computed dependencies
	field1, ok := cv.fields["base"]
	assert.True(t, ok)
	assert.Len(t, field1.Dependencies, 1)
	assert.Equal(t, "input", field1.Dependencies[0].Scope)

	// Verify doubled depends on computed.base
	field2, ok := cv.fields["doubled"]
	assert.True(t, ok)
	assert.Len(t, field2.Dependencies, 1)
	assert.Equal(t, "computed", field2.Dependencies[0].Scope)
	assert.Equal(t, "base", field2.Dependencies[0].Name)

	// Verify quadrupled depends on computed.doubled
	field3, ok := cv.fields["quadrupled"]
	assert.True(t, ok)
	assert.Len(t, field3.Dependencies, 1)
	assert.Equal(t, "computed", field3.Dependencies[0].Scope)
	assert.Equal(t, "doubled", field3.Dependencies[0].Name)
}

// TestComputedField_DirtyMarking tests the dirty flag mechanism
func TestComputedField_DirtyMarking(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "field1", Type: "int", Eval: "input.x"},
		{Name: "field2", Type: "int", Eval: "MUL(computed.field1, 2)"},
	}

	cv := NewComputedValues(computedFields)

	// Initially all fields should be dirty
	field1, _ := cv.GetField("field1")
	assert.True(t, field1.dirty, "New computed fields should be dirty")

	field2, _ := cv.GetField("field2")
	assert.True(t, field2.dirty, "New computed fields should be dirty")

	// Mark field1 as not dirty (simulating evaluation)
	field1.dirty = false

	// Mark dirty should mark it dirty again
	cv.MarkDirty("field1")
	field1, _ = cv.GetField("field1")
	assert.True(t, field1.dirty, "MarkDirty should set dirty flag")
}

// TestComputedField_MultipleScopeDependencies tests dependencies across all scopes
func TestComputedField_MultipleScopeDependencies(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "total", Type: "int", Eval: "ADD(input.a, context.b)"},
		{Name: "average", Type: "int", Eval: "DIV(computed.total, 2)"},
		{Name: "check_output", Type: "bool", Eval: "GT(output.result, 0)"},
	}

	cv := NewComputedValues(computedFields)

	// Check total has dependencies from input and context
	field1, ok := cv.fields["total"]
	assert.True(t, ok)
	assert.Len(t, field1.Dependencies, 2)

	// Check average depends on computed.total
	field2, ok := cv.fields["average"]
	assert.True(t, ok)
	assert.Len(t, field2.Dependencies, 1)
	assert.Equal(t, "computed", field2.Dependencies[0].Scope)

	// Check check_output depends on output
	field3, ok := cv.fields["check_output"]
	assert.True(t, ok)
	assert.Len(t, field3.Dependencies, 1)
	assert.Equal(t, "output", field3.Dependencies[0].Scope)
}

// TestComputedField_TypeMapping tests correct type mapping
func TestComputedField_TypeMapping(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "bool_field", Type: "bool", Eval: "GT(input.x, 0)"},
		{Name: "int_field", Type: "int", Eval: "input.y"},
		{Name: "string_field", Type: "string", Eval: "input.z"},
		{Name: "float_field", Type: "float", Eval: "input.w"},
	}

	cv := NewComputedValues(computedFields)

	// Check bool type
	field1, ok := cv.fields["bool_field"]
	assert.True(t, ok)
	assert.Equal(t, flows.TypeBool, field1.Type)

	// Check int type
	field2, ok := cv.fields["int_field"]
	assert.True(t, ok)
	assert.Equal(t, flows.TypeInt, field2.Type)

	// Check string type
	field3, ok := cv.fields["string_field"]
	assert.True(t, ok)
	assert.Equal(t, flows.TypeString, field3.Type)

	// Check float type
	field4, ok := cv.fields["float_field"]
	assert.True(t, ok)
	assert.Equal(t, flows.TypeFloat, field4.Type)
}

// TestComputedField_UnknownField tests error handling for unknown fields
func TestComputedField_UnknownField(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "existing", Type: "int", Eval: "input.x"},
	}

	cv := NewComputedValues(computedFields)

	// Get existing field should work
	_, err := cv.GetField("existing")
	assert.NoError(t, err)

	// Get unknown field should return error
	_, err = cv.GetField("unknown")
	assert.Error(t, err)
}

// TestComputedField_Has tests the Has method
func TestComputedField_Has(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "field1", Type: "int", Eval: "input.x"},
		{Name: "field2", Type: "bool", Eval: "GT(input.y, 0)"},
	}

	cv := NewComputedValues(computedFields)

	// Check existing fields
	assert.True(t, cv.Has("field1"))
	assert.True(t, cv.Has("field2"))

	// Check non-existing field
	assert.False(t, cv.Has("field3"))
}

// TestComputedField_EmptyComputedFields tests with no computed fields
func TestComputedField_EmptyComputedFields(t *testing.T) {
	cv := NewComputedValues(nil)
	assert.NotNil(t, cv)
	assert.False(t, cv.Has("anything"))
}

// TestComputedField_ComplexExpressionParsing tests parsing of complex expressions
func TestComputedField_ComplexExpressionParsing(t *testing.T) {
	computedFields := []flows.ComputedField{
		{Name: "complex", Type: "bool", Eval: "AND(GT(input.x, 0), LT(input.x, 100), EQ(input.status, \"active\"))"},
	}

	cv := NewComputedValues(computedFields)
	field, ok := cv.fields["complex"]
	assert.True(t, ok)

	// Should extract dependencies even from complex expressions
	assert.Greater(t, len(field.Dependencies), 0)
	// First dependency should be x
	assert.Equal(t, "x", field.Dependencies[0].Name)
}

// TestComputedField_CircularDependencyDetection tests that circular dependencies are tracked
func TestComputedField_CircularDependencyDetection(t *testing.T) {
	// This test documents that circular dependencies CAN be created in ComputedValues
	// but should be caught by the linter before execution
	computedFields := []flows.ComputedField{
		{Name: "a", Type: "int", Eval: "computed.b"},
		{Name: "b", Type: "int", Eval: "computed.c"},
		{Name: "c", Type: "int", Eval: "computed.a"},
	}

	cv := NewComputedValues(computedFields)

	// ComputedValues will accept these (validation is linter's job)
	assert.True(t, cv.Has("a"))
	assert.True(t, cv.Has("b"))
	assert.True(t, cv.Has("c"))

	// Each field has dependencies
	fieldA, _ := cv.GetField("a")
	assert.Len(t, fieldA.Dependencies, 1)
	assert.Equal(t, "computed", fieldA.Dependencies[0].Scope)
}
