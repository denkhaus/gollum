# ACP Session Persistence Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build complete session management system with database persistence using Ent ORM, enabling session restoration, listing, forking, and lifecycle management across all Gollum channels.

**Architecture:** Three-layer approach with Ent ORM for persistence, Repository interface for data access, and extended SessionManager for business logic. ACP-Service implements protocol methods (list/load/resume/close/fork) using the extended SessionManager.

**Tech Stack:** Ent ORM (facebook/ent), SQLite/PostgreSQL/MySQL drivers, go-acp for ACP protocol, samber/do/v2 for DI, existing session/sessionManager infrastructure.

---

## Chunk 1: Persistence Foundation

### Task 1: Create persistence package structure

**Files:**
- Create: `pkg/session/persistence/ent/schema/session.go`
- Create: `pkg/session/persistence/ent/schema/message.go`
- Create: `pkg/session/persistence/ent/schema/supervisor_config.go`
- Create: `pkg/session/persistence/ent/generate.go`

- [ ] **Step 1: Create SessionEntity schema**

```go
package schema

import (
    "time"
    
    "entgo.io/ent"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
    "github.com/google/uuid"
)

// Session holds the schema definition for the Session entity.
type Session struct {
    ent.Schema
}

// Fields of the Session.
func (Session) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("id", uuid.UUID).
            Default(uuid.New).
            Unique(),
        field.UUID("session_id", uuid.UUID).
            Unique(),
        field.UUID("channel_id", uuid.UUID),
        field.UUID("agent_id", uuid.UUID),
        field.String("cwd").
            Default("."),
        field.Time("created_at").
            Default(time.Now).
            Immutable(),
        field.Time("updated_at").
            Default(time.Now).
            UpdateDefault(time.Now),
        field.Time("closed_at").
            Optional().
            Nillable(),
        field.Enum("state").
            Values("active", "closed", "archived").
            Default("active"),
    }
}

// Edges of the Session.
func (Session) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("messages", Message.Type).
            StorageKey(edge.Field("session_id")),
        edge.To("supervisor", SupervisorConfig.Type).
            Unique(),
    }
}

// Indexes of the Session.
func (Session) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("session_id"),
        index.Fields("channel_id"),
    }
}
```

- [ ] **Step 2: Create MessageEntity schema**

```go
package schema

import (
    "time"
    
    "entgo.io/ent"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
    "github.com/google/uuid"
)

// Message holds the schema definition for the Message entity.
type Message struct {
    ent.Schema
}

// Fields of the Message.
func (Message) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("id", uuid.UUID).
            Default(uuid.New).
            Unique(),
        field.UUID("session_id", uuid.UUID),
        field.Enum("type").
            Values("user_chat", "agent_chat", "tool_request", "tool_response", "thinking", "system_info", "error"),
        field.Text("content"),
        field.Time("timestamp").
            Default(time.Now).
            Immutable(),
        field.JSON("metadata", map[string]interface{}{}).
            Optional(),
    }
}

// Edges of the Message.
func (Message) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("session", Session.Type).
            Ref("messages").
            Field("session_id").
            Unique().
            Required(),
    }
}

// Indexes of the Message.
func (Message) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("session_id"),
        index.Fields("timestamp"),
    }
}
```

- [ ] **Step 3: Create SupervisorConfigEntity schema**

```go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
    "github.com/google/uuid"
)

// SupervisorConfig holds the schema definition for the SupervisorConfig entity.
type SupervisorConfig struct {
    ent.Schema
}

// Fields of the SupervisorConfig.
func (SupervisorConfig) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("id", uuid.UUID).
            Default(uuid.New).
            Unique(),
        field.UUID("session_id", uuid.UUID).
            Unique(),
        field.String("llm_model").
            Default(""),
        field.String("llm_provider").
            Default(""),
        field.Float32("temperature").
            Default(0.7),
        field.Int("max_tokens").
            Default(4096),
        field.Text("system_prompt").
            Optional(),
        field.JSON("config_json", map[string]interface{}{}).
            Optional(),
    }
}

// Edges of the SupervisorConfig.
func (SupervisorConfig) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("session", Session.Type).
            Ref("supervisor").
            Field("session_id").
            Unique().
            Required(),
    }
}
```

- [ ] **Step 4: Create Ent generate file**

```go
//go:generate go run -mod=mod entgo.io/cmd/ent generate ./schema
package ent
```

- [ ] **Step 5: Run ent generate**

```bash
cd pkg/session/persistence
go generate ./ent
```

Expected: Ent code generated in `pkg/session/persistence/ent/`

- [ ] **Step 6: Commit persistence schema**

```bash
git add pkg/session/persistence/
git commit -m "feat(persistence): add Ent schema for session persistence"
```

### Task 2: Create repository interface and types

**Files:**
- Create: `pkg/session/repository/repository.go`
- Create: `pkg/session/repository/errors.go`

