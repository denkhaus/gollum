# Channel Factory Refactoring Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor ApplicationService to remove tight coupling with TUI by implementing a channel factory pattern using dependency injection.

**Architecture:** Channels register themselves as DI providers with names matching `channel_*` pattern. ChannelFacade discovers these providers and creates channel instances via `CreateChannel(identifier, opts...)`. ApplicationService requests channels by identifier constant.

**Tech Stack:** Go 1.26+, samber/do/v2 (DI), gomock (mocking), testify/assert

**Reference Spec:** `docs/superpowers/specs/2026-04-12-channel-factory-refactor-design.md`

---

## Chunk 1: Foundation Types

### Task 1: Create Channel Types File

**Files:**
- Create: `pkg/channel/types.go`
- Test: `pkg/channel/types_test.go`

- [ ] **Step 1: Write failing tests for channel types**

```go
// pkg/channel/types_test.go
package channel

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChannelIdentifier_String(t *testing.T) {
	id := ChannelIdentifier("tui")
	assert.Equal(t, "tui", string(id))
}

func TestChannelFactory_CreatesChannel(t *testing.T) {
	factory := func(opts ...ChannelOption) (Channel, error) {
		return &mockChannel{id: uuid.New()}, nil
	}

	ch, err := factory()
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func TestChannelOption_Apply(t *testing.T) {
	ch := &mockChannel{id: uuid.New()}
	opt := &mockOption{applyFunc: func(c Channel) error {
		return nil
	}}

	err := opt.Apply(ch)
	assert.NoError(t, err)
}

// mockChannel for testing
type mockChannel struct {
	id uuid.UUID
}

func (m *mockChannel) ID() uuid.UUID                                  { return m.id }
func (m *mockChannel) OnMessage(msg Message)                           {}
func (m *mockChannel) OnLog(entry shared.LogEntry)                     {}
func (m *mockChannel) OnAgentLifecycle(event AgentLifecycleEvent)      {}

// mockOption for testing
type mockOption struct {
	applyFunc func(Channel) error
}

func (m *mockOption) Apply(ch Channel) error {
	return m.applyFunc(ch)
}

func TestChannelStarter_Interface(t *testing.T) {
	// Compile-time check that ChannelStarter is a valid interface
	var _ ChannelStarter = (*mockStarterChannel)(nil)
}

type mockStarterChannel struct {
	id uuid.UUID
}

func (m *mockStarterChannel) ID() uuid.UUID                              { return m.id }
func (m *mockStarterChannel) OnMessage(msg Message)                       {}
func (m *mockStarterChannel) OnLog(entry shared.LogEntry)                 {}
func (m *mockStarterChannel) OnAgentLifecycle(event AgentLifecycleEvent) {}
func (m *mockStarterChannel) Start(ctx context.Context) error            { return nil }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/channel/types_test.go -v`
Expected: FAIL with "undefined: ChannelIdentifier", "undefined: ChannelFactory", "undefined: ChannelOption", "undefined: ChannelStarter"

- [ ] **Step 3: Implement channel types**

```go
// pkg/channel/types.go
package channel

import (
	"context"
)

// ChannelOption is the interface for channel configuration options.
// Each channel package implements its own option types that satisfy this interface.
type ChannelOption interface {
	Apply(channel.Channel) error
}

// ChannelFactory creates a channel instance with options.
// Registered as named providers in DI (e.g., "channel_tui", "channel_acp").
type ChannelFactory func(opts ...ChannelOption) (channel.Channel, error)

// ChannelIdentifier is the const type each channel exports.
// Provides type safety - no magic strings.
type ChannelIdentifier string

// ChannelStarter is the interface for channels that manage their own lifecycle.
// Channels like TUI implement this to run their main loop.
type ChannelStarter interface {
	channel.Channel
	Start(ctx context.Context) error
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/channel/... -v -run TestChannel`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/channel/types.go pkg/channel/types_test.go
git commit -m "feat(channel): add ChannelOption, ChannelFactory, ChannelIdentifier, ChannelStarter types"
```

---

### Task 2: Generate Channel Mocks

**Files:**
- Create: `pkg/channel/generate.go`
- Generated: `pkg/channel/channel_mock.go`

- [ ] **Step 1: Create generate.go with mockgen directives**

```go
// pkg/channel/generate.go
//go:generate go run go.uber.org/mock/mockgen -source=interface.go -destination=channel_mock.go -package=channel github.com/denkhaus/gollum/pkg/channel Channel,ChannelFacade
package channel
```

- [ ] **Step 2: Run mockgen**

Run: `cd pkg/channel && go generate`
Expected: Creates `channel_mock.go` with MockChannel and MockChannelFacade

- [ ] **Step 3: Verify mocks compile**

Run: `go build ./pkg/channel/...`
Expected: Success, no errors

- [ ] **Step 4: Commit**

```bash
git add pkg/channel/generate.go pkg/channel/channel_mock.go
git commit -m "feat(channel): add mockgen for channel interfaces"
```

---

## Chunk 2: TUI Channel Infrastructure

### Task 3: Add TUI Channel Identifier

**Files:**
- Modify: `pkg/tui/channel.go` (add Identifier constant)
- Test: `pkg/tui/channel_test.go`

- [ ] **Step 1: Write failing test for Identifier constant**

```go
// pkg/tui/channel_test.go
package tui

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/stretchr/testify/assert"
)

