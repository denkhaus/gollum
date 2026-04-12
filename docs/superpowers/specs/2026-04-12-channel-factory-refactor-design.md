# Channel Factory Refactoring Design

**Date:** 2026-04-12
**Status:** Draft
**Author:** AI Agent
**Reviewer:** Pending

## Overview

Refactor the ApplicationService to remove tight coupling with the TUI by implementing a channel factory pattern. This enables any number of channels (TUI, ACP, web, CLI, etc.) to be instantiated through a unified interface using dependency injection.

## Background

### Current State

The `pkg/app/service.go` file contains TODOs indicating architectural concerns:

```go
// TODO: this should not be part of the application service.
// The application service should instantiate a channel through the channel facade instead of instantiating the TUI loop itself.
// The TUI itself is only one of many possible channels to interact with.
```

**Problems:**
- ApplicationService directly imports and creates TUI components
- TUI-specific code mixed with application logic
- Adding new channels requires modifying ApplicationService
- Tight coupling prevents testing with mock channels
- Violates separation of concerns principle

### Existing Channel Infrastructure

The codebase already has:
- `pkg/channel` package with Channel interface and ChannelFacade
- `pkg/tui/channel.go` implementing TUIChannel
- `pkg/acp/service.go` implementing ACP as a channel
- DI container using `samber/do/v2`

## Goals

1. **Separation of Concerns**: ApplicationService should not know about specific channel implementations
2. **Extensibility**: Add new channels without modifying ApplicationService or ChannelFacade
3. **Testability**: Use dependency injection for clean testing with mocks
4. **No Magic Strings**: Use constants defined in channel packages
5. **No Circular Dependencies**: Careful package placement to avoid import cycles

## Design

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    main.go                              │
│  - Creates DI container                                  │
│  - Calls RegisterChannels() for each channel             │
└───────────────┬─────────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────┐
│              DI Container                                │
│  Named providers:                                         │
│  - channel_tui → TUIChannel factory                      │
│  - channel_acp → ACPChannel factory                      │
└───────────────┬─────────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────┐
│            ChannelFacade                                 │
│  - Discovers channel_* providers from DI                 │
│  - CreateChannel(identifier, opts...)                   │
└───────────────┬─────────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────┐
│         ApplicationService                               │
│  - Requests: channelFacade.CreateChannel(tui.Identifier)│
│  - No direct TUI imports                                 │
└─────────────────────────────────────────────────────────┘
```

### Component Design

#### 1. Channel Types (`pkg/channel/types.go`)

```go
// ChannelOption is the interface for channel configuration options
type ChannelOption interface {
    Apply(channel.Channel) error
}

// ChannelFactory creates a channel instance with options
type ChannelFactory func(opts ...ChannelOption) (channel.Channel, error)

// ChannelIdentifier is the const type each channel exports
type ChannelIdentifier string

// ChannelStarter is the interface for channels that manage their own lifecycle
type ChannelStarter interface {
    Start(ctx context.Context) error
}
```

#### 2. Channel Registration Pattern

Each channel package exports:

**Identifier constant:**
```go
// pkg/tui/channel.go
const Identifier = channel.ChannelIdentifier("tui")
```

**Registration function:**
```go
// pkg/tui/registration.go
func RegisterChannels(injector do.Injector) error {
    do.ProvideNamed(injector, "channel_tui", func(opts ...channel.ChannelOption) (channel.Channel, error) {
        return NewTUIChannel(opts...)
    })
    return nil
}
```

**Type-safe options:**
```go
// pkg/tui/options.go
type TUIOption struct {
    applyFunc func(*TUIChannel) error
}

func (o TUIOption) Apply(ch channel.Channel) error {
    if tuiCh, ok := ch.(*TUIChannel); ok {
        return o.applyFunc(tuiCh)
    }
    return fmt.Errorf("TUI option applied to wrong channel type")
}