- [ ] **Step 1: Create repository interface**

```go
package repository

import (
    "context"
    "time"
    
    "github.com/denkhaus/gollum/pkg/channel"
    "github.com/denkhaus/gollum/pkg/shared"
    "github.com/google/uuid"
)

// SessionRepository defines the interface for session persistence operations.
type SessionRepository interface {
    // Create persists a new session.
    Create(ctx context.Context, session *shared.Session) error
    
    // Get retrieves a session by ID.
    Get(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error)
    
    // List retrieves sessions with optional filtering.
    List(ctx context.Context, filter *SessionFilter) ([]*shared.Session, error)
    
    // Update updates an existing session.
    Update(ctx context.Context, session *shared.Session) error
    
    // Delete removes a session.
    Delete(ctx context.Context, sessionID uuid.UUID) error
    
    // Fork creates a copy of a session.
    Fork(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error)
    
    // Close marks a session as closed.
    Close(ctx context.Context, sessionID uuid.UUID) error
    
    // Archive removes old sessions.
    Archive(ctx context.Context, olderThan time.Duration) (int64, error)
    
    // AddMessage adds a message to a session.
    AddMessage(ctx context.Context, sessionID uuid.UUID, msg channel.Message) error
    
    // GetMessages retrieves messages from a session.
    GetMessages(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]channel.Message, error)
    
    // Exists checks if a session exists.
    Exists(ctx context.Context, sessionID uuid.UUID) (bool, error)
}

// SessionFilter defines filtering options for List operations.
type SessionFilter struct {
    ChannelID     uuid.UUID
    AgentID       uuid.UUID
    State         string
    CreatedAfter  time.Time
    CreatedBefore time.Time
}
```

- [ ] **Step 2: Create repository errors**

```go
package repository

import (
    "errors"
    "github.com/google/uuid"
)

var (
    // ErrSessionNotFound indicates a session was not found.
    ErrSessionNotFound = errors.New("session not found")
    
    // ErrSessionClosed indicates a session is already closed.
    ErrSessionClosed = errors.New("session already closed")
    
    // ErrSessionCannotFork indicates a session cannot be forked.
    ErrSessionCannotFork = errors.New("session cannot be forked")
    
    // ErrDatabaseConnection indicates a database connection failure.
    ErrDatabaseConnection = errors.New("database connection failed")
    
    // ErrDatabaseMigration indicates a migration failure.
    ErrDatabaseMigration = errors.New("database migration failed")
)

// SessionError wraps errors with session context.
type SessionError struct {
    SessionID uuid.UUID
    Op        string
    Err       error
}

func (e *SessionError) Error() string {
    return fmt.Sprintf("session %s: %s failed: %v", e.SessionID, e.Op, e.Err)
}

func (e *SessionError) Unwrap() error {
    return e.Err
}
```

- [ ] **Step 3: Commit repository interface**

```bash
git add pkg/session/repository/
git commit -m "feat(repository): add session repository interface and error types"
```

### Task 3: Implement EntRepository

**Files:**
- Create: `pkg/session/persistence/entrepo.go`

- [ ] **Step 1: Create EntRepository struct and constructor**

```go
package persistence

import (
    "context"
    "fmt"
    
    "entgo.io/ent/dialect/sql"
    "github.com/denkhaus/gollum/pkg/channel"
    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/denkhaus/gollum/pkg/repository"
    "github.com/denkhaus/gollum/pkg/session/persistence/ent"
    "github.com/google/uuid"
)

// EntRepository implements SessionRepository using Ent ORM.
type EntRepository struct {
    client *ent.Client
    logger logger.LoggerService
}

// NewEntRepository creates a new EntRepository with the specified driver and DSN.
func NewEntRepository(driver string, dsn string, log logger.LoggerService) (*EntRepository, error) {
    client, err := ent.Open(driver, dsn)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", repository.ErrDatabaseConnection, err)
    }
    
    return &EntRepository{
        client: client,
        logger: log,
    }, nil
}

// Close closes the database connection.
func (r *EntRepository) Close() error {
    return r.client.Close()
}
```

- [ ] **Step 2: Implement Create method**

```go
func (r *EntRepository) Create(ctx context.Context, session *shared.Session) error {
    tx, err := r.client.Tx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Create session
    sessionEnt, err := tx.Session.
        Create().
        SetSessionID(session.ID).
        SetChannelID(session.ChannelID).
        SetAgentID(session.SupervisorID).
        SetCwd(session.Cwd).
        SetState("active").
        Save(ctx)
    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }
    
    // Create supervisor config if available
    if session.SupervisorID != uuid.Nil {
        _, err = tx.SupervisorConfig.
            Create().
            SetSessionID(session.ID).
            SetLLMModel(session.Model).
            SetLLMProvider(session.Provider).
            SetTemperature(session.Temperature).
            SetMaxTokens(session.MaxTokens).
            Save(ctx)
        if err != nil {
            return fmt.Errorf("failed to create supervisor config: %w", err)
        }
    }
    
    return tx.Commit()
}
```