func TestIdentifier(t *testing.T) {
	id := Identifier
	assert.Equal(t, channel.ChannelIdentifier("tui"), id)
	assert.NotEmpty(t, string(id))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/tui/... -v -run TestIdentifier`
Expected: FAIL with "undefined: Identifier"

- [ ] **Step 3: Add Identifier constant to channel.go**

```go
// pkg/tui/channel.go
// Add after imports, before TUIChannel struct:

// Identifier is the unique channel identifier for registration.
const Identifier = channel.ChannelIdentifier("tui")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/tui/... -v -run TestIdentifier`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/tui/channel.go pkg/tui/channel_test.go
git commit -m "feat(tui): add Identifier constant for channel registration"
```

---

### Task 4: Create TUI Options Package

**Files:**
- Create: `pkg/tui/options.go`
- Test: `pkg/tui/options_test.go`

- [ ] **Step 1: Write failing tests for TUI options**

```go
// pkg/tui/options_test.go
package tui

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTUIOption_WithMessageChan(t *testing.T) {
	ch := NewTUIChannel(nil)
	msgChan := make(chan channel.Message, 10)

	opt := WithMessageChan(msgChan)
	err := opt.Apply(ch)

	require.NoError(t, err)
	assert.Equal(t, msgChan, ch.GetMessageChan())
}

func TestTUIOption_WithLoggerService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	ch := NewTUIChannel(nil)

	opt := WithLoggerService(mockLogger)
	err := opt.Apply(ch)

	require.NoError(t, err)
	assert.Equal(t, mockLogger, ch.GetLogger())
}

func TestTUIOption_WithMarkdownRenderer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRenderer := markdown.NewMockRenderer(ctrl)
	ch := NewTUIChannel(nil)

	opt := WithMarkdownRenderer(mockRenderer)
	err := opt.Apply(ch)

	require.NoError(t, err)
	assert.Equal(t, mockRenderer, ch.GetRenderer())
}

func TestTUIOption_WrongChannelType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	opt := WithMessageChan(make(chan channel.Message))

	// Apply to wrong channel type
	wrongCh := &mockOtherChannel{id: uuid.New()}
	err := opt.Apply(wrongCh)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TUI option applied to wrong channel type")
}

func TestTUIOptions_Multiple(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockRenderer := markdown.NewMockRenderer(ctrl)
	msgChan := make(chan channel.Message, 10)

	ch := NewTUIChannel(nil)

	// Apply multiple options
	opts := []TUIOption{
		WithMessageChan(msgChan),
		WithLoggerService(mockLogger),
		WithMarkdownRenderer(mockRenderer),
	}

	for _, opt := range opts {
		err := opt.Apply(ch)
		require.NoError(t, err)
	}

	assert.Equal(t, msgChan, ch.GetMessageChan())
	assert.Equal(t, mockLogger, ch.GetLogger())
	assert.Equal(t, mockRenderer, ch.GetRenderer())
}

// mockOtherChannel for testing type mismatch
type mockOtherChannel struct {
	id uuid.UUID
}

func (m *mockOtherChannel) ID() uuid.UUID                              { return m.id }
func (m *mockOtherChannel) OnMessage(msg channel.Message)               {}
func (m *mockOtherChannel) OnLog(entry shared.LogEntry)                 {}
func (m *mockOtherChannel) OnAgentLifecycle(event channel.AgentLifecycleEvent) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/tui/... -v -run TestTUIOption`
Expected: FAIL with "undefined: WithMessageChan", "undefined: WithLoggerService", etc.

- [ ] **Step 3: Implement TUI options**

```go
// pkg/tui/options.go
package tui

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
)

// TUIOption implements channel.ChannelOption for TUI-specific configuration.
type TUIOption struct {
	applyFunc func(*TUIChannel) error
}

// Apply implements channel.ChannelOption.
func (o TUIOption) Apply(ch channel.Channel) error {
	tuiCh, ok := ch.(*TUIChannel)
	if !ok {
		return fmt.Errorf("TUI option applied to wrong channel type (got %T, expected *TUIChannel)", ch)
	}
	return o.applyFunc(tuiCh)
}

// WithMessageChan sets the message channel for TUI communication.
func WithMessageChan(ch chan<- channel.Message) TUIOption {
	return TUIOption{applyFunc: func(c *TUIChannel) error {
		c.messageChan = ch
		return nil
	}}
}

// WithLoggerService sets the logger service.
func WithLoggerService(logger logger.LoggerService) TUIOption {
	return TUIOption{applyFunc: func(c *TUIChannel) error {
		c.logger = logger
		return nil
	}}
}

// WithMarkdownRenderer sets the markdown renderer.
func WithMarkdownRenderer(renderer markdown.Renderer) TUIOption {
	return TUIOption{applyFunc: func(c *TUIChannel) error {
		c.renderer = renderer
		return nil
	}}
}
```

- [ ] **Step 4: Add getter methods to TUIChannel for testing**

```go
// pkg/tui/channel.go
// Add these getter methods at the end of the file:

// GetMessageChan returns the message channel (for testing).
func (c *TUIChannel) GetMessageChan() chan<- channel.Message {
	return c.messageChan
}

// GetLogger returns the logger service (for testing).
func (c *TUIChannel) GetLogger() logger.LoggerService {
	return c.logger
}

// GetRenderer returns the markdown renderer (for testing).
func (c *TUIChannel) GetRenderer() markdown.Renderer {
	return c.renderer
}
```

- [ ] **Step 5: Update TUIChannel struct to support options**

```go
// pkg/tui/channel.go
// Update TUIChannel struct:

type TUIChannel struct {
	id          uuid.UUID
	messageChan chan<- channel.Message
	agentID     uuid.UUID
	agentRole   string
	logger      logger.LoggerService    // Add this
	renderer    markdown.Renderer       // Add this
	executor    shared.Agent            // Add this for Start method
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./pkg/tui/... -v -run TestTUIOption`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/tui/options.go pkg/tui/options_test.go pkg/tui/channel.go
git commit -m "feat(tui): add TUIOption type and functional option constructors"
```

---

### Task 5: Update TUIChannel Constructor

**Files:**
- Modify: `pkg/tui/channel.go` (update NewTUIChannel to accept options)

- [ ] **Step 1: Write failing test for options-based constructor**

```go
// pkg/tui/channel_test.go
func TestNewTUIChannel_WithOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockRenderer := markdown.NewMockRenderer(ctrl)
	msgChan := make(chan channel.Message, 10)

	ch := NewTUIChannel(
		WithMessageChan(msgChan),
		WithLoggerService(mockLogger),
		WithMarkdownRenderer(mockRenderer),
	)

	assert.NotNil(t, ch)
	assert.Equal(t, msgChan, ch.GetMessageChan())
	assert.Equal(t, mockLogger, ch.GetLogger())
	assert.Equal(t, mockRenderer, ch.GetRenderer())
}

