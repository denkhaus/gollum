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
func loadEnvVar(key string) string {
	// Load .env file (sets environment variables from .env into process)
	godotenv.Load()
	// Read value from environment (was set by godotenv.Load or is in actual environment)
	return os.Getenv(key)
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
	// Load API key from environment (optional for local development)
	godotenv.Load()
	exaKey := os.Getenv("EXA_API_KEY")

	// Build Exa search URL
	var exaURL string
	if exaKey != "" {
		exaURL = "https://mcp.exa.ai/mcp?exaApiKey=" + exaKey
	} else {
		// No API key configured - use empty URL for local testing
		exaURL = "https://mcp.exa.ai/mcp"
	}

	return mcp.NewStreamableHTTP(ctx, exaURL,
		mcp.WithStreamableHTTPClientInfo("exa-search-client", "1.0.0"),
	)
}

// NewTavilySearchMCPClient creates a new MCP client for Tavily search
func NewTavilySearchMCPClient(ctx context.Context) (*mcp.Client, error) {
	// Load API key from environment (optional for local development)
	godotenv.Load()
	tavilyKey := os.Getenv("TAVILY_API_KEY")

	// Build Tavily search URL
	var tavilyURL string
	if tavilyKey != "" {
		tavilyURL = "https://mcp.tavily.com/mcp/?tavilyApiKey=" + tavilyKey
	} else {
		// No API key configured - use empty URL for local testing
		tavilyURL = "https://mcp.tavily.com/mcp"
	}

	return mcp.NewStreamableHTTP(ctx, tavilyURL,
		mcp.WithStreamableHTTPClientInfo("tavily-search-client", "0.2.16"),
	)
}