- [ ] **Step 3: Implement Get method**

```go
func (r *EntRepository) Get(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error) {
    sessionEnt, err := r.client.Session.
        Query().
        Where(ent.SessionSessionID(sessionID)).
        WithMessages().
        WithSupervisor().
        Only(ctx)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", repository.ErrSessionNotFound, err)
    }
    
    return r.entityToSession(sessionEnt)
}
```

- [ ] **Step 4: Implement List method**

```go
func (r *EntRepository) List(ctx context.Context, filter *repository.SessionFilter) ([]*shared.Session, error) {
    query := r.client.Session.Query()
    
    if filter != nil {
        if filter.ChannelID != uuid.Nil {
            query.Where(ent.SessionChannelID(filter.ChannelID))
        }
        if filter.AgentID != uuid.Nil {
            query.Where(ent.SessionAgentID(filter.AgentID))
        }
        if filter.State != "" {
            query.Where(ent.SessionState(filter.State))
        }
        if !filter.CreatedAfter.IsZero() {
            query.Where(ent.SessionCreatedAtGTE(filter.CreatedAfter))
        }
        if !filter.CreatedBefore.IsZero() {
            query.Where(ent.SessionCreatedAtLTE(filter.CreatedBefore))
        }
    }
    
    entities, err := query.
        Order(ent.Asc(ent.SessionCreatedAt)).
        All(ctx)
    if err != nil {
        return nil, err
    }
    
    sessions := make([]*shared.Session, len(entities))
    for i, ent := range entities {
        session, err := r.entityToSession(ent)
        if err != nil {
            return nil, err
        }
        sessions[i] = session
    }
    
    return sessions, nil
}
```

- [ ] **Step 5: Implement Fork method**

```go
func (r *EntRepository) Fork(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error) {
    tx, err := r.client.Tx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()
    
    // Get source session
    source, err := tx.Session.
        Query().
        Where(ent.SessionSessionID(sessionID)).
        WithSupervisor().
        Only(ctx)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", repository.ErrSessionNotFound, err)
    }
    
    // Create new session
    newID := uuid.New()
    newSession, err := tx.Session.
        Create().
        SetSessionID(newID).
        SetChannelID(source.ChannelID).
        SetAgentID(source.AgentID).
        SetCwd(source.Cwd).
        SetState("active").
        Save(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to create forked session: %w", err)
    }
    
    // Copy supervisor config
    if source.Edges.Supervisor != nil {
        _, err = tx.SupervisorConfig.
            Create().
            SetSessionID(newID).
            SetLLMModel(source.Edges.Supervisor.LLMModel).
            SetLLMProvider(source.Edges.Supervisor.LLMProvider).
            SetTemperature(source.Edges.Supervisor.Temperature).
            SetMaxTokens(source.Edges.Supervisor.MaxTokens).
            Save(ctx)
        if err != nil {
            return nil, fmt.Errorf("failed to copy supervisor config: %w", err)
        }
    }
    
    if err := tx.Commit(); err != nil {
        return nil, err
    }
    
    return r.Get(ctx, newID)
}
```

- [ ] **Step 6: Implement Close method**

```go
func (r *EntRepository) Close(ctx context.Context, sessionID uuid.UUID) error {
    now := time.Now()
    _, err := r.client.Session.
        Update().
        Where(ent.SessionSessionID(sessionID)).
        SetState("closed").
        SetClosedAt(&now).
        Save(ctx)
    return err
}
```

- [ ] **Step 7: Implement Exists method**

```go
func (r *EntRepository) Exists(ctx context.Context, sessionID uuid.UUID) (bool, error) {
    return r.client.Session.
        Query().
        Where(ent.SessionSessionID(sessionID)).
        Exist(ctx)
}
```

- [ ] **Step 8: Add helper method entityToSession**

```go
func (r *EntRepository) entityToSession(ent *ent.Session) (*shared.Session, error) {
    ctx, cancel := context.WithCancel(context.Background())
    
    supervisorID := uuid.Nil
    if ent.Edges.Supervisor != nil {
        supervisorID = ent.Edges.Supervisor.ID
    }
    
    return &shared.Session{
        ID:           ent.SessionID,
        ChannelID:    ent.ChannelID,
        SupervisorID: supervisorID,
        Cwd:          ent.Cwd,
        Context:      ctx,
        CancelFunc:   cancel,
        CreatedAt:    ent.CreatedAt,
    }, nil
}
```

- [ ] **Step 9: Write repository tests**

