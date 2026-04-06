# Multisession Implementation Plan

**Based on**: 2026-04-06-multisession-design.md
**Created**: 2026-04-06
**Status**: Ready for Execution

---

## Overview

This plan breaks down the multisession architecture implementation into executable tasks. Each task is designed to be completed independently with clear acceptance criteria.

---

## Phase 1: Foundation (Message Structure & Session Manager)

### Task 1.1: Extended Message Structure
**File**: `pkg/channel/interface.go`
**Estimate**: 30 minutes

**Changes**:
```go
type Message struct {
    ID        uuid.UUID
    Type      MessageType
    AgentID   uuid.UUID
    AgentRole string
    SessionID string      // ADD THIS
    ChannelID uuid.UUID   // ADD THIS
    Content   string
    Timestamp time.Time
    Metadata  map[string]any
}
```

**Acceptance**:
- [ ] Message struct has SessionID and ChannelID fields
- [ ] All existing code compiles (uses empty defaults for now)
- [ ] Tests pass

---

### Task 1.2: Session Manager Component
**File**: `pkg/session/manager.go` (NEW)
**Estimate**: 2 hours

**Create**:
```go
package session

type SessionManager interface {
    CreateSession(ctx context.Context, channelID uuid.UUID) (*Session, error)
    GetSession(sessionID string) (*Session, bool)
    CloseSession(sessionID string) error
    GetSessionsByChannel(channelID uuid.UUID) []*Session
}

type Session struct {
    ID           string
    ChannelID    uuid.UUID
    SupervisorID uuid.UUID
    Context      context.Context
    CancelFunc   context.CancelFunc
    CreatedAt    time.Time
}
```

**Acceptance**:
- [ ] SessionManager interface defined
- [ ] Basic implementation with in-memory storage
- [ ] Session creation generates unique IDs
- [ ] DI provider registered
- [ ] Unit tests for basic operations

---

### Task 1.3: Targeted Message Routing in Facade
**File**: `pkg/channel/facade.go`
**Estimate**: 1 hour

**Change**:
```go
// FROM: Broadcast to all channels
func (p *channelFacadeImpl) DisplayMessage(msg Message) {
    p.mu.RLock()
    channel, exists := p.channels[msg.ChannelID]  // Route by ChannelID
    p.mu.RUnlock()

    if !exists {
        p.logger.Warn("channel not found", zap.String("channel_id", msg.ChannelID.String()))
        return
    }

    channel.OnMessage(msg)
}
```

**Acceptance**:
- [ ] DisplayMessage routes by msg.ChannelID
- [ ] Logs warning when channel not found
- [ ] No broadcast behavior
- [ ] Tests verify targeted routing

---

## Phase 2: Middleware & Agent Factory

### Task 2.1: Session-Aware Channel Middleware
**File**: `pkg/channel/middleware.go`
**Estimate**: 1 hour

**Change**:
```go
type ChannelMiddleware struct {
    facade     ChannelFacade
    agentID    uuid.UUID
    agentRole  string
    sessionID  string      // ADD
    channelID  uuid.UUID   // ADD
}

func NewChannelMiddleware(
    facade ChannelFacade,
    agentID uuid.UUID,
    agentRole string,
    sessionID string,      // ADD PARAM
    channelID uuid.UUID,   // ADD PARAM
) *ChannelMiddleware {
    return &ChannelMiddleware{
        facade:     facade,
        agentID:    agentID,
        agentRole:  agentRole,
        sessionID:  sessionID,
        channelID:  channelID,
    }
}

// Inject SessionID/ChannelID in ContentBlockMiddleware
// Inject SessionID/ChannelID in ToolMiddleware
```

**Acceptance**:
- [ ] Middleware stores sessionID and channelID
- [ ] ContentBlockMiddleware injects SessionID/ChannelID
- [ ] ToolMiddleware injects SessionID/ChannelID
- [ ] Provider interface updated
- [ ] Tests verify injection

---

### Task 2.2: Update AgentFactory for Session Context
**File**: `pkg/agents/factory.go`
**Estimate**: 1.5 hours

**Change**:
- Update `CreateAgent()` to accept optional sessionID/channelID
- Pass session context to middleware creation
- Support multiple supervisor agents per session

**Acceptance**:
- [ ] AgentFactory supports session context
- [ ] Middleware created with session parameters
- [ ] Multiple supervisors can coexist
- [ ] Tests verify session isolation

---

