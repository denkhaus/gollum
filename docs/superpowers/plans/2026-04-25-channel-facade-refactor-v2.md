# Channel Facade Refactor - Pragmatic SOC

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the channel facade to improve separation of concerns by extracting business logic (session/supervisor management) while keeping infrastructure concerns (registry, routing) in the facade.

**Architecture:** Extract only `InputHandler` for session/supervisor logic. Keep provider discovery, channel registry, and message routing in `ChannelFacade` where they belong. No unnecessary indirection.

**Tech Stack:** Go 1.23+, samber/do/v2 (DI), gollem SDK, existing channel infrastructure

---

## Context & Problem Statement

**Current Issues:**
1. `ChannelFacade.SubmitInput()` embeds session/supervisor creation (business logic in infrastructure)
2. `DisplayMessage()` documentation unclear (name is correct, just needs docs)
3. Log routing behavior undocumented (ACP broadcasts, TUI single-session)

**What's NOT a problem:**
- Provider discovery in facade ✅ (infrastructure concern)
- Channel registry in facade ✅ (infrastructure concern)
- Message routing in facade ✅ (facade pattern = coordination)

**Why This Matters:**
- Business logic in facade → hard to test, hard to swap implementations
- Undocumented behavior → developer confusion, inconsistent implementations
- Extract InputHandler → can mock for testing, swap for different strategies

**Files Involved:**
- `pkg/channel/interface.go` - Interface definitions
- `pkg/channel/facade.go` - Current implementation
- `pkg/channel/handler.go` - NEW: InputHandler
- `pkg/channel/handler_test.go` - NEW: Tests
- `pkg/channel/log_routing.go` - NEW: Documentation
- `pkg/channel/facade_test.go` - Update tests
- `pkg/channel/facade_mock.go` - Regenerate mock

---

## File Structure After Refactor

```
pkg/channel/
├── interface.go              # ChannelFacade (updated), Channel, etc.
├── handler.go                # NEW: InputHandler interface + impl
├── handler_test.go           # NEW: Tests for InputHandler
├── facade.go                 # Updated: delegates SubmitInput to InputHandler
├── facade_test.go            # Updated: Uses InputHandler in tests
├── log_routing.go            # NEW: Log routing documentation
├── types.go                  # Existing types (unchanged)
└── facade_mock.go            # Regenerated mock
```

**What we DON'T need:**
- ~~MessageRouter~~ - Message routing belongs in facade
- ~~ChannelRegistry~~ - Provider discovery belongs in facade
- ~~facadeAdapter~~ - Unnecessary indirection layer

---

## Chunk 1: Extract InputHandler (The Key Win)

This extracts the business logic (session/supervisor management) from the facade.

### Task 1: Create InputHandler interface and implementation

**Files:**
- Create: `pkg/channel/handler.go`
- Create: `pkg/channel/handler_test.go`
- Modify: `pkg/channel/facade.go:171-197` (delegate to InputHandler)

- [ ] **Step 1: Write failing test for InputHandler**

