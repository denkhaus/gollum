# Multisession Architecture Design

**Date**: 2026-04-06
**Status**: Draft Specification
**Author**: Brainstorming Session

---

## Executive Summary

This document specifies the design for implementing full ACP (Agent Client Protocol) multisession support in Gollum. The goal is to transform Gollum into a fully compliant ACP server that supports multiple concurrent sessions, each with its own Supervisor Agent.

---

## Problem Statement

### Current Limitations

1. **Single Session per Connection**: Only one `activeSession` is tracked per ACP connection
2. **Singleton Supervisor Agent**: All requests route to a single shared Supervisor Agent
3. **Broadcast Messaging**: `DisplayMessage()` sends to all channels indiscriminately
4. **No Session Context**: Messages lack SessionID/ChannelID correlation
5. **Channel Middleware not Session-Aware**: Middleware instances are per-Agent, not per-Session

### ACP Standard Requirements

The ACP protocol requires:
- Multiple concurrent sessions per connection
- Each session has its own context and agent lifecycle
- `session/new` creates sessions explicitly
- `Prompt` requests specify which session they belong to
- Messages must be routable to specific sessions

---

## Goals

1. ✅ **Full ACP Compliance**: Support all ACP session management features
2. ✅ **Multiple Concurrent Sessions**: Multiple sessions can be active simultaneously
3. ✅ **Session-Isolated Supervisor Agents**: Each session has its own Supervisor Agent
4. ✅ **Targeted Message Routing**: Messages route to specific channels/sessions, not broadcast
5. ✅ **Session-Aware Commands**: Commands execute in session context
6. ✅ **Clean Architecture**: Minimal changes, maintainable code

---

## Current Architecture Analysis

### Message Structure (Current)
```go
type Message struct {
    ID        uuid.UUID
    Type      MessageType
    AgentID   uuid.UUID
    AgentRole string
    Content   string
    Timestamp time.Time
    Metadata  map[string]any
}
```

**Problem**: No SessionID or ChannelID - cannot correlate messages to sessions.

### ChannelFacade (Current)
```go
// Broadcasts to ALL channels
func (p *channelFacadeImpl) DisplayMessage(msg Message) {
    for _, channel := range p.channels {
        channel.OnMessage(msg)
    }
}

// Single supervisor for everything
func (p *channelFacadeImpl) SubmitInput(ctx context.Context, channelID uuid.UUID, input string) {
    supervisor := p.registry.GetSupervisorAgent()  // Singleton!
    supervisor.Execute(ctx, input)
}
```

**Problems**:
- Broadcast instead of targeted routing
- Singleton supervisor agent
- channelID parameter ignored for routing

### ChannelMiddleware (Current)
```go
type ChannelMiddleware struct {
    facade    ChannelFacade
    agentID   uuid.UUID
    agentRole string
}
```

**Problem**: One middleware per Agent, not per (Agent, Session) combination.

### ACP Service (Current)
```go
type acpServiceImpl struct {
    activeSession   *shared.ACPSession  // Only ONE session
    activeSessionMu sync.RWMutex
}
```

**Problem**: Tracks only single active session.

---

## Design Decisions

### Decision 1: Extended Message Structure

**Rationale**: Messages must carry session and channel information for routing.

```go
type Message struct {
    ID        uuid.UUID
    Type      MessageType
    AgentID   uuid.UUID
    AgentRole string
    SessionID string      // NEW: ACP Session ID
    ChannelID uuid.UUID   // NEW: Target Channel ID
    Content   string
    Timestamp time.Time
    Metadata  map[string]any
}
```

### Decision 2: Session-Aware Channel Middleware

**Rationale**: Middleware instances are created per (Agent, Session) pair, not per Agent.

```go
type ChannelMiddleware struct {
    facade     ChannelFacade
    agentID    uuid.UUID
    agentRole  string
    sessionID  string      // NEW: Session this middleware belongs to
    channelID  uuid.UUID   // NEW: Channel this session belongs to
}
```

The middleware automatically injects SessionID and ChannelID into all messages it creates.

### Decision 3: Session Manager Component

**Rationale**: Central component to manage sessions and their associated Supervisor Agents.

```go
type SessionManager interface {
    CreateSession(ctx context.Context, channelID uuid.UUID) (*Session, error)
    GetSession(sessionID string) (*Session, bool)
    CloseSession(sessionID string) error
    GetSessionsByChannel(channelID uuid.UUID) []*Session
}

type Session struct {
    ID           string
    ChannelID    uuid.UUID
    SupervisorID uuid.UUID   // Unique supervisor for this session
    Context      context.Context
    CancelFunc   context.CancelFunc
    CreatedAt    time.Time
}
```

### Decision 4: Targeted Message Routing

**Rationale**: Messages route to specific channels based on ChannelID, not broadcast.

```go
func (p *channelFacadeImpl) DisplayMessage(msg Message) {
    // Route to specific channel based on msg.ChannelID
    channel, exists := p.channels[msg.ChannelID]
    if !exists {
        p.logger.Warn("channel not found")
        return
    }
    channel.OnMessage(msg)
}
```

### Decision 5: Session-Aware Facade Methods

**Rationale**: Facade methods need session context to route correctly.

```go
// SubmitInput creates/retrieves session and routes to session-specific supervisor
func (p *channelFacadeImpl) SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (InputResult, error)

// CommandManager executes commands with session context
func (m *CommandManager) Execute(ctx context.Context, sessionID string, input string) (bool, string, error)
```

---

## Component Changes

### 1. pkg/channel/interface.go

**Changes**:
- Add `SessionID string` to Message struct
- Add `ChannelID uuid.UUID` to Message struct

**Impact**: All Message consumers must be updated.

