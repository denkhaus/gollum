# ACP Integration Improvements

**Date**: 2026-04-06
**Status**: Analysis Complete
**Reference**: [ironpark/acp-go](https://github.com/ironpark/acp-go)

---

## Executive Summary

This document identifies gaps, misconceptions, and improvement opportunities in Gollum's ACP (Agent Client Protocol) implementation based on analysis of the [ironpark/acp-go](https://github.com/ironpark/acp-go) reference implementation.

**Critical Findings**:
- ✗ Log messages are not forwarded to ACP client
- ✗ Client-side methods (fs, terminal) not implemented
- ✗ Session lifecycle unclear
- ✗ No tool/permission support

---

## 1. Log Message Transport to Client

### Current State

**File**: `pkg/acp/service.go:149-157`

```go
func (s *acpServiceImpl) OnLog(entry channel.LogEntry) {
    s.logger.Debug("log entry from agent system",
        zap.String("level", entry.Level),
        zap.String("message", entry.Message),
    )
    // TODO: Optionally forward logs to ACP client if needed
}
```

**Problem**: Log entries from the agent system are logged locally but NOT streamed to the ACP client.

### Solution

According to the ACP standard, logs should be sent via `session/update` notifications using `AgentMessageChunk`:

```go
func (s *acpServiceImpl) OnLog(entry channel.LogEntry) {
    s.activeSessionMu.RLock()
    session := s.activeSession
    s.activeSessionMu.RUnlock()

    if session == nil {
        s.logger.Warn("received log but no active session")
        return
    }

    // Stream log to ACP client
    stream := acppkg.NewSessionStream(s.client, session.SessionID)
    logMsg := fmt.Sprintf("[%s] %s", entry.Level, entry.Message)
    if err := stream.SendText(session.Context, logMsg); err != nil {
        s.logger.Error("failed to stream log to ACP client", zap.Error(err))
    }
}
```

**Priority**: Medium
**Complexity**: Low (1-2 hours)

---

## 2. Unimplemented Methods

### Agent Interface Methods

| Method | Status | Priority | Notes |
|--------|--------|----------|-------|
| `Authenticate()` | Partial | Medium | Returns nil (no auth) - OK for now |
| `SetSessionMode()` | Stub only | Low | Only logs, no effect |
| `SetSessionConfigOption()` | Stub only | Low | Only logs, no effect |
| **`LoadSession()`** | **Missing** | **High** | Capability claims `LoadSession: false` |

### LoadSession Implementation

**Current**: `AgentCapabilities.LoadSession = false`

**Required Implementation**:

```go
func (s *acpServiceImpl) LoadSession(ctx context.Context, params *acppkg.LoadSessionRequest) (*acppkg.LoadSessionResponse, error) {
    // Restore session from store or return error if not supported
    return nil, fmt.Errorf("session loading not implemented")
}
```

---

## 3. Critical Gaps & Misconceptions

### Gap 1: Missing Client-Side Methods

**Problem**: ACP is **bidirectional**. The client can send requests to the agent:

| Client Method | Purpose | Status |
|---------------|---------|--------|
| `fs/read_text_file` | Read files from client | ❌ Not implemented |
| `fs/write_text_file` | Write files to client | ❌ Not implemented |
| `terminal/create` | Create terminal session | ❌ Not implemented |
| `terminal/output` | Send terminal output | ❌ Not implemented |
| `session/request_permission` | Permission workflow | ❌ Not implemented |

**Impact**: Gollum cannot interact with the client's filesystem or terminals, which are core ACP features.

**Solution**: Implement the `acppkg.Client` interface methods that the agent calls:

```go
// Example: Reading a file from the client
func (s *acpServiceImpl) ReadTextFile(ctx context.Context, params *acppkg.ReadTextFileRequest) (*acppkg.ReadTextFileResponse, error) {
    // This would be called by the agent to read files from the client
    return s.client.ReadTextFile(ctx, params)
}
```

**Priority**: High
**Complexity**: High (requires architecture changes)

---

### Gap 2: Session Lifecycle Confusion

**Comment in code**:
```go
// TODO: Determine when NewSession callback fires (Initialize/SetSessionMode/Prompt?)
```

**Clarification from ACP Standard**:

1. `Initialize()` - Protocol handshake, NO session created
2. First `Prompt()` call - Session is created via `NewSession` callback
3. `session/new` callback fires BEFORE the prompt is processed

**Current Implementation Issue**:
- We track only ONE `activeSession` per connection
- ACP supports MULTIPLE sessions per connection

**Architecture Issue**:
```go
activeSession *shared.ACPSession  // ONE session per connection
```

Should be:
```go
sessions map[acppkg.SessionID]*shared.ACPSession  // Multiple sessions
```

**Priority**: Medium
**Complexity**: Medium

---

### Gap 3: No Tool/Permission Support

**Problem**: ACP defines a structured workflow for tool calls with user permissions:

```
Agent → session/request_permission → Client displays to user
User approves → Agent proceeds with tool call
```

**Current**: Our `Prompt()` method only sends text, no tool definitions or permission requests.

**Required**: Support for tool definitions and permission workflow in prompts.

**Priority**: Medium
**Complexity**: High

---

### Gap 4: OnAgentLifecycle Not Implemented

**File**: `pkg/acp/service.go:159-167`

```go
func (s *acpServiceImpl) OnAgentLifecycle(event channel.AgentLifecycleEvent) {
    s.logger.Debug("agent lifecycle event", ...)
    // TODO: Optionally notify ACP client of agent changes
}
```

**Opportunity**: When agents are added/removed, notify the ACP client via `session/update`.

**Priority**: Low
**Complexity**: Low

---

## 4. Architecture Overview

### Current Implementation

```
┌─────────────────────────────────────────────────────────────┐
│                    Gollum ACP Service                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │           Channel Interface (Inward)                  │  │
│  │  • OnMessage()     ← Receive from agents             │  │
│  │  • OnLog()         ← Receive logs                    │  │
│  │  • OnAgentLifecycle() ← Receive agent events         │  │
│  └────────────────────┬─────────────────────────────────┘  │
│                       │                                      │
│  ┌────────────────────▼─────────────────────────────────┐  │
│  │         ACP Agent Interface (Outward)                 │  │
│  │  • Initialize()     → Handshake with client          │  │
│  │  • Prompt()         → Receive user input              │  │
│  │  • Cancel()         → Cancel in-flight prompt        │  │
│  └────────────────────┬─────────────────────────────────┘  │
└───────────────────────┼──────────────────────────────────────┘
                        │
                        ▼
              ┌─────────────────┐
              │  ACP Client      │
              │  (Editor/IDE)    │
              └─────────────────┘
```

### Missing Pieces

```
                    ┌─────────────────────────────────────┐
                    │     MISSING: Client Methods         │
                    │  • ReadTextFile()                   │
                    │  • WriteTextFile()                  │
                    │  • CreateTerminal()                 │
                    │  • RequestPermission()              │
                    └─────────────────────────────────────┘
```

---

## 5. Recommended Implementation Order

### Phase 1: Quick Wins (1-2 days) ✅ COMPLETE

1. ✅ **Log Forwarding** - Implemented via LogForwarder interface
   - Created `shared.LogForwarder` interface for decoupled log routing
   - Embedded in `channel.ChannelFacade` interface
   - Integrated with DI container for logger wiring
   - Added comprehensive tests for session-aware log routing
   - See: `pkg/shared/log_forwarder.go`, `pkg/acp/service.go:OnLog()`

2. ✅ **Document Session Lifecycle** - Added clarifying comments
   - Documented session creation on first Prompt() call
   - Clarified relationship between Initialize and NewSession callback
   - Added lifecycle documentation in service.go
   - See: `pkg/acp/service.go:109-123`

3. ✅ **OnAgentLifecycle Notification** - Implemented
   - Lifecycle events now streamed to ACP client via session/update
   - Supports agent_added, agent_removed, agent_updated events
   - Integrated with channel system event propagation
   - See: `pkg/acp/service.go:159-183`

### Phase 2: Core Functionality (3-5 days)

4. ✅ **Multi-Session Support** - Track multiple sessions
5. ✅ **LoadSession Implementation** - Add session resumption
6. ✅ **SetSessionMode/Config** - Implement basic handling

### Phase 3: Advanced Features (1-2 weeks)

7. ✅ **Client Methods** - Implement fs/terminal operations
8. ✅ **Tool/Permission Support** - Add tool call workflow
9. ✅ **Authentication** - Implement proper auth

---

## 6. Transport Architecture

### Overview

Gollum's ACP implementation supports two transport modes for communication with ACP clients:

- **Stdio Transport** - Default, single-client mode using stdin/stdout
- **HTTP Transport** - Multi-client mode using HTTP + Server-Sent Events (SSE)

### Transport Type Enum

**File**: `pkg/acp/transport.go`

```go
type TransportType string

const (
    TransportStdio TransportType = "stdio"
    TransportHTTP TransportType = "http"
)

func ParseTransportType(s string) (TransportType, error)
```

The `ParseTransportType` function validates transport type strings and returns an error for invalid values.

### Stdio Transport

**Configuration**: Via channel options or CLI flags

```go
// Channel options
acp.WithStdin(os.Stdin)
acp.WithStdout(os.Stdout)
acp.WithTransport(acp.TransportStdio)
```

**Usage**: Local development, single-client scenarios

**Behavior**:
- Reads ACP protocol messages from stdin
- Writes responses to stdout
- Blocks until client disconnects
- Single active session per connection

**CLI**: `gollum acp` (default) or `gollum acp --transport stdio`

### HTTP Transport

**Configuration**: Via channel options or CLI flags

```go
// Channel options
acp.WithTransport(acp.TransportHTTP)
acp.WithHost("0.0.0.0")
acp.WithPort(8080)
```

**Usage**: Multi-client servers, production deployments

**Behavior**:
- HTTP server with SSE for real-time streaming
- Supports multiple concurrent sessions
- Each session gets unique context and cancellation
- Graceful shutdown with configurable timeout

**CLI**: `gollum acp --transport http --host 0.0.0.0 --port 8080`

### Transport Selection

Transport type is selected via:

1. **Configuration Service** - Centralized config with defaults
2. **CLI Flags** - Command-line override with env var support
3. **Channel Options** - Programmatic configuration

**Priority**: CLI flags > Config > Defaults

### Implementation Details

#### Service Start Method

**File**: `pkg/acp/service.go`

```go
func (s *acpServiceImpl) Start(ctx context.Context) error {
    if s.transportType == TransportHTTP {
        return s.startHTTP(ctx)
    }
    return s.startStdio(ctx)
}
```

The `Start` method routes to the appropriate transport implementation based on `s.transportType`.

#### HTTP Server Startup

**File**: `pkg/cli/acp.go`

The CLI command starts an HTTP server in a goroutine with:
- Configurable timeouts (Read: 15s, Write: 15s, Idle: 60s)
- Graceful shutdown with configurable timeout (default: 10s)
- Ready signaling for dependency management
- Proper error handling and logging

#### Channel Factory

The channel facade creates ACP channels with transport-specific options:

```go
ch, err := facade.CreateChannel(acp.Identifier,
    acp.WithTransport(transportType),
    acp.WithHost(host),
    acp.WithPort(port),
    acp.WithStdin(os.Stdin),
    acp.WithStdout(os.Stdout),
)
```

### Session Management

**HTTP Mode**: Multi-session support via `acppkg.MemoryStore`
- Each HTTP connection creates a new session
- Sessions are tracked by SessionID
- Independent context cancellation per session

**Stdio Mode**: Single-session support
- One session per stdio connection
- Session created on first `Prompt()` call

### Environment Variables

All transport configuration supports environment variables:

- `GOLLUM_ACP_TRANSPORT` - Transport type (`stdio` or `http`)
- `GOLLUM_ACP_HOST` - HTTP server bind address
- `GOLLUM_ACP_PORT` - HTTP server port
- `GOLLUM_ACP_SHUTDOWN_TIMEOUT` - Graceful shutdown timeout

### Testing

Both transport modes have comprehensive test coverage:

- **Stdio**: `pkg/acp/transport_test.go` - `TestStdioModeStart`
- **HTTP**: `pkg/acp/transport_test.go` - `TestHTTPModeStart`
- **Factory**: `pkg/acp/options_test.go` - Transport option validation

---

## 7. Reference Resources

- **ACP Standard**: [ironpark/acp-go](https://github.com/ironpark/acp-go)
- **DeepWiki Documentation**: Available via MCP `deepwiki` tool
- **Current Implementation**: `pkg/acp/service.go`, `pkg/acp/connection.go`
- **Transport Implementation**: `pkg/acp/transport.go`, `pkg/cli/acp.go`

---

## 7. Open Questions

1. Should we support multiple concurrent sessions per connection?
2. Do we need terminal support for our use case?
3. Should file operations (fs/*) be implemented or marked as unsupported?

---

## Appendix: ACP Method Reference

### Agent Methods (Implemented)

| Method | Implemented | Notes |
|--------|-------------|-------|
| `Initialize` | ✅ | Returns capabilities |
| `Authenticate` | ⚠️ | Stub only |
| `SetSessionMode` | ⚠️ | Stub only |
| `SetSessionConfigOption` | ⚠️ | Stub only |
| `Prompt` | ✅ | Core execution loop |
| `Cancel` | ✅ | Cancels in-flight prompt |
| `LoadSession` | ❌ | Not implemented |

### Client Methods (NOT Implemented)

| Method | Status | Priority |
|--------|--------|----------|
| `SessionUpdate` | ✅ | Used for streaming |
| `RequestPermission` | ❌ | High |
| `ReadTextFile` | ❌ | High |
| `WriteTextFile` | ❌ | High |
| `CreateTerminal` | ❌ | Medium |
| `TerminalOutput` | ❌ | Medium |