```go
// pkg/channel/handler_test.go
package channel

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/command"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputHandler_HandleInput_SlashCommand(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	sessionID := "test-session"

	// Setup mocks
	cmdMgr := command.NewMockCommandManager(t)
	cmdMgr.EXPECT().Execute(ctx, sessionID, "/help").Return(true, "Help text", nil)

	sm := session.NewMockSessionManager(t)
	reg := registry.NewMockAgentRegistry(t)
	af := shared.NewMockAgentFactory(t)
	logger := NewMockLoggerService(t)

	handler := NewInputHandler(cmdMgr, sm, reg, af, logger)

	result, err := handler.HandleInput(ctx, channelID, sessionID, "/help")

	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.True(t, result.IsCommand)
	assert.Equal(t, "Help text", result.Response)
}

func TestInputHandler_HandleInput_NonCommand(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	sessionID := "test-session"

	// Setup mocks
	cmdMgr := command.NewMockCommandManager(t)
	cmdMgr.EXPECT().Execute(ctx, sessionID, "hello").Return(false, "", nil)

	sm := session.NewMockSessionManager(t)
	mockSession := session.NewMockManagedSession(t)
	sm.EXPECT().GetOrCreateSession(sessionID, channelID).Return(mockSession, nil)

	// Supervisor setup
	af := shared.NewMockAgentFactory(t)
	mockSupervisor := shared.NewMockSupervisorAgent(t)
	mockSession.EXPECT().GetOrCreateSupervisor(af).Return(mockSupervisor, nil)
	mockSession.EXPECT().Context().Return(ctx)

	// Execute response
	mockSupervisor.EXPECT().Execute(ctx, mock.MatchedBy(func(input interface{}) bool {
		return true
	})).Return(&gollem.Response{Texts: []string{"Hello back"}}, nil)

	logger := NewMockLoggerService(t)

	handler := NewInputHandler(cmdMgr, sm, reg, af, logger)

	result, err := handler.HandleInput(ctx, channelID, sessionID, "hello")

	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.False(t, result.IsCommand)
	assert.Equal(t, "Hello back", result.Response)
}

func TestInputHandler_CancelInput(t *testing.T) {
	sessionID := "test-session"

	sm := session.NewMockSessionManager(t)
	mockSession := session.NewMockManagedSession(t)
	sm.EXPECT().GetSession(sessionID).Return(mockSession, true)
	mockSession.EXPECT().CancelFunc()

	handler := NewInputHandler(nil, sm, nil, nil, nil)

	err := handler.CancelInput(sessionID)
	require.NoError(t, err)
}
```

Run: `go test ./pkg/channel -run TestInputHandler -v`
Expected: FAIL with "type InputHandler has no field HandleInput" and "undefined: NewInputHandler"

- [ ] **Step 2: Create InputHandler interface and implementation**

```go
// pkg/channel/handler.go
package channel

import (
	"context"
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/command"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// InputHandler handles user input by managing sessions, supervisors, and command execution.
// This interface encapsulates the business logic for processing user input.
//
// Why separate from ChannelFacade?
// - Session/supervisor management is business logic, not infrastructure
// - Enables testing without mocking the entire facade
// - Allows swapping input handling strategies (e.g., different session models)
type InputHandler interface {
	// HandleInput processes user input and returns the result.
	// channelID assigns agent messages to a channel.
	// sessionID identifies the session for this interaction.
	HandleInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (*InputResult, error)

	// CancelInput cancels an in-flight input for the given session.
	CancelInput(sessionID string) error
}

// inputHandlerImpl implements InputHandler
type inputHandlerImpl struct {
	commandManager command.Manager
	sessionManager session.SessionManager
	registry       registry.AgentRegistry
	agentFactory   shared.AgentFactory
	logger         logger.LoggerService
}

// NewInputHandler creates a new InputHandler instance
func NewInputHandler(
	cm command.Manager,
	sm session.SessionManager,
	reg registry.AgentRegistry,
	af shared.AgentFactory,
	log logger.LoggerService,
) InputHandler {
	return &inputHandlerImpl{
		commandManager: cm,
		sessionManager: sm,
		registry:       reg,
		agentFactory:   af,
		logger:         log,
	}
}

// HandleInput processes user input by checking commands, managing sessions, and executing supervisors
func (h *inputHandlerImpl) HandleInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (*InputResult, error) {
	// First check if it's a slash command
	handled, response, err := h.commandManager.Execute(ctx, sessionID, input)
	if handled {
		return &InputResult{
			Handled:   true,
			IsCommand: true,
			Response:  response,
			Error:     err,
		}, nil
	}

	// Get or create session for this interaction
	sess, err := h.sessionManager.GetOrCreateSession(sessionID, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create session: %w", err)
	}

	// Get or create supervisor for this session (lazy, thread-safe)
	supervisor, err := sess.GetOrCreateSupervisor(h.agentFactory)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create supervisor: %w", err)
	}

	// Execute supervisor agent with session context
	resp, err := supervisor.Execute(sess.Context, gollem.Text(input))
	if err != nil {
		return &InputResult{
			Handled: true,
			Error:   err,
		}, nil
	}

	// Extract response
	var content string
	if resp != nil && len(resp.Texts) > 0 {
		content = strings.Join(resp.Texts, "\n")
	}

	return &InputResult{
		Handled:  true,
		Response: content,
	}, nil
}

// CancelInput cancels an in-flight input for the given session
func (h *inputHandlerImpl) CancelInput(sessionID string) error {
	session, ok := h.sessionManager.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Cancel the session context
	session.CancelFunc()

	return nil
}

// Compile-time check
var _ InputHandler = (*inputHandlerImpl)(nil)
```

