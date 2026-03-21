# MCP Parallel Loading - Integration Test

## Manual Verification

To verify the parallel initialization is working:

1. Configure multiple MCP servers in your `~/.config/gollum/mcp.json`:
```json
{
  "mcpServers": {
    "server1": {
      "command": "sleep",
      "args": ["0.2"],
      "enabled": true
    },
    "server2": {
      "command": "sleep",
      "args": ["0.2"],
      "enabled": true
    },
    "server3": {
      "command": "sleep",
      "args": ["0.2"],
      "enabled": true
    }
  }
}
```

2. Add logging to measure startup time:
```go
// In main.go or wherever registry is created
start := time.Now()
registry, err := do.Invoke[*registry.MCPRegistry](injector)
log.Printf("MCP registry initialized in %v", time.Since(start))
```

3. Run the application and observe:
   - With 3 servers × 200ms each:
     - Sequential: ~600ms
     - Parallel: ~200ms (semaphore allows all 3)

## Expected Log Output

```
INFO  MCP client initialization starting (parallel)  total_servers=3 enabled=3 max_parallel=5
DEBUG Attempting to create MCP client (parallel)  mcp_server=server1
DEBUG Attempting to create MCP client (parallel)  mcp_server=server2
DEBUG Attempting to create MCP client (parallel)  mcp_server=server3
WARN  failed to create MCP client  mcp_server=server1 error=...
WARN  failed to create MCP client  mcp_server=server2 error=...
WARN  failed to create MCP client  mcp_server=server3 error=...
INFO  MCP client initialization completed  enabled_attempted=3 success_count=0 failed_count=3
```

Note the interleaved DEBUG logs indicating parallel execution.
