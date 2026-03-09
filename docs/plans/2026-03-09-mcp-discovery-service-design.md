# MCP Discovery Service Design

**Date:** 2026-03-09
**Status:** Design Approved
**Related:** 2025-03-08-flow-executor-consolidation-design.md

## Problem Statement

The flow executor currently returns "mcp step execution not yet implemented" for MCP steps. To implement MCP step execution, we need a centralized service that:

1. Discovers and registers MCP servers from configuration files
2. Creates MCP clients dynamically (replacing hardcoded functions in `pkg/mcp/mcp.go`)
3. Provides ToolSets for agent creation via `gollem.WithToolSets()`
4. Integrates with the existing DI container

## Current State

**Hardcoded MCP clients** (`pkg/mcp/mcp.go`):
- `NewExaSearchMCPClient()` - Hardcoded Exa search client
- `NewTavilySearchMCPClient()` - Hardcoded Tavily search client
- `NewForgejoMCPClient()` - Hardcoded Forgejo client
- Each manually loads env vars and creates specific client type

**MCP config** (`.gollum/mcp.json`):
- Contains server definitions with command, args, env, type, url, headers
- Has `enabled` field for filtering
- Not currently used by the codebase

**Executor** (`pkg/flows/executor/executor.go:415-417`):
- `executeMCPStep()` returns "not yet implemented" error

## Solution

Create an MCP discovery service following DI patterns that loads MCP servers from config files and provides ToolSets for agent creation.

## Architecture

```
pkg/mcp/
├── config/
│   ├── loader.go           # ConfigLoader service
│   ├── loader_test.go      # Config loading tests
│   └── types.go            # MCPServerConfig struct
├── registry/
│   ├── registry.go         # MCPRegistry service
│   └── registry_test.go    # Registry tests
└── mcp.go                  # Package exports (interfaces)
```

### Component Overview

**ConfigLoader** (`pkg/mcp/config/`):
- Loads and merges mcp.json from project and global locations
- Filters to enabled servers only
- Provides server configurations to registry

**MCPRegistry** (`pkg/mcp/registry/`):
- Creates MCP clients from config (eager initialization)
- Implements gollem.ToolSet interface for each client
- Provides ToolSets for agent creation
- Manages client lifecycle (close on shutdown)

### Public Interfaces

```go
// ConfigLoader defines the config loading service
type ConfigLoader interface {
    Load() (map[string]MCPServerConfig, error)
}

// MCPServerConfig defines a single MCP server configuration
type MCPServerConfig struct {
    Command string            `json:"command"`
    Args    []string          `json:"args"`
    Env     map[string]string `json:"env"`
    Type    string            `json:"type"` // "stdio" or "http"
    URL     string            `json:"url"`  // for http type
    Headers map[string]string `json:"headers"`
    Enabled bool              `json:"enabled"`
}

// MCPRegistry defines the MCP client registry service
type MCPRegistry interface {
    // GetToolSets returns all ToolSets for gollem.WithToolSets()
    GetToolSets() []gollem.ToolSet
    // Close shuts down all active MCP clients
    Close() error
}
```

### Private Implementation

```go
// configLoaderImpl is the private ConfigLoader implementation
type configLoaderImpl struct {
    projectPath string  // .gollum/mcp.json
    globalPath  string  // ~/.config/gollum/mcp.json
}

// NewConfigLoader is the DI constructor
func NewConfigLoader(injector do.Injector) (ConfigLoader, error)

// mcpRegistryImpl is the private MCPRegistry implementation
type mcpRegistryImpl struct {
    clients map[string]*mcp.Client  // serverName -> client
    tools   []gollem.ToolSet        // All tool sets
    loader  ConfigLoader
    logger  logger.Logger
}

// NewMCPRegistry is the DI constructor
func NewMCPRegistry(injector do.Injector) (MCPRegistry, error)
```

## Config Loading Behavior

### File Locations

