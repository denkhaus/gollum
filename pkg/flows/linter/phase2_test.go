package linter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		Computed: &flows.ComputedBlock{
			Bools: []flows.ComputedFieldDef{
				{Name: "is_open", Eval: "INVALID(context.pr.state,"},
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
		},
		Computed: &flows.ComputedBlock{
			Bools: []flows.ComputedFieldDef{
				{Name: "is_open", Eval: "EQ(context.status, 'open')"},
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
						Input: &flows.CallInputBlock{
							Strings: []flows.CallInputParam{
								{
									Name:       "dir",
									AssignFrom: "${target}",
								},
							}, // template notation - should error
						},
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, result.Valid)
	assert.True(t, hasErrorCode(result, flows.ErrTemplateNotation), "should have template notation error for call input with ${}")
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
						Output: &flows.CallOutputBlock{
							Strings: []flows.CallOutputParam{
								{
									Name:    "result",
									AssignTo: "${score}",
								},
							}, // template notation - should error
						},
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, result.Valid)
	assert.True(t, hasErrorCode(result, flows.ErrTemplateNotation), "should have template notation error for call output with ${}")
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
							{Name: "input", AssignFrom: "${data}"}, // template notation - should error
						},
					},
				},
			},
		},
	}

	result := Lint(flow)
	assert.False(t, result.Valid)
	assert.True(t, hasErrorCode(result, flows.ErrTemplateNotation), "should have template notation error for step param with ${}")
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
	// Use artifact directory for test flow files (saved when -artifacts flag is used)
	tmpDir := t.ArtifactDir()

	// Create a simple sub-flow file
	subFlowContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="sub-flow" version="1.0">
	<input>
		<string name="dir" required="true" />
	</input>
	<output>
		<string name="result" />
	</output>
	<states>
		<state name="process" initial="true">
			<transitions>
				<transition to="done" />
			</transitions>
		</state>
		<state name="done" />
	</states>
</flow>
`
	subFlowPath := filepath.Join(tmpDir, "sub-flow.xml")
	require.NoError(t, os.WriteFile(subFlowPath, []byte(subFlowContent), 0644))

	// Create the main flow
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
						Input: &flows.CallInputBlock{
							Strings: []flows.CallInputParam{
								{
									Name:       "dir",
									AssignFrom: "input.target",
								},
							}, // valid bare notation
						},
					},
				},
			},
		},
	}

	// Lint with the temporary directory as the flow path
	result := LintPath(filepath.Join(tmpDir, "main.xml"), flow)
	if !result.Valid {
		t.Logf("Errors: %+v", result.Errors)
		for _, e := range result.Errors {
			t.Logf("  - Code: %s, Message: %s, Context: %s", e.Code, e.Message, e.Context)
		}
	}
	assert.True(t, result.Valid, "should be valid with bare notation and valid scope")
}

func TestPhase2_ValidAbsoluteVarRefInCallOutput(t *testing.T) {
	// Use artifact directory for test flow files (saved when -artifacts flag is used)
	tmpDir := t.ArtifactDir()

	// Create a simple sub-flow file
	subFlowContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="sub-flow" version="1.0">
	<input>
		<string name="input" />
	</input>
	<output>
		<int name="result" />
	</output>
	<states>
		<state name="process" initial="true">
			<transitions>
				<transition to="done" />
			</transitions>
		</state>
		<state name="done" />
	</states>
</flow>
`
	subFlowPath := filepath.Join(tmpDir, "sub-flow.xml")
	require.NoError(t, os.WriteFile(subFlowPath, []byte(subFlowContent), 0644))

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
						Output: &flows.CallOutputBlock{
							Ints: []flows.CallOutputParam{
								{
									Name:    "result",
									AssignTo: "context.score",
								},
							}, // valid bare notation
						},
					},
				},
			},
		},
	}

	result := LintPath(filepath.Join(tmpDir, "main.xml"), flow)
	if !result.Valid {
		t.Logf("Errors: %+v", result.Errors)
		for _, e := range result.Errors {
			t.Logf("  - Code: %s, Message: %s, Context: %s", e.Code, e.Message, e.Context)
		}
	}
	assert.True(t, result.Valid, "should be valid with bare notation and valid scope")
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
						Prompt: "Error: ${sys.error.Message} in ${sys.error.StepName}", // error context via sys scope
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