Run: `go test ./pkg/channel -run TestInputHandler -v`
Expected: PASS

- [ ] **Step 3: Update ChannelFacade to use InputHandler**

```go
// pkg/channel/facade.go (modify channelFacadeImpl struct)

type channelFacadeImpl struct {
	mu             sync.RWMutex
	channels       map[uuid.UUID]Channel
	inputHandler   InputHandler           // NEW: Delegate business logic
	commandManager command.Manager        // Keep for compatibility
	registry       registry.AgentRegistry
	agentFactory   shared.AgentFactory
	sessionManager session.SessionManager
	logger         logger.LoggerService
	providers      map[ChannelIdentifier]ChannelFactory
}

// Update NewChannelFacade to create InputHandler
func NewChannelFacade(injector do.Injector) (ChannelFacade, error) {
	cm := do.MustInvoke[command.ManagerService](injector)
	reg := do.MustInvoke[registry.AgentRegistry](injector)
	af := do.MustInvoke[shared.AgentFactory](injector)
	sm := do.MustInvoke[session.SessionManager](injector)
	log := do.MustInvoke[logger.LoggerService](injector)

	// Create InputHandler for business logic
	inputHandler := NewInputHandler(cm, sm, reg, af, log)

	return &channelFacadeImpl{
		commandManager: cm,
		registry:       reg,
		agentFactory:   af,
		sessionManager: sm,
		channels:       make(map[uuid.UUID]Channel),
		logger:         log,
		inputHandler:   inputHandler, // NEW
	}, nil
}

// Update SubmitInput to delegate to InputHandler
func (p *channelFacadeImpl) SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (*InputResult, error) {
	return p.inputHandler.HandleInput(ctx, channelID, sessionID, input)
}

// Update CancelInput to delegate to InputHandler
func (p *channelFacadeImpl) CancelInput(sessionID string) error {
	return p.inputHandler.CancelInput(sessionID)
}
```

Run: `go test ./pkg/channel -v`
Expected: PASS (existing tests still work)

- [ ] **Step 4: Update interface documentation**

```go
// pkg/channel/interface.go (update ChannelFacade documentation)

// ChannelFacade is the central coordinator for all channels.
// It implements shared.LogForwarder for routing logs to channels.
//
// RESPONSIBILITIES:
// - Channel registry: RegisterChannel, UnregisterChannel
// - Provider discovery: DiscoverProviders, CreateChannel
// - Message routing: DisplayMessage, NotifyAgentLifecycle, ForwardLog
// - Input handling: SubmitInput, CancelInput (delegates to InputHandler)
//
// DESIGN NOTE: SubmitInput/CancelInput delegate to InputHandler which encapsulates
// the business logic of session and supervisor management. This separation enables
// easier testing and allows swapping input handling strategies.
type ChannelFacade interface {
	shared.LogForwarder

	// DisplayMessage sends a message to the channel specified in msg.ChannelID.
	// The name "Display" reflects the middleware's intent: showing agent output to the user.
	// This is NOT a broadcast - it routes to exactly one channel.
	DisplayMessage(msg Message)

	// SubmitInput handles user input from any channel.
	// Delegates to InputHandler which manages sessions, supervisors, and command execution.
	SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (*InputResult, error)

	// ... rest of interface unchanged
}
```

- [ ] **Step 5: Commit InputHandler extraction**

```bash
git add pkg/channel/handler.go pkg/channel/handler_test.go pkg/channel/facade.go pkg/channel/interface.go
git commit -m "refactor(channel): extract InputHandler for session/supervisor business logic

Extracts session and supervisor management from ChannelFacade into a focused
InputHandler interface. This separates business logic from infrastructure concerns.

Benefits:
- Cleaner separation of concerns
- Easier to test (mock InputHandler vs entire facade)
- Enables swapping input handling strategies

ChannelFacade now delegates SubmitInput/CancelInput to InputHandler while
retaining responsibility for provider discovery, channel registry, and message routing."
```

