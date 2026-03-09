// Package mcp provides the Model Context Protocol (MCP) discovery service
// for the Gollum application.
//
// The MCP discovery service enables automatic loading and management of MCP
// servers from configuration files, allowing agents to access external tools
// and resources through the standardized MCP protocol.
//
// # Overview
//
// MCP (Model Context Protocol) is an open protocol that enables AI applications
// to connect to external data sources and tools. This package provides:
//
//   - Config-based MCP server discovery via mcp.json files
//   - Support for multiple transport types (stdio, SSE, streamable HTTP)
//   - Automatic client initialization and lifecycle management
//   - Dynamic tool set registration with the agent executor
//
// # Configuration
//
// MCP servers are configured in JSON files:
//   - Project-specific: .gollum/mcp.json
//   - Global: ~/.config/gollum/mcp.json
//
// Project configuration overrides global settings for matching server names.
//
// # Configuration Format
//
// The mcp.json file follows this structure:
//
//	{
//	    "mcpServers": {
//	        "server-name": {
//	            "type": "stdio",  // or "sse" or "streamable-http"
//	            "command": "path-to-executable",  // for stdio
//	            "args": ["--arg1", "arg2"],      // for stdio
//	            "url": "http://localhost:8080/mcp", // for sse/streamable-http
//	            "env": {
//	                "API_KEY": "$API_KEY",           // env var expansion
//	                "TOKEN": "static-value",
//	                "DYNAMIC": "$(echo generated)"   // shell command
//	            },
//	            "headers": {                         // for HTTP clients
//	                "Authorization": "$AUTH_TOKEN"
//	            },
//	            "enabled": true
//	        }
//	    }
//	}
//
// # Interpolation
//
// Configuration values support shell command and environment variable interpolation:
//
//   - Shell commands: $(command) - Executes command and uses output
//   - Environment variables: $VAR or ${VAR}
//   - Combined: $PREFIX-$(echo hello)
//
// Shell commands have a configurable timeout (default: 5s, min: 1s, max: 60s)
// controlled by GOLLUM_MCP_COMMAND_TIMEOUT_SECONDS environment variable.
//
// # Transport Types
//
// ## stdio (Standard Input/Output)
//
// Communicates with local MCP executables via stdin/stdout. Suitable for:
//   - Command-line tools
//   - Local server processes
//   - Wrappers around HTTP APIs
//
// Example:
//
//	{
//	    "type": "stdio",
//	    "command": "npx",
//	    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path"]
//	}
//
// ## sse (Server-Sent Events)
//
// Connects to remote MCP servers via HTTP SSE. Suitable for:
//   - Cloud-hosted MCP servers
//   - Long-lived connections
//   - Server-to-client streaming
//
// Example:
//
//	{
//	    "type": "sse",
//	    "url": "https://api.example.com/mcp",
//	    "headers": {
//	        "Authorization": "Bearer $TOKEN"
//	    }
//	}
//
// ## streamable-http
//
// Similar to SSE but optimized for streaming responses. Suitable for:
//   - High-throughput scenarios
//   - Real-time data streaming
//   - Custom transport implementations
//
// # Usage
//
// The MCP registry is automatically initialized via dependency injection.
// All configured and enabled MCP servers are automatically available to agents
// through the tool set registry.
//
// No manual client creation is required - simply add server configurations
// to mcp.json files and they will be discovered and initialized automatically.
//
// # Architecture
//
// The MCP discovery service consists of two main packages:
//
//   - config: Configuration loading and interpolation
//   - registry: Client initialization and management
//
// See individual package documentation for details.
package mcp
