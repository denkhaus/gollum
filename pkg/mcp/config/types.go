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

	// Enabled determines if this server should be loaded.
	// Defaults to true when not specified in JSON.
	Enabled bool `json:"enabled"`

	// AllowedSystemEnv is a list of system environment variable names
	// that are allowed to be passed to the MCP server.
	// This provides security by only whitelisting safe variables.
	// If empty or null, a default safe list is used (PATH, HOME, USER, etc.)
	AllowedSystemEnv []string `json:"allowedSystemEnv,omitempty"`
}

// MCPConfigFile represents the root structure of mcp.json
type MCPConfigFile struct {
	// MCPServers is the map of server configurations
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`

	// DefaultAllowedSystemEnv is the default whitelist for system env vars.
	// Applied to servers that don't specify their own allowedSystemEnv.
	// If empty or null, a built-in safe default is used.
	DefaultAllowedSystemEnv []string `json:"defaultAllowedSystemEnv"`
}
