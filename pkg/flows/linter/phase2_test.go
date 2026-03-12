package linter

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

// hasErrorCode checks if result contains a specific error code
func hasErrorCode(result *flows.LinterResult, code flows.ErrorCode) bool {
	for _, err := range result.Errors {
		if err.Code == code {
			return true
		}
	}
	return false
}

func TestPhase2_InvalidExpression(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		Context: &flows.ContextBlock{
			Computeds: []flows.ComputedField{
				{Name: "is_open", When: "INVALID(context.pr.state,"},
			},
		},
	}

	result := Lint(flow)

	assert.False(t, result.Valid)
	// Should have an expression error
	hasExprError := false
	for _, err := range result.Errors {
		if err.Code == flows.ErrInvalidExpr {
			hasExprError = true
			break
		}
	}
	assert.True(t, hasExprError, "should have expression error")
}

func TestPhase2_FieldExists(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		Context: &flows.ContextBlock{
			Strings: []flows.ContextField{
				{Name: "status"},
			},
			Computeds: []flows.ComputedField{
				{Name: "is_open", When: "EQ(context.status, 'open')"},
			},
		},
	}

	result := Lint(flow)

	// Should validate successfully - field exists
	hasFieldNotFound := false
	for _, err := range result.Errors {
		if err.Code == flows.ErrFieldNotFound {
			hasFieldNotFound = true
			break
		}
	}
	assert.False(t, hasFieldNotFound, "should not have field not found error")
}

// ========== Bare Variable Reference Tests ==========

func TestPhase2_BareVarRefInCallInput(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{Strings: []flows.FieldDef{{Name: "target"}}},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Calls: []flows.Call{
					{
						Ref: "sub-flow",
						Input: []flows.CallField{
							{Name: "dir", Value: "${target}"}, // bare - should error
						},
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, result.Valid)
	assert.True(t, hasErrorCode(result, flows.ErrRelativePath), "should have relative path error for bare call input")
}

func TestPhase2_BareVarRefInCallOutput(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		Context: &flows.ContextBlock{
			Ints: []flows.ContextField{{Name: "score"}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Calls: []flows.Call{
					{
						Ref: "sub-flow",
						Output: []flows.CallField{
							{Name: "result", Value: "${score}"}, // bare - should error
						},
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, result.Valid)
	assert.True(t, hasErrorCode(result, flows.ErrRelativePath), "should have relative path error for bare call output")
}

func TestPhase2_BareVarRefInStepParam(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		Context: &flows.ContextBlock{
			Strings: []flows.ContextField{{Name: "data"}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type: "func",
						Name: "process",
						Params: []flows.StepParam{
							{Name: "input", Value: "${data}"}, // bare - should error
						},
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, result.Valid)
	assert.True(t, hasErrorCode(result, flows.ErrRelativePath), "should have relative path error for bare step param")
}

func TestPhase2_BareVarRefInStepPrompt(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		Context: &flows.ContextBlock{
			Ints: []flows.ContextField{{Name: "count"}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "coordinator",
						Prompt: "Process ${count} items", // bare - should error
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, result.Valid)
	assert.True(t, hasErrorCode(result, flows.ErrRelativePath), "should have relative path error for bare step prompt")
}

func TestPhase2_ValidAbsoluteVarRefInCallInput(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{Strings: []flows.FieldDef{{Name: "target"}}},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Calls: []flows.Call{
					{
						Ref: "sub-flow",
						Input: []flows.CallField{
							{Name: "dir", Value: "${input.target}"}, // valid absolute path
						},
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, hasErrorCode(result, flows.ErrRelativePath), "should not have relative path error for valid input prefix")
}

func TestPhase2_ValidAbsoluteVarRefInCallOutput(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		Context: &flows.ContextBlock{
			Ints: []flows.ContextField{{Name: "score"}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Calls: []flows.Call{
					{
						Ref: "sub-flow",
						Output: []flows.CallField{
							{Name: "result", Value: "${context.score}"}, // valid absolute path
						},
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, hasErrorCode(result, flows.ErrRelativePath), "should not have relative path error for valid context prefix")
}

func TestPhase2_ValidAbsoluteVarRefInStepPrompt(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{Strings: []flows.FieldDef{{Name: "name"}}},
		Output: &flows.OutputBlock{},
		Context: &flows.ContextBlock{
			Ints: []flows.ContextField{{Name: "count"}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "coordinator",
						Prompt: "Process ${input.name} with ${context.count} items", // valid absolute paths
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, hasErrorCode(result, flows.ErrRelativePath), "should not have relative path error for valid prefixes in prompt")
}

func TestPhase2_ErrorPrefixVarRefIsValid(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "coordinator",
						Prompt: "Error: ${error.message} in ${error.step_name}", // error prefix is valid
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, hasErrorCode(result, flows.ErrRelativePath), "should not have relative path error for error prefix")
}

func TestPhase2_OutputPrefixVarRefIsValid(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{Strings: []flows.FieldDef{{Name: "result"}}},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "coordinator",
						Prompt: "Result: ${output.result}", // output prefix is valid
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, hasErrorCode(result, flows.ErrRelativePath), "should not have relative path error for output prefix")
}

func TestPhase2_MultipleBareVarRefsInPrompt(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		Context: &flows.ContextBlock{
			Ints:    []flows.ContextField{{Name: "count"}},
			Strings: []flows.ContextField{{Name: "name"}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:   "llm",
						Agent:  "coordinator",
						Prompt: "Process ${count} items for ${name}", // both bare - should get 2 errors
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, result.Valid)

	// Count errors with ErrRelativePath
	errorCount := 0
	for _, err := range result.Errors {
		if err.Code == flows.ErrRelativePath {
			errorCount++
		}
	}
	assert.Equal(t, 2, errorCount, "should have 2 relative path errors")
}
