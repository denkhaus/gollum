# ACP Session Persistence Design

## Overview

Complete session management system with persistence using Ent ORM framework. Enables session restoration, listing, forking, and lifecycle management across all Gollum channels (ACP, TUI, future).

## Architecture

### Component Structure

**1. Ent ORM Layer** (`pkg/session/persistence`)
- Ent-Client with configurable database driver (SQLite, PostgreSQL, MySQL)
- Entities: `SessionEntity`, `MessageEntity`, `SupervisorConfigEntity`
- Schema migrations for database management

**2. Session Repository** (`pkg/session/repository`)
- `SessionRepository` interface for CRUD operations
- Ent-based implementation for persistence
- Methods: `Create`, `Get`, `List`, `Update`, `Delete`, `Fork`

**3. Session Manager Extension** (`pkg/session`)
- Existing `SessionManager` extended with repository methods
- Runtime state remains in-memory (current behavior)
- Persistence adds restart recovery capability

**4. ACP Integration** (`pkg/acp/service`)
- ACP-Service uses extended SessionManager methods
- ACP protocol methods implemented: `session/list`, `session/load`, `session/resume`, `session/close`, `session/fork`
- No persistence logic in ACP-Service

## Database Schema

### SessionEntity

```go
ID          uuid.UUID  // Primary Key
SessionID   uuid.UUID  // Unique Session ID
ChannelID   uuid.UUID  // Associated Channel
AgentID     uuid.UUID  // Associated Agent
Cwd         string     // Working Directory
CreatedAt   time.Time  // Creation timestamp
UpdatedAt   time.Time  // Last update timestamp
ClosedAt    *time.Time // Optional: When closed
State       string     // active, closed, archived

// Relations:
Messages    []MessageEntity      // 1:N relation
Supervisor  SupervisorConfigEntity // 1:1 relation
```

### MessageEntity

```go
ID          uuid.UUID  // Primary Key
SessionID   uuid.UUID  // Foreign Key to Session
Type        string     // user_chat, agent_chat, tool_request, etc.
Content     string     // Message content
Timestamp   time.Time  // Timestamp
Metadata     string     // JSON: tool_name, duration, etc.

// Relations:
Session     SessionEntity // N:1 relation
```

### SupervisorConfigEntity

```go
ID              uuid.UUID  // Primary Key
SessionID       uuid.UUID  // Foreign Key to Session
LLMModel        string     // Model name
LLMProvider     string     // Provider name
Temperature     float32    // LLM parameter
MaxTokens       int        // LLM parameter
SystemPrompt    string     // System prompt
ConfigJSON      string     // Additional config as JSON

// Relations:
Session         SessionEntity // 1:1 relation
```

### Indexes

- `idx_session_session_id` on SessionEntity.SessionID
- `idx_session_channel_id` on SessionEntity.ChannelID
- `idx_message_session_id` on MessageEntity.SessionID
- `idx_message_timestamp` on MessageEntity.Timestamp

## Repository Interface

```go
type SessionRepository interface {
    // CRUD operations
    Create(ctx context.Context, session *shared.Session) error
    Get(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error)
    List(ctx context.Context, filter *SessionFilter) ([]*shared.Session, error)
    Update(ctx context.Context, session *shared.Session) error
    Delete(ctx context.Context, sessionID uuid.UUID) error
    
    // Session-specific operations
    Fork(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error)
    Close(ctx context.Context, sessionID uuid.UUID) error
    Archive(ctx context.Context, olderThan time.Duration) (int64, error)
    
    // Message operations
    AddMessage(ctx context.Context, sessionID uuid.UUID, msg channel.Message) error
    GetMessages(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]channel.Message, error)
    
    // Query helpers
    Exists(ctx context.Context, sessionID uuid.UUID) (bool, error)
}

type SessionFilter struct {
    ChannelID    uuid.UUID
    AgentID      uuid.UUID
    State        string
    CreatedAfter time.Time
    CreatedBefore time.Time
}
```

