// Package config provides MCP configuration types.
package config

// MCPServerConfig defines a single MCP server configuration from mcp.json
type MCPServerConfig struct {
	// Command is the executable to run (for stdio type)
	Command string `json:"command"`

	// Args are command-line arguments (for stdio type)
	Args []string `json:"args"`

	// Env contains environment variables (key=value or references)
	Env map[string]string `json:"env"`

	// Type is the transport type: "stdio" or "http"
	Type string `json:"type"`

	// URL is the endpoint URL (for http type)
	URL string `json:"url"`

	// Headers contains HTTP headers (for http type)
	Headers map[string]string `json:"headers"`

	// Enabled determines if this server should be loaded
	Enabled bool `json:"enabled"`
}
