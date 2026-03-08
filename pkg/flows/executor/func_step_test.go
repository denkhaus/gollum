package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFuncStep_StringsToUpper(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test-func",
		Version: "1.0",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "text", Required: true}},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result"}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:     "func",
						Function: "strings.ToUpper",
						Params:   []flows.StepParam{{Name: "s", Value: "${input.text}"}},
						Output:   &flows.StepOutput{Assign: "${output.result}"},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	exec.SetInput(map[string]any{"text": "hello"})
	step := &flow.States[0].Steps[0]

	err := exec.(*flowExecutorImpl).executeFuncStep(step, "init")

	require.NoError(t, err)
	result, ok := exec.(*flowExecutorImpl).ctx.GetOutputField("result")
	require.True(t, ok)
	assert.Equal(t, "HELLO", result)
}

func TestExecuteFuncStep_StringsToLower(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test-func",
		Version: "1.0",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "text", Required: true}},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result"}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:     "func",
						Function: "strings.ToLower",
						Params:   []flows.StepParam{{Name: "s", Value: "${input.text}"}},
						Output:   &flows.StepOutput{Assign: "${output.result}"},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	exec.SetInput(map[string]any{"text": "HELLO"})
	step := &flow.States[0].Steps[0]

	err := exec.(*flowExecutorImpl).executeFuncStep(step, "init")

	require.NoError(t, err)
	result, ok := exec.(*flowExecutorImpl).ctx.GetOutputField("result")
	require.True(t, ok)
	assert.Equal(t, "hello", result)
}

func TestExecuteFuncStep_StringsContains(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test-func",
		Version: "1.0",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{
				{Name: "text", Required: true},
				{Name: "substr", Required: true},
			},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result"}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:     "func",
						Function: "strings.Contains",
						Params: []flows.StepParam{
							{Name: "s", Value: "${input.text}"},
							{Name: "substr", Value: "${input.substr}"},
						},
						Output: &flows.StepOutput{Assign: "${output.result}"},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	exec.SetInput(map[string]any{"text": "hello world", "substr": "world"})
	step := &flow.States[0].Steps[0]

	err := exec.(*flowExecutorImpl).executeFuncStep(step, "init")

	require.NoError(t, err)
	result, ok := exec.(*flowExecutorImpl).ctx.GetOutputField("result")
	require.True(t, ok)
	assert.Equal(t, true, result)
}

func TestExecuteFuncStep_Len_String(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test-func",
		Version: "1.0",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "text", Required: true}},
		},
		Output: &flows.OutputBlock{
			Ints: []flows.FieldDef{{Name: "length"}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:     "func",
						Function: "len",
						Params:   []flows.StepParam{{Name: "v", Value: "${input.text}"}},
						Output:   &flows.StepOutput{Assign: "${output.length}"},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	exec.SetInput(map[string]any{"text": "hello"})
	step := &flow.States[0].Steps[0]

	err := exec.(*flowExecutorImpl).executeFuncStep(step, "init")

	require.NoError(t, err)
	result, ok := exec.(*flowExecutorImpl).ctx.GetOutputField("length")
	require.True(t, ok)
	assert.Equal(t, 5, result)
}