```go
package persistence

import (
    "context"
    "testing"
    "time"
    
    "github.com/denkhaus/gollum/pkg/session/persistence/ent/enttest"
    "github.com/denkhaus/gollum/pkg/shared"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestEntRepository_Create(t *testing.T) {
    ctx := context.Background()
    client := enttest.Open(t, ent.SQLite, "file:ent?mode=memory&cache=shared")
    defer client.Close()
    
    repo := NewEntRepositoryFromClient(client, nil)
    
    session := &shared.Session{
        ID:        uuid.New(),
        ChannelID: uuid.New(),
        Cwd:       "/test",
    }
    
    err := repo.Create(ctx, session)
    assert.NoError(t, err)
    
    // Verify session was created
    exists, err := repo.Exists(ctx, session.ID)
    assert.NoError(t, err)
    assert.True(t, exists)
}

func TestEntRepository_Get_NotFound(t *testing.T) {
    ctx := context.Background()
    client := enttest.Open(t, ent.SQLite, "file:ent?mode=memory&cache=shared")
    defer client.Close()
    
    repo := NewEntRepositoryFromClient(client, nil)
    
    _, err := repo.Get(ctx, uuid.New())
    assert.Error(t, err)
}

func TestEntRepository_Fork(t *testing.T) {
    ctx := context.Background()
    client := enttest.Open(t, ent.SQLite, "file:ent?mode=memory&cache=shared")
    defer client.Close()
    
    repo := NewEntRepositoryFromClient(client, nil)
    
    // Create source session
    source := &shared.Session{
        ID:        uuid.New(),
        ChannelID: uuid.New(),
        Cwd:       "/test",
    }
    require.NoError(t, repo.Create(ctx, source))
    
    // Fork session
    forked, err := repo.Fork(ctx, source.ID)
    assert.NoError(t, err)
    assert.NotEqual(t, source.ID, forked.ID)
    assert.Equal(t, source.Cwd, forked.Cwd)
}
```

- [ ] **Step 10: Run repository tests**

```bash
cd pkg/session/persistence
go test -v ./...
```

Expected: All tests pass

- [ ] **Step 11: Commit EntRepository implementation**

```bash
git add pkg/session/persistence/
git commit -m "feat(persistence): implement EntRepository with CRUD operations"
```

---

## Chunk 2: SessionManager Integration

### Task 4: Extend SessionManager interface

**Files:**
- Modify: `pkg/session/manager.go`

- [ ] **Step 1: Add persistence methods to SessionManager interface**

```go
type SessionManager interface {
    // ... existing methods
    
    // LoadSession loads a session from persistence.
    LoadSession(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error)
    
    // ListSessions lists all sessions.
    ListSessions(ctx context.Context) ([]*shared.Session, error)
    
    // ResumeSession resumes a closed session.
    ResumeSession(ctx context.Context, sessionID uuid.UUID) error
    
    // ForkSession creates a copy of a session.
    ForkSession(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error)
}
```

- [ ] **Step 2: Update sessionManagerImpl struct**

```go
type sessionManagerImpl struct {
    sessions sync.Map
    repo     repository.SessionRepository  // New field
}
```

- [ ] **Step 3: Update NewSessionManager constructor**

```go
func NewSessionManager(injector do.Injector) (SessionManager, error) {
    repo, err := do.Invoke[repository.SessionRepository](injector)
    if err != nil {
        return nil, err
    }
    
    return &sessionManagerImpl{
        repo: repo,
    }, nil
}
```

- [ ] **Step 4: Implement LoadSession method**

```go
func (p *sessionManagerImpl) LoadSession(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error) {
    // Check if already in memory
    if val, ok := p.sessions.Load(sessionID.String()); ok {
        return val.(*shared.Session), nil
    }
    
    // Load from repository
    session, err := p.repo.Get(ctx, sessionID)
    if err != nil {
        return nil, err
    }
    
    // Add to in-memory cache
    p.sessions.Store(sessionID.String(), session)
    
    return session, nil
}
```

- [ ] **Step 5: Implement ListSessions method**

```go
func (p *sessionManagerImpl) ListSessions(ctx context.Context) ([]*shared.Session, error) {
    return p.repo.List(ctx, nil)
}
```

- [ ] **Step 6: Implement ResumeSession method**

```go
func (p *sessionManagerImpl) ResumeSession(ctx context.Context, sessionID uuid.UUID) error {
    // Load session (from repo or cache)
    session, err := p.LoadSession(ctx, sessionID)
    if err != nil {
        return err
    }
    
    // Check if closed
    if session.CancelFunc == nil {
        return repository.ErrSessionClosed
    }
    
    // Create new context
    ctx, cancel := context.WithCancel(context.Background())
    session.Context = ctx
    session.CancelFunc = cancel
    
    // Update in repository
    return p.repo.Update(ctx, session)
}
```

- [ ] **Step 7: Implement ForkSession method**

```go
func (p *sessionManagerImpl) ForkSession(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error) {
    // Fork in repository
    newSession, err := p.repo.Fork(ctx, sessionID)
    if err != nil {
        return nil, err
    }
    
    // Add to in-memory cache
    p.sessions.Store(newSession.ID.String(), newSession)
    
    return newSession, nil
}
```

