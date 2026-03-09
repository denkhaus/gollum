// Package config provides MCP server configuration loading and interpolation.
//
// This package handles:
//   - Loading mcp.json files from project and global locations
//   - Merging project and global configurations (project overrides global)
//   - Filtering enabled/disabled servers
//   - Shell command interpolation with timeout protection
//   - Environment variable expansion
//
// # Configuration Loading
//
// The ConfigLoader interface provides a simple Load() method that returns
// a map of server configurations keyed by server name:
//
//	loader := config.NewConfigLoader(injector)
//	servers, err := loader.Load()
//	// servers["server-name"] -> MCPServerConfig
//
// # Configuration Sources
//
// Config files are loaded from two locations (in order of precedence):
//
//   1. Project: .gollum/mcp.json (higher priority)
//   2. Global: ~/.config/gollum/mcp.json (lower priority)
//
// When a server name exists in both files, the project configuration
// completely replaces the global configuration for that server.
//
// # Interpolation
//
// Configuration values support dynamic interpolation:
//
// ## Shell Commands
//
//	"env": {
//	    "RESULT": "$(echo hello)"  // Executes: echo hello
//	}
//
// Commands run via sh -c with a configurable timeout.
// On timeout or error, the original string is preserved.
//
// ## Environment Variables
//
//	"env": {
//	    "API_KEY": "$MY_API_KEY",      // Simple form
//	        "TOKEN": "${MY_TOKEN}"      // Braced form
//	}
//
// Undefined variables expand to empty strings.
//
// ## Combined
//
//	"env": {
//	    "FULL_PATH": "$HOME/$(basename $PWD)"
//	}
//
// # Security
//
// Shell command interpolation executes commands with the same privileges
// as the Gollum process. Only load mcp.json files from trusted sources.
//
// # Configuration Schema
//
// See MCPServerConfig in types.go for the complete schema definition.
package config
