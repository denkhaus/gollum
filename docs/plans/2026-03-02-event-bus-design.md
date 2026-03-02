# Event Bus Design

## Overview

This document describes the design for a centralized event bus that enables decoupled communication between services in the Gollum application. The event bus supports both synchronous and asynchronous event handling with retry logic for reliability.

## Motivation

The application needs a way for services to react to state changes without tight coupling. Example scenario:

1. User executes `change_directory` tool
2. WorkspaceManager updates current path
3. SkillService discovers new skills
4. All idle agents update their system prompts with new skills

Currently, this requires direct service-to-service calls. An event bus provides:
- **Decoupling**: Publishers don't know about subscribers
- **Extensibility**: Plugins can subscribe to events without modifying core
- **Testability**: Event handlers can be tested in isolation

## Architecture

### Event Flow Example

```
ChangeDirectory Tool
    ↓ publish("workspace.directory_changed")
    ├── [SYNC] WorkspaceService → saves CWD
    └── [SYNC] SkillService → discovers skills
           ↓ (if new skills) publish("skills.updated")
           └── [ASYNC] AgentRegistry → updates idle agents
```

### Sync vs Async

- **Synchronous handlers**: Block the publisher until complete. Use for critical operations that must complete before continuing.
- **Asynchronous handlers**: Run in goroutines, publisher continues immediately. Use for non-critical operations.

### Delivery Guarantee

**At-Least-Once** delivery with retry:
- Event is sent → Handler acknowledges (returns nil) → Done
- Handler returns error → Retry with exponential backoff → Until success or max retries
- Duplicates only possible if ACK is lost (rare in in-process bus)

## Package Structure

```
pkg/events/
├── bus.go           # Bus interface and SubscriptionOptions
├── event.go         # Event interface, TypedEvent struct, NewEvent()
├── generic.go       # SubscribeTyped[T], PublishTyped[T] helper functions
├── memory_bus.go    # memoryBus struct (private implementation)
├── provider.go      # NewBusProvider for DI
└── types.go         # Event type constants and Payload structs
```

## Core Interfaces

### Event Types and Payloads

```go
// pkg/events/types.go

package events

// Event types are centrally defined constants
const (
    // Workspace events
    EventDirectoryChanged = "workspace.directory_changed"

    // Skill events
    EventSkillsUpdated    = "skills.updated"
    EventSkillsDiscovered = "skills.discovered"

    // Agent events
    EventAgentSpawned     = "agent.spawned"
    EventAgentRemoved     = "agent.removed"
    EventAgentIdle        = "agent.idle"
    EventAgentBusy        = "agent.busy"

    // Configuration events
    EventConfigChanged    = "config.changed"

    // Plugin events
    EventPluginLoaded     = "plugin.loaded"
    EventPluginUnloaded   = "plugin.unloaded"
)

// Payload structs - typed for safety
type DirectoryChangedPayload struct {
    OldPath string
    NewPath string
}

type SkillsUpdatedPayload struct {
    Skills []SkillInfo
}

// ... other payload types
```

### Event Interface

```go
// pkg/events/event.go

package events

import "time"

// Event is the base interface for storage
type Event interface {
    Type() string
    Timestamp() time.Time
    Source() string
}

// TypedEvent wraps payload with type safety
type TypedEvent[T any] struct {
    eventType string
    payload   T
    timestamp time.Time
    source    string
}

func (e *TypedEvent[T]) Type() string        { return e.eventType }
func (e *TypedEvent[T]) Payload() T          { return e.payload }
func (e *TypedEvent[T]) Timestamp() time.Time { return e.timestamp }
func (e *TypedEvent[T]) Source() string      { return e.source }

// NewEvent creates a typed event
func NewEvent[T any](eventType, source string, payload T) *TypedEvent[T] {
    return &TypedEvent[T]{
        eventType: eventType,
        payload:   payload,
        timestamp: time.Now(),
        source:    source,
    }
}
```

### Bus Interface

```go
// pkg/events/bus.go

package events

import "context"

// SubscriptionOption configures how events are delivered
type SubscriptionOption func(*SubscriptionConfig)

type SubscriptionConfig struct {
    Async     bool
    Priority  int  // Higher = executed first within sync/async group
}

func WithAsync() SubscriptionOption {
    return func(c *SubscriptionConfig) { c.Async = true }
}

func WithPriority(priority int) SubscriptionOption {
    return func(c *SubscriptionConfig) { c.Priority = priority }
}

// Handler for internal storage (untyped)
type Handler func(ctx context.Context, event Event) error

// Bus is the central event dispatcher
type Bus interface {
    Subscribe(eventType string, handler Handler, opts ...SubscriptionOption) (string, error)
    Unsubscribe(subscriptionID string) error
    Publish(ctx context.Context, event Event) error
}
```

### Generic Helper Functions

