# MCP Parallel Loading Design

**Date:** 2026-03-20
**Author:** Claude (via brainstorming session)
**Status:** Draft

## Problem Statement

The current MCP client initialization is sequential. When multiple MCP servers are configured, the startup time is the sum of all individual connection times. For example, with 5 servers taking 200ms each, startup takes 1000ms.

## Goal

Reduce MCP client initialization time through parallel loading with semaphore-based concurrency limiting.

## Architecture

### Current Behavior

```
Server 1: 200ms ──────────────────┐
Server 2:        200ms ────────────┤
Server 3:              200ms ──────┤ Startup: 1000ms
Server 4:                    200ms ┤
Server 5:                          200ms ─
```

### Proposed Behavior (Semaphore with 5 workers)

```
Server 1: 200ms ─┐
Server 2: 200ms ─┤
Server 3: 200ms ─┼─> Startup: ~200ms (slowest server)
Server 4: 200ms ─┤
Server 5: 200ms ─┘
```

## Implementation Changes

### File: `pkg/mcp/registry/registry.go`

**Add to `mcpRegistryImpl` struct:**

```go
type mcpRegistryImpl struct {
    clients   map[string]*mcp.Client
    tools     []gollem.ToolSet
    loader    mcpconfig.ConfigLoader
    appConfig appconfig.ConfigService
    logger    logger.LoggerService

    // Semaphore for limiting concurrent connections
    sem       chan struct{}  // Buffer size = max parallel connections
}
```

**Update `NewMCPRegistry` constructor:**

```go
func NewMCPRegistry(injector do.Injector) (MCPRegistry, error) {
    ...
    p := &mcpRegistryImpl{
        clients:   make(map[string]*mcp.Client),
        tools:     make([]gollem.ToolSet, 0),
        loader:    loader,
        appConfig: appConfig,
        logger:    log,
        sem:       make(chan struct{}, 5),  // Max 5 parallel MCP connections
    }
    ...
}
```

**Replace `initializeClients` with parallel version:**

```go
func (p *mcpRegistryImpl) initializeClients(ctx context.Context) error {
    allConfigs, err := p.loader.LoadAll()
    if err != nil {
        return err
    }

    // Filter to enabled servers only
    configs := make(map[string]mcpconfig.MCPServerConfig)
    for name, cfg := range allConfigs {
        if cfg.Enabled {
            configs[name] = cfg
        }
    }

    // Count enabled servers for logging
    enabledCount := len(configs)
    mcpConfig := p.appConfig.GetMCPConfig()
    p.logger.Info("MCP client initialization starting (parallel)",
        zap.Int("total_servers", enabledCount),
        zap.Int("max_parallel", cap(p.sem)),
        zap.Duration("init_timeout", mcpConfig.GetClientInitTimeout()),
    )

    // Semaphore-bounded parallel initialization
    var wg sync.WaitGroup
    var mu sync.Mutex       // Protects p.clients, p.tools
    errors := make([]error, 0)

    for name, cfg := range configs {
        wg.Add(1)
        go func(name string, cfg mcpconfig.MCPServerConfig) {
            defer wg.Done()

            p.sem <- struct{}{}  // Acquire semaphore
            defer func() { <-p.sem }()  // Release semaphore

            p.logger.Debug("Attempting to create MCP client (parallel)",
                zap.String("mcp_server", name),
                zap.String("type", cfg.Type),
            )

            client, err := p.createClient(ctx, name, cfg)
            if err != nil {
                p.logger.Warn("failed to create MCP client",
                    zap.String("mcp_server", name),
                    zap.String("type", cfg.Type),
                    zap.Error(err),
                )
                mu.Lock()
                errors = append(errors, fmt.Errorf("%s: %w", name, err))
                mu.Unlock()
                return
            }

            mu.Lock()
            p.clients[name] = client
            p.tools = append(p.tools, client) // mcp.Client implements gollem.ToolSet
            mu.Unlock()

            p.logger.Debug("MCP client created successfully",
                zap.String("mcp_server", name))
        }(name, cfg)
    }

    wg.Wait()

    // Log summary
    successCount := len(p.clients)
    failedCount := len(errors)

    p.logger.Info("MCP client initialization completed",
        zap.Int("enabled_attempted", enabledCount),
        zap.Int("success_count", successCount),
        zap.Int("failed_count", failedCount),
    )

    // Log all errors (but don't fail initialization)
    for _, err := range errors {
        p.logger.Warn("MCP client initialization error", zap.Error(err))
    }

    return nil
}
```

**Add imports:**

```go
import (
    "sync"  // Add this
    ...
)
```

### Configuration

The semaphore buffer size (5) can be made configurable via config in a future iteration if needed. For now, 5 is a reasonable default:
- Allows 5 stdio processes to run in parallel
- Prevents overwhelming the system with too many concurrent processes
- Balances speed vs resource usage

## Error Handling

- **Individual server failures**: Logged as warnings, don't abort entire initialization
- **All servers fail**: Logger warns, but app continues (no MCP tools available)
- **Context timeout**: Respects the existing `initTimeout` from config

## Testing

### Unit Tests

Update existing tests to handle parallel execution:
- Test concurrent client creation
- Test semaphore limiting
- Test error isolation (one failure doesn't stop others)

### Integration Tests

- Test with multiple MCP servers
- Measure startup time improvement
- Verify all tools are available after initialization

## Performance Impact

**Expected improvement:**
- Sequential: N × avg_connection_time
- Parallel (with 5 workers): max_connection_time (with slight overhead)

**Example:**
- 5 servers × 200ms each = 1000ms → ~200ms (5× faster)
- 10 servers × 200ms each = 2000ms → ~400ms (5× faster, limited by semaphore)

## Migration Path

1. Implement parallel initialization
2. Run existing tests to verify no regressions
3. Add new tests for parallel behavior
4. Deploy and monitor startup times

## Out of Scope

- On-demand MCP client loading (determined to not provide benefit, as connection is needed anyway to get tool specs)
- Dynamic reconfiguration (would require restart anyway)
- Per-agent tool filtering (existing `AllowedTools` mechanism handles this)

## Future Enhancements

- Make semaphore buffer size configurable
- Add metrics for initialization time
- Consider connection pooling for frequently used servers
