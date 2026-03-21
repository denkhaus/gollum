// Package registry provides MCP client registry and lifecycle management.
//
// This package handles:
//   - MCP client initialization from configuration
//   - Client lifecycle (creation, storage, cleanup)
//   - Tool set exposure to the agent executor
//   - Graceful error handling for unavailable servers
//
// # Registry Interface
//
// The MCPRegistry interface provides:
//
//	type MCPRegistry interface {
//	    GetToolSets() []gollem.ToolSet
//	    Close() error
//	}
//
// # Client Initialization
//
// The registry is initialized via dependency injection:
//
//	registry := do.MustInvoke[mcpregistry.MCPRegistry](injector)
//	toolSets := registry.GetToolSets()
//
// All configured and enabled MCP servers are automatically initialized
// when the registry is created. Failed servers are logged but don't
// prevent the application from starting.
//
// # Parallel Initialization
//
// The registry initializes MCP clients in parallel using a semaphore
// to limit concurrent connections. The default semaphore buffer size
// is 5, meaning up to 5 MCP clients will be created simultaneously.
// Individual client failures are logged as warnings but do not prevent
// other clients from being initialized.
//
// # Client Types
//
// The registry supports three MCP client types:
//
//   - stdio: Local executables communicating via stdin/stdout
//   - sse: Remote servers via Server-Sent Events
//   - streamable-http: Streaming HTTP connections
//
// See the package documentation for configuration examples.
//
// # Error Handling
//
// Client initialization failures are handled gracefully:
//
//   - Failed servers log a warning and are skipped
//   - Other servers continue to initialize
//   - The application starts successfully even if all MCP servers fail
//   - Tool sets only include successfully initialized clients
//
// # Lifecycle Management
//
// The registry maintains ownership of all MCP clients:
//
//   - Clients are created during registry initialization
//   - Clients remain active for the application lifetime
//   - Close() shuts down all clients gracefully
//
// # Integration with Agent Executor
//
// MCP tool sets are automatically integrated with the agent system:
//
//	toolSets := registry.GetToolSets()
//	// Pass to agent configuration
//	config.ToolSets = append(config.ToolSets, toolSets...)
//
// Each MCP client implements gollem.ToolSet, exposing tools from
// the connected MCP server to agents.
//
// # Thread Safety
//
// The registry is thread-safe after initialization:
//
//   - GetToolSets() returns a snapshot and can be called concurrently
//   - Close() is idempotent and safe to call multiple times
//   - Individual clients manage their own concurrency
package registry