```go
// pkg/events/generic.go

package events

import (
    "context"
    "fmt"
)

// TypedHandler receives typed payload - no casting needed
type TypedHandler[T any] func(ctx context.Context, payload T) error

// SubscribeTyped is a generic helper function
func SubscribeTyped[T any](bus Bus, eventType string, handler TypedHandler[T], opts ...SubscriptionOption) (string, error) {
    wrapper := func(ctx context.Context, event Event) error {
        typed, ok := event.(*TypedEvent[T])
        if !ok {
            return fmt.Errorf("unexpected event type for %s", eventType)
        }
        return handler(ctx, typed.Payload())
    }
    return bus.Subscribe(eventType, wrapper, opts...)
}

// PublishTyped is a generic helper function
func PublishTyped[T any](bus Bus, ctx context.Context, eventType, source string, payload T) error {
    return bus.Publish(ctx, NewEvent(eventType, source, payload))
}
```

### Memory Bus Implementation

```go
// pkg/events/memory_bus.go

package events

import (
    "context"
    "sort"
    "sync"
    "time"

    "github.com/denkhaus/gollum/pkg/config"
    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/google/uuid"
)

type subscription struct {
    id       string
    handler  Handler
    async    bool
    priority int
}

type memoryBus struct {
    logger logger.LoggerService
    config *config.EventsConfig
    mu     sync.RWMutex
    subs   map[string][]*subscription
}

func newMemoryBus(logger logger.LoggerService, cfg *config.EventsConfig) *memoryBus {
    return &memoryBus{
        logger: logger,
        config: cfg,
        subs:   make(map[string][]*subscription),
    }
}

func (b *memoryBus) Subscribe(eventType string, handler Handler, opts ...SubscriptionOption) (string, error) {
    cfg := &SubscriptionConfig{}
    for _, opt := range opts {
        opt(cfg)
    }

    sub := &subscription{
        id:       uuid.New().String(),
        handler:  handler,
        async:    cfg.Async,
        priority: cfg.Priority,
    }

    b.mu.Lock()
    b.subs[eventType] = append(b.subs[eventType], sub)
    b.mu.Unlock()

    return sub.id, nil
}

func (b *memoryBus) Unsubscribe(subscriptionID string) error {
    b.mu.Lock()
    defer b.mu.Unlock()

    for eventType, subs := range b.subs {
        for i, sub := range subs {
            if sub.id == subscriptionID {
                b.subs[eventType] = append(subs[:i], subs[i+1:]...)
                return nil
            }
        }
    }
    return nil
}

func (b *memoryBus) Publish(ctx context.Context, event Event) error {
    b.mu.RLock()
    subs := b.subs[event.Type()]
    b.mu.RUnlock()

    if len(subs) == 0 {
        return nil
    }

    // Sort by priority (descending)
    sorted := make([]*subscription, len(subs))
    copy(sorted, subs)
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i].priority > sorted[j].priority
    })

    // Execute sync handlers first
    var asyncHandlers []*subscription
    for _, sub := range sorted {
        if !sub.async {
            if err := b.executeWithRetry(ctx, sub, event); err != nil {
                b.logger.Error("sync handler failed",
                    "subscription", sub.id,
                    "event", event.Type(),
                    "error", err)
            }
        } else {
            asyncHandlers = append(asyncHandlers, sub)
        }
    }

    // Execute async handlers in background
    for _, sub := range asyncHandlers {
        go b.executeWithRetry(context.Background(), sub, event)
    }

    return nil
}

func (b *memoryBus) executeWithRetry(ctx context.Context, sub *subscription, event Event) error {
    var lastErr error
    delay := time.Duration(b.config.RetryDelayMs) * time.Millisecond

    for attempt := 0; attempt <= b.config.MaxRetries; attempt++ {
        if err := sub.handler(ctx, event); err != nil {
            lastErr = err
            if attempt < b.config.MaxRetries {
                b.logger.Warn("handler failed, retrying",
                    "subscription", sub.id,
                    "attempt", attempt+1,
                    "delay", delay)
                time.Sleep(delay)
                delay *= time.Duration(b.config.RetryBackoff)
                continue
            }
        }
        return nil
    }
    return lastErr
}
```

## Configuration

Add to `pkg/config/service.go`:

```go
// EventsConfig holds configuration for the event bus
type EventsConfig struct {
    MaxRetries   int `envconfig:"MAX_RETRIES" default:"3"`
    RetryDelayMs int `envconfig:"RETRY_DELAY_MS" default:"100"`
    RetryBackoff int `envconfig:"RETRY_BACKOFF" default:"2"`
}

// Add to serviceImpl struct:
type serviceImpl struct {
    // ... existing fields ...
    Events EventsConfig `envconfig:"EVENTS"`
}

// Add getter:
func (s *serviceImpl) GetEventsConfig() *EventsConfig {
    return &s.Events
}

// Add to ConfigService interface:
GetEventsConfig() *EventsConfig
```

**Environment Variables:**
```bash
GOLLUM_EVENTS_MAX_RETRIES=3
GOLLUM_EVENTS_RETRY_DELAY_MS=100
GOLLUM_EVENTS_RETRY_BACKOFF=2
```

## DI Integration

### Provider