---

## Chunk 2: Add comprehensive documentation

This chunk adds documentation for log routing and clarifies DisplayMessage.

### Task 2: Create log routing documentation

**Files:**
- Create: `pkg/channel/log_routing.go`

- [ ] **Step 1: Create log routing guide**

```go
// Package channel provides the channel abstraction layer for Gollum.
//
// LOG ROUTING BEHAVIOR:
//
// Channels receive log entries via the OnLog(entry) method. The routing
// behavior depends on the channel type and use case:
//
// SINGLE-SESSION CHANNELS (e.g., TUI):
// - Receive all logs for display in the UI
// - Logs are not session-scoped
// - Example: TUI shows all system logs in a dedicated panel
//
// MULTI-SESSION CHANNELS (e.g., ACP):
// - MUST route logs to specific sessions when entry.SessionID is set
// - MAY broadcast system-wide logs (no session ID) to all active sessions
// - Example: ACP forwards session-specific logs to that session only,
//            but broadcasts agent lifecycle events to all sessions
//
// LOG ENTRY ROUTING:
// - entry.ChannelID determines which channel receives the log
// - entry.SessionID provides session-specific routing (optional)
// - Channels MUST handle logs even if no session is active
//
// IMPLEMENTATION NOTES:
// - Use non-blocking sends to prevent log system deadlocks
// - Filter/handle logs appropriately for the channel type
// - Document channel-specific log behavior in channel documentation
package channel

// The content above was previously in a separate file. Now integrated with package docs.
```

- [ ] **Step 2: Update Channel interface documentation**

```go
// pkg/channel/interface.go (update Channel interface documentation)

// Channel is the interface that all channel implementations must satisfy.
//
// Channels receive data FROM the system via three methods:
// - OnMessage(msg): Agent responses, tool executions, thinking blocks
// - OnLog(entry): Log entries from the system (see package docs for routing behavior)
// - OnAgentLifecycle(event): Agent registration/removal notifications
//
// Channels send data TO the system via the ChannelFacade:
// - SubmitInput(): User input from the channel
//
// IMPLEMENTATION GUIDELINES:
// 1. Use non-blocking sends for channels to prevent deadlocks
// 2. Handle all message types (MessageType* constants)
// 3. Document session handling behavior (single vs multi-session)
// 4. Implement Start() to block until channel is complete
// 5. See package documentation for log routing behavior
type Channel interface {
	// ... methods unchanged
}
```

- [ ] **Step 3: Update ACP OnLog documentation**

```go
// pkg/acp/service.go (update OnLog documentation)

// OnLog receives log entries from the agent system and forwards them to ACP sessions.
//
// Log routing behavior:
// - Session-specific logs (entry.SessionID set) → routed to that session only
// - System-wide logs (no session ID) → broadcast to all active sessions
//
// See pkg/channel package documentation for general log routing patterns.
func (s *acpServiceImpl) OnLog(entry shared.LogEntry) {
	// ... existing implementation
}
```

- [ ] **Step 4: Update TUI OnLog documentation**

```go
// pkg/tui/channel.go (update OnLog documentation)

// OnLog handles log entries from the channel system.
//
// Log routing behavior for TUI (single-session channel):
// - All logs are displayed in the TUI UI
// - No session routing (TUI uses empty session ID)
//
// See pkg/channel package documentation for general log routing patterns.
func (c *TUIChannel) OnLog(entry shared.LogEntry) {
	// ... existing implementation
}
```

- [ ] **Step 5: Commit documentation**

```bash
git add pkg/channel/log_routing.go pkg/channel/interface.go pkg/acp/service.go pkg/tui/channel.go
git commit -m "docs(channel): add comprehensive log routing documentation

Documents log routing behavior for different channel types:
- Single-session channels (TUI): display all logs
- Multi-session channels (ACP): route session logs, broadcast system logs

Updates Channel interface, ACP service, and TUI channel with log routing
documentation. Adds package-level documentation in log_routing.go."
```