- [ ] **Step 8: Update CreateSession to persist sessions**

```go
func (p *sessionManagerImpl) CreateSession(ctx *shared.SessionContext) (*shared.Session, error) {
    sessionCtx, cancel := context.WithCancel(context.Background())
    session := &shared.Session{
        ID:         ctx.SessionID,
        ChannelID:  ctx.ChannelID,
        Context:    sessionCtx,
        CancelFunc: cancel,
        CreatedAt:  time.Now(),
        Cwd:        ctx.Cwd,
    }
    
    // Persist to repository
    if err := p.repo.Create(context.Background(), session); err != nil {
        cancel()
        return nil, fmt.Errorf("failed to persist session: %w", err)
    }
    
    p.sessions.Store(ctx.SessionID.String(), session)
    return session, nil
}
```

- [ ] **Step 9: Write SessionManager integration tests**

```go
func TestSessionManager_LoadSession(t *testing.T) {
    ctx := context.Background()
    
    mockRepo := testutil.NewMockRepository()
    manager := NewSessionManagerWithRepo(mockRepo)
    
    testSession := &shared.Session{
        ID:        uuid.New(),
        ChannelID: uuid.New(),
        Cwd:       "/test",
    }
    
    mockRepo.EXPECT().Get(ctx, testSession.ID).Return(testSession, nil)
    
    loaded, err := manager.LoadSession(ctx, testSession.ID)
    assert.NoError(t, err)
    assert.Equal(t, testSession.ID, loaded.ID)
}
```

- [ ] **Step 10: Run SessionManager tests**

```bash
cd pkg/session
go test -v -run TestSessionManager
```

Expected: All tests pass

- [ ] **Step 11: Commit SessionManager integration**

```bash
git add pkg/session/
git commit -m "feat(session): integrate repository into SessionManager"
```

---

## Chunk 3: ACP Methods

### Task 5: Implement ACP session/list

**Files:**
- Modify: `pkg/acp/service.go`

- [ ] **Step 1: Add ListSessions method to acpServiceImpl**

```go
// ListSessions implements ACP session/list protocol method.
func (s *acpServiceImpl) ListSessions(ctx context.Context) ([]*shared.Session, error) {
    return s.sessionManager.ListSessions(ctx)
}
```

- [ ] **Step 2: Update ACPService interface**

```go
type ACPService interface {
    // ... existing methods
    // ACP session management methods
    ListSessions(ctx context.Context) ([]*shared.Session, error)
    LoadSession(ctx context.Context, sessionID acppkg.SessionID) (*shared.ACPSession, error)
    ResumeSession(ctx context.Context, sessionID acppkg.SessionID) error
    CloseSession(ctx context.Context, sessionID acppkg.SessionID) error
    ForkSession(ctx context.Context, sessionID acppkg.SessionID) (acppkg.SessionID, error)
}
```

- [ ] **Step 3: Write test for ListSessions**

```go
func TestACPService_ListSessions(t *testing.T) {
    ctx := context.Background()
    
    service := setupTestService(t)
    mockMgr := testutil.NewMockSessionManager()
    service.sessionManager = mockMgr
    
    testSessions := []*shared.Session{
        {ID: uuid.New(), ChannelID: uuid.New()},
        {ID: uuid.New(), ChannelID: uuid.New()},
    }
    
    mockMgr.EXPECT().ListSessions(ctx).Return(testSessions, nil)
    
    sessions, err := service.ListSessions(ctx)
    assert.NoError(t, err)
    assert.Len(t, sessions, 2)
}
```

- [ ] **Step 4: Run test**

```bash
cd pkg/acp
go test -v -run TestACPService_ListSessions
```

Expected: Test passes

- [ ] **Step 5: Commit**

```bash
git add pkg/acp/service.go pkg/acp/service_test.go
git commit -m "feat(acp): add session/list implementation"
```

### Task 6: Implement ACP session/load

**Files:**
- Modify: `pkg/acp/service.go`

- [ ] **Step 1: Implement LoadSession method**

```go
// LoadSession loads an existing session (ACP: session/load).
func (s *acpServiceImpl) LoadSession(ctx context.Context, sessionID acppkg.SessionID) (*shared.ACPSession, error) {
    uuidSessionID, err := uuid.Parse(string(sessionID))
    if err != nil {
        return nil, fmt.Errorf("invalid session ID: %w", err)
    }
    
    session, err := s.sessionManager.LoadSession(ctx, uuidSessionID)
    if err != nil {
        return nil, err
    }
    
    return &shared.ACPSession{
        Context:    session.Context,
        CancelFunc: session.CancelFunc,
        SessionID:  session.ID,
        Cwd:        session.Cwd,
    }, nil
}
```

- [ ] **Step 2: Write test**