func TestNewTUIChannel_NoOptions(t *testing.T) {
	ch := NewTUIChannel()
	assert.NotNil(t, ch)
	// Defaults should be nil/zero values
	assert.Nil(t, ch.GetMessageChan())
	assert.Nil(t, ch.GetLogger())
	assert.Nil(t, ch.GetRenderer())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/tui/... -v -run TestNewTUIChannel_WithOptions`
Expected: FAIL because NewTUIChannel doesn't accept variadic options

- [ ] **Step 3: Update NewTUIChannel to accept options**

```go
// pkg/tui/channel.go
// Replace the existing NewTUIChannel function:

// NewTUIChannel creates a new TUIChannel instance with optional configuration.
// Options can be provided to configure the channel's dependencies.
func NewTUIChannel(opts ...TUIOption) *TUIChannel {
	ch := &TUIChannel{
		id:        uuid.New(),
		agentID:   uuid.Nil, // Will be set by SetAgentInfo
		agentRole: "assistant",
		// messageChan, logger, renderer default to nil
	}

	for _, opt := range opts {
		opt.Apply(ch) // TUIOption.Apply handles the type assertion
	}

	return ch
}
```

**Note:** Keep the old `NewTUIChannel(messageChan)` for backward compatibility during transition, or update all call sites. Check usage first.

- [ ] **Step 4: Check for existing usage of NewTUIChannel**

Run: `grep -r "NewTUIChannel" --include="*.go" pkg/ cmd/`
Expected: Find all call sites to update

- [ ] **Step 5: Update existing call sites if needed**

If there are existing call sites, update them to use options:
```go
// Old: NewTUIChannel(msgChan)
// New: NewTUIChannel(WithMessageChan(msgChan))
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./pkg/tui/... -v -run TestNewTUIChannel`
Expected: PASS

- [ ] **Step 7: Run all TUI tests to ensure no breakage**

Run: `go test ./pkg/tui/... -v`
Expected: All PASS

- [ ] **Step 8: Commit**

```bash
git add pkg/tui/channel.go pkg/tui/channel_test.go
git commit -m "refactor(tui): update NewTUIChannel to accept options pattern"
```

---

### Task 6: Create TUI Channel Registration

**Files:**
- Create: `pkg/tui/registration.go`
- Test: `pkg/tui/registration_test.go`

- [ ] **Step 1: Write failing test for RegisterChannels**

```go
// pkg/tui/registration_test.go
package tui

import (
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterChannels(t *testing.T) {
	injector := do.New()

	err := RegisterChannels(injector)
	require.NoError(t, err)

	// Verify the factory was registered
	factory, err := do.InvokeNamed[func(opts ...channel.ChannelOption) (channel.Channel, error)](injector, "channel_tui")
	assert.NoError(t, err)
	assert.NotNil(t, factory)

	// Verify factory creates TUIChannel
	ch, err := factory()
	assert.NoError(t, err)
	assert.NotNil(t, ch)

	// Type check
	_, ok := ch.(*TUIChannel)
	assert.True(t, ok, "Factory should create *TUIChannel")
}

func TestRegisterChannels_Duplicate(t *testing.T) {
	injector := do.New()

	// First registration should succeed
	err := RegisterChannels(injector)
	require.NoError(t, err)

	// Second registration should fail (DI constraint)
	err = RegisterChannels(injector)
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/tui/... -v -run TestRegisterChannels`
Expected: FAIL with "undefined: RegisterChannels"

- [ ] **Step 3: Implement RegisterChannels function**

```go
// pkg/tui/registration.go
package tui

import (
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/samber/do/v2"
)

// RegisterChannels registers the TUI channel with the DI container.
// Call this during application initialization before creating services.
func RegisterChannels(injector do.Injector) error {
	do.ProvideNamed(injector, "channel_tui", func(opts ...channel.ChannelOption) (channel.Channel, error) {
		// Convert channel.ChannelOption to TUIOption and create channel
		ch := NewTUIChannel()
		for _, opt := range opts {
			if err := opt.Apply(ch); err != nil {
				return nil, err
			}
		}
		return ch, nil
	})
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/tui/... -v -run TestRegisterChannels`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/tui/registration.go pkg/tui/registration_test.go
git commit -m "feat(tui): add RegisterChannels function for DI registration"
```

---

## Chunk 3: ChannelFacade Extensions

### Task 7: Add Provider Discovery to ChannelFacade

**Files:**
- Modify: `pkg/channel/facade.go` (add providers field and DiscoverProviders method)
- Test: `pkg/channel/facade_discovery_test.go`

- [ ] **Step 1: Write failing test for provider discovery**

```go
// pkg/channel/facade_discovery_test.go
package channel

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDiscoverProviders_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof("Discovered channel: %s", "test").Times(1)

	// Create injector with registered channel
	injector := do.New()
	do.ProvideNamed(injector, "channel_test", func(opts ...ChannelOption) (Channel, error) {
		return &mockChannel{id: uuid.New()}, nil
	})

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)
	assert.NotNil(t, facade.providers)
	assert.Contains(t, facade.providers, ChannelIdentifier("test"))
}

func TestDiscoverProviders_NoChannels(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Warn("No channel providers discovered - channels may not be available").Times(1)

	// Create empty injector
	injector := do.New()

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)
	assert.NotNil(t, facade.providers)
	assert.Empty(t, facade.providers)
}

func TestDiscoverProviders_MultipleChannels(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).Times(2)

	// Create injector with multiple channels
	injector := do.New()
	do.ProvideNamed(injector, "channel_tui", func(opts ...ChannelOption) (Channel, error) {
		return &mockChannel{id: uuid.New()}, nil
	})
	do.ProvideNamed(injector, "channel_acp", func(opts ...ChannelOption) (Channel, error) {
		return &mockChannel{id: uuid.New()}, nil
	})

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)
	assert.Len(t, facade.providers, 2)
	assert.Contains(t, facade.providers, ChannelIdentifier("tui"))
	assert.Contains(t, facade.providers, ChannelIdentifier("acp"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/channel/... -v -run TestDiscoverProviders`
Expected: FAIL with "undefined: providers", "method DiscoverProviders not defined"

- [ ] **Step 3: Add providers field to channelFacadeImpl**

```go
// pkg/channel/facade.go
// Update the struct:

type channelFacadeImpl struct {
	mu             sync.RWMutex
	channels       map[uuid.UUID]Channel
	commandManager command.Manager
	registry       registry.AgentRegistry
	agentFactory   shared.AgentFactory
	sessionManager session.SessionManager
	logger         logger.LoggerService
	providers      map[ChannelIdentifier]ChannelFactory // Add this
}
```

- [ ] **Step 4: Implement DiscoverProviders method**

```go
// pkg/channel/facade.go
// Add this method to channelFacadeImpl:

// DiscoverProviders scans the DI container for channel providers.
// Looks for named providers matching the "channel_*" pattern.
func (f *channelFacadeImpl) DiscoverProviders(injector do.Injector) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.providers = make(map[ChannelIdentifier]ChannelFactory)

	// NOTE: samber/do/v2 doesn't expose ListServices directly
	// We need to iterate known channel names or use reflection
	// For now, implement a simple approach with known channels
	// This can be extended with reflection if needed

	knownChannels := []string{"tui", "acp"} // Can be extended
	foundAny := false

	for _, name := range knownChannels {
		providerName := "channel_" + name
		factory, err := do.InvokeNamed[ChannelFactory](injector, providerName)
		if err != nil {
			// Provider not registered, skip
			continue
		}

		identifier := ChannelIdentifier(name)
		f.providers[identifier] = factory
		f.logger.Infof("Discovered channel: %s", identifier)
		foundAny = true
	}

	if !foundAny {
		f.logger.Warn("No channel providers discovered - channels may not be available")
	}

	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/channel/... -v -run TestDiscoverProviders`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/channel/facade.go pkg/channel/facade_discovery_test.go
git commit -m "feat(channel): add DiscoverProviders method for channel discovery"
```

---

### Task 8: Add CreateChannel Method to ChannelFacade

**Files:**
- Modify: `pkg/channel/facade.go` (add CreateChannel method)
- Test: `pkg/channel/facade_create_test.go`

- [ ] **Step 1: Write failing tests for CreateChannel**

```go
// pkg/channel/facade_create_test.go
package channel

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateChannel_ValidIdentifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.ProvideNamed(injector, "channel_test", func(opts ...ChannelOption) (Channel, error) {
		return &mockChannel{id: uuid.New()}, nil
	})

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)

	ch, err := facade.CreateChannel(ChannelIdentifier("test"))
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func TestCreateChannel_UnknownIdentifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any()).AnyTimes()

	facade := &channelFacadeImpl{
		channels:  make(map[uuid.UUID]Channel),
		logger:    mockLogger,
		providers: make(map[ChannelIdentifier]ChannelFactory),
	}

	_, err := facade.CreateChannel(ChannelIdentifier("unknown"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown channel identifier")
	assert.Contains(t, err.Error(), "test")
}

func TestCreateChannel_WithOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()

	// Track if option was applied
	optionApplied := false

	injector := do.New()
	do.ProvideNamed(injector, "channel_test", func(opts ...ChannelOption) (Channel, error) {
		ch := &mockChannel{id: uuid.New()}
		for _, opt := range opts {
			if err := opt.Apply(ch); err != nil {
				return nil, err
			}
			optionApplied = true
		}
		return ch, nil
	})

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)

	opt := &mockOption{applyFunc: func(c Channel) error { return nil }}
	ch, err := facade.CreateChannel(ChannelIdentifier("test"), opt)

	assert.NoError(t, err)
	assert.NotNil(t, ch)
	assert.True(t, optionApplied, "Option should be applied")
}

func TestCreateChannel_OptionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.ProvideNamed(injector, "channel_test", func(opts ...ChannelOption) (Channel, error) {
		return &mockChannel{id: uuid.New()}, nil
	})

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)

	// Option that returns error
	badOpt := &mockOption{applyFunc: func(c Channel) error {
		return assert.AnError
	}}

	_, err = facade.CreateChannel(ChannelIdentifier("test"), badOpt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to apply option")
}

