package executor

import (
	"context"
	"fmt"
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMCPTool is a mock MCP tool for testing
type mockMCPTool struct {
	name        string
	description string
	runFunc     func(ctx context.Context, args map[string]any) (map[string]any, error)
}

func (m *mockMCPTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        m.name,
		Description: m.description,
	}
}

func (m *mockMCPTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, args)
	}
	return map[string]string{}, nil
}

// mockMCPToolSet is a mock ToolSet for testing
type mockMCPToolSet struct {
	tools []gollem.Tool
}

func (m *mockMCPToolSet) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	specs := make([]gollem.ToolSpec, len(m.tools))
	for i, tool := range m.tools {
		specs[i] = tool.Spec()
	}
	return specs, nil
}

func (m *mockMCPToolSet) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	for _, tool := range m.tools {
		if tool.Spec().Name == name {
			return tool.Run(ctx, args)
		}
	}
	return nil, fmt.Errorf("tool not found: %s", name)
}

// mockMCPRegistryWithTools is a mock MCP registry for testing
type mockMCPRegistryWithTools struct {
	toolSets []gollem.ToolSet
}

func (m *mockMCPRegistryWithTools) GetToolSets() []gollem.ToolSet {
	return m.toolSets
}

func (m *mockMCPRegistryWithTools) Close() error {
	return nil
}

func TestExecuteMCPStep_ToolCall(t *testing.T) {
	// Create a mock MCP tool that simulates tavily.search
	mockTool := &mockMCPTool{
		name:        "tavily.search",
		description: "Search the web",
		runFunc: func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Verify parameters
			query, ok := args["query"].(string)
			require.NoError(t, err)
			assert.Equal(t, "test search query", query)

			// Return mock search results
			return map[string]any{
				"results": []map[string]string{
					{"title": "Test Result 1", "url": "https://example.com/1"},
					{"title": "Test Result 2", "url": "https://example.com/2"},
				},
			}, nil
		},
	}

	// Create a tool set with the mock tool
	toolSet := &mockMCPToolSet{tools: []gollem.Tool{mockTool}}
	registry := &mockMCPRegistryWithTools{toolSets: []gollem.ToolSet{toolSet}}

	// Create test flow with MCP step
	flow := &flows.Flow{
		Name:    "test-mcp",
		Version: "1.0",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "query", Required: true}},
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
						Type:   "mcp",
						Tool:   "tavily.search",
						Params: []flows.StepParam{{Name: "query", Value: "test search query"}},
						Output: &flows.StepOutput{Assign: "${output.result}"},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	// Create executor with mock registry
	provider := &testBashToolProvider{}
	injector := setupTestDIWithBashProviderAndMCPRegistry(t, provider, registry)

	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	exec.SetInput(map[string]string{"query": "test search query"})

	// Execute the MCP step
	step := &flow.States[0].Steps[0]
	err := exec.(*flowExecutorImpl).executeMCPStep(step, "init")

	require.NoError(t, err)

	// Verify output was mapped
	result, err := exec.(*flowExecutorImpl).ctx.GetOutputField("result")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExecuteMCPStep_MultipleParams(t *testing.T) {
	// Create a mock MCP tool that accepts multiple parameters
	mockTool := &mockMCPTool{
		name:        "test.tool",
		description: "Test tool with multiple params",
		runFunc: func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Verify multiple parameters were passed
			assert.Equal(t, "value1", args["param1"])
			assert.Equal(t, "value2", args["param2"])
			assert.Equal(t, "123", args["param3"])

			return map[string]any{
				"success": true,
			}, nil
		},
	}

	toolSet := &mockMCPToolSet{tools: []gollem.Tool{mockTool}}
	registry := &mockMCPRegistryWithTools{toolSets: []gollem.ToolSet{toolSet}}

	// Create test flow with multiple params
	flow := &flows.Flow{
		Name:    "test-mcp-multi",
		Version: "1.0",
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type: "mcp",
						Tool: "test.tool",
						Params: []flows.StepParam{
							{Name: "param1", Value: "value1"},
							{Name: "param2", Value: "value2"},
							{Name: "param3", Value: "123"},
						},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	provider := &testBashToolProvider{}
	injector := setupTestDIWithBashProviderAndMCPRegistry(t, provider, registry)

	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)

	step := &flow.States[0].Steps[0]
	err := exec.(*flowExecutorImpl).executeMCPStep(step, "init")

	require.NoError(t, err)
}

func TestExecuteMCPStep_ToolNotFound(t *testing.T) {
	// Create an empty registry (no tools)
	registry := &mockMCPRegistryWithTools{toolSets: []gollem.ToolSet{}}

	flow := &flows.Flow{
		Name:    "test-mcp-notfound",
		Version: "1.0",
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type: "mcp",
						Tool: "nonexistent.tool",
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	provider := &testBashToolProvider{}
	injector := setupTestDIWithBashProviderAndMCPRegistry(t, provider, registry)

	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)

	step := &flow.States[0].Steps[0]
	err := exec.(*flowExecutorImpl).executeMCPStep(step, "init")

	// Should return MCPError with "tool not found"
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool not found")
}

func TestExecuteMCPStep_TemplateSubstitution(t *testing.T) {
	// Test that template variables are substituted in parameters
	receivedParams := make(map[string]any)

	mockTool := &mockMCPTool{
		name:        "test.tool",
		description: "Test tool",
		runFunc: func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Capture received params for verification
			for k, v := range args {
				receivedParams[k] = v
			}
			return map[string]string{}, nil
		},
	}

	toolSet := &mockMCPToolSet{tools: []gollem.Tool{mockTool}}
	registry := &mockMCPRegistryWithTools{toolSets: []gollem.ToolSet{toolSet}}

	flow := &flows.Flow{
		Name:    "test-mcp-template",
		Version: "1.0",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "name", Required: true}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type: "mcp",
						Tool: "test.tool",
						Params: []flows.StepParam{
							{Name: "static", Value: "fixed value"},
							{Name: "from_input", Value: "${input.name}"},
						},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	provider := &testBashToolProvider{}
	injector := setupTestDIWithBashProviderAndMCPRegistry(t, provider, registry)

	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	exec.SetInput(map[string]string{"name": "Alice"})

	step := &flow.States[0].Steps[0]
	err := exec.(*flowExecutorImpl).executeMCPStep(step, "init")

	require.NoError(t, err)

	// Verify template substitution worked
	assert.Equal(t, "fixed value", receivedParams["static"])
	assert.Equal(t, "Alice", receivedParams["from_input"])
}

func TestExecuteMCPStep_ToolExecutionError(t *testing.T) {
	// Create a mock tool that returns an error
	mockTool := &mockMCPTool{
		name:        "failing.tool",
		description: "Tool that fails",
		runFunc: func(ctx context.Context, args map[string]any) (map[string]any, error) {
			return nil, fmt.Errorf("tool execution failed")
		},
	}

	toolSet := &mockMCPToolSet{tools: []gollem.Tool{mockTool}}
	registry := &mockMCPRegistryWithTools{toolSets: []gollem.ToolSet{toolSet}}

	flow := &flows.Flow{
		Name:    "test-mcp-error",
		Version: "1.0",
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type: "mcp",
						Tool: "failing.tool",
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	provider := &testBashToolProvider{}
	injector := setupTestDIWithBashProviderAndMCPRegistry(t, provider, registry)

	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)

	step := &flow.States[0].Steps[0]
	err := exec.(*flowExecutorImpl).executeMCPStep(step, "init")

	// Should return MCPError with tool execution error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool execution failed")
}