```go
func TestACPService_LoadSession(t *testing.T) {
    ctx := context.Background()
    
    service := setupTestService(t)
    mockMgr := testutil.NewMockSessionManager()
    service.sessionManager = mockMgr
    
    testID := uuid.New()
    testSession := &shared.Session{
        ID:  testID,
        Cwd: "/test",
    }
    
    mockMgr.EXPECT().LoadSession(ctx, testID).Return(testSession, nil)
    
    acpSessionID := acppkg.SessionID(testID.String())
    loaded, err := service.LoadSession(ctx, acpSessionID)
    assert.NoError(t, err)
    assert.Equal(t, testID, loaded.SessionID)
}
```

- [ ] **Step 3: Run test**

```bash
cd pkg/acp
go test -v -run TestACPService_LoadSession
```

Expected: Test passes

- [ ] **Step 4: Commit**

```bash
git add pkg/acp/service.go pkg/acp/service_test.go
git commit -m "feat(acp): add session/load implementation"
```

### Task 7: Implement ACP session/resume

**Files:**
- Modify: `pkg/acp/service.go`

- [ ] **Step 1: Implement ResumeSession method**

```go
// ResumeSession resumes a closed session (ACP: session/resume).
func (s *acpServiceImpl) ResumeSession(ctx context.Context, sessionID acppkg.SessionID) error {
    uuidSessionID, err := uuid.Parse(string(sessionID))
    if err != nil {
        return fmt.Errorf("invalid session ID: %w", err)
    }
    
    return s.sessionManager.ResumeSession(ctx, uuidSessionID)
}
```

- [ ] **Step 2: Write test**

```go
func TestACPService_ResumeSession(t *testing.T) {
    ctx := context.Background()
    
    service := setupTestService(t)
    mockMgr := testutil.NewMockSessionManager()
    service.sessionManager = mockMgr
    
    testID := uuid.New()
    mockMgr.EXPECT().ResumeSession(ctx, testID).Return(nil)
    
    acpSessionID := acppkg.SessionID(testID.String())
    err := service.ResumeSession(ctx, acpSessionID)
    assert.NoError(t, err)
}
```

- [ ] **Step 3: Run test**

```bash
cd pkg/acp
go test -v -run TestACPService_ResumeSession
```

Expected: Test passes

- [ ] **Step 4: Commit**

```bash
git add pkg/acp/service.go pkg/acp/service_test.go
git commit -m "feat(acp): add session/resume implementation"
```

### Task 8: Implement ACP session/close

**Files:**
- Modify: `pkg/acp/service.go`

- [ ] **Step 1: Implement CloseSession method**

```go
// CloseSession closes a session (ACP: session/close).
func (s *acpServiceImpl) CloseSession(ctx context.Context, sessionID acppkg.SessionID) error {
    uuidSessionID, err := uuid.Parse(string(sessionID))
    if err != nil {
        return fmt.Errorf("invalid session ID: %w", err)
    }
    
    return s.sessionManager.CloseSession(uuidSessionID)
}
```

- [ ] **Step 2: Write test**

```go
func TestACPService_CloseSession(t *testing.T) {
    ctx := context.Background()
    
    service := setupTestService(t)
    mockMgr := testutil.NewMockSessionManager()
    service.sessionManager = mockMgr
    
    testID := uuid.New()
    mockMgr.EXPECT().CloseSession(testID).Return(nil)
    
    acpSessionID := acppkg.SessionID(testID.String())
    err := service.CloseSession(ctx, acpSessionID)
    assert.NoError(t, err)
}
```

- [ ] **Step 3: Run test**

```bash
cd pkg/acp
go test -v -run TestACPService_CloseSession
```

Expected: Test passes

- [ ] **Step 4: Commit**

```bash
git add pkg/acp/service.go pkg/acp/service_test.go
git commit -m "feat(acp): add session/close implementation"
```

### Task 9: Implement ACP session/fork

**Files:**
- Modify: `pkg/acp/service.go`

- [ ] **Step 1: Implement ForkSession method**

```go
// ForkSession copies a session (ACP: session/fork).
func (s *acpServiceImpl) ForkSession(ctx context.Context, sessionID acppkg.SessionID) (acppkg.SessionID, error) {
    uuidSessionID, err := uuid.Parse(string(sessionID))
    if err != nil {
        return "", fmt.Errorf("invalid session ID: %w", err)
    }
    
    newSession, err := s.sessionManager.ForkSession(ctx, uuidSessionID)
    if err != nil {
        return "", err
    }
    
    return acppkg.SessionID(newSession.ID.String()), nil
}
```

- [ ] **Step 2: Write test**

```go
func TestACPService_ForkSession(t *testing.T) {
    ctx := context.Background()
    
    service := setupTestService(t)
    mockMgr := testutil.NewMockSessionManager()
    service.sessionManager = mockMgr
    
    testID := uuid.New()
    newID := uuid.New()
    newSession := &shared.Session{ID: newID, Cwd: "/test"}
    
    mockMgr.EXPECT().ForkSession(ctx, testID).Return(newSession, nil)
    
    acpSessionID := acppkg.SessionID(testID.String())
    forkedID, err := service.ForkSession(ctx, acpSessionID)
    assert.NoError(t, err)
    assert.Equal(t, newID.String(), string(forkedID))
}
```