---

## Chunk 3: Regenerate mocks and update tests

This chunk regenerates the facade mock and updates tests to use InputHandler.

### Task 3: Regenerate mock and update tests

**Files:**
- Regenerate: `pkg/channel/facade_mock.go`
- Create: `pkg/channel/handler_mock.go`
- Update: `pkg/channel/facade_test.go`

- [ ] **Step 1: Install mockgen if needed**

```bash
which mockgen || go install github.com/golang/mock/mockgen@latest
```

- [ ] **Step 2: Regenerate facade mock**

```bash
mockgen -source=pkg/channel/interface.go -destination=pkg/channel/facade_mock.go -package=channel github.com/denkhaus/gollum/pkg/channel ChannelFacade
```

Expected: Mock regenerated successfully

- [ ] **Step 3: Generate InputHandler mock**

```bash
mockgen -source=pkg/channel/handler.go -destination=pkg/channel/handler_mock.go -package=channel github.com/denkhaus/gollum/pkg/channel InputHandler
```

Expected: Mock generated successfully

- [ ] **Step 4: Update facade tests to use InputHandler mock**

```go
// pkg/channel/facade_test.go (example test update)

func TestChannelFacade_SubmitInput_DelegatesToHandler(t *testing.T) {
	mockHandler := NewMockInputHandler(t)
	mockFacade := &channelFacadeImpl{
		inputHandler: mockHandler,
		channels:     make(map[uuid.UUID]Channel),
		logger:       NewMockLoggerService(t),
	}

	ctx := context.Background()
	channelID := uuid.New()
	sessionID := "test"
	input := "hello"

	expectedResult := &InputResult{Handled: true, Response: "Hi there"}
	mockHandler.EXPECT().HandleInput(ctx, channelID, sessionID, input).Return(expectedResult, nil)

	result, err := mockFacade.SubmitInput(ctx, channelID, sessionID, input)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}
```

- [ ] **Step 5: Run all channel tests**

```bash
go test ./pkg/channel/... -v
```

Expected: All tests PASS

- [ ] **Step 6: Commit mocks and tests**

```bash
git add pkg/channel/facade_mock.go pkg/channel/handler_mock.go pkg/channel/facade_test.go
git commit -m "test(channel): regenerate mocks and update tests for InputHandler

Regenerates ChannelFacade mock after interface documentation update.
Generates new InputHandler mock for testing.
Updates facade tests to use InputHandler mock for more focused testing."
```

---

## Chunk 4: Create migration guide

This chunk creates a simple migration guide for the refactor.

### Task 4: Create migration guide

**Files:**
- Create: `docs/channel-facade-migration.md`

- [ ] **Step 1: Create migration guide**

```markdown
# Channel Facade Refactor - Migration Guide

## Summary

The `ChannelFacade.SubmitInput` method now delegates to a new `InputHandler` interface that encapsulates session and supervisor management logic.

## What Changed

### Before (Business logic in facade)
```go
type ChannelFacade interface {
    SubmitInput(ctx, channelID, sessionID, input) (*InputResult, error)
    // SubmitInput contained: command handling, session creation, supervisor creation
}
```

### After (Delegates to InputHandler)
```go
// New interface for business logic
type InputHandler interface {
    HandleInput(ctx, channelID, sessionID, input) (*InputResult, error)
    CancelInput(sessionID) error
}

// ChannelFacade delegates to InputHandler
type ChannelFacade interface {
    SubmitInput(...) (*InputResult, error)  // Now delegates to InputHandler
    CancelInput(...) error                   // Now delegates to InputHandler
    // ... other methods unchanged
}
```

## Impact

### For Channel Consumers (ACP, TUI, etc.)

**No changes required!** The `ChannelFacade` interface maintains backward compatibility.

```go
// This still works exactly as before
facade.SubmitInput(ctx, channelID, sessionID, input)
facade.CancelInput(sessionID)
```

### For Testing

You can now mock `InputHandler` directly instead of the entire facade:

```go
// Before: Mock entire facade
mockFacade := channel.NewMockChannelFacade(ctrl)
mockFacade.EXPECT().SubmitInput(...)