## ACP Methods

```go
// ListSessions lists all available sessions (ACP: session/list)
func (s *acpServiceImpl) ListSessions(ctx context.Context) ([]*shared.Session, error)

// LoadSession loads an existing session (ACP: session/load)
func (s *acpServiceImpl) LoadSession(ctx context.Context, sessionID acppkg.SessionID) (*shared.ACPSession, error)

// ResumeSession resumes a session (ACP: session/resume)
func (s *acpServiceImpl) ResumeSession(ctx context.Context, sessionID acppkg.SessionID) error

// CloseSession closes a session (ACP: session/close)
func (s *acpServiceImpl) CloseSession(ctx context.Context, sessionID acppkg.SessionID) error

// ForkSession copies a session (ACP: session/fork)
func (s *acpServiceImpl) ForkSession(ctx context.Context, sessionID acppkg.SessionID) (acppkg.SessionID, error)
```

## Dependency Injection

### Repository Provider

```go
func NewSessionRepository(injector do.Injector) (SessionRepository, error) {
    cfg := do.MustInvoke[config.ConfigService](injector)
    log := do.MustInvoke[logger.LoggerService](injector)
    
    dbConfig := cfg.GetDatabaseConfig()
    repo, err := NewEntRepository(dbConfig.Driver, dbConfig.DSN)
    if err != nil {
        return nil, err
    }
    
    return repo, nil
}
```

### Extended SessionManager

```go
func NewSessionManager(injector do.Injector) (SessionManager, error) {
    repo, err := do.Invoke[SessionRepository](injector)
    if err != nil {
        return nil, err
    }
    
    return &sessionManagerImpl{
        repo: repo,
    }, nil
}
```

## Error Handling

### Error Types

```go
var (
    ErrSessionNotFound     = errors.New("session not found")
    ErrSessionClosed       = errors.New("session already closed")
    ErrSessionCannotFork   = errors.New("session cannot be forked")
    ErrDatabaseConnection  = errors.New("database connection failed")
    ErrDatabaseMigration   = errors.New("database migration failed")
)

type SessionError struct {
    SessionID uuid.UUID
    Op        string
    Err       error
}
```

### Graceful Degradation

```go
// Fallback to in-memory if persistence fails
func (m *sessionManagerImpl) CreateSession(ctx *shared.SessionContext) (*shared.Session, error) {
    session, err := m.repo.Create(ctx, sessionCtx)
    if err != nil {
        m.logger.Warn("session persistence failed, using in-memory only")
        return m.createInMemorySession(sessionCtx)
    }
    return session, nil
}
```

## Implementation Phases

**Phase 1: Persistence Foundation**
- Create `pkg/session/persistence` package
- Define Ent entities
- Create migrations
- Define `SessionRepository` interface
- Implement `EntRepository`

**Phase 2: SessionManager Integration**
- Extend `SessionManager` interface
- Implement repository integration
- Hybrid in-memory + persistence
- Error handling

**Phase 3: ACP Methods**
- Implement `ListSessions`
- Implement `LoadSession`
- Implement `ResumeSession`
- Implement `CloseSession`
- Implement `ForkSession`

**Phase 4: Configuration & DI**
- Database config in ConfigService
- Provider for SessionRepository
- Update SessionManager provider
- Update DI registry

**Phase 5: Testing**
- Repository unit tests
- SessionManager integration tests
- ACP service tests
- E2E tests

**Phase 6: Documentation**
- GoDoc comments
- README for session persistence
- Migration documentation

**Estimate:** 3-5 days for complete implementation

## Testing Strategy

### Repository Tests
- In-memory SQLite for fast tests
- Test CRUD operations
- Test Fork, List with filters

### Integration Tests
- SessionManager + Repository
- Test persistence flow

### ACP Service Tests
- Test ACP protocol methods
- Test SessionID format conversion

### E2E Tests
- Full session lifecycle
- Initialize → New → Prompt → Close → Load → Resume