- [ ] **Step 3: Run test**

```bash
cd pkg/acp
go test -v -run TestACPService_ForkSession
```

Expected: Test passes

- [ ] **Step 4: Commit**

```bash
git add pkg/acp/service.go pkg/acp/service_test.go
git commit -m "feat(acp): add session/fork implementation"
```

---

## Chunk 4: Configuration & DI

### Task 10: Add database configuration

**Files:**
- Create: `pkg/config/database.go`
- Modify: `pkg/config/service.go`

- [ ] **Step 1: Create database config types**

```go
package config

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
    Driver string // "sqlite", "postgres", "mysql"
    DSN    string // Database connection string
}

// GetDatabaseConfig returns the database configuration.
func (c *ConfigService) GetDatabaseConfig() DatabaseConfig {
    return DatabaseConfig{
        Driver: c.viper.GetString("database.driver"),
        DSN:    c.viper.GetString("database.dsn"),
    }
}
```

- [ ] **Step 2: Add default database configuration**

```go
func (c *ConfigService) initDefaults() {
    // ... existing defaults
    
    c.viper.SetDefault("database.driver", "sqlite")
    c.viper.SetDefault("database.dsn", "gollum.db")
}
```

- [ ] **Step 3: Write tests**

```go
func TestConfigService_GetDatabaseConfig(t *testing.T) {
    cfg := NewTestConfig(t)
    
    dbConfig := cfg.GetDatabaseConfig()
    assert.Equal(t, "sqlite", dbConfig.Driver)
    assert.Equal(t, "gollum.db", dbConfig.DSN)
}
```

- [ ] **Step 4: Run tests**

```bash
cd pkg/config
go test -v -run TestConfigService_GetDatabaseConfig
```

Expected: Test passes

- [ ] **Step 5: Commit**

```bash
git add pkg/config/
git commit -m "feat(config): add database configuration"
```

### Task 11: Create repository provider

**Files:**
- Create: `pkg/session/persistence/provider.go`

- [ ] **Step 1: Create DI provider**

```go
package persistence

import (
    "github.com/denkhaus/gollum/pkg/config"
    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/samber/do/v2"
)

// NewSessionRepository creates a SessionRepository with DI.
func NewSessionRepository(injector do.Injector) (repository.SessionRepository, error) {
    cfg := do.MustInvoke[config.ConfigService](injector)
    log := do.MustInvoke[logger.LoggerService](injector)
    
    dbConfig := cfg.GetDatabaseConfig()
    
    repo, err := NewEntRepository(dbConfig.Driver, dbConfig.DSN, log)
    if err != nil {
        return nil, err
    }
    
    log.Info("Session repository initialized",
        "driver", dbConfig.Driver,
        "dsn", dbConfig.DSN,
    )
    
    return repo, nil
}
```

- [ ] **Step 2: Write provider test**

```go
func TestNewSessionRepository(t *testing.T) {
    injector := do.New()
    do.Provide(injector, testutil.NewMockLogger)
    do.Provide(injector, testutil.NewMockConfig)
    
    repo, err := NewSessionRepository(injector)
    assert.NoError(t, err)
    assert.NotNil(t, repo)
}
```

- [ ] **Step 3: Run test**

```bash
cd pkg/session/persistence
go test -v -run TestNewSessionRepository
```

Expected: Test passes

- [ ] **Step 4: Commit**

```bash
git add pkg/session/persistence/provider.go
git commit -m "feat(persistence): add SessionRepository DI provider"
```

### Task 12: Update SessionManager provider

**Files:**
- Modify: `pkg/session/manager.go`

- [ ] **Step 1: Update NewSessionManager to use repository**

```go
func NewSessionManager(injector do.Injector) (SessionManager, error) {
    repo, err := do.Invoke[repository.SessionRepository](injector)
    if err != nil {
        // Fallback to in-memory only if repository not available
        repo = nil
    }
    
    return &sessionManagerImpl{
        repo: repo,
    }, nil
}
```

- [ ] **Step 2: Update CreateSession to handle nil repo**

```go
func (p *sessionManagerImpl) CreateSession(ctx *shared.SessionContext) (*shared.Session, error) {
    sessionCtx, cancel := context.WithCancel(context.Background())
    session := &shared.Session{
        ID:         ctx.SessionID,
        ChannelID:  ctx.ChannelID,
        Context:    sessionCtx,
        CancelFunc: cancel,
        CreatedAt:  time.Now(),
        Cwd:        ctx.Cwd,
    }
    
    // Persist to repository if available
    if p.repo != nil {
        if err := p.repo.Create(context.Background(), session); err != nil {
            p.logger.Warn("session persistence failed, using in-memory only",
                "error", err,
                "session_id", ctx.SessionID,
            )
        }
    }
    
    p.sessions.Store(ctx.SessionID.String(), session)
    return session, nil
}
```