### Task 2.3: Session-Specific Supervisor Creation
**File**: `pkg/agents/factory.go`
**Estimate**: 1 hour

**Add method**:
```go
func (f *defaultAgentFactory) CreateSupervisorForSession(
    ctx context.Context,
    sessionID string,
    channelID uuid.UUID,
) (shared.Agent, error) {
    // Create supervisor agent for this specific session
    // Register in SessionManager
    // Return agent reference
}
```

**Acceptance**:
- [ ] Supervisor creation per session
- [ ] SessionManager integration
- [ ] Supervisor tracked per session
- [ ] Tests verify isolation

---

## Phase 3: ACP Integration

### Task 3.1: Remove Active Session Pattern
**File**: `pkg/acp/service.go`
**Estimate**: 1 hour

**Remove**:
- `activeSession` field
- `activeSessionMu` field
- `setActiveSession()` method
- `clearActiveSession()` method

**Acceptance**:
- [ ] Single-session tracking removed
- [ ] Code compiles
- [ ] No references to activeSession remain

---

### Task 3.2: Integrate SessionManager in ACP Service
**File**: `pkg/acp/service.go`
**Estimate**: 2 hours

**Change**:
```go
type acpServiceImpl struct {
    // Remove: activeSession, activeSessionMu
    // Add:
    sessionManager session.SessionManager  // ADD
}

func NewAcpService(injector do.Injector) (shared.ACPService, error) {
    // ...
    sessionManager := do.MustInvoke[session.SessionManager](injector)  // ADD
    // ...
}
```

**Acceptance**:
- [ ] SessionManager injected
- [ ] Service uses SessionManager for session lookups
- [ ] DI configured correctly

---

### Task 3.3: Update Prompt() for Multi-Session
**File**: `pkg/acp/service.go`
**Estimate**: 2 hours

**Change**:
```go
func (s *acpServiceImpl) Prompt(ctx context.Context, params *acppkg.PromptRequest) (*acppkg.PromptResponse, error) {
    // Get session from params.SessionID
    session, ok := s.store.Get(params.SessionID)
    if !ok {
        return nil, fmt.Errorf("session not found")
    }

    // Get session-specific supervisor
    supervisor, err := s.sessionManager.GetSession(session.SessionID)
    if err != nil {
        return nil, err
    }

    agent, err := s.registry.GetAgent(supervisor.SupervisorID)
    if err != nil {
        return nil, err
    }

    // Execute with session context
    // ...
}
```

**Acceptance**:
- [ ] Prompt uses explicit SessionID
- [ ] Routes to session-specific supervisor
- [ ] No activeSession dependency
- [ ] Tests verify correct routing

---

### Task 3.4: Update OnMessage() for Multi-Session
**File**: `pkg/acp/service.go`
**Estimate**: 1.5 hours

**Change**:
```go
func (s *acpServiceImpl) OnMessage(msg channel.Message) {
    // Use msg.SessionID to look up session
    session, ok := s.store.Get(acppkg.SessionID(msg.SessionID))
    if !ok {
        s.logger.Warn("session not found", zap.String("session_id", msg.SessionID))
        return
    }

    // Stream to correct session
    stream := acppkg.NewSessionStream(s.client, session.SessionID)
    stream.SendText(session.Context, msg.Content)
}
```

**Acceptance**:
- [ ] OnMessage uses msg.SessionID
- [ ] Streams to correct session
- [ ] Handles missing sessions gracefully
- [ ] Tests verify routing

---

## Phase 4: Commands & Integration

### Task 4.1: Session-Aware Commands
**File**: `pkg/channel/command.go`
**Estimate**: 1 hour

**Change**:
```go
type CommandHandler func(ctx context.Context, sessionID string, args string) (string, error)

func (m *commandManagerImpl) Execute(ctx context.Context, sessionID string, input string) (bool, string, error) {
    // Parse command
    // Execute with sessionID context
}
```

**Acceptance**:
- [ ] CommandHandler includes sessionID
- [ ] Execute passes sessionID to handlers
- [ ] Existing commands updated
- [ ] Tests verify session context

---

### Task 4.2: Update Facade SubmitInput()
**File**: `pkg/channel/facade.go`
**Estimate**: 1.5 hours

**Change**:
```go
func (p *channelFacadeImpl) SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (InputResult, error) {
    // Check for command
    handled, response, err := p.commandManager.Execute(ctx, sessionID, input)
    if handled {
        return InputResult{Handled: true, IsCommand: true, Response: response, Error: err}, nil
    }

    // Get or create session
    session, err := p.sessionManager.CreateSession(ctx, channelID)
    if err != nil {
        return InputResult{}, err
    }

    // Get session supervisor
    // Execute...
}
```