```go
// pkg/events/provider.go

package events

import (
    "github.com/denkhaus/gollum/pkg/config"
    "github.com/denkhaus/gollum/pkg/di"
    "github.com/samber/do"
)

func NewBusProvider(i *do.Injector) (Bus, error) {
    logger := di.MustInvokeLogger(i)
    cfgService := do.MustInvoke[config.ConfigService](i)
    return newMemoryBus(logger, cfgService.GetEventsConfig()), nil
}
```

### Registration

```go
// pkg/di/container.go

func registerServices(injector *do.Injector) {
    // ... existing registrations ...

    // Event Bus - must be registered before services that use it
    do.Provide(injector, events.NewBusProvider)
}
```

## Usage Examples

### Publishing Events (Tool)

```go
// pkg/tools/change_directory.go

func (t *ChangeDirectoryTool) Execute(ctx context.Context, input string) (string, error) {
    oldPath, _ := os.Getwd()

    // ... validation and directory change logic ...

    err := events.PublishTyped(t.eventBus, ctx,
        events.EventDirectoryChanged,
        shared.ToolNameChangeDirectory,
        events.DirectoryChangedPayload{
            OldPath: oldPath,
            NewPath: newPath,
        },
    )

    if err != nil {
        t.logService.Warn("failed to publish directory changed event", "error", err)
    }

    return fmt.Sprintf("Changed directory to %s", newPath), nil
}
```

### Subscribing to Events (Service)

Each service defines its own source name and subscribes during initialization:

```go
// pkg/workspace/service.go

package workspace

const SourceName = "workspace_service"

type serviceImpl struct {
    eventBus events.Bus
    // ...
}

func NewServiceProvider(i *do.Injector) (Service, error) {
    bus := do.MustInvoke[events.Bus](i)
    logger := di.MustInvokeLogger(i)

    svc := &serviceImpl{
        eventBus: bus,
        logger:   logger,
    }

    if err := svc.subscribeToEvents(); err != nil {
        return nil, fmt.Errorf("failed to subscribe to events: %w", err)
    }

    return svc, nil
}

func (s *serviceImpl) subscribeToEvents() error {
    _, err := events.SubscribeTyped(s.eventBus, events.EventDirectoryChanged,
        func(ctx context.Context, payload events.DirectoryChangedPayload) error {
            s.mu.Lock()
            s.currentPath = payload.NewPath
            s.mu.Unlock()
            return nil
        },
        events.WithSync(),
        events.WithPriority(100),
    )
    return err
}
```

### Event Chaining (SkillService)

```go
// pkg/skills/service.go

package skills

const SourceName = "skill_service"

func (s *serviceImpl) subscribeToEvents() error {
    _, err := events.SubscribeTyped(s.eventBus, events.EventDirectoryChanged,
        s.handleDirectoryChanged,
        events.WithSync(),
        events.WithPriority(50),
    )
    return err
}

func (s *serviceImpl) handleDirectoryChanged(ctx context.Context, payload events.DirectoryChangedPayload) error {
    newSkills, err := s.DiscoverInPath(payload.NewPath)
    if err != nil {
        return err
    }

    if len(newSkills) > 0 {
        return events.PublishTyped(s.eventBus, ctx,
            events.EventSkillsUpdated,
            SourceName,
            events.SkillsUpdatedPayload{Skills: newSkills},
        )
    }
    return nil
}
```

### Async Handler (AgentRegistry)

```go
// pkg/registry/service.go

package registry

const SourceName = "agent_registry"

func (s *serviceImpl) subscribeToEvents() error {
    _, err := events.SubscribeTyped(s.eventBus, events.EventSkillsUpdated,
        s.handleSkillsUpdated,
        events.WithAsync(), // Non-blocking
    )
    return err
}

func (s *serviceImpl) handleSkillsUpdated(ctx context.Context, payload events.SkillsUpdatedPayload) error {
    s.mu.RLock()
    defer s.mu.RUnlock()

    for _, agent := range s.agents {
        if agent.IsIdle() {
            go agent.UpdateSystemPrompt(ctx, s.buildPromptWithSkills(payload.Skills))
        }
    }
    return nil
}
```

## Design Principles

1. **Agnostic Events Package**: The events package knows nothing about specific services or tools. It only defines the bus infrastructure and event type constants.

2. **Source Names in Own Package**: Each service defines its own `SourceName` constant. Tools use existing `shared.ToolName*` constants.

3. **Type Safety**: Generic helper functions (`SubscribeTyped`, `PublishTyped`) provide compile-time type checking for payloads.

4. **Self-Contained Subscriptions**: Each service subscribes to its own events during initialization. No central subscription setup in main.

5. **DI Singleton**: The Bus is a singleton via DI, ensuring all services use the same instance.

## Future Considerations

- **Metrics**: Add event publishing/handling metrics for monitoring
- **Dead Letter Queue**: Store failed events after max retries for analysis
- **Event Persistence**: Optional event logging for debugging/audit
- **Plugin API**: Expose event bus to plugins for extensibility
