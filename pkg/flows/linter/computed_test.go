package linter

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestComputedChecker_Check_Valid(t *testing.T) {
	checker := NewComputedChecker()

	flow := &flows.Flow{
		Input: &flows.InputBlock{
			Ints: []flows.FieldDef{{Name: "x"}},
		},
		Computed: &flows.ComputedBlock{
			Bools: []flows.ComputedFieldDef{
				{Name: "is_large", Type: "bool", Eval: "GT(input.x, 10)"},
			},
		},
	}

	result := &flows.LinterResult{}
	checker.Check(flow, result)
	result.Valid = len(result.Errors) == 0

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestComputedChecker_Check_EmptyName(t *testing.T) {
	checker := NewComputedChecker()

	flow := &flows.Flow{
		Computed: &flows.ComputedBlock{
		Bools: []flows.ComputedFieldDef{
				{Name: "", Type: "bool", Eval: "GT(input.x, 10)"},
			},
		},
	}

	result := &flows.LinterResult{}
	checker.Check(flow, result)
	result.Valid = len(result.Errors) == 0

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, flows.ErrInvalidExpr, result.Errors[0].Code)
}

func TestComputedChecker_Check_InvalidType(t *testing.T) {
	checker := NewComputedChecker()

	flow := &flows.Flow{
		Input: &flows.InputBlock{
			Ints: []flows.FieldDef{{Name: "x"}},
		},
		Computed: &flows.ComputedBlock{
		Bools: []flows.ComputedFieldDef{
				{Name: "test", Type: "invalid", Eval: "GT(input.x, 10)"},
			},
		},
	}

	result := &flows.LinterResult{}
	checker.Check(flow, result)
	result.Valid = len(result.Errors) == 0

	assert.False(t, result.Valid)
	// Should have invalid type error
	assert.Greater(t, len(result.Errors), 0)
}

func TestComputedChecker_Check_NoEval(t *testing.T) {
	checker := NewComputedChecker()

	flow := &flows.Flow{
		Computed: &flows.ComputedBlock{
		Bools: []flows.ComputedFieldDef{
				{Name: "test", Type: "bool", Eval: ""},
			},
		},
	}

	result := &flows.LinterResult{}
	checker.Check(flow, result)
	result.Valid = len(result.Errors) == 0

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, flows.ErrInvalidExpr, result.Errors[0].Code)
}

func TestComputedChecker_Check_InvalidExpression(t *testing.T) {
	checker := NewComputedChecker()

	flow := &flows.Flow{
		Computed: &flows.ComputedBlock{
		Bools: []flows.ComputedFieldDef{
				{Name: "test", Type: "bool", Eval: "INVALID(input.x, 10)"},
			},
		},
	}

	result := &flows.LinterResult{}
	checker.Check(flow, result)
	result.Valid = len(result.Errors) == 0

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, flows.ErrInvalidExpr, result.Errors[0].Code)
}

func TestComputedChecker_Check_UndefinedReference(t *testing.T) {
	checker := NewComputedChecker()

	flow := &flows.Flow{
		Computed: &flows.ComputedBlock{
		Bools: []flows.ComputedFieldDef{
				{Name: "test", Type: "bool", Eval: "GT(input.undef, 10)"},
			},
		},
	}

	result := &flows.LinterResult{}
	checker.Check(flow, result)
	result.Valid = len(result.Errors) == 0

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, flows.ErrFieldNotFound, result.Errors[0].Code)
}

func TestComputedChecker_Check_CircularDependency(t *testing.T) {
	checker := NewComputedChecker()

	flow := &flows.Flow{
		Computed: &flows.ComputedBlock{
		Bools: []flows.ComputedFieldDef{
				{Name: "a", Type: "bool", Eval: "GT(computed.b, 0)"},
				{Name: "b", Type: "bool", Eval: "GT(computed.c, 0)"},
				{Name: "c", Type: "bool", Eval: "GT(computed.a, 0)"},
			},
		},
	}

	result := &flows.LinterResult{}
	checker.Check(flow, result)
	result.Valid = len(result.Errors) == 0

	assert.False(t, result.Valid)
	assert.Greater(t, len(result.Errors), 0)
	// Should have circular dependency error
	found := false
	for _, err := range result.Errors {
		if err.Code == flows.ErrCircularDeps {
			found = true
			break
		}
	}
	assert.True(t, found, "Should have circular dependency error")
}

func TestComputedChecker_Check_NoComputed(t *testing.T) {
	checker := NewComputedChecker()

	flow := &flows.Flow{
		Input: &flows.InputBlock{},
	}

	result := &flows.LinterResult{}
	checker.Check(flow, result)
	result.Valid = len(result.Errors) == 0

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}
