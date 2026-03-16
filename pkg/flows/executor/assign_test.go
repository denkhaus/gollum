package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssignStep_FromInputToOutput tests assigning from input to output
func TestAssignStep_FromInputToOutput(t *testing.T) {
	flow := &flows.Flow{
		Name:    "assign-test",
		Version: "1.0",
		Input:   &flows.InputBlock{Strings: []flows.FieldDef{{Name: "message"}}},
		Output:  &flows.OutputBlock{Strings: []flows.FieldDef{{Name: "result"}}},
		States: []flows.State{
			{
				Name:    "assign",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:     "func",
						Function: "assign",
						Params: []flows.StepParam{
							{Name: "from", Value: "input.message"},
							{Name: "to", Value: "output.result"},
						},
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

	err := exec.SetInput(map[string]string{"message": "Hello World"})
	require.NoError(t, err)

	err = exec.Run()
	require.NoError(t, err)

	result, err := exec.GetContext().GetOutputField("result")
	require.NoError(t, err)
	assert.Equal(t, "Hello World", result)
}

// TestAssignStep_FromComputedToOutput tests assigning from computed to output
func TestAssignStep_FromComputedToOutput(t *testing.T) {
	flow := &flows.Flow{
		Name:    "computed-assign-test",
		Version: "1.0",
		Input:   &flows.InputBlock{Ints: []flows.FieldDef{{Name: "a"}, {Name: "b"}}},
		Output:  &flows.OutputBlock{Ints: []flows.FieldDef{{Name: "sum"}}},
		Context: &flows.ContextBlock{Ints: []flows.ContextField{{Name: "sum"}}},
		Computed: &flows.ComputedBlock{
			Ints: []flows.ComputedFieldDef{
				{Name: "sum", Type: "int", Eval: "ADD(input.a, input.b)"},
			},
		},
		States: []flows.State{
			{
				Name:    "assign",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:     "func",
						Function: "assign",
						Params: []flows.StepParam{
							{Name: "from", Value: "computed.sum"},
							{Name: "to", Value: "output.sum"},
						},
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

	err := exec.SetInput(map[string]string{"a": "10", "b": "5"})
	require.NoError(t, err)

	err = exec.Run()
	require.NoError(t, err)

	result, err := exec.GetContext().GetOutputField("sum")
	require.NoError(t, err)
	// Should be 15 (10 + 5)
	// Note: may be int or int64 depending on implementation
	assert.Equal(t, 15, result)
}

// TestAssignStep_WithDirectValue tests assigning a direct value
func TestAssignStep_WithDirectValue(t *testing.T) {
	flow := &flows.Flow{
		Name:    "value-assign-test",
		Version: "1.0",
		Output:  &flows.OutputBlock{Strings: []flows.FieldDef{{Name: "message"}}},
		States: []flows.State{
			{
				Name:    "assign",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:     "func",
						Function: "assign",
						Params: []flows.StepParam{
							{Name: "value", Value: "Hello from assign!"},
							{Name: "to", Value: "output.message"},
						},
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

	err := exec.Run()
	require.NoError(t, err)

	result, err := exec.GetContext().GetOutputField("message")
	require.NoError(t, err)
	assert.Equal(t, "Hello from assign!", result)
}

// TestAssignStep_ToContext tests assigning to context
func TestAssignStep_ToContext(t *testing.T) {
	flow := &flows.Flow{
		Name:    "context-assign-test",
		Version: "1.0",
		Input:   &flows.InputBlock{Strings: []flows.FieldDef{{Name: "value"}}},
		Context: &flows.ContextBlock{Strings: []flows.ContextField{{Name: "stored"}}},
		States: []flows.State{
			{
				Name:    "assign",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:     "func",
						Function: "assign",
						Params: []flows.StepParam{
							{Name: "from", Value: "input.value"},
							{Name: "to", Value: "context.stored"},
						},
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

	err := exec.SetInput(map[string]string{"value": "test123"})
	require.NoError(t, err)

	err = exec.Run()
	require.NoError(t, err)

	result, err := exec.GetContext().GetContextField("stored")
	require.NoError(t, err)
	assert.Equal(t, "test123", result)
}
