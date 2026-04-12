package executor

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBashToolRunner is a simple mock for testing without using gomock
type mockBashToolRunner struct {
	runFunc func(ctx context.Context, args map[string]any) (map[string]any, error)
}

func (m *mockBashToolRunner) Name() string {
	return "bash"
}

func (m *mockBashToolRunner) Description() string {
	return "Mock bash tool"
}

func (m *mockBashToolRunner) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "bash",
		Description: "Mock bash tool",
	}
}

func (m *mockBashToolRunner) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, args)
	}
	return map[string]any{
		"stdout":    "",
		"stderr":    "",
		"exit_code": 0,
	}, nil
}

// mockBashToolProvider is a simple provider for testing
type mockBashToolProvider struct {
	tool *mockBashToolRunner
}

func (m *mockBashToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	if m.tool != nil {
		return m.tool
	}
	return &mockBashToolRunner{}
}

func TestExecuteShellStep_SuccessfulExecution(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test-shell",
		Version: "1.0",
		Output:  &flows.OutputBlock{Strings: []flows.FieldDef{{Name: "greeting"}}},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type: "shell",
						Cmd:  "echo 'Hello World'",
						Result: &flows.StepResult{
							Paths: []flows.ResultPath{
								{Path: "stdout", AssignTo: "output.greeting"},
							},
						},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	mockTool := &mockBashToolRunner{
		runFunc: func(ctx context.Context, args map[string]any) (map[string]any, error) {
			assert.Equal(t, "echo 'Hello World'", args["command"])
			return map[string]any{
				"stdout":    "Hello World\n",
				"stderr":    "",
				"exit_code": 0,
			}, nil
		},
	}

	provider := &mockBashToolProvider{tool: mockTool}
	// Create fresh injector with all dependencies
	injector := setupTestDIWithBashProvider(t, provider)

	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	step := &flow.States[0].Steps[0]
	err := exec.(*flowExecutorImpl).executeShellStep(context.Background(), step, "init")

	require.NoError(t, err)
	result, err := exec.(*flowExecutorImpl).ctx.GetOutputField("greeting")
	require.NoError(t, err)
	assert.Equal(t, "Hello World\n", result)
}

func TestExecuteShellStep_WithInputVariable(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test-shell-var",
		Version: "1.0",
		Input:   &flows.InputBlock{Strings: []flows.FieldDef{{Name: "name"}}},
		Output:  &flows.OutputBlock{Strings: []flows.FieldDef{{Name: "greeting"}}},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type: "shell",
						Cmd:  "echo 'Hello ${input.name}'",
						Result: &flows.StepResult{
							Paths: []flows.ResultPath{{Path: "stdout", AssignTo: "output.greeting"}},
						},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	mockTool := &mockBashToolRunner{
		runFunc: func(ctx context.Context, args map[string]any) (map[string]any, error) {
			assert.Equal(t, "echo 'Hello Claude'", args["command"])
			return map[string]any{
				"stdout":    "Hello Claude\n",
				"stderr":    "",
				"exit_code": 0,
			}, nil
		},
	}

	provider := &mockBashToolProvider{tool: mockTool}
	// Create fresh injector with all dependencies
	injector := setupTestDIWithBashProvider(t, provider)

	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	err := exec.SetInput(map[string]string{"name": "Claude"})
	require.NoError(t, err)

	step := &flow.States[0].Steps[0]
	err = exec.(*flowExecutorImpl).executeShellStep(context.Background(), step, "init")

	require.NoError(t, err)
	result, err := exec.(*flowExecutorImpl).ctx.GetOutputField("greeting")
	require.NoError(t, err)
	assert.Equal(t, "Hello Claude\n", result)
}

func TestExecuteShellStep_WithTimeout(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test-timeout",
		Version: "1.0",
		Output:  &flows.OutputBlock{Strings: []flows.FieldDef{{Name: "result"}}},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:    "shell",
						Cmd:     "sleep 1",
						Timeout: "500ms",
						Result:  &flows.StepResult{AssignTo: "output.result"},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	mockTool := &mockBashToolRunner{
		runFunc: func(ctx context.Context, args map[string]any) (map[string]any, error) {
			timeout, ok := args["timeout"]
			require.True(t, ok)
			assert.Equal(t, 0.5, timeout)
			return map[string]any{
				"stdout":    "",
				"exit_code": 0,
			}, nil
		},
	}

	provider := &mockBashToolProvider{tool: mockTool}
	// Create fresh injector with all dependencies
	injector := setupTestDIWithBashProvider(t, provider)

	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	step := &flow.States[0].Steps[0]
	err := exec.(*flowExecutorImpl).executeShellStep(context.Background(), step, "init")

	require.NoError(t, err)
}

func TestSubstituteTemplate_InputVariables(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{Strings: []flows.FieldDef{{Name: "name"}}},
		States: []flows.State{{Name: "init"}},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	err := exec.SetInput(map[string]string{"name": "Claude"})
	require.NoError(t, err)

	result := exec.GetContext().SubstituteTemplate("echo 'Hello ${input.name}'")

	assert.Equal(t, "echo 'Hello Claude'", result)
}

func TestSubstituteTemplate_ContextVariables(t *testing.T) {
	flow := &flows.Flow{
		Name: "test",
		Context: &flows.ContextBlock{
			Strings: []flows.ContextField{{Name: "project_dir", Default: "/tmp/project"}},
		},
		States: []flows.State{{Name: "init"}},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	exec.(*flowExecutorImpl).ctx = newContext(flow.Input, flow.Output, flow.Context, nil)
	// Initialize context with default values manually for this test
	if flow.Context != nil {
		for _, field := range flow.Context.Strings {
			_ = exec.(*flowExecutorImpl).ctx.SetContextField(field.Name, field.Default)
		}
	}

	result := exec.GetContext().SubstituteTemplate("ls ${context.project_dir}")

	assert.Equal(t, "ls /tmp/project", result)
}

func TestSubstituteTemplate_OutputVariables(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Output: &flows.OutputBlock{Strings: []flows.FieldDef{{Name: "result"}}},
		States: []flows.State{{Name: "init"}},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	// Set output value before substitution
	exec.(*flowExecutorImpl).ctx = newContext(flow.Input, flow.Output, flow.Context, nil)
	_ = exec.(*flowExecutorImpl).ctx.SetOutputField("result", "success")

	result := exec.GetContext().SubstituteTemplate("echo 'Status: ${output.result}'")

	assert.Equal(t, "echo 'Status: success'", result)
}

func TestSubstituteTemplate_MultipleVariables(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test",
		Input:   &flows.InputBlock{Strings: []flows.FieldDef{{Name: "name"}, {Name: "action"}}},
		Context: &flows.ContextBlock{Strings: []flows.ContextField{{Name: "env", Default: "prod"}}},
		Output:  &flows.OutputBlock{Strings: []flows.FieldDef{{Name: "result"}}},
		States:  []flows.State{{Name: "init"}},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	err := exec.SetInput(map[string]string{"name": "app", "action": "deploy"})
	require.NoError(t, err)
	// Initialize context with default values manually for this test
	if flow.Context != nil {
		for _, field := range flow.Context.Strings {
			_ = exec.(*flowExecutorImpl).ctx.SetContextField(field.Name, field.Default)
		}
	}
	_ = exec.(*flowExecutorImpl).ctx.SetOutputField("result", "pending")

	cmd := "${input.action} ${input.name} in ${context.env}, status: ${output.result}"
	result := exec.GetContext().SubstituteTemplate(cmd)

	assert.Equal(t, "deploy app in prod, status: pending", result)
}
