package linter

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestOutputBindingsChecker_Valid(t *testing.T) {
	flow := &flows.Flow{
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "msg", Type: flows.TypeString}},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{
				{Name: "result", AssignFrom: "input.msg", Type: flows.TypeString},
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
				{Name: "result", AssignFrom: "invalidformat", Type: flows.TypeString},
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
				{Name: "result", AssignFrom: "computed.nonexistent", Type: flows.TypeString},
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
			{Name: "result", AssignFrom: "invalid.field", Type: flows.TypeString},
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
				{Name: "a", AssignFrom: "output.b", Type: flows.TypeInt},
				{Name: "b", AssignFrom: "output.a", Type: flows.TypeInt},
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
				{Name: "result1", AssignFrom: "computed.sum", Type: flows.TypeInt},
				{Name: "result2", AssignFrom: "computed.sum", Type: flows.TypeInt},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.NotEmpty(t, result.Warnings)
	assert.Contains(t, result.Warnings[0].Message, "multiple output fields")
}

func TestOutputBindingsChecker_StepOutputAssignInvalidDollarSyntax(t *testing.T) {
	flow := &flows.Flow{
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result", Type: flows.TypeString}},
		},
		States: []flows.State{
			{
				Name: "init",
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "test",
						Result: &flows.StepResult{AssignTo: "${output.result}"},
					},
				},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "invalid assign syntax")
	assert.Contains(t, result.Errors[0].Message, "without ${}")
}

func TestOutputBindingsChecker_StepOutputAssignInvalidScope(t *testing.T) {
	flow := &flows.Flow{
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result", Type: flows.TypeString}},
		},
		States: []flows.State{
			{
				Name: "init",
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "test",
						Result: &flows.StepResult{AssignTo: "input.result"},
					},
				},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "invalid scope")
	assert.Contains(t, result.Errors[0].Message, "must be 'output.field'")
}

func TestOutputBindingsChecker_StepOutputAssignFieldNotFound(t *testing.T) {
	flow := &flows.Flow{
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result", Type: flows.TypeString}},
		},
		States: []flows.State{
			{
				Name: "init",
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "test",
						Result: &flows.StepResult{AssignTo: "output.nonexistent"},
					},
				},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "does not exist")
}

func TestOutputBindingsChecker_StepOutputAssignValid(t *testing.T) {
	flow := &flows.Flow{
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result", Type: flows.TypeString}},
		},
		States: []flows.State{
			{
				Name: "init",
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "test",
						Result: &flows.StepResult{AssignTo: "output.result"},
					},
				},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.Empty(t, result.Errors)
}

func TestOutputBindingsChecker_StepOutputAssignContextScopeValid(t *testing.T) {
	flow := &flows.Flow{
		Context: &flows.ContextBlock{
			Strings: []flows.ContextField{{Name: "temp_result", Type: flows.TypeString}},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result", Type: flows.TypeString}},
		},
		States: []flows.State{
			{
				Name: "init",
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "test",
						Result: &flows.StepResult{AssignTo: "context.temp_result"},
					},
				},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.Empty(t, result.Errors)
}

func TestOutputBindingsChecker_StepOutputAssignContextScopeFieldNotFound(t *testing.T) {
	flow := &flows.Flow{
		Context: &flows.ContextBlock{
			Strings: []flows.ContextField{{Name: "temp_result", Type: flows.TypeString}},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result", Type: flows.TypeString}},
		},
		States: []flows.State{
			{
				Name: "init",
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "test",
						Result: &flows.StepResult{AssignTo: "context.nonexistent"},
					},
				},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	// Should have error for non-existent context field
	if len(result.Errors) == 0 {
		t.Fatal("expected errors but got none")
	}
	assert.Contains(t, result.Errors[0].Message, "does not exist", "error message should mention field does not exist")
}

func TestOutputBindingsChecker_AssignFromWithTemplateNotation(t *testing.T) {
	// Test E009: assignFrom should not use ${} notation
	flow := &flows.Flow{
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "source", Type: flows.TypeString}},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "target", Type: flows.TypeString, AssignFrom: "${input.source}"}},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.NotEmpty(t, result.Errors)
	assert.Equal(t, "E009", string(result.Errors[0].Code))
	assert.Contains(t, result.Errors[0].Message, "invalid assignFrom syntax")
	assert.Contains(t, result.Errors[0].Message, "without ${}")
}

func TestOutputBindingsChecker_AssignFromMissingScopePrefix(t *testing.T) {
	// Test E013: assignFrom must have scope prefix
	flow := &flows.Flow{
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "source", Type: flows.TypeString}},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "target", Type: flows.TypeString, AssignFrom: "source"}},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "must use scope.field notation")
}

func TestOutputBindingsChecker_StepOutputAssignMissingScopePrefix(t *testing.T) {
	// Test E013: assignTo must have scope prefix
	flow := &flows.Flow{
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result", Type: flows.TypeString}},
		},
		States: []flows.State{
			{
				Name: "init",
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "test",
						Result: &flows.StepResult{AssignTo: "result"},
					},
				},
			},
		},
	}

	checker := NewOutputBindingsChecker()
	result := &flows.LinterResult{}
	checker.Check(flow, result)

	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "must use scope.field notation")
}