// After: Mock just the input handler (more focused)
mockHandler := channel.NewMockInputHandler(ctrl)
mockHandler.EXPECT().HandleInput(...)
```

### For Extensibility

You can provide custom input handling by implementing `InputHandler`:

```go
// Custom input handler with different session management
type CustomInputHandler struct {
    // ... custom fields
}

func (h *CustomInputHandler) HandleInput(ctx, channelID, sessionID, input) (*InputResult, error) {
    // Custom session/supervisor logic
}

// Inject custom handler (requires constructor modification)
facade := NewChannelFacadeWithHandler(customHandler)
```

## Benefits

1. **Better SOC**: Business logic separated from infrastructure
2. **Easier testing**: Mock InputHandler instead of entire facade
3. **Extensibility**: Swap input handling strategies without touching facade
4. **No breaking changes**: Existing code continues to work

## Questions?

See `pkg/channel` package documentation for log routing behavior and implementation guidelines.
```

- [ ] **Step 2: Commit migration guide**

```bash
git add docs/channel-facade-migration.md
git commit -m "docs(channel): add migration guide for InputHandler refactor

Documents the refactoring that extracted InputHandler for session/supervisor
management. Notes that no code changes are required for consumers - the
ChannelFacade interface remains backward compatible."
```

---

## Chunk 5: Final verification

This chunk verifies the refactor is complete and working.

### Task 5: Final verification and cleanup

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -v 2>&1 | grep -E "(PASS|FAIL|ok|FAIL)"
```

Expected: All packages PASS

- [ ] **Step 2: Run build**

```bash
go build ./...
```

Expected: Success, no errors

- [ ] **Step 3: Run linter**

```bash
golangci-lint run ./pkg/channel/... || true
```

Expected: No new warnings (fix if any)

- [ ] **Step 4: Verify ACP and TUI still work**

```bash
go test ./pkg/acp/... ./pkg/tui/... -v
```

Expected: All tests PASS

- [ ] **Step 5: Check for unused code**

```bash
goimports -w ./pkg/channel/
go vet ./pkg/channel/...
```

Expected: No issues

- [ ] **Step 6: Final commit**

```bash
git add -A
git commit -m "refactor(channel): complete InputHandler extraction refactor

Completes the ChannelFacade SOC refactoring:

Changes:
- Extracted InputHandler for session/supervisor business logic
- Added comprehensive log routing documentation
- Regenerated mocks for InputHandler and ChannelFacade
- Created migration guide

Benefits:
- Cleaner separation: business logic vs infrastructure
- Better testability: mock InputHandler directly
- Extensibility: swap input handling strategies
- No breaking changes: facade interface backward compatible

All tests passing, ACP and TUI verified working."
```

---

## Success Criteria

- [ ] InputHandler interface created with tests
- [ ] ChannelFacade delegates SubmitInput/CancelInput to InputHandler
- [ ] Log routing behavior documented (package docs, ACP, TUI)
- [ ] Mocks regenerated (facade_mock.go, handler_mock.go)
- [ ] Migration guide created
- [ ] Full test suite passes: `go test ./...`
- [ ] Build succeeds: `go build ./...`
- [ ] ACP and TUI verified working

---

## Comparison: Original Plan vs Pragmatic Refactor

| Aspect | Original Plan (Overengineered) | Pragmatic Refactor |
|--------|-------------------------------|-------------------|
| Interfaces | 4 (Handler, Router, Registry, Adapter) | 1 (Handler only) |
| Lines of code | ~1000+ | ~400 |
| Indirection layers | Adapter adds complexity | Direct delegation |
| Breaking changes | Multiple method renames | None |
| Extensibility | High (but YAGNI?) | Focused on actual need |
| Testability | Great | Great (for what matters) |
| Maintenance burden | High | Low |

**Key insight:** Message routing and provider discovery ARE facade responsibilities. We only needed to extract the business logic (session/supervisor management).

---

## References

- Original analysis: Channel facade had business logic mixed with infrastructure
- SOLID principles: Single Responsibility (business vs infrastructure)
- Related docs: `docs/channel-facade-migration.md`
- Related skills: @superpowers:test-driven-development
