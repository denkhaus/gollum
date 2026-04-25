# Channel Facade Refactor - Separation of Concerns

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the channel facade system to improve separation of concerns by splitting the monolithic ChannelFacade into focused, single-responsibility interfaces.

**Architecture:** Extract InputHandler for session/supervisor logic, split ChannelFacade into ChannelRegistry (lifecycle) and MessageRouter (distribution), keep DisplayMessage for middleware compatibility with improved documentation, and document log routing behavior.

**Tech Stack:** Go 1.23+, samber/do/v2 (DI), gollem SDK, existing channel infrastructure

---

## Context & Problem Statement

**Current Issues:**
1. `ChannelFacade` has 7+ responsibilities (registry, factory, routing, business logic)
2. `SubmitInput()` embeds session/supervisor creation (business logic in a facade)
3. `DisplayMessage()` name is misunderstood (actually correct - middleware "displays" output)
4. Log forwarding behavior is inconsistent (ACP broadcasts, TUI single-session)

**Why This Matters:**
- Violates Single Responsibility Principle → harder to test, maintain, extend
- Business logic in facade → tight coupling to session/supervisor implementation
- Misleading documentation (not the name!) → developer confusion
  - `DisplayMessage` name is correct for middleware semantics ("showing" output)
  - Issue: Documentation doesn't clarify it routes to ONE channel, not broadcast

**Files Involved:**
- `pkg/channel/interface.go` - Interface definitions
- `pkg/channel/facade.go` - Current monolithic implementation
- `pkg/channel/facade_test.go` - Tests to update
- `pkg/acp/service.go` - ACP channel implementation
- `pkg/tui/channel.go` - TUI channel implementation
- `pkg/channel/facade_mock.go` - Mock to regenerate

---

## File Structure After Refactor

```
pkg/channel/
├── interface.go              # Core interfaces (Channel, ChannelFactory, etc.)
├── registry.go               # NEW: ChannelRegistry interface + impl
├── router.go                 # NEW: MessageRouter interface + impl
├── handler.go                # NEW: InputHandler interface + impl
├── facade.go                 # DEPRECATED: Kept for backward compat, composes new interfaces
├── facade_adapter.go         # NEW: Adapter implementing old facade via new interfaces
├── log_routing.go            # NEW: Log routing documentation and helper types
├── types.go                  # Existing types (Message, InputResult, etc.)
├── facade_test.go            → split into registry_test.go, router_test.go, handler_test.go
└── facade_mock.go            → regenerate for new interfaces
```

---

## Chunk 1: Extract InputHandler Interface

This chunk extracts the input handling logic (session/supervisor management) from the facade into a focused interface.

### Task 1: Create InputHandler interface and implementation

**Files:**
- Create: `pkg/channel/handler.go`
- Create: `pkg/channel/handler_test.go`
- Modify: `pkg/channel/facade.go:171-197` (remove SubmitInput implementation)

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
		// Match gollem.Text input
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
```

Run: `go test ./pkg/channel -run TestInputHandler -v`
Expected: FAIL with "type InputHandler has no field HandleInput" and "undefined: NewInputHandler"

- [ ] **Step 2: Create InputHandler interface**

```go
// pkg/channel/handler.go
package channel

