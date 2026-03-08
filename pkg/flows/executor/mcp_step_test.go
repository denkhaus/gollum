package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
)

// NOTE: MCP Step implementation requires:
// 1. MCP client registry (similar to FlowRegistry)
// 2. Tool discovery (list available tools from MCP servers)
// 3. Tool execution with parameter marshaling
// 4. Error handling for MCP-specific errors
//
// This is deferred pending MCP client infrastructure.
// The tests below show the expected interface.

func TestExecuteMCPStep_ToolCall(t *testing.T) {
	t.Skip("MCP step requires MCP client registry - deferred")

	_ = &flows.Flow{
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
						Type:     "mcp",
						Tool:     "tavily.search",
						Params:   []flows.StepParam{{Name: "query", Value: "${input.query}"}},
						Output:   &flows.StepOutput{Assign: "${output.result}"},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	// Expected implementation:
	// 1. Create MCP client registry
	// 2. Register MCP clients (tavily, exa, forgejo, etc.)
	// 3. Create executor with registry
	// 4. Execute MCP step
	// 5. Verify output is mapped correctly

	t.Log("MCP step implementation requires:")
	t.Log("  - MCP client registry: NewMCPClientRegistry() *MCPClientRegistry")
	t.Log("  - Client registration: registry.Register(name, mcp.Client)")
	t.Log("  - Tool discovery: client.ListTools() ([]Tool, error)")
	t.Log("  - Tool execution: client.CallTool(name, args) (result, error)")
}

func TestExecuteMCPStep_MultipleParams(t *testing.T) {
	t.Skip("MCP step requires MCP client registry - deferred")

	// Test tool call with multiple parameters
	t.Log("Should support parameter substitution from input, context, and output")
}

func TestExecuteMCPStep_ErrorHandling(t *testing.T) {
	t.Skip("MCP step requires MCP client registry - deferred")

	// Test MCP-specific error handling
	t.Log("Should handle tool not found, parameter errors, connection errors")
}
