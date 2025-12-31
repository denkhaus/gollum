// Package mcp provides MCP (Model Context Protocol) client implementations.
package mcp

import (
	"context"

	"github.com/m-mizutani/gollem/mcp"
)

// NewBrainMCPClient creates a new MCP client for the brain service
func NewBrainMCPClient(ctx context.Context) (*mcp.Client, error) {
	return mcp.NewStreamableHTTP(ctx, "http://localhost:8555/mcp",
		mcp.WithStreamableHTTPClientInfo("brain-mcp-client", "1.0.0"),
	)
}
