package app

import (
	"context"

	"github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/m-mizutani/gollem"
)

// convertToolSetsToAllowedTools converts MCP ToolSets to AllowedTools string slice.
// Each tool is formatted as "server_name/tool_name" for MCP tools.
func convertToolSetsToAllowedTools(ctx context.Context, toolSets []gollem.ToolSet, mcpRegistry registry.MCPRegistry) []string {
	// Get tool names directly from registry in "server_name/tool_name" format
	return mcpRegistry.GetToolNames()
}
