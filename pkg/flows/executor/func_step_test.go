package executor

import (
	"context"
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
						Params:   []flows.StepParam{{Name: "s", Value: "input.text"}},
						Result:   &flows.StepResult{AssignTo: "output.result"},
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
	err := exec.SetInput(map[string]string{"text": "hello"})
	require.NoError(t, err)
	step := &flow.States[0].Steps[0]

	err = exec.(*flowExecutorImpl).executeFuncStep(context.Background(), step, "init")

	require.NoError(t, err)
	result, err := exec.(*flowExecutorImpl).ctx.GetOutputField("result")
	require.NoError(t, err)
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
						Params:   []flows.StepParam{{Name: "s", Value: "input.text"}},
						Result:   &flows.StepResult{AssignTo: "output.result"},
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
	err := exec.SetInput(map[string]string{"text": "HELLO"})
	require.NoError(t, err)
	step := &flow.States[0].Steps[0]

	err = exec.(*flowExecutorImpl).executeFuncStep(context.Background(), step, "init")

	require.NoError(t, err)
	result, err := exec.(*flowExecutorImpl).ctx.GetOutputField("result")
	require.NoError(t, err)
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
			Bools: []flows.FieldDef{{Name: "result"}},
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
							{Name: "s", Value: "input.text"},
							{Name: "substr", Value: "input.substr"},
						},
						Result: &flows.StepResult{AssignTo: "output.result"},
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
	err := exec.SetInput(map[string]string{"text": "hello world", "substr": "world"})
	require.NoError(t, err)
	step := &flow.States[0].Steps[0]

	err = exec.(*flowExecutorImpl).executeFuncStep(context.Background(), step, "init")

	require.NoError(t, err)
	result, err := exec.(*flowExecutorImpl).ctx.GetOutputField("result")
	require.NoError(t, err)
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
						Params:   []flows.StepParam{{Name: "v", Value: "input.text"}},
						Result:   &flows.StepResult{AssignTo: "output.length"},
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
	err := exec.SetInput(map[string]string{"text": "hello"})
	require.NoError(t, err)
	step := &flow.States[0].Steps[0]

	err = exec.(*flowExecutorImpl).executeFuncStep(context.Background(), step, "init")

	require.NoError(t, err)
	result, err := exec.(*flowExecutorImpl).ctx.GetOutputField("length")
	require.NoError(t, err)
	assert.Equal(t, 5, result)
}
