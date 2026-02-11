// Package mcp provides MCP (Model Context Protocol) client implementations.
package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/m-mizutani/gollem/mcp"
)

// loadEnvVar loads an environment variable from .env file or returns default value
func loadEnvVar(key string) (string, error) {
	godotenv.Load()
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("env key %s not found", key)
	}

	return value, nil
}

// NewBrainMCPClient creates a new MCP client for brain service
func NewBrainMCPClient(ctx context.Context) (*mcp.Client, error) {
	// Local brain MCP client for supervisor agents
	return mcp.NewStreamableHTTP(ctx, "http://localhost:8555/mcp",
		mcp.WithStreamableHTTPClientInfo("brain-mcp-client", "1.0.0"),
	)
}

// NewExaSearchMCPClient creates a new MCP client for Exa search
func NewExaSearchMCPClient(ctx context.Context) (*mcp.Client, error) {
	exaKey, err := loadEnvVar("EXA_API_KEY")
	if err != nil {
		return nil, err
	}

	exaURL := "https://mcp.exa.ai/mcp?exaApiKey=" + exaKey

	return mcp.NewStreamableHTTP(ctx, exaURL,
		mcp.WithStreamableHTTPClientInfo("exa-search-client", "1.0.0"),
	)
}

// NewTavilySearchMCPClient creates a new MCP client for Tavily search
func NewTavilySearchMCPClient(ctx context.Context) (*mcp.Client, error) {
	tavilyKey, err := loadEnvVar("TAVILY_API_KEY")
	if err != nil {
		return nil, err
	}

	tavilyURL := "https://mcp.tavily.com/mcp/?tavilyApiKey=" + tavilyKey

	return mcp.NewStreamableHTTP(ctx, tavilyURL,
		mcp.WithStreamableHTTPClientInfo("tavily-search-client", "0.2.16"),
	)
}

func NewForgejoMCPClient(ctx context.Context) (*mcp.Client, error) {
	forgejoAccessToken, err := loadEnvVar("FORGEJO_ACCESS_TOKEN")
	if err != nil {
		return nil, err
	}

	return mcp.NewStdio(ctx, "forgejo-mcp", []string{
		"--transport",
		"stdio",
		"--url",
		"https://git.cluster.mirtuell.net",
	},
		mcp.WithStdioClientInfo("forgejo-client", "1.0.0"),
		mcp.WithEnvVars([]string{fmt.Sprintf("FORGEJO_ACCESS_TOKEN=%s", forgejoAccessToken)}),
	)
}
