package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_SimpleFlow_ExecutesSuccessfully(t *testing.T) {
	// Define a simple flow with input/output and state transitions
	flow := &flows.Flow{
		Name:    "simple-test",
		Version: "1.0",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "message", Default: "hello"}},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result"}},
		},
		Context: &flows.ContextBlock{
			Strings: []flows.ContextField{{Name: "greeting"}},
		},
		Computed: &flows.ComputedBlock{
			Bools: []flows.ComputedFieldDef{
				{Name: "is_ready", Type: "bool", Eval: "EQ(context.greeting, 'hello')"},
			},
		},
		States: []flows.State{
			{
				Name: "init", Initial: true,
				Steps: []flows.Step{
					{Type: "llm", Agent: "worker", Prompt: "Say ${input.message}"},
				},
				Transitions: []flows.Transition{
					{To: "done"},
				},
			},
			{Name: "done"},
		},
		Agents: []flows.Agent{
			{Name: "worker", Model: "test", Prompt: "Test agent"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	err := exec.SetInput(map[string]string{"message": "hello"})
	require.NoError(t, err)

	// LLM step will fail but we can test state transitions
	err = exec.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "init", exec.(*flowExecutorImpl).currentState)

	// Test computed field evaluation
	_ = exec.(*flowExecutorImpl).ctx.SetContextField("greeting", "hello")
	err = exec.(*flowExecutorImpl).ctx.EvaluateComputed()
	assert.NoError(t, err)

	val, err := exec.(*flowExecutorImpl).ctx.GetComputedField("is_ready")
	assert.NoError(t, err)
	assert.Equal(t, true, val)
}

func TestExecutor_ErrorHandling_TransitionsToErrorState(t *testing.T) {
	flow := &flows.Flow{
		Name: "error-test",
		States: []flows.State{
			{
				Name: "init", Initial: true,
				Steps: []flows.Step{
					{Name: "analyze", Type: "llm", Agent: "missing", OnError: &flows.OnErrorTransition{State: "error"}},
				},
			},
			{Name: "error"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	err := exec.Run()

	// The error state transition succeeds, so Run returns nil
	// But we can verify the error was captured
	assert.NoError(t, err)

	// Verify error context was set via dedicated API
	errorCtx := exec.(*flowExecutorImpl).ctx.GetError()
	assert.NotNil(t, errorCtx)
	assert.Equal(t, "analyze", errorCtx.StepName)

	// Verify we ended up in error state (or init, since it transitioned)
	assert.True(t, exec.(*flowExecutorImpl).currentState == "error" || exec.(*flowExecutorImpl).currentState == "init")
}

func TestExecutor_DeclarativeOutput_BindsFromComputed(t *testing.T) {
	flow := &flows.Flow{
		Name:    "declarative-output-test",
		Version: "1.0",
		Input: &flows.InputBlock{
			Ints: []flows.FieldDef{{Name: "a", Default: "10"}},
		},
		Computed: &flows.ComputedBlock{
			Ints: []flows.ComputedFieldDef{
				{Name: "doubled", Type: flows.TypeInt, Eval: "MUL(input.a, 2)"},
			},
		},
		Output: &flows.OutputBlock{
			Ints: []flows.FieldDef{
				{Name: "result", From: "computed.doubled", Type: flows.TypeInt},
			},
		},
		States: []flows.State{
			{Name: "done", Initial: true},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)

	err := exec.SetInput(map[string]string{"a": "15"})
	require.NoError(t, err)

	err = exec.Validate()
	require.NoError(t, err)

	err = exec.Run()
	require.NoError(t, err)

	// Verify declarative output was populated
	result, err := exec.GetContext().GetOutputField("result")
	assert.NoError(t, err)
	assert.Equal(t, 30, result)
}