- [ ] **Step 3: Write test**

```go
func TestSessionManager_WithRepository(t *testing.T) {
    mockRepo := testutil.NewMockRepository()
    manager := NewSessionManagerWithRepo(mockRepo)
    
    sessionCtx := &shared.SessionContext{
        SessionID: uuid.New(),
        ChannelID: uuid.New(),
    }
    
    session, err := manager.CreateSession(sessionCtx)
    assert.NoError(t, err)
    assert.NotNil(t, session)
}
```

- [ ] **Step 4: Run test**

```bash
cd pkg/session
go test -v -run TestSessionManager_WithRepository
```

Expected: Test passes

- [ ] **Step 5: Commit**

```bash
git add pkg/session/manager.go
git commit -m "feat(session): integrate repository into SessionManager provider"
```

---

## Chunk 5: Final Integration & Testing

### Task 13: Register providers in DI

**Files:**
- Modify: `pkg/app/injector.go` or `pkg/app/service.go`

- [ ] **Step 1: Add repository provider to DI registry**

```go
func NewInjector() (do.Injector, error) {
    injector := do.New()
    
    // ... existing providers
    
    // Session persistence providers
    do.Provide(injector, persistence.NewSessionRepository)
    
    return injector, nil
}
```

- [ ] **Step 2: Run application to test DI integration**

```bash
just build
./gollum --help
```

Expected: Application starts successfully

- [ ] **Step 3: Commit**

```bash
git add pkg/app/
git commit -m "feat(app): register SessionRepository provider in DI"
```

### Task 14: Write E2E tests

**Files:**
- Create: `pkg/acp/e2e_session_test.go`

- [ ] **Step 1: Create E2E test for session lifecycle**

```go
func TestACP_SessionLifecycle_E2E(t *testing.T) {
    ctx := context.Background()
    
    service := setupE2EService(t)
    
    // 1. Initialize
    initResp, err := service.Initialize(ctx, &acppkg.InitializeRequest{})
    assert.NoError(t, err)
    
    // 2. New Session
    newResp, err := service.CreateSession(ctx, "/home/user/project")
    assert.NoError(t, err)
    sessionID := newResp.SessionID
    
    // 3. List Sessions
    sessions, err := service.ListSessions(ctx)
    assert.NoError(t, err)
    assert.Len(t, sessions, 1)
    
    // 4. Fork Session
    forkedID, err := service.ForkSession(ctx, sessionID)
    assert.NoError(t, err)
    assert.NotEmpty(t, forkedID)
    
    // 5. Close Session
    err = service.CloseSession(ctx, sessionID)
    assert.NoError(t, err)
    
    // 6. Load Session
    loaded, err := service.LoadSession(ctx, sessionID)
    assert.NoError(t, err)
    assert.NotNil(t, loaded)
    
    // 7. Resume Session
    err = service.ResumeSession(ctx, sessionID)
    assert.NoError(t, err)
}
```

- [ ] **Step 2: Run E2E test**

```bash
cd pkg/acp
go test -v -run TestACP_SessionLifecycle_E2E
```

Expected: All lifecycle steps pass

- [ ] **Step 3: Commit**

```bash
git add pkg/acp/e2e_session_test.go
git commit -m "test(acp): add E2E test for session lifecycle"
```

### Task 15: Add documentation

**Files:**
- Create: `pkg/session/persistence/README.md`

- [ ] **Step 1: Create persistence README**

```markdown
# Session Persistence

This package provides database persistence for Gollum sessions using Ent ORM.

## Supported Databases

- SQLite (default)
- PostgreSQL
- MySQL

## Configuration

```yaml
database:
  driver: sqlite
  dsn: gollum.db
```

## Schema

The schema consists of three main entities:

- **Session**: Core session data (ID, Channel, Agent, Cwd, State)
- **Message**: Message history for each session
- **SupervisorConfig**: LLM configuration for sessions

## Usage

Sessions are automatically persisted when created via SessionManager. Use the repository interface directly for custom queries.
```

- [ ] **Step 2: Update main README**

```markdown
## Session Management

Gollum supports persistent session management across all channels. Sessions can be:

- Created and resumed across restarts
- Listed and queried
- Forked to create variations
- Closed and archived

See [pkg/session/persistence/README.md](../session/persistence/) for details.
```

- [ ] **Step 3: Commit**

```bash
git add pkg/session/persistence/README.md README.md
git commit -m "docs: add session persistence documentation"
```

---

## Completion Checklist

- [ ] All tasks completed
- [ ] All tests pass
- [ ] Documentation updated
- [ ] Code committed with clear messages
- [ ] Build succeeds
- [ ] E2E tests pass

**Estimated Total Time:** 3-5 days for complete implementation