### 2. pkg/channel/middleware.go

**Changes**:
- Add `sessionID string` and `channelID uuid.UUID` to ChannelMiddleware struct
- Update `NewChannelMiddleware()` constructor
- Update `CreateChannelMiddleware()` in provider interface
- Inject SessionID/ChannelID in ContentBlockMiddleware
- Inject SessionID/ChannelID in ToolMiddleware

**Impact**: AgentFactory must pass session/channel IDs when creating middleware.

### 3. pkg/channel/facade.go

**Changes**:
- Change `DisplayMessage()` from broadcast to targeted routing
- Update `SubmitInput()` signature to include sessionID
- Integrate SessionManager for session creation/retrieval
- Route to session-specific Supervisor Agent (not singleton)
- Update `CancelInput()` to handle session cancellation

**Impact**: Major refactoring of core routing logic.

### 4. pkg/agents/factory.go

**Changes**:
- Update `CreateChannelMiddleware()` calls to include sessionID/channelID
- Support creating multiple supervisor agents (one per session)

**Impact**: Agent creation must be session-aware.

### 5. pkg/acp/service.go

**Changes**:
- Remove `activeSession` single-session tracking
- Use SessionManager for multi-session support
- Update `Prompt()` to work with explicit sessionID
- Update `OnMessage()` to route to correct session based on SessionID in Message
- Remove `setActiveSession()`/`clearActiveSession()` methods

**Impact**: Core ACP logic changes from single to multi-session.

### 6. pkg/session/manager.go (NEW)

**Changes**:
- Create new SessionManager component
- Implement session lifecycle (create, get, close)
- Integrate with AgentRegistry for supervisor creation
- Track supervisor agents per session

**Impact**: New component, integrates with existing systems.

### 7. pkg/channel/command.go

**Changes**:
- Update CommandHandler signature: `func(ctx context.Context, sessionID string, args string) (string, error)`
- Update CommandManager.Execute() to include sessionID

**Impact**: All command handlers must be updated.

---

## Message Flow

### Prompt Flow (Client → Agent System)

```
1. ACP Client sends PromptRequest{SessionID: "abc123", Prompt: "..."}
       ↓
2. ACP Service.Prompt() receives sessionID
       ↓
3. SessionManager.GetSession("abc123") → Session
       ↓
4. Get Session.SupervisorID → AgentID
       ↓
5. AgentRegistry.GetAgent(AgentID) → Supervisor
       ↓
6. Supervisor.Execute(ctx, input)
       ↓
7. ChannelMiddleware (session-aware) receives output
       ↓
8. Middleware creates Message{SessionID: "abc123", ChannelID: xyz, Content: "..."}
       ↓
9. Facade.DisplayMessage(msg) → routes to Channel xyz
       ↓
10. Channel.OnMessage(msg) → ACP Service.OnMessage()
       ↓
11. ACP Service streams to correct ACP session via SessionStream
```

### OnMessage Flow (Agent System → ACP Client)

```
1. Supervisor Agent generates output
       ↓
2. ChannelMiddleware (created for this session) intercepts
       ↓
3. Middleware injects SessionID and ChannelID
       ↓
4. Facade.DisplayMessage(msg) routes to ACP channel
       ↓
5. ACP Service.OnMessage(msg) receives msg with SessionID
       ↓
6. Look up session from SessionID
       ↓
7. Stream to ACP client via NewSessionStream(client, sessionID)
```

---

## Open Questions

1. **Command Execution Context**: How do commands access the session context for operations like workspace, tools, etc.?

2. **Session Lifecycle**: When exactly should a session be closed?
   - After Prompt completes?
   - After timeout?
   - Explicit close from client?

3. **Session Initialization**: Does `Initialize()` create a session, or does the first `Prompt()`?

4. **Agent Registry Integration**: How do we track multiple supervisor agents in the registry?

5. **Backward Compatibility**: How do we maintain compatibility with existing channels (Discord, Slack) that don't use sessions?

6. **Session State**: What session state needs to be persisted (if any)?

7. **Error Handling**: What happens when a message arrives for a non-existent session?

---

## Implementation Plan (High-Level)

### Phase 1: Foundation (Week 1)
1. Extended Message structure
2. Session Manager component (basic version)
3. Update ChannelFacade for targeted routing

### Phase 2: Middleware & Agent Factory (Week 2)
4. Session-aware ChannelMiddleware
5. Update AgentFactory for session-specific middleware
6. Multiple supervisor agent support

### Phase 3: ACP Integration (Week 3)
7. Update ACP Service for multi-session
8. Remove activeSession pattern
9. Session lifecycle in ACP context

### Phase 4: Commands & Testing (Week 4)
10. Session-aware commands
11. Integration testing
12. Documentation

---

## Appendix: ACP Protocol Reference

### Session Lifecycle

1. **Connection Established**
   - Client connects to server
   - `Initialize()` handshake

2. **Session Creation** (Client-initiated)
   - Client sends `session/new` request
   - Server creates Session with unique SessionID
   - Server creates Supervisor Agent for this session
   - Server responds with SessionID

3. **Prompt Execution**
   - Client sends `Prompt` request with SessionID
   - Server routes to session-specific Supervisor
   - Supervisor processes and streams responses
   - Responses sent via `session/update` notifications

4. **Session Termination**
   - Client sends `session/close` (optional)
   - Or server closes after timeout
   - Supervisor Agent cleaned up
   - Session resources released

---

## Related Documents

- [ACP Improvements](../acp_improvements.md) - ACP integration gaps and improvements
- [Channel Architecture](../architecture/channel.md) - Channel system design (if exists)
- [Agent Registry](../architecture/agents.md) - Agent registry design (if exists)