**Acceptance**:
- [ ] SubmitInput accepts sessionID
- [ ] Creates session via SessionManager
- [ ] Routes to session-specific supervisor
- [ ] Tests verify flow

---

### Task 4.3: Integration Tests
**File**: `pkg/acp/integration_test.go` (NEW)
**Estimate**: 3 hours

**Create tests for**:
- Multiple concurrent sessions
- Message routing between sessions
- Session isolation
- Session lifecycle (create, use, close)

**Acceptance**:
- [ ] Multi-session scenario tests
- [ ] Isolation verified
- [ ] Routing verified
- [ ] All tests pass

---

### Task 4.4: Documentation
**Files**: Various
**Estimate**: 2 hours

**Update**:
- README with multisession info
- Code comments for session flow
- Architecture diagrams
- Migration guide (if needed)

**Acceptance**:
- [ ] README updated
- [ ] Code documented
- [ ] Diagrams created/updated
- [ ] Examples provided

---

## Task Dependencies

```
Phase 1 (Foundation)
├── Task 1.1 (Message structure) → MUST BE FIRST
├── Task 1.2 (Session Manager) → independent
└── Task 1.3 (Targeted routing) → needs Task 1.1

Phase 2 (Middleware)
├── Task 2.1 (Session-aware middleware) → needs Task 1.1
├── Task 2.2 (AgentFactory update) → needs Task 2.1
└── Task 2.3 (Session-specific supervisor) → needs Task 1.2, Task 2.2

Phase 3 (ACP Integration)
├── Task 3.1 (Remove activeSession) → independent, can be done anytime
├── Task 3.2 (Integrate SessionManager) → needs Task 1.2, Task 3.1
├── Task 3.3 (Update Prompt) → needs Task 1.2, Task 2.3, Task 3.2
└── Task 3.4 (Update OnMessage) → needs Task 1.1, Task 3.2

Phase 4 (Commands & Integration)
├── Task 4.1 (Session-aware commands) → needs Task 1.1
├── Task 4.2 (Update SubmitInput) → needs Task 1.2, Task 2.3, Task 4.1
├── Task 4.3 (Integration tests) → needs all previous tasks
└── Task 4.4 (Documentation) → needs all implementation tasks
```

---

## Execution Order

**Sprint 1** (Foundation):
1. Task 1.1: Extended Message Structure
2. Task 1.2: Session Manager Component
3. Task 1.3: Targeted Message Routing

**Sprint 2** (Middleware):
4. Task 2.1: Session-Aware Channel Middleware
5. Task 2.2: Update AgentFactory
6. Task 2.3: Session-Specific Supervisor Creation

**Sprint 3** (ACP Integration):
7. Task 3.1: Remove Active Session Pattern (can be done earlier)
8. Task 3.2: Integrate SessionManager
9. Task 3.3: Update Prompt()
10. Task 3.4: Update OnMessage()

**Sprint 4** (Finalize):
11. Task 4.1: Session-Aware Commands
12. Task 4.2: Update Facade SubmitInput()
13. Task 4.3: Integration Tests
14. Task 4.4: Documentation

---

## Risk Mitigation

### Risk: Breaking Existing Channels
**Mitigation**: Add feature flag or gradual rollout
**Fallback**: Keep broadcast mode for channels without SessionID

### Risk: Session Leaks
**Mitigation**: Implement session timeout/cleanup
**Monitoring**: Add session count metrics

### Risk: Performance with Many Sessions
**Mitigation**: Session pooling, limits on concurrent sessions
**Testing**: Load testing with 100+ sessions

---

## Success Criteria

- [ ] Multiple ACP sessions can run concurrently
- [ ] Each session has isolated Supervisor Agent
- [ ] Messages route to correct sessions
- [ ] No broadcast behavior
- [ ] All tests pass
- [ ] Documentation complete
- [ ] Backward compatibility maintained (where possible)

---

## Next Steps

1. Review this plan with team
2. Create tasks in tracking system
3. Begin with Sprint 1, Task 1.1
4. Daily standups to track progress
5. Adjust estimates as we learn

**Estimated Total Time**: 18-22 hours across 4 sprints
**Recommended Timeline**: 2 weeks with parallel work possible
