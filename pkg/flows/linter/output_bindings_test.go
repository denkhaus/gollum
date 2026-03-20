package linter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/denkhaus/gollum/pkg/flows"
)

func TestOutputBindingsChecker_Valid(t *testing.T) {
	flow := &flows.Flow{
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "msg", Type: flows.TypeString}},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{
				{Name: "result", From: "input.msg", Type: flows.TypeString},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.Empty(t, result.Errors)
	assert.Empty(t, result.Warnings)
}

func TestOutputBindingsChecker_InvalidFieldReference(t *testing.T) {
	flow := &flows.Flow{
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{
				{Name: "result", From: "invalidformat", Type: flows.TypeString},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "invalid")
}

func TestOutputBindingsChecker_FieldNotFound(t *testing.T) {
	flow := &flows.Flow{
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{
				{Name: "result", From: "computed.nonexistent", Type: flows.TypeString},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "does not exist")
}

func TestOutputBindingsChecker_InvalidScope(t *testing.T) {
	flow := &flows.OutputBlock{
		Strings: []flows.FieldDef{
			{Name: "result", From: "invalid.field", Type: flows.TypeString},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(&flows.Flow{Output: flow}, result)

	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "invalid scope")
}

func TestOutputBindingsChecker_CircularDependency(t *testing.T) {
	flow := &flows.Flow{
		Output: &flows.OutputBlock{
			Ints: []flows.FieldDef{
				{Name: "a", From: "output.b", Type: flows.TypeInt},
				{Name: "b", From: "output.a", Type: flows.TypeInt},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "circular")
}

func TestOutputBindingsChecker_DuplicateSource(t *testing.T) {
	flow := &flows.Flow{
		Computed: &flows.ComputedBlock{
			Ints: []flows.ComputedFieldDef{
				{Name: "sum", Type: flows.TypeInt, Eval: "ADD(input.a, input.b)"},
			},
		},
		Output: &flows.OutputBlock{
			Ints: []flows.FieldDef{
				{Name: "result1", From: "computed.sum", Type: flows.TypeInt},
				{Name: "result2", From: "computed.sum", Type: flows.TypeInt},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.NotEmpty(t, result.Warnings)
	assert.Contains(t, result.Warnings[0].Message, "multiple output fields")
}