func TestCreateChannel_FactoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.ProvideNamed(injector, "channel_test", func(opts ...ChannelOption) (Channel, error) {
		return nil, assert.AnError
	})

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)

	_, err = facade.CreateChannel(ChannelIdentifier("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create channel")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/channel/... -v -run TestCreateChannel`
Expected: FAIL with "method CreateChannel not defined"

- [ ] **Step 3: Implement CreateChannel method**

```go
// pkg/channel/facade.go
// Add this method to channelFacadeImpl:

// CreateChannel creates a channel instance by identifier with options.
func (f *channelFacadeImpl) CreateChannel(identifier ChannelIdentifier, opts ...ChannelOption) (Channel, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	factory, ok := f.providers[identifier]
	if !ok {
		// Provide helpful error listing available channels
		available := make([]string, 0, len(f.providers))
		for id := range f.providers {
			available = append(available, string(id))
		}
		return nil, fmt.Errorf("unknown channel identifier: %s (available: %v)", identifier, available)
	}

	ch, err := factory(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel %s: %w", identifier, err)
	}

	// Apply options with error handling
	for _, opt := range opts {
		if err := opt.Apply(ch); err != nil {
			return nil, fmt.Errorf("failed to apply option to channel %s: %w", identifier, err)
		}
	}

	return ch, nil
}
```

- [ ] **Step 4: Update ChannelFacade interface**

```go
// pkg/channel/interface.go
// Add to ChannelFacade interface:

type ChannelFacade interface {
	shared.LogForwarder

	// DisplayMessage sends a message to all registered channels
	DisplayMessage(msg Message)

	// SubmitInput handles user input from any channel
	SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (InputResult, error)

	// CancelInput cancels an in-flight input for the given session
	CancelInput(sessionID string) error

	// RegisterChannel adds a channel to receive events
	RegisterChannel(channel Channel) error

	// UnregisterChannel removes a channel
	UnregisterChannel(channelID uuid.UUID) error

	// NotifyAgentLifecycle broadcasts agent lifecycle event
	NotifyAgentLifecycle(agentID uuid.UUID, role string, added bool)

	// DiscoverProviders scans DI for channel providers
	DiscoverProviders(injector do.Injector) error

	// CreateChannel creates a channel instance by identifier with options
	CreateChannel(identifier ChannelIdentifier, opts ...ChannelOption) (Channel, error)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/channel/... -v -run TestCreateChannel`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/channel/facade.go pkg/channel/facade_create_test.go pkg/channel/interface.go
git commit -m "feat(channel): add CreateChannel method for abstract channel instantiation"
```

---

## Chunk 4: ApplicationService Integration

### Task 9: Update ApplicationService to Use ChannelFactory

**Files:**
- Modify: `pkg/app/service.go` (add runChannel, remove runTUI/runInteractiveLoop)
- Test: `pkg/app/service_channel_test.go`

- [ ] **Step 1: Write failing test for runChannel method**

```go
// pkg/app/service_channel_test.go
package app

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/tui"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRunChannel_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockRegistry := NewMockFlowRegistry(ctrl)
	mockFSM := NewMockFileStateManager(ctrl)

	// Mock channel that doesn't error
	mockCh := &mockStarterChannel{id: uuid.New()}
	mockFacade.EXPECT().CreateChannel(tui.Identifier, gomock.Any()).Return(mockCh, nil)
	mockFacade.EXPECT().RegisterChannel(mockCh).Return(nil)
	mockCh.EXPECT().Start(ctx).Return(nil)

	svc := &applicationServiceImpl{
		logService:       mockLogger,
		channelFacade:    mockFacade,
		flowRegistry:     mockRegistry,
		fsm:              mockFSM,
	}

	err := svc.runChannel(ctx, tui.Identifier)
	assert.NoError(t, err)
}

func TestRunChannel_CreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockRegistry := NewMockFlowRegistry(ctrl)
	mockFSM := NewMockFileStateManager(ctrl)

	mockFacade.EXPECT().CreateChannel(tui.Identifier, gomock.Any()).
		Return(nil, assert.AnError)

	svc := &applicationServiceImpl{
		logService:    mockLogger,
		channelFacade: mockFacade,
		flowRegistry:  mockRegistry,
		fsm:           mockFSM,
	}

	err := svc.runChannel(ctx, tui.Identifier)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create channel")
}

func TestRunChannel_RegisterError_Cleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockRegistry := NewMockFlowRegistry(ctrl)
	mockFSM := NewMockFileStateManager(ctrl)

	mockCh := &mockStarterChannel{id: uuid.New()}
	mockFacade.EXPECT().CreateChannel(tui.Identifier, gomock.Any()).Return(mockCh, nil)
	mockFacade.EXPECT().RegisterChannel(mockCh).Return(assert.AnError)
	mockFacade.EXPECT().UnregisterChannel(mockCh.ID()).Return(nil)

	svc := &applicationServiceImpl{
		logService:    mockLogger,
		channelFacade: mockFacade,
		flowRegistry:  mockRegistry,
		fsm:           mockFSM,
	}

	err := svc.runChannel(ctx, tui.Identifier)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register channel")
}

// mockStarterChannel for testing
type mockStarterChannel struct {
	id        uuid.UUID
	startCall func(ctx context.Context) error
}

func (m *mockStarterChannel) ID() uuid.UUID                              { return m.id }
func (m *mockStarterChannel) OnMessage(msg channel.Message)               {}
func (m *mockStarterChannel) OnLog(entry shared.LogEntry)                 {}
func (m *mockStarterChannel) OnAgentLifecycle(event channel.AgentLifecycleEvent) {}
func (m *mockStarterChannel) Start(ctx context.Context) error {
	if m.startCall != nil {
		return m.startCall(ctx)
	}
	return nil
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/app/... -v -run TestRunChannel`
Expected: FAIL with "undefined: runChannel"

- [ ] **Step 3: Implement runChannel method**

```go
// pkg/app/service.go
// Add this method to applicationServiceImpl:

// runChannel creates and runs a channel by identifier.
func (p *applicationServiceImpl) runChannel(ctx context.Context, identifier channel.ChannelIdentifier, opts ...channel.ChannelOption) error {
	ch, err := p.channelFacade.CreateChannel(identifier, opts...)
	if err != nil {
		return fmt.Errorf("failed to create channel %s: %w", identifier, err)
	}

	if err := p.channelFacade.RegisterChannel(ch); err != nil {
		return fmt.Errorf("failed to register channel %s: %w", identifier, err)
	}

	// Ensure cleanup on failure
	defer func() {
		if err != nil {
			p.channelFacade.UnregisterChannel(ch.ID())
		}
	}()

	// Start channel lifecycle if supported
	if starter, ok := ch.(channel.ChannelStarter); ok {
		return starter.Start(ctx)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/app/... -v -run TestRunChannel`
Expected: PASS

- [ ] **Step 5: Update Run method to use runChannel**

```go
// pkg/app/service.go
// Find the Run method and replace the TUI fallback:

func (p *applicationServiceImpl) Run(ctx context.Context) error {
	// Create .gollum directory first (this initializes p.gollumDir)
	if err := p.ensureGollumDirectory(); err != nil {
		return fmt.Errorf("failed to create .gollum directory: %w", err)
	}

	// Enable file logging (LoggerService handles logs/ subdir and cleanup)
	if err := p.logService.EnableFileLogging(p.gollumDir, p.sessionID); err != nil {
		return fmt.Errorf("failed to enable file logging: %w", err)
	}
	defer func() {
		if err := p.logService.CloseFileLogging(); err != nil {
			p.logService.Warnf("failed to close file logging: %v", err)
		}
	}()

	// Prime FileStateManager
	if err := p.primeFileStateManager(ctx); err != nil {
		return err
	}

	// Check for default flow first
	defaultFlow, err := p.flowRegistry.GetDefaultFlow()
	if err == nil && defaultFlow != nil {
		// Default flow found, execute it using flow.Execute()
		p.logService.Infof("Default flow found: %s", defaultFlow.Name)
		result, err := defaultFlow.Execute(ctx, make(map[string]any))
		if err != nil {
			return fmt.Errorf("default flow execution failed: %w", err)
		}
		// Display flow outputs
		if len(result.Outputs) > 0 {
			p.logService.Info("Flow outputs:")
			for name, value := range result.Outputs {
				p.logService.Infof("  %s: %v", name, value)
			}
		}
		p.logService.Info("Default flow completed successfully")
		return nil
	}
	p.logService.Info("no default flow found -> run tui")

	// No default flow, run TUI channel
	return p.runChannel(ctx, tui.Identifier,
		tui.WithLoggerService(p.logService),
		tui.WithMarkdownRenderer(p.markdownRenderer),
	)
}
```

- [ ] **Step 6: Add import for tui package**

```go
// pkg/app/service.go
// Add to imports:
import (
	// ... existing imports ...
	"github.com/denkhaus/gollum/pkg/tui"
)
```

- [ ] **Step 7: Remove old runTUI and runInteractiveLoop methods**

Find and remove:
- `func (p *applicationServiceImpl) runTUI(ctx context.Context) error`
- `func (p *applicationServiceImpl) runInteractiveLoop(ctx context.Context, agent shared.Agent) error`

- [ ] **Step 8: Run all app tests**

Run: `go test ./pkg/app/... -v`
Expected: All PASS (some may need updating)

- [ ] **Step 9: Commit**

```bash
git add pkg/app/service.go pkg/app/service_channel_test.go
git commit -m "refactor(app): use channel factory instead of direct TUI instantiation"
```

---

### Task 10: Update ApplicationService.NewService to Call DiscoverProviders

**Files:**
- Modify: `pkg/app/service.go` (update NewService)
- Test: update existing tests

- [ ] **Step 1: Update NewService to discover channels**

```go
// pkg/app/service.go
// Find NewService function and add provider discovery:

func NewService(injector do.Injector) (ApplicationService, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm := do.MustInvoke[state.FileStateManager](injector)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)
	promptMgr := do.MustInvoke[manager.PromptManager](injector)
	agentFactory := do.MustInvoke[shared.AgentFactory](injector)
	markdownRenderer := do.MustInvoke[markdown.Renderer](injector)
	workspaceService := do.MustInvoke[workspace.Service](injector)
	mcpRegistry := do.MustInvoke[mcpregistry.MCPRegistry](injector)
	channelFacade := do.MustInvoke[channel.ChannelFacade](injector)
	flowExecutorService := do.MustInvoke[executor.FlowExecutorService](injector)
	flowRegistry := do.MustInvoke[flowregistry.FlowRegistry](injector)

	// Discover channel providers from DI
	if err := channelFacade.DiscoverProviders(injector); err != nil {
		return nil, fmt.Errorf("failed to discover channel providers: %w", err)
	}

	return &applicationServiceImpl{
		sessionID:        uuid.New(),
		logService:       logService,
		fsm:              fsm,
		agentRegistry:    agentRegistry,
		workspaceService: workspaceService,
		promptMgr:        promptMgr,
		agentFactory:     agentFactory,
		markdownRenderer: markdownRenderer,
		mcpRegistry:      mcpRegistry,
		channelFacade:    channelFacade,
		flowExecutorService: flowExecutorService,
		flowRegistry:        flowRegistry,
	}, nil
}
```

- [ ] **Step 2: Update tests to mock DiscoverProviders**

Find all tests that create applicationServiceImpl and add:
```go
mockFacade.EXPECT().DiscoverProviders(gomock.Any()).Return(nil).Times(1)
```

- [ ] **Step 3: Run tests**

Run: `go test ./pkg/app/... -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add pkg/app/service.go
git commit -m "feat(app): discover channel providers during service initialization"
```

---

## Chunk 5: Main.go Integration

### Task 11: Add Channel Registration to main.go

**Files:**
- Modify: `cmd/gollum/main.go`
- Test: manual testing required

- [ ] **Step 1: Read current main.go**

Run: `head -50 cmd/gollum/main.go`
Understand the current structure

- [ ] **Step 2: Add registerChannels function**

```go
// cmd/gollum/main.go
// Add after imports, before main:

// registerChannels registers all available channels with the DI container.
func registerChannels(injector do.Injector) error {
	if err := tui.RegisterChannels(injector); err != nil {
		return fmt.Errorf("failed to register TUI channel: %w", err)
	}
	// Add other channels here as they become available
	// if err := acp.RegisterChannels(injector); err != nil {
	//     return fmt.Errorf("failed to register ACP channel: %w", err)
	// }
	return nil
}
```

- [ ] **Step 3: Add import for tui package**

```go
// cmd/gollum/main.go
// Add to imports:
import (
	// ... existing imports ...
	"github.com/denkhaus/gollum/pkg/tui"
)
```

- [ ] **Step 4: Call registerChannels in main**

```go
// cmd/gollum/main.go
// In main function, after creating injector:

func main() {
	ctx := context.Background()
	container := di.NewContainer()
	injector := container.RegisterServices(ctx)
	defer container.Shutdown()

	// Register all channels
	if err := registerChannels(injector); err != nil {
		log.Fatalf("Failed to register channels: %v", err)
	}

	// ... rest of main ...
}
```

- [ ] **Step 5: Test compilation**

Run: `go build ./cmd/gollum/`
Expected: Success

- [ ] **Step 6: Manual smoke test**

Run: `./gollum --help`
Expected: Help message displayed

- [ ] **Step 7: Commit**

```bash
git add cmd/gollum/main.go
git commit -m "feat(main): register channels during application initialization"
```

---

## Chunk 6: TUI Channel Start Method

### Task 12: Implement TUIChannel.Start Method

**Files:**
- Modify: `pkg/tui/channel.go` (add Start method)
- Test: `pkg/tui/channel_start_test.go`

- [ ] **Step 1: Write failing test for Start method**

```go
// pkg/tui/channel_start_test.go
package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTUIChannel_Start_ContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockExecutor := shared.NewMockAgent(ctrl)

	// Cancel immediately to exit TUI
	cancel()

	ch := NewTUIChannel(
		WithLoggerService(mockLogger),
	)
	ch.executor = mockExecutor

	// Start should return due to context cancellation
	err := ch.Start(ctx)
	// TUI should exit cleanly when context is cancelled
	assert.NoError(t, err)
}

func TestTUIChannel_Start_MissingDependencies(t *testing.T) {
	ctx := context.Background()

	ch := NewTUIChannel()
	// No logger, renderer, or executor set

	err := ch.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/tui/... -v -run TestTUIChannel_Start`
Expected: FAIL with "method Start not defined"

- [ ] **Step 3: Implement Start method**

```go
// pkg/tui/channel.go
// Add this method to TUIChannel:

// Start runs the TUI program with this channel's configuration.
func (c *TUIChannel) Start(ctx context.Context) error {
	// Validate required dependencies
	if c.logger == nil {
		return fmt.Errorf("missing logger service")
	}
	if c.renderer == nil {
		return fmt.Errorf("missing markdown renderer")
	}
	if c.executor == nil {
		return fmt.Errorf("missing agent executor")
	}

	// Create the TUI model
	model := NewModel(ctx, c.executor)
	WithTUIChannel()(&model)
	WithLoggerService(c.logger)(&model)
	WithMarkdownRenderer(c.renderer)(&model)

	// Create and run the TUI program
	prog := tea.NewProgram(model,
		tea.WithContext(ctx),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := prog.Run()
	return err
}
```

- [ ] **Step 4: Add required imports**

```go
// pkg/tui/channel.go
// Add to imports:
import (
	// ... existing imports ...
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)
```

- [ ] **Step 5: Update NewTUIChannel to initialize struct properly**

```go
// pkg/tui/channel.go
// Update the struct initialization in NewTUIChannel:

func NewTUIChannel(opts ...TUIOption) *TUIChannel {
	ch := &TUIChannel{
		id:        uuid.New(),
		agentID:   uuid.Nil,
		agentRole: "assistant",
		// Other fields remain nil until options are applied
	}

	for _, opt := range opts {
		// TUIOption.Apply handles the type assertion internally
		_ = opt.Apply(ch)
	}

	return ch
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./pkg/tui/... -v -run TestTUIChannel_Start`
Expected: PASS

- [ ] **Step 7: Integration test - verify TUI still works**

Run: Manually test the application
```bash
go build ./cmd/gollum/
./gollum
```
Expected: TUI starts normally

- [ ] **Step 8: Commit**

```bash
git add pkg/tui/channel.go pkg/tui/channel_start_test.go
git commit -m "feat(tui): add Start method for channel lifecycle management"
```

---

## Chunk 7: Cleanup and Verification

### Task 13: Remove TODO Comments

**Files:**
- Modify: `pkg/app/service.go` (remove TODOs)

- [ ] **Step 1: Find all TODOs in service.go**

Run: `grep -n "TODO" pkg/app/service.go`
Expected: Find the TODO comments about TUI instantiation

- [ ] **Step 2: Remove the TODO comments**

Remove these lines from `runTUI` and `runInteractiveLoop`:
```go
// TODO: this should not be part of the application service.
// The application service should instantiate a channel through the channel facade instead of instantiating the TUI loop itself.
// The TUI itself is only one of many possible channels to interact with.
```

Note: These methods should have been removed in Task 9. If they still exist, remove them now.

- [ ] **Step 3: Verify no TODOs remain**

Run: `grep -n "TODO" pkg/app/service.go`
Expected: No results

- [ ] **Step 4: Commit**

```bash
git add pkg/app/service.go
git commit -m "refactor(app): remove resolved TODO comments"
```

---

### Task 14: Verify No Circular Dependencies

**Files:**
- None (verification task)

- [ ] **Step 1: Check import graph**

Run: `go mod graph | grep gollum | sort | uniq`
Expected: No cycles visible

- [ ] **Step 2: Build all packages**

Run: `go build ./...`
Expected: Success, no import cycle errors

- [ ] **Step 3: Verify dependency structure**

Confirm:
- `pkg/channel` does NOT import `pkg/tui` or `pkg/acp`
- `pkg/tui` imports `pkg/channel`
- `pkg/acp` imports `pkg/channel`
- `pkg/app` imports `pkg/channel` and identifier constants from `pkg/tui`, `pkg/acp`

- [ ] **Step 4: Document dependency structure**

If all checks pass, no commit needed

---

### Task 15: Run Full Test Suite

**Files:**
- None (verification task)

- [ ] **Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

- [ ] **Step 2: Check coverage**

Run: `go test ./pkg/channel/... -coverprofile=/tmp/channel_coverage.out && go tool cover -func=/tmp/channel_coverage.out | tail -20`
Expected: Coverage >= 80%

Run: `go test ./pkg/tui/... -coverprofile=/tmp/tui_coverage.out && go tool cover -func=/tmp/tui_coverage.out | tail -20`
Expected: Coverage >= 80%

Run: `go test ./pkg/app/... -coverprofile=/tmp/app_coverage.out && go tool cover -func=/tmp/app_coverage.out | tail -20`
Expected: Coverage >= 80%

- [ ] **Step 3: Fix any coverage gaps**

If coverage < 80%, add tests for uncovered code

- [ ] **Step 4: Commit any new tests**

```bash
git add <test files>
git commit -m "test: improve coverage to meet 80% threshold"
```

---

### Task 16: Manual Verification

**Files:**
- None (verification task)

- [ ] **Step 1: Build application**

Run: `go build ./cmd/gollum/`
Expected: Success

- [ ] **Step 2: Test TUI starts**

Run: `./gollum`
Expected: TUI starts normally

- [ ] **Step 3: Test TUI functionality**

In TUI:
- Send a message
- Verify response
- Exit with Ctrl+D or Ctrl+C

Expected: All functions work

- [ ] **Step 4: Test error handling**

Run: `./gollum --invalid-flag`
Expected: Helpful error message

- [ ] **Step 5: Verify log files created**

Run: `ls -la .gollum/logs/`
Expected: Log files present

---

## Task 17: Create Documentation

**Files:**
- Create: `docs/channel-architecture.md` (or update existing docs)

- [ ] **Step 1: Create channel architecture documentation**

```markdown
# Channel Architecture

## Overview

Gollum uses a channel-based architecture to support multiple user interfaces (TUI, ACP, web, CLI, etc.) through a unified factory pattern.

## Channel Pattern

### Registering a Channel

Each channel package registers itself with the DI container:

```go
// In pkg/mychannel/registration.go
func RegisterChannels(injector do.Injector) error {
    do.ProvideNamed(injector, "channel_mychannel", func(opts ...channel.ChannelOption) (channel.Channel, error) {
        return NewMyChannel(opts...)
    })
    return nil
}
```

### Creating a Channel

ApplicationService creates channels through the ChannelFacade:

```go
ch, err := channelFacade.CreateChannel(mychannel.Identifier,
    mychannel.WithConfig(value),
)
```

### Channel Lifecycle

Channels implement the `ChannelStarter` interface to manage their own lifecycle:

```go
type ChannelStarter interface {
    channel.Channel
    Start(ctx context.Context) error
}
```

## Adding a New Channel

1. Create channel package: `pkg/mychannel/`
2. Implement `channel.Channel` interface
3. Define `Identifier` constant
4. Implement `ChannelOption` pattern
5. Create `RegisterChannels()` function
6. Register in `cmd/gollum/main.go`

## Existing Channels

- **TUI** (`pkg/tui/`): Terminal user interface
- **ACP** (`pkg/acp/`): Advanced Context Provider integration

## References

- Design: `docs/superpowers/specs/2026-04-12-channel-factory-refactor-design.md`
- Implementation: `docs/superpowers/plans/2026-04-12-channel-factory-refactor.md`
```

- [ ] **Step 2: Update main README if needed**

Add section about channel architecture if not present

- [ ] **Step 3: Commit documentation**

```bash
git add docs/channel-architecture.md README.md
git commit -m "docs: add channel architecture documentation"
```

---

## Final Verification Checklist

- [ ] All tests pass (`go test ./...`)
- [ ] Coverage >= 80% for modified packages
- [ ] TUI starts and functions normally
- [ ] No circular dependencies
- [ ] No TODO comments remain in service.go
- [ ] Documentation updated
- [ ] Code follows Go programming guidelines
- [ ] Code follows DI guidelines
- [ ] Code follows testing guidelines
- [ ] All commits have descriptive messages
- [ ] Git history is clean

---

## Rollback Plan

If issues arise:

1. **Identify the breaking commit** - Use `git bisect` if needed
2. **Revert to last working state** - `git revert <commit>`
3. **Fix the issue** - Address the root cause
4. **Re-apply changes** - Move forward with fixes

Common rollback scenarios:
- **TUI doesn't start**: Check `Start()` method implementation
- **Tests fail**: Check mock expectations and option application
- **Import cycles**: Review dependency structure
- **Runtime errors**: Check nil pointer dereferences in options

---

## Notes for Future Channel Implementers

1. **Always use the ChannelOption pattern** - Don't accept raw dependencies in constructors
2. **Register in main.go** - Add your `RegisterChannels` call to `registerChannels()`
3. **Implement ChannelStarter** - If your channel has a lifecycle, implement this interface
4. **Test with mocks** - Use gomock for clean testing
5. **Handle errors gracefully** - Provide helpful error messages with available channels listed
6. **Document your options** - Each `WithX()` function should have clear documentation

---

**Total Estimated Time:** 8-12 hours

**Chunks:** 7 (for incremental review and execution)