func WithMessageChan(ch chan<- channel.Message) TUIOption {
    return TUIOption{applyFunc: func(c *TUIChannel) error {
        c.messageChan = ch
        return nil
    }}
}

func WithLoggerService(logger logger.LoggerService) TUIOption {
    return TUIOption{applyFunc: func(c *TUIChannel) error {
        c.logger = logger
        return nil
    }}
}

func WithMarkdownRenderer(renderer markdown.Renderer) TUIOption {
    return TUIOption{applyFunc: func(c *TUIChannel) error {
        c.renderer = renderer
        return nil
    }}
}
```

#### 3. ChannelFacade Extensions

```go
// pkg/channel/facade.go
type channelFacadeImpl struct {
    // existing fields...
    providers map[channel.ChannelIdentifier]channel.ChannelFactory
}

func (f *channelFacadeImpl) DiscoverProviders(injector do.Injector) error {
    f.providers = make(map[channel.ChannelIdentifier]channel.ChannelFactory)

    // Scan for named providers matching "channel_*"
    providers := getChannelProviders(injector)

    for identifier, factory := range providers {
        f.providers[identifier] = factory
        f.logService.Infof("Discovered channel: %s", identifier)
    }

    return nil
}

func (f *channelFacadeImpl) CreateChannel(
    identifier channel.ChannelIdentifier,
    opts ...channel.ChannelOption,
) (channel.Channel, error) {
    factory, ok := f.providers[identifier]
    if !ok {
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

    for _, opt := range opts {
        if err := opt.Apply(ch); err != nil {
            return nil, fmt.Errorf("failed to apply option to channel %s: %w", identifier, err)
        }
    }

    return ch, nil
}
```

#### 4. ApplicationService Changes

```go
// pkg/app/service.go
func (p *applicationServiceImpl) runChannel(
    ctx context.Context,
    identifier channel.ChannelIdentifier,
    opts ...channel.ChannelOption,
) error {
    ch, err := p.channelFacade.CreateChannel(identifier, opts...)
    if err != nil {
        return fmt.Errorf("failed to create channel %s: %w", identifier, err)
    }

    if err := p.channelFacade.RegisterChannel(ch); err != nil {
        return fmt.Errorf("failed to register channel %s: %w", identifier, err)
    }
    defer func() {
        if err != nil {
            p.channelFacade.UnregisterChannel(ch.ID())
        }
    }()

    if starter, ok := ch.(channel.ChannelStarter); ok {
        return starter.Start(ctx)
    }

    return nil
}

func (p *applicationServiceImpl) Run(ctx context.Context) error {
    // ... existing logging and directory setup ...

    // Check for default flow
    if defaultFlow, err := p.flowRegistry.GetDefaultFlow(); err == nil && defaultFlow != nil {
        result, err := defaultFlow.Execute(ctx, make(map[string]any))
        if err != nil {
            return fmt.Errorf("default flow execution failed: %w", err)
        }
        // Display outputs...
        return nil
    }

    // No default flow - run TUI channel
    return p.runChannel(ctx, tui.Identifier,
        tui.WithLoggerService(p.logService),
        tui.WithMarkdownRenderer(p.markdownRenderer),
    )
}
```

#### 5. TUIChannel Lifecycle

```go
// pkg/tui/channel.go
func (c *TUIChannel) Start(ctx context.Context) error {
    model := tui.NewModel(ctx, c.executor)
    tui.WithTUIChannel()(&model)
    tui.WithLoggerService(c.logger)(&model)
    tui.WithMarkdownRenderer(c.renderer)(&model)

    prog := tea.NewProgram(model,
        tea.WithContext(ctx),
        tea.WithAltScreen(),
        tea.WithMouseCellMotion(),
    )

    _, err := prog.Run()
    return err
}
```

#### 6. Main.go Setup

```go
// cmd/gollum/main.go
func main() {
    ctx := context.Background()
    container := di.NewContainer()
    injector := container.RegisterServices(ctx)

    // Register all channels
    if err := registerChannels(injector); err != nil {
        log.Fatalf("Failed to register channels: %v", err)
    }

    // Rest of main...
}

func registerChannels(injector do.Injector) error {
    if err := tui.RegisterChannels(injector); err != nil {
        return fmt.Errorf("failed to register TUI channel: %w", err)
    }
    if err := acp.RegisterChannels(injector); err != nil {
        return fmt.Errorf("failed to register ACP channel: %w", err)
    }
    return nil
}
```

### Dependency Structure

```
pkg/shared
    ↑
    ├── pkg/channel (imports shared for LogEntry)
    ↑
    ├── pkg/tui (imports channel, shared)
    ├── pkg/acp (imports channel, shared)
    └── pkg/app (imports channel, shared, tui.Identifier/acp.Identifier)
```

**No circular dependencies:**
- TUI implements ChannelOption, but only depends on channel.Channel interface
- Channel package doesn't import TUI or ACP
- App package only imports identifiers (constants), not implementations

## Error Handling

### Channel Creation Errors

```go
// Provides helpful error listing available channels
if !ok {
    available := make([]string, 0, len(f.providers))
    for id := range f.providers {
        available = append(available, string(id))
    }
    return nil, fmt.Errorf("unknown channel identifier: %s (available: %v)", identifier, available)
}
```

### Option Type Mismatch

```go
func (o TUIOption) Apply(ch channel.Channel) error {
    if tuiCh, ok := ch.(*TUIChannel); ok {
        return o.applyFunc(tuiCh)
    }
    return fmt.Errorf("TUI option applied to wrong channel type (got %T, expected *TUIChannel)", ch)
}
```

### Discovery Failures

```go
if len(providers) == 0 {
    f.logService.Warn("No channel providers discovered - channels may not be available")
}
```

### Cleanup on Failure

```go
defer func() {
    if err != nil {
        p.channelFacade.UnregisterChannel(ch.ID())
    }
}()
```

## Testing Strategy

### Mock Generation

```go
// pkg/channel/generate.go
//go:generate go run go.uber.org/mock/mockgen -source=interface.go -destination=channel_mock.go -package=channel github.com/denkhaus/gollum/pkg/channel Channel,ChannelFacade
```

### Unit Tests

**pkg/channel/facade_test.go:**
- TestCreateChannel_ValidIdentifier
- TestCreateChannel_UnknownIdentifier
- TestCreateChannel_OptionError
- TestDiscoverProviders_NoChannels

**pkg/tui/options_test.go:**
- TestTUIOption_WithMessageChan
- TestTUIOption_WithLoggerService
- TestTUIOption_WrongChannelType
- TestTUIOptions_Multiple

### Integration Tests

**pkg/app/service_integration_test.go:**
- TestApplicationService_ChannelDiscovery
- TestApplicationService_CreateTUIChannel

### Test Helpers

```go
// pkg/channel/test_helpers.go
func setupTestFacade(t *testing.T, ctrl *gomock.Controller, mockLogger *logger.MockLoggerService) *channelFacadeImpl {
    facade := NewChannelFacade(mockLogger)
    mockLogger.EXPECT().Infof(gomock.Any, gomock.Any).AnyTimes()
    mockLogger.EXPECT().Warnf(gomock.Any, gomock.Any).AnyTimes()
    return facade
}
```

## Implementation Plan

### Phase 1: Foundation (No Breaking Changes)

1. **Add channel types** (`pkg/channel/types.go`)
   - Add `ChannelOption` interface
   - Add `ChannelFactory` type
   - Add `ChannelIdentifier` type

2. **Create channel registration infrastructure**
   - `pkg/tui/registration.go` - RegisterChannels function
   - `pkg/tui/options.go` - TUIOption and functional option constructors
   - `pkg/tui/generate.go` - Add mockgen directive

3. **Extend ChannelFacade**
   - Add `DiscoverProviders(injector)` method
   - Add `CreateChannel(identifier, opts...)` method
   - Add providers map to struct

4. **Update DI container**
   - Ensure ChannelFacade is created before channels are registered
   - Test discovery mechanism

### Phase 2: Integration (Breaking Changes)

5. **Update main.go**
   - Add channel registration calls
   - Ensure registration happens before service creation

6. **Refactor ApplicationService**
   - Replace direct TUI instantiation with `channelFacade.CreateChannel()`
   - Remove `runTUI()` and `runInteractiveLoop()` methods
   - Add `runChannel()` method

7. **Implement ChannelStarter**
   - Add interface to `pkg/channel/types.go`
   - Implement `Start(ctx)` in TUIChannel
   - Move TUI lifecycle management into channel

8. **Update imports and dependencies**
   - Remove direct TUI imports from ApplicationService
   - Ensure no circular dependencies

### Phase 3: Testing and Documentation

9. **Write comprehensive tests**
   - Unit tests for ChannelFacade methods
   - Unit tests for TUI options
   - Integration tests for ApplicationService
   - Ensure >= 80% coverage

10. **Update documentation**
    - Document channel registration pattern
    - Update README with channel architecture
    - Add examples for adding new channels

### Phase 4: Verification

11. **Manual testing**
    - Run TUI and verify functionality
    - Test channel discovery
    - Test error handling

12. **Clean up**
    - Remove old TUI-specific code from ApplicationService
    - Remove unused imports
    - Verify no TODOs remain

## File Changes

### New Files

- `pkg/channel/types.go` - ChannelOption, ChannelFactory, ChannelIdentifier
- `pkg/tui/registration.go` - RegisterChannels function
- `pkg/tui/options.go` - TUIOption and constructors
- `pkg/tui/generate.go` - Mock generation directive
- `pkg/channel/test_helpers.go` - Test helper functions

### Modified Files

- `pkg/channel/facade.go` - Add discovery and creation methods
- `pkg/app/service.go` - Refactor to use channel factory
- `cmd/gollum/main.go` - Add channel registration
- `pkg/di/container.go` - Adjust initialization order if needed

## Benefits

1. **Separation of Concerns**: ApplicationService no longer knows about TUI specifics
2. **Extensibility**: Add channels without modifying existing code
3. **Testability**: Clean DI-based testing with mocks
4. **Type Safety**: Compile-time constants, no magic strings
5. **No Circular Dependencies**: Careful package placement
6. **Clean Architecture**: Each channel is self-contained

## Migration Notes

### Backward Compatibility

- **TUI remains default behavior** when no default flow exists
- Existing TUI functionality fully preserved
- No breaking changes to flow definitions or loading
- Default flow is opt-in (user creates `.gollum/flows/default/main.xml`)

### To Be Removed

- Direct TUI instantiation in ApplicationService
- `runTUI()` and `runInteractiveLoop()` methods in ApplicationService
- TUI-specific imports from ApplicationService

## Verification Checklist

- [ ] TUI still works with `gollum` command
- [ ] Channel discovery works correctly
- [ ] Error messages are helpful and informative
- [ ] All tests pass (unit + integration)
- [ ] Coverage >= 80%
- [ ] No circular dependencies
- [ ] No TODOs remain in service.go
- [ ] Manual testing confirms functionality

## References

- `pkg/channel/interface.go` - Channel interface definition
- `pkg/channel/facade.go` - ChannelFacade implementation
- `pkg/tui/channel.go` - TUIChannel implementation
- `pkg/acp/service.go` - ACPChannel implementation
- `docs/superpowers/specs/2026-03-13-cli-design.md` - Related CLI design
- `/home/denkhaus/dev/kb/guides/guide.golang.testing.md` - Testing guidelines
- `/home/denkhaus/dev/kb/guides/guide.golang.di.md` - DI guidelines