- **Project-local**: `.gollum/mcp.json` (relative to current working directory)
- **Global**: `~/.config/gollum/mcp.json` (expanded from home directory)

### Merge Behavior: Project Overrides Global

1. Load global config first (if exists)
2. Load project config (if exists)
3. Merge with project values overriding global on conflict
4. Filter to only `enabled: true` servers

### Error Handling

- Missing files → not an error (empty config)
- Invalid JSON → error (fail fast)
- Failed client creation → log warning, continue (don't block startup)

## MCP Client Creation

### Supported Types

The gollem library (`github.com/m-mizutani/gollem/mcp`) supports three client types:

1. **Stdio**: `NewStdio(ctx, path, args, WithEnvVars, WithStdioClientInfo)`
2. **StreamableHTTP**: `NewStreamableHTTP(ctx, url, WithStreamableHTTPHeaders, WithStreamableHTTPClientInfo)`
3. **SSE**: `NewSSE(ctx, url, WithSSEHeaders, WithSSEClientInfo)`

### Client Initialization

```go
func createClient(ctx context.Context, name string, cfg MCPServerConfig) (*mcp.Client, error) {
    switch cfg.Type {
    case "stdio", "":
        envVars := buildEnvVars(cfg.Env)
        return mcp.NewStdio(ctx, cfg.Command, cfg.Args,
            mcp.WithEnvVars(envVars),
            mcp.WithStdioClientInfo("gollum", "1.0.0"),
        )
    case "http":
        return mcp.NewStreamableHTTP(ctx, cfg.URL,
            mcp.WithStreamableHTTPHeaders(cfg.Headers),
            mcp.WithStreamableHTTPClientInfo("gollum", "1.0.0"),
        )
    default:
        return nil, fmt.Errorf("unsupported MCP type: %s", cfg.Type)
    }
}
```

## Executor Integration

### FlowExecutor Changes

```go
type flowExecutorImpl struct {
    // ... existing fields ...
    mcpRegistry registry.MCPRegistry  // ← New field
}

func NewFlowExecutor(injector do.Injector) (FlowExecutorService, error) {
    // ... existing dependencies ...
    mcpRegistry := do.MustInvoke[registry.MCPRegistry](injector)

    return &flowExecutorServiceImpl{
        // ... existing fields ...
        mcpRegistry: mcpRegistry,
    }, nil
}
```

### MCP Step Execution

```go
func (p *flowExecutorImpl) executeMCPStep(step *flows.Step, stateName string) error {
    // step.Tool format: "server.tool" (e.g., "tavily.search")
    serverName, toolName, err := parseToolName(step.Tool)
    if err != nil {
        return &MCPError{Server: step.Tool, Step: stateName, Err: err}
    }

    // Build args from step params
    args := make(map[string]any)
    for _, param := range step.Params {
        value := p.substituteTemplate(param.Value)
        args[param.Name] = value
    }

    // Find and execute tool
    toolSets := p.mcpRegistry.GetToolSets()
    for _, toolSet := range toolSets {
        specs, _ := toolSet.Specs(context.Background())
        for _, spec := range specs {
            if spec.Name == step.Tool {
                result, err := toolSet.Run(context.Background(), step.Tool, args)
                if err != nil {
                    return &MCPError{Server: serverName, Tool: toolName, Step: stateName, Err: err}
                }

                // Map result to output
                if step.Output != nil && step.Output.Assign != "" {
                    fieldName := extractFieldName(step.Output.Assign)
                    p.ctx.SetOutputField(fieldName, result)
                }
                return nil
            }
        }
    }

    return &MCPError{Server: serverName, Tool: toolName, Step: stateName, Err: fmt.Errorf("tool not found")}
}
```

### MCP Error Type

```go
// MCPError represents an MCP step execution error
type MCPError struct {
    Server string
    Tool   string
    Step   string
    Err    error
}

func (e *MCPError) Error() string {
    return fmt.Sprintf("MCP step '%s' failed: server=%s tool=%s: %v", e.Step, e.Server, e.Tool, e.Err)
}
```

## DI Registration

### Container Updates

```go
// pkg/di/container.go

func (p *containerImpl) RegisterServices(_ context.Context) do.Injector {
    // ... existing services ...

    // MCP
    do.Provide(p.injector, config.NewConfigLoader)
    do.Provide(p.injector, registry.NewMCPRegistry)

    // Flows
    do.Provide(p.injector, flowregistry.NewFlowRegistryService)
    do.Provide(p.injector, executor.NewFlowExecutor)

    // ... existing services ...
}
```

## Implementation Plan (TDD Approach)

### Phase 1: Config Package
1. Write `config_test.go`:
   - Test loading valid JSON (project and global)
   - Test merge behavior (project overrides global)
   - Test missing files (not errors)
   - Test invalid JSON (error)
   - Test `enabled: false` filtering
2. Implement `config/loader.go` to pass tests
3. Implement `config/types.go` for MCPServerConfig struct

### Phase 2: Registry Service
1. Write `registry_test.go`:
   - Test registry initialization with mocked config
   - Test successful client creation (stdio type)
   - Test successful client creation (HTTP type)
   - Test failed client creation (log warning, continue)
   - Test `GetToolSets()` returns all ToolSets
   - Test `Close()` shuts down all clients
2. Implement `registry/registry.go` to pass tests
3. Add mock for `config.ConfigLoader` in `pkg/mocks/generate.go`

### Phase 3: Executor Integration
1. Write `mcp_step_test.go`:
   - Test successful MCP tool execution
   - Test tool not found error
   - Test parameter substitution
   - Test output mapping
2. Implement `executeMCPStep()` in `executor.go`
3. Update `NewFlowExecutor` to inject MCPRegistry

### Phase 4: DI Registration
1. Add to `pkg/di/container.go`
2. Write integration test: `pkg/flows/executor/mcp_integration_test.go`
3. Remove deprecated `pkg/mcp/mcp.go`

### Phase 5: Verification
1. Create test flow with MCP step
2. Run with actual MCP server (e.g., fetch)
3. Verify tool execution and output mapping

## Files Created

```
pkg/mcp/config/loader.go
pkg/mcp/config/loader_test.go
pkg/mcp/config/types.go
pkg/mcp/registry/registry.go
pkg/mcp/registry/registry_test.go
pkg/mcp/mcp.go
```

## Files Modified

```
pkg/flows/executor/executor.go
pkg/flows/executor/errors.go
pkg/flows/executor/mcp_step_test.go
pkg/di/container.go
pkg/mocks/generate.go
```

## Files Deleted

```
pkg/mcp/mcp.go (old hardcoded clients)
pkg/mcp/mcp_test.go (if exists)
```

## Design Decisions

### Eager vs Lazy Initialization
**Decision:** Eager initialization at registry creation time
**Rationale:** `gollem.WithToolSets()` requires all ToolSets upfront at agent creation, not at tool call time

### Config Merge Behavior
**Decision:** Project config overrides global config
**Rationale:** Allows per-project customization while maintaining user defaults

### Error Handling
**Decision:** Failed servers log warnings but don't block startup
**Rationale:** Fail-fast on actual tool calls, not during initialization. More flexible for development.

### Schema Scope
**Decision:** Only include fields we can actually implement
**Rationale:** The gollem library doesn't support timeout/concurrency fields. Those can be added later when needed with wrapper logic.

## Success Criteria

1. ✅ MCP servers loaded from `.gollum/mcp.json` and `~/.config/gollum/mcp.json`
2. ✅ Project config overrides global config
3. ✅ Enabled servers create MCP clients at startup
4. ✅ ToolSets available for agent creation
5. ✅ MCP steps execute successfully in flows
6. ✅ Proper error handling with MCPError type
7. ✅ All tests pass (TDD approach)
8. ✅ Old hardcoded `pkg/mcp/mcp.go` removed
9. ✅ DI patterns followed (public interface, private impl, `p` receiver)
