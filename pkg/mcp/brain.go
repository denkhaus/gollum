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

func NewExaSearchMCPClient(ctx context.Context) (*mcp.Client, error) {
	return mcp.NewStreamableHTTP(ctx, "https://mcp.exa.ai/mcp?exaApiKey=6772266d-fde8-4154-b32a-46e893628a8c",
		mcp.WithStreamableHTTPClientInfo("exa-search-client", "1.0.0"),
	)
}

func NewTavilySearchMCPClient(ctx context.Context) (*mcp.Client, error) {
	return mcp.NewStreamableHTTP(ctx, "https://mcp.tavily.com/mcp/?tavilyApiKey=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",
		mcp.WithStreamableHTTPClientInfo("tavily-search-client", "0.2.16"),
	)

}

func NewForgejoMCPClient(ctx context.Context) (*mcp.Client, error) {
	return mcp.NewStdio(ctx, "forgejo-mcp", []string{
		"--transport",
		"stdio",
		"--url",
		"https://git.cluster.mirtuell.net",
	},
		mcp.WithStdioClientInfo("forgejo-client", "1.0.0"),
		mcp.WithEnvVars([]string{"FORGEJO_ACCESS_TOKEN=b415ffa48423f9c6ad82197c9e487723af505525"}),
	)
}