import (
	"context"
	"fmt"

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
Expected: PASS (all tests passing)

- [ ] **Step 3: Commit InputHandler implementation**

```bash
git add pkg/channel/handler.go pkg/channel/handler_test.go
git commit -m "refactor(channel): extract InputHandler interface for session/supervisor logic

Extracts input handling logic from ChannelFacade into a focused InputHandler
interface. This separates business logic (session/supervisor management) from
facade coordination concerns.

 BREAKING_CHANGE: ChannelFacade.SubmitInput moved to InputHandler.HandleInput"
```

---

## Chunk 2: Create MessageRouter interface

This chunk extracts message routing logic (DisplayMessage, NotifyAgentLifecycle, ForwardLog) from the facade.

### Task 2: Create MessageRouter interface and implementation

**NOTE:** After investigation, `DisplayMessage` should be **kept** (not renamed to `RouteMessage`) because:
1. It's used by `ChannelMiddleware` which semantically "displays" agent output
2. The name is correct for the middleware's intent (showing to user)
3. Renaming would break middleware compatibility
4. The issue is documentation, not naming - we need to clarify it routes to one channel

**Files:**
- Create: `pkg/channel/router.go`
- Create: `pkg/channel/router_test.go`
- Modify: `pkg/channel/facade.go:133-141, 206-238` (remove routing methods)

- [ ] **Step 1: Write failing test for MessageRouter**

```go
// pkg/channel/router_test.go
package channel

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageRouter_DisplayMessage(t *testing.T) {
	// Setup mock channel
	mockCh := NewMockChannel(t)
	mockCh.EXPECT().ID().Return(uuid.New())
	mockCh.EXPECT().OnMessage(mock.MatchedBy(func(msg Message) bool {
		return msg.Content == "test message"
	})).Return()

	router := NewMessageRouter(logger.NewMockLoggerService(t))
	err := router.RegisterChannel(mockCh)
	require.NoError(t, err)

	msg := Message{
		ChannelID: mockCh.ID(),
		Content:   "test message",
	}

	router.DisplayMessage(msg)
}

func TestMessageRouter_NotifyAgentLifecycle(t *testing.T) {
	mockCh := NewMockChannel(t)
	channelID := uuid.New()
	mockCh.EXPECT().ID().Return(channelID)

	router := NewMessageRouter(logger.NewMockLoggerService(t))
	router.RegisterChannel(mockCh)

	event := AgentLifecycleEvent{
		ChannelID: channelID,
		AgentID:   uuid.New(),
		Added:     true,
	}

	mockCh.EXPECT().OnAgentLifecycle(mock.MatchedBy(func(e AgentLifecycleEvent) bool {
		return e.Added == true
	}))

	router.NotifyAgentLifecycle(event.AgentID, event.ChannelID, "", "assistant", true)
}

func TestMessageRouter_ForwardLog(t *testing.T) {
	mockCh := NewMockChannel(t)
	channelID := uuid.New()
	mockCh.EXPECT().ID().Return(channelID)

	router := NewMessageRouter(logger.NewMockLoggerService(t))
	router.RegisterChannel(mockCh)

	entry := shared.LogEntry{
		ChannelID: channelID,
		Level:     "info",
		Message:   "test log",
	}

	mockCh.EXPECT().OnLog(mock.MatchedBy(func(e shared.LogEntry) bool {
		return e.Message == "test log"
	}))

	router.ForwardLog(entry)
}
```

Run: `go test ./pkg/channel -run TestMessageRouter -v`
Expected: FAIL with "undefined: NewMessageRouter" and "no method RouteMessage"

- [ ] **Step 2: Create MessageRouter interface**

```go
// pkg/channel/router.go
package channel

import (
	"fmt"
	"sync"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MessageRouter handles message distribution to registered channels.
// This interface encapsulates the logic for routing messages, logs, and events.
type MessageRouter interface {
	// DisplayMessage sends a message to the channel specified in msg.ChannelID.
	// NOTE: This is NOT a broadcast - it routes to exactly one channel.
	// The name "Display" reflects the intent: showing agent output to the user.
	DisplayMessage(msg Message)

	// NotifyAgentLifecycle sends agent lifecycle event to the specified channel
	NotifyAgentLifecycle(agentID uuid.UUID, channelID uuid.UUID, sessionID string, role string, added bool)

	// ForwardLog implements shared.LogForwarder for channel-based log routing
	ForwardLog(entry shared.LogEntry)

	// RegisterChannel adds a channel to receive events
	RegisterChannel(channel Channel) error

	// UnregisterChannel removes a channel from the registry
	UnregisterChannel(channelID uuid.UUID) error
}

// messageRouterImpl implements MessageRouter
type messageRouterImpl struct {
	mu       sync.RWMutex
	channels map[uuid.UUID]Channel
	logger   logger.LoggerService
}

// NewMessageRouter creates a new MessageRouter instance
func NewMessageRouter(log logger.LoggerService) MessageRouter {
	return &messageRouterImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   log,
	}
}

// DisplayMessage sends a message to the specific channel by ID.
// The name "Display" reflects the middleware's intent: showing agent output to the user.
// This is NOT a broadcast - it routes to exactly one channel (msg.ChannelID).
func (r *messageRouterImpl) DisplayMessage(msg Message) {
	r.mu.RLock()
	channel, exists := r.channels[msg.ChannelID]
	r.mu.RUnlock()

	if !exists {
		r.logger.Warn("channel not found for message",
			zap.String("channel_id", msg.ChannelID.String()),
		)
		return
	}

	channel.OnMessage(msg)
}

// NotifyAgentLifecycle sends agent lifecycle event to specific channel
func (r *messageRouterImpl) NotifyAgentLifecycle(agentID uuid.UUID, channelID uuid.UUID, sessionID string, role string, added bool) {
	event := AgentLifecycleEvent{
		AgentID:   agentID,
		Role:      role,
		Added:     added,
		SessionID: sessionID,
		ChannelID: channelID,
	}

	r.mu.RLock()
	targetChannel, exists := r.channels[channelID]
	r.mu.RUnlock()

	if !exists {
		r.logger.Warn("channel not found for agent lifecycle event",
			zap.String("channel_id", channelID.String()),
			zap.String("session_id", sessionID),
			zap.String("agent_id", agentID.String()),
		)
		return
	}

	targetChannel.OnAgentLifecycle(event)
}

// ForwardLog implements shared.LogForwarder for channel-based log routing
func (r *messageRouterImpl) ForwardLog(entry shared.LogEntry) {
	r.mu.RLock()
	targetChannel, exists := r.channels[entry.ChannelID]
	r.mu.RUnlock()

	if !exists {
		r.logger.Warn("channel not found for log entry",
			zap.String("channel_id", entry.ChannelID.String()),
			zap.String("session_id", entry.SessionID),
		)
		return
	}

	// Forward to specific channel only
	targetChannel.OnLog(entry)
}

// RegisterChannel adds a channel to receive events
func (r *messageRouterImpl) RegisterChannel(channel Channel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.channels[channel.ID()]; exists {
		return fmt.Errorf("channel %s already registered", channel.ID())
	}
	r.channels[channel.ID()] = channel
	return nil
}

// UnregisterChannel removes a channel from the registry
func (r *messageRouterImpl) UnregisterChannel(channelID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.channels, channelID)
	return nil
}

// Compile-time check
var _ MessageRouter = (*messageRouterImpl)(nil)
var _ shared.LogForwarder = (*messageRouterImpl)(nil)
```

Run: `go test ./pkg/channel -run TestMessageRouter -v`
Expected: PASS

- [ ] **Step 3: Commit MessageRouter implementation**

```bash
git add pkg/channel/router.go pkg/channel/router_test.go
git commit -m "refactor(channel): extract MessageRouter interface for distribution logic

Extracts message routing logic from ChannelFacade into a focused MessageRouter
interface. Keeps DisplayMessage name for middleware compatibility (correct semantic:
'displaying' agent output to user). Adds documentation clarifying targeted routing."
```

---

## Chunk 3: Create ChannelRegistry interface

This chunk extracts channel provider discovery and creation logic.

### Task 3: Create ChannelRegistry interface and implementation

**Files:**
- Create: `pkg/channel/registry.go`
- Create: `pkg/channel/registry_test.go`
- Modify: `pkg/channel/facade.go:60-98` (remove provider methods)

- [ ] **Step 1: Write failing test for ChannelRegistry**

```go
// pkg/channel/registry_test.go
package channel

import (
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelRegistry_DiscoverProviders(t *testing.T) {
	injector := do.New()
	dummyFactory := func(opts ...ChannelOption) (Channel, error) {
		return &mockChannel{id: uuid.New()}, nil
	}

	// Register a mock provider
	do.ProvideNamedValue(injector, "channel_test", dummyFactory)

	registry := NewChannelRegistry(NewMockLoggerService(t))
	err := registry.DiscoverProviders(injector)
	require.NoError(t, err)
}

func TestChannelRegistry_CreateChannel(t *testing.T) {
	registry := NewChannelRegistry(NewMockLoggerService(t))

	// Register a test factory manually
	testID := ChannelIdentifier("test")
	testFactory := func(opts ...ChannelOption) (Channel, error) {
		return &mockChannel{id: uuid.New()}, nil
	}

	registry.(*channelRegistryImpl).providers[testID] = testFactory

	ch, err := registry.CreateChannel(testID)
	require.NoError(t, err)
	assert.NotNil(t, ch)
}
```

Run: `go test ./pkg/channel -run TestChannelRegistry -v`
Expected: FAIL with "undefined: NewChannelRegistry"

- [ ] **Step 2: Create ChannelRegistry interface**

```go
// pkg/channel/registry.go
package channel

import (
	"fmt"
	"strings"
	"sync"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/samber/do/v2"
)

// ChannelRegistry manages channel provider discovery and creation.
type ChannelRegistry interface {
	// DiscoverProviders scans DI for channel providers
	DiscoverProviders(injector do.Injector) error

	// CreateChannel creates a channel instance by identifier with options
	CreateChannel(identifier ChannelIdentifier, opts ...ChannelOption) (Channel, error)
}

// channelRegistryImpl implements ChannelRegistry
type channelRegistryImpl struct {
	mu        sync.RWMutex
	providers map[ChannelIdentifier]ChannelFactory
	logger    logger.LoggerService
}

// NewChannelRegistry creates a new ChannelRegistry instance
func NewChannelRegistry(log logger.LoggerService) ChannelRegistry {
	return &channelRegistryImpl{
		providers: make(map[ChannelIdentifier]ChannelFactory),
		logger:    log,
	}
}

// DiscoverProviders scans the DI container for channel providers
func (r *channelRegistryImpl) DiscoverProviders(injector do.Injector) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers = make(map[ChannelIdentifier]ChannelFactory)

	services := injector.ListProvidedServices()
	foundAny := false

	for _, service := range services {
		if !strings.HasPrefix(service.Service, ProviderPrefix) {
			continue
		}

		channelName := strings.TrimPrefix(service.Service, ProviderPrefix)
		if channelName == "" {
			continue
		}

		factory, err := do.InvokeNamed[ChannelFactory](injector, service.Service)
		if err != nil {
			r.logger.Debugf("Service %s found but is not a ChannelFactory: %v", service.Service, err)
			continue
		}

		identifier := ChannelIdentifier(channelName)
		r.providers[identifier] = factory
		r.logger.Infof("Discovered channel: %s (from service: %s)", identifier, service.Service)
		foundAny = true
	}

	if !foundAny {
		r.logger.Warn("No channel providers discovered - channels may not be available")
	}

	return nil
}

// CreateChannel creates a channel instance by identifier with options
func (r *channelRegistryImpl) CreateChannel(identifier ChannelIdentifier, opts ...ChannelOption) (Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, ok := r.providers[identifier]
	if !ok {
		available := make([]string, 0, len(r.providers))
		for id := range r.providers {
			available = append(available, string(id))
		}
		return nil, fmt.Errorf("unknown channel identifier: %s (available: %v)", identifier, available)
	}

	ch, err := factory(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel %s: %w", identifier, err)
	}

	for _, opt := range opts {
		if err := opt.Apply(ch); err != nil {
			return nil, fmt.Errorf("failed to apply option to channel %s: %w", identifier, err)
		}
	}

	return ch, nil
}

// Compile-time check
var _ ChannelRegistry = (*channelRegistryImpl)(nil)
```

Run: `go test ./pkg/channel -run TestChannelRegistry -v`
Expected: PASS

- [ ] **Step 3: Commit ChannelRegistry implementation**

```bash
git add pkg/channel/registry.go pkg/channel/registry_test.go
git commit -m "refactor(channel): extract ChannelRegistry interface for provider management

Extracts channel provider discovery and creation logic from ChannelFacade
into a focused ChannelRegistry interface.

 BREAKING_CHANGE: ChannelFacade.DiscoverProviders/CreateChannel moved to ChannelRegistry"
```

---

## Chunk 4: Update ChannelFacade to compose new interfaces

This chunk updates the original ChannelFacade to use composition instead of direct implementation.

### Task 4: Create backward-compatible facade adapter

**Files:**
- Create: `pkg/channel/facade_adapter.go`
- Modify: `pkg/channel/interface.go:98-126` (update ChannelFacade documentation)

- [ ] **Step 1: Write failing test for backward compatibility**

```go
// pkg/channel/facade_adapter_test.go
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

func TestFacadeAdapter_BackwardCompatibility(t *testing.T) {
	injector := do.New()

	// Setup mocks
	cmdMgr := command.NewMockCommandManager(t)
	sm := session.NewMockSessionManager(t)
	reg := registry.NewMockAgentRegistry(t)
	af := shared.NewMockAgentFactory(t)
	logger := NewMockLoggerService(t)

	// Create facade via new pattern
	facade, err := NewChannelFacadeCompose(injector, cmdMgr, sm, reg, af, logger)
	require.NoError(t, err)

	// Test old interface still works
	channelID := uuid.New()
	sessionID := "test"

	// SubmitInput should work
	result, err := facade.SubmitInput(context.Background(), channelID, sessionID, "/help")
	require.NoError(t, err)
	assert.NotNil(t, result)

	// DisplayMessage should work
	msg := Message{ChannelID: channelID, Content: "test"}
	facade.DisplayMessage(msg)

	// RegisterChannel should work
	mockCh := NewMockChannel(t)
	mockCh.EXPECT().ID().Return(channelID)
	err = facade.RegisterChannel(mockCh)
	require.NoError(t, err)
}
```

Run: `go test ./pkg/channel -run TestFacadeAdapter -v`
Expected: FAIL with "undefined: NewChannelFacadeCompose"

- [ ] **Step 2: Create facade adapter composing new interfaces**

```go
// pkg/channel/facade_adapter.go
package channel

import (
	"github.com/denkhaus/gollum/pkg/command"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
)

// NewChannelFacadeCompose creates a ChannelFacade by composing the new focused interfaces.
// This maintains backward compatibility while using the refactored implementation.
func NewChannelFacadeCompose(
	injector do.Injector,
	cmdMgr command.Manager,
	sessMgr session.SessionManager,
	reg registry.AgentRegistry,
	af shared.AgentFactory,
	log logger.LoggerService,
) (ChannelFacade, error) {
	handler := NewInputHandler(cmdMgr, sessMgr, reg, af, log)
	router := NewMessageRouter(log)
	registry := NewChannelRegistry(log)

	if err := registry.DiscoverProviders(injector); err != nil {
		return nil, err
	}

	return &facadeAdapter{
		handler:  handler,
		router:   router,
		registry: registry,
	}, nil
}

// facadeAdapter implements ChannelFacade by composing focused interfaces
type facadeAdapter struct {
	handler  InputHandler
	router   MessageRouter
	registry ChannelRegistry
}

// DisplayMessage sends a message to the channel specified in msg.ChannelID
// Delegates to MessageRouter which handles the routing
func (a *facadeAdapter) DisplayMessage(msg Message) {
	a.router.DisplayMessage(msg)
}

// SubmitInput handles user input via InputHandler
func (a *facadeAdapter) SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (*InputResult, error) {
	return a.handler.HandleInput(ctx, channelID, sessionID, input)
}

// CancelInput cancels an in-flight input via InputHandler
func (a *facadeAdapter) CancelInput(sessionID string) error {
	return a.handler.CancelInput(sessionID)
}

// RegisterChannel adds a channel via MessageRouter
func (a *facadeAdapter) RegisterChannel(channel Channel) error {
	return a.router.RegisterChannel(channel)
}

// UnregisterChannel removes a channel via MessageRouter
func (a *facadeAdapter) UnregisterChannel(channelID uuid.UUID) error {
	return a.router.UnregisterChannel(channelID)
}

// NotifyAgentLifecycle sends event via MessageRouter
func (a *facadeAdapter) NotifyAgentLifecycle(agentID uuid.UUID, channelID uuid.UUID, sessionID string, role string, added bool) {
	a.router.NotifyAgentLifecycle(agentID, channelID, sessionID, role, added)
}

// ForwardLog forwards log entry via MessageRouter
func (a *facadeAdapter) ForwardLog(entry shared.LogEntry) {
	a.router.ForwardLog(entry)
}

// DiscoverProviders scans DI for channel providers via ChannelRegistry
func (a *facadeAdapter) DiscoverProviders(injector do.Injector) error {
	return a.registry.DiscoverProviders(injector)
}

// CreateChannel creates a channel via ChannelRegistry
func (a *facadeAdapter) CreateChannel(identifier ChannelIdentifier, opts ...ChannelOption) (Channel, error) {
	return a.registry.CreateChannel(identifier, opts...)
}

// Compile-time check
var _ ChannelFacade = (*facadeAdapter)(nil)
```

Run: `go test ./pkg/channel -run TestFacadeAdapter -v`
Expected: PASS

- [ ] **Step 3: Update facade.go to use new composition**

```go
// pkg/channel/facade.go (modify NewChannelFacade function)

// NewChannelFacade creates a new channel facade service using the composed implementation.
// This function now uses the refactored interfaces internally while maintaining
// backward compatibility with the existing ChannelFacade interface.
func NewChannelFacade(injector do.Injector) (ChannelFacade, error) {
	cm := do.MustInvoke[command.ManagerService](injector)
	reg := do.MustInvoke[registry.AgentRegistry](injector)
	af := do.MustInvoke[shared.AgentFactory](injector)
	sm := do.MustInvoke[session.SessionManager](injector)
	log := do.MustInvoke[logger.LoggerService](injector)

	return NewChannelFacadeCompose(injector, cm, sm, reg, af, log)
}
```

Run: `go test ./pkg/channel -v`
Expected: PASS (all existing tests still pass)

- [ ] **Step 4: Commit facade adapter**

```bash
git add pkg/channel/facade_adapter.go pkg/channel/facade_adapter_test.go pkg/channel/facade.go
git commit -m "refactor(channel): add facade adapter for backward compatibility

Creates facadeAdapter that composes the new focused interfaces (InputHandler,
MessageRouter, ChannelRegistry) to maintain backward compatibility with existing
ChannelFacade interface. Updates NewChannelFacade to use composed implementation."
```

---

## Chunk 5: Update documentation and add log routing guide

This chunk documents the log routing behavior and updates interface documentation.

### Task 5: Create log routing documentation

**Files:**
- Create: `pkg/channel/log_routing.go`
- Modify: `pkg/channel/interface.go:78-96` (update Channel documentation)

- [ ] **Step 1: Create log routing guide**

```go
// pkg/channel/log_routing.go
package channel

// Log Routing Behavior
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
//
// EXAMPLE: ACP log forwarding (broadcasts system logs, routes session logs)
//
//	func (s *acpServiceImpl) OnLog(entry shared.LogEntry) {
//	    if entry.SessionID != "" {
//	        // Route to specific session
//	        session := s.store.Get(acppkg.SessionID(entry.SessionID))
//	        stream.SendText(session.Context, formatLog(entry))
//	    } else {
//	        // Broadcast to all sessions
//	        for _, session := range s.store.List() {
//	            stream.SendText(session.Context, formatLog(entry))
//	        }
//	    }
//	}
//
// EXAMPLE: TUI log forwarding (single channel, no sessions)
//
//	func (c *TUIChannel) OnLog(entry shared.LogEntry) {
//	    select {
//	    case c.messageChan <- formatLog(entry):
//	    default:
//	        // Drop if full to prevent deadlock
//	    }
//	}
const LogRoutingDocumentation = ""
```

- [ ] **Step 2: Update Channel interface documentation**

```go
// pkg/channel/interface.go (update Channel interface comments)

// Channel is the interface that all channel implementations must satisfy.
//
// Channels receive data FROM the system via three methods:
// - OnMessage(msg): Agent responses, tool executions, thinking blocks
// - OnLog(entry): Log entries from the system (see log_routing.go for behavior)
// - OnAgentLifecycle(event): Agent registration/removal notifications
//
// Channels send data TO the system via the ChannelFacade:
// - SubmitInput(): User input from the channel
//
// IMPLEMENTATION GUIDELINES:
// 1. Use non-blocking sends for channels to prevent deadlocks
// 2. Handle all message types ( MessageType* constants)
// 3. Document session handling behavior (single vs multi-session)
// 4. Implement Start() to block until channel is complete
//
// See log_routing.go for log routing behavior documentation.
type Channel interface {
	// ... existing methods unchanged
}
```

- [ ] **Step 3: Commit documentation updates**

```bash
git add pkg/channel/log_routing.go pkg/channel/interface.go
git commit -m "docs(channel): add log routing behavior documentation

Documents log routing behavior for different channel types (single-session
vs multi-session). Updates Channel interface documentation with implementation
guidelines. Adds log_routing.go with examples for ACP and TUI patterns."
```

---

## Chunk 6: Update ACP service implementation

This chunk updates the ACP service to use the renamed facade method.

### Task 6: Update ACP to use RouteMessage

**Files:**
- Modify: `pkg/acp/service.go` (update facade method calls if any)
- Modify: `pkg/acp/service_test.go` (update tests)

- [ ] **Step 1: Check for DisplayMessage usage in ACP**

```bash
grep -n "DisplayMessage" /home/denkhaus/dev/gomodules/gollum/pkg/acp/*.go
```

Expected: No direct usage (ACP only calls facade.SubmitInput via Prompt())

- [ ] **Step 2: Verify ACP tests still pass**

```bash
go test ./pkg/acp -v
```

Expected: PASS (ACP doesn't use DisplayMessage, so no changes needed)

- [ ] **Step 3: Add comment documenting ACP log routing**

```go
// pkg/acp/service.go (update OnLog documentation)

// OnLog receives log entries from the agent system and forwards them to ACP sessions.
//
// Log routing behavior:
// - Session-specific logs (entry.SessionID set) → routed to that session only
// - System-wide logs (no session ID) → broadcast to all active sessions
// See pkg/channel/log_routing.go for general log routing documentation.
func (s *acpServiceImpl) OnLog(entry shared.LogEntry) {
	// ... existing implementation
}
```

- [ ] **Step 4: Commit ACP documentation update**

```bash
git add pkg/acp/service.go
git commit -m "docs(acp): document log routing behavior

Adds documentation for ACP's log routing pattern (broadcast system logs,
route session logs). References pkg/channel/log_routing.go."
```

---

## Chunk 7: Update TUI channel implementation

This chunk ensures TUI works correctly with the refactored facade.

### Task 7: Verify TUI compatibility

**Files:**
- Verify: `pkg/tui/channel.go`
- Test: `pkg/tui/channel_test.go`

- [ ] **Step 1: Check TUI facade usage**

```bash
grep -n "facade\." /home/denkhaus/dev/gomodules/gollum/pkg/tui/channel.go
```

Expected: Only facade.SubmitInput() used (already compatible)

- [ ] **Step 2: Add TUI log routing documentation**

```go
// pkg/tui/channel.go (update OnLog documentation)

// OnLog handles log entries from the channel system.
//
// Log routing behavior for TUI (single-session channel):
// - All logs are displayed in the TUI UI
// - No session routing (TUI uses empty session ID)
// See pkg/channel/log_routing.go for general log routing documentation.
func (c *TUIChannel) OnLog(entry shared.LogEntry) {
	// ... existing implementation
}
```

- [ ] **Step 3: Verify TUI tests pass**

```bash
go test ./pkg/tui -v
```

Expected: PASS

- [ ] **Step 4: Commit TUI documentation update**

```bash
git add pkg/tui/channel.go
git commit -m "docs(tui): document log routing behavior

Adds documentation for TUI's single-session log routing pattern.
References pkg/channel/log_routing.go."
```

---

## Chunk 8: Regenerate mocks and finalize

This chunk regenerates mocks for the new interfaces and finalizes the refactor.

### Task 8: Regenerate mocks and run full test suite

**Files:**
- Regenerate: `pkg/channel/facade_mock.go`
- Create: `pkg/channel/handler_mock.go`
- Create: `pkg/channel/router_mock.go`
- Create: `pkg/channel/registry_mock.go`

- [ ] **Step 1: Install mockgen if needed**

```bash
which mockgen || go install github.com/golang/mock/mockgen@latest
```

- [ ] **Step 2: Regenerate facade mock**

```bash
mockgen -source=pkg/channel/interface.go -destination=pkg/channel/facade_mock.go -package=channel github.com/denkhaus/gollum/pkg/channel ChannelFacade
```

Expected: Mock regenerated successfully

- [ ] **Step 3: Generate handler mock**

```bash
mockgen -source=pkg/channel/handler.go -destination=pkg/channel/handler_mock.go -package=channel github.com/denkhaus/gollum/pkg/channel InputHandler
```

Expected: Mock generated successfully

- [ ] **Step 4: Generate router mock**

```bash
mockgen -source=pkg/channel/router.go -destination=pkg/channel/router_mock.go -package=channel github.com/denkhaus/gollum/pkg/channel MessageRouter
```

Expected: Mock generated successfully

- [ ] **Step 5: Generate registry mock**

```bash
mockgen -source=pkg/channel/registry.go -destination=pkg/channel/registry_mock.go -package=channel github.com/denkhaus/gollum/pkg/channel ChannelRegistry
```

Expected: Mock generated successfully

- [ ] **Step 6: Run full channel test suite**

```bash
go test ./pkg/channel/... -v
```

Expected: PASS (all tests passing)

- [ ] **Step 7: Run full project test suite**

```bash
go test ./... -v 2>&1 | grep -E "(PASS|FAIL|ok|FAIL)"
```

Expected: All packages PASS

- [ ] **Step 8: Commit mocks**

```bash
git add pkg/channel/*_mock.go
git commit -m "test(channel): regenerate mocks for refactored interfaces

Regenerates mocks for InputHandler, MessageRouter, and ChannelRegistry
interfaces. Updates facade mock for compatibility."
```

---

## Chunk 9: Migration guide and final documentation

This chunk creates a migration guide for consumers of the channel facade.

### Task 9: Create migration guide

**Files:**
- Create: `docs/channel-facade-migration.md`

- [ ] **Step 1: Create migration guide**

```markdown
# Channel Facade Migration Guide

## Summary

The `ChannelFacade` interface has been refactored to improve separation of concerns. The interface remains backward compatible, but internal implementation now uses composed, focused interfaces.

## What Changed

### Before (Monolithic Facade)
```go
type ChannelFacade interface {
    // Registry + Factory + Routing + Business Logic all mixed
    RegisterChannel(...) error
    CreateChannel(...) (Channel, error)
    DisplayMessage(msg Message)
    SubmitInput(...) (*InputResult, error)
    // ... 7+ methods
}
```

### After (Composed Interfaces)
```go
// Focused interfaces
type InputHandler interface {
    HandleInput(...) (*InputResult, error)
    CancelInput(...) error
}

type MessageRouter interface {
    RouteMessage(msg Message)
    NotifyAgentLifecycle(...)
    ForwardLog(...)
    RegisterChannel(...) error
    UnregisterChannel(...) error
}

type ChannelRegistry interface {
    DiscoverProviders(...) error
    CreateChannel(...) (Channel, error)
}

// ChannelFacade composes these (backward compatible)
type ChannelFacade interface {
    // All original methods still work
}
```

## Breaking Changes

### Interface Split (No API Changes)
- `DisplayMessage()` remains unchanged - name is correct for middleware semantics
- `SubmitInput()` now implemented by `InputHandler`
- `DisplayMessage()` now implemented by `MessageRouter`
- `CreateChannel()` now implemented by `ChannelRegistry`

### Interface Split
- `SubmitInput()` now implemented by `InputHandler`
- `DisplayMessage()` now implemented by `MessageRouter`
- `CreateChannel()` now implemented by `ChannelRegistry`

## Migration Steps

### For Channel Consumers (ACP, TUI, etc.)

**No changes required!** The `ChannelFacade` interface maintains backward compatibility.

```go
// This still works
facade.SubmitInput(ctx, channelID, sessionID, input)
facade.DisplayMessage(msg)
```

### For New Code

Prefer using the focused interfaces directly:

```go
// Instead of
facade.SubmitInput(ctx, channelID, sessionID, input)
facade.DisplayMessage(msg)

// Use (if you have access to the focused interfaces)
handler.HandleInput(ctx, channelID, sessionID, input)
router.DisplayMessage(msg)
```

**Note:** `DisplayMessage` keeps its name - it correctly describes the middleware's intent
of "displaying" agent output to the user. The implementation routes to one channel, which is
documented in the interface.

### For Facade Testing

Use the new focused mocks:

```go
// Before
mockFacade := channel.NewMockChannelFacade(ctrl)
mockFacade.EXPECT().SubmitInput(...)

// After (more focused testing)
mockHandler := channel.NewMockInputHandler(ctrl)
mockHandler.EXPECT().HandleInput(...)
```

## Benefits

1. **Better testability**: Mock only what you need
2. **Clearer responsibilities**: Each interface has one job
3. **Easier extension**: Add new routing logic without touching input handling
4. **Type safety**: Compile-time checks for correct interface usage

## Questions?

See `pkg/channel/log_routing.go` for log routing behavior documentation.
```

- [ ] **Step 2: Commit migration guide**

```bash
git add docs/channel-facade-migration.md
git commit -m "docs(channel): add migration guide for facade refactor

Documents the refactoring from monolithic ChannelFacade to composed
focused interfaces (InputHandler, MessageRouter, ChannelRegistry).
Includes migration steps and examples."
```

---

## Chunk 10: Final verification and cleanup

This chunk verifies everything works and cleans up any deprecated code.

### Task 10: Final verification

**Files:**
- Verify: All tests pass
- Verify: No deprecated code remains

- [ ] **Step 1: Run build**

```bash
go build ./...
```

Expected: Success, no errors

- [ ] **Step 2: Run linter**

```bash
golangci-lint run ./pkg/channel/... || true
```

Expected: No new warnings (fix if any)

- [ ] **Step 3: Verify DI registration**

```bash
grep -rn "NewChannelFacade" /home/denkhaus/dev/gomodules/gollum/pkg/ --include='*.go' | grep -v test | grep -v mock
```

Expected: All usages still work with composed implementation

- [ ] **Step 4: Check for unused imports**

```bash
goimports -w ./pkg/channel/
```

Expected: Imports cleaned up

- [ ] **Step 5: Final commit**

```bash
git add -A
git commit -m "refactor(channel): finalize facade SOC refactor

Completes the ChannelFacade separation of concerns refactor:
- InputHandler: session/supervisor management
- MessageRouter: message/log/event distribution
- ChannelRegistry: provider discovery and creation
- Backward compatible via facade adapter

All tests passing, migration guide added."
```

---

## Success Criteria

- [ ] All focused interfaces have tests (handler_test.go, router_test.go, registry_test.go)
- [ ] Backward compatibility maintained (facade_adapter passes all old tests)
- [ ] ACP and TUI work without changes
- [ ] Mocks regenerated for all new interfaces
- [ ] Migration guide created
- [ ] Log routing behavior documented
- [ ] Full test suite passes: `go test ./...`
- [ ] Build succeeds: `go build ./...`

---

## References

- Original issue: Channel facade has too many responsibilities
- SOLID principles: Single Responsibility, Interface Segregation
- Related docs: `pkg/channel/log_routing.go`, `docs/channel-facade-migration.md`
- Related skills: @superpowers:test-driven-development, @superpowers:brainstorming
