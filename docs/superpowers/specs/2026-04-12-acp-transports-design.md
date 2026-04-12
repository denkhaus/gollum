# ACP Transport Support Design

**Date:** 2026-04-12
**Status:** Approved
**Author:** Claude (with user approval)

## Overview

Extend the Gollum ACP command to support multiple transport types (stdio and HTTP) while maintaining backward compatibility. The HTTP transport enables multi-client server deployment scenarios.

## Problem Statement

Currently, the ACP command only supports stdio transport (stdin/stdout), which limits deployment to local subprocess scenarios. Adding HTTP transport support enables:
- Multi-client agent servers (one instance serves multiple clients)
- Web-based IDE integration
- Remote agent access

## Architecture

### Transport Type System

```go
type TransportType int

const (
    TransportStdio TransportType = iota  // Default
    TransportHTTP
)
```

### Connection Creation Flow

1. CLI parses transport flags → creates transport options
2. `facade.CreateChannel()` receives transport options
3. `NewConnection()` reads transport option
4. Creates appropriate go-acp connection:
   - **Stdio**: `NewAgentSideConnection(service, stdin, stdout)`
   - **HTTP**: `NewAgentSideConnection(service, nil, nil, WithTransport(httpTransport))`

### Multi-Client Support

The go-acp `HTTPServerTransport` inherently supports multiple concurrent connections. Each HTTP request creates its own transport instance, and our existing session store handles session separation.

## Components

### Modified Files

**pkg/acp/options.go**
- Add `TransportType` enum
- Add `WithTransport(t TransportType)` option
- Add `WithHost(host string)` option
- Add `WithPort(port int)` option

**pkg/acp/connection.go**
- Update `NewConnection()` to accept transport options
- Create `HTTPServerTransport` when HTTP selected
- Return `http.Handler` for HTTP mode

**pkg/acp/channel.go**
- Update `Start()` method to handle HTTP mode
- Return handler for HTTP server startup

**pkg/cli/acp.go**
- Add `--transport {stdio|http}` flag (default: stdio)
- Add `--host <address>` flag (default: 0.0.0.0)
- Add `--port <number>` flag (default: 8080)
- For HTTP mode: start HTTP server with connection's handler
- For stdio mode: use existing direct start flow

### New Files

**pkg/acp/transport.go**
- Transport type enum definition
- Transport validation logic

**pkg/acp/transport_test.go**
- Unit tests for transport options
- HTTP transport creation tests

**pkg/acp/http_transport_test.go**
- Integration tests for HTTP transport
- Multi-client connection tests

## Data Flow

### HTTP Mode
```
CLI: gollum acp --transport http --port 8080
  → Parse flags, create WithTransport(TransportHTTP), WithPort(8080)
  → facade.CreateChannel(acp.Identifier, transportOpts...)
  → NewConnection() creates HTTPServerTransport
  → Returns Connection with http.Handler
  → http.ListenAndServe(":8080", handler)
  → Client connects → HTTP transport handles connection
  → ACP messages flow via HTTP + SSE
```

### Stdio Mode (unchanged)
```
CLI: gollum acp
  → Default TransportStdio
  → NewConnection(stdin, stdout)
  → Direct connection start
```

## Configuration

### CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--transport` | string | `stdio` | Transport type: `stdio` or `http` |
| `--host` | string | `0.0.0.0` | HTTP server bind address |
| `--port` | int | `8080` | HTTP server port |

### Validation

- Transport type: must be `stdio` or `http`
- Port range: 1-65535
- Host format: valid IP address or hostname

## Error Handling

### Transport Errors

**Invalid transport type:**
```
Error: unsupported transport type: <value>
Supported transports: stdio, http
```

**HTTP startup errors:**
- Port in use: `failed to start HTTP server: port 8080 already in use`
- Invalid host: `invalid host address: <value>`
- Permission denied: `failed to bind to port <port>: permission denied (try port > 1024)`

### Shutdown

Graceful shutdown on SIGTERM/SIGINT (already handled by CLI):
- HTTP server: `Shutdown(ctx)` with timeout
- Close existing connections
- Release resources

## Testing Strategy

### Unit Tests

**pkg/acp/transport_test.go**
- Test `WithTransport()` option application
- Test `WithHost()` and `WithPort()` options
- Test option combinations
- Test transport validation

**pkg/acp/connection_test.go** (extend)
- Test HTTP transport creation
- Test stdio transport remains default
- Test handler returned for HTTP mode

### Integration Tests

**pkg/acp/http_transport_test.go**
- Start HTTP server with test connection
- Send/receive ACP messages over HTTP
- Test multiple concurrent connections
- Test connection lifecycle

### Manual Testing

```bash
# Test stdio (existing)
gollum acp

# Test HTTP with defaults
gollum acp --transport http

# Test HTTP with custom host/port
gollum acp --transport http --host localhost --port 9000

# Test invalid transport (should error)
gollum acp --transport invalid
```

## Implementation Phases

### Phase 1: Foundation
- Create `pkg/acp/transport.go`
- Modify `pkg/acp/options.go` - add transport options
- Modify `pkg/acp/connection.go` - support transport selection
- Modify `pkg/acp/channel.go` - handle HTTP mode

### Phase 2: CLI Integration
- Modify `pkg/cli/acp.go` - add flags
- Add HTTP server startup logic
- Add signal-safe shutdown for HTTP server

### Phase 3: Testing
- Add unit tests
- Add integration tests
- Test multi-client concurrent connections

## Backward Compatibility

- **No breaking changes**: stdio remains default transport
- Existing CLI usage continues to work: `gollum acp`
- HTTP transport is opt-in via `--transport http` flag

## Security Considerations

- HTTP transport binds to `0.0.0.0` by default (all interfaces) - may need warning
- No authentication built into transport layer (ACP authentication handles this)
- Consider future HTTPS/TLS support for production deployments

## Future Enhancements

- HTTPS/TLS support for secure transport
- WebSocket transport as alternative to SSE
- CORS configuration for browser-based clients
- Connection limits and rate limiting
- Health check endpoints for HTTP server

## Success Criteria

- [x] Design approved by user
- [ ] All transport types functional (stdio, http)
- [ ] Multi-client connections work correctly
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Manual testing successful
