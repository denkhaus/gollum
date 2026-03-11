# Event Bus

## Overview

The Gollum event bus provides decoupled communication between services using a publish/subscribe pattern. Services can publish events when state changes occur, and other services can subscribe to those events to react accordingly.

## Key Features

- **Type-safe**: Generic helper functions provide compile-time type checking for event payloads
- **Sync/Async handlers**: Choose between blocking and non-blocking event handling
- **Priority ordering**: Control the order in which handlers execute
- **Automatic retry**: Failed handlers are retried with exponential backoff
- **DI integration**: Event bus is a singleton via dependency injection

## Quick Start

### Publishing Events

Tools and services publish events when state changes occur:

```go
import (
    "github.com/denkhaus/gollum/pkg/events"
    "github.com/denkhaus/gollum/pkg/shared"
)

// Publish a directory changed event
err := events.PublishTyped(eventBus, ctx,
    string(events.EventDirectoryChanged),
    string(shared.ToolNameChangeDirectory),
    events.DirectoryChangedPayload{
        OldPath: "/old/path",
        NewPath: "/new/path",
    },
)
```

### Subscribing to Events

Services subscribe to events during initialization:

```go
import (
    "github.com/denkhaus/gollum/pkg/events"
)

func (s *myService) subscribeToEvents() error {
    _, err := events.SubscribeTyped[events.DirectoryChangedPayload](
        s.eventBus,
        string(events.EventDirectoryChanged),
        s.handleDirectoryChanged,
        events.WithSync(),
        events.WithPriority(100),
    )
    return err
}

func (s *myService) handleDirectoryChanged(ctx context.Context, payload events.DirectoryChangedPayload) error {
    s.log.Info("Directory changed", "old", payload.OldPath, "new", payload.NewPath)
    // Update service state...
    return nil
}
```

## Event Types

Event types are defined in `pkg/events/types.go`:

| Event Type | Payload | Description |
|------------|---------|-------------|
| `EventDirectoryChanged` | `DirectoryChangedPayload` | Working directory changed |
| `EventSkillsUpdated` | `SkillsUpdatedPayload` | Skills list updated |
| `EventSkillsDiscovered` | `SkillsDiscoveredPayload` | New skills discovered |
| `EventAgentSpawned` | `AgentSpawnedPayload` | New agent created |
| `EventAgentRemoved` | `AgentRemovedPayload` | Agent removed |
| `EventAgentIdle` | `AgentStatePayload` | Agent became idle |
| `EventAgentBusy` | `AgentStatePayload` | Agent became busy |
| `EventAgentResumed` | `AgentStatePayload` | Agent resumed |
| `EventAgentPaused` | `AgentStatePayload` | Agent paused |
| `EventConfigChanged` | `ConfigChangedPayload` | Configuration changed |
| `EventPluginLoaded` | `PluginLoadedPayload` | Plugin loaded |
| `EventPluginUnloaded` | `PluginUnloadedPayload` | Plugin unloaded |

## Subscription Options

### Sync vs Async

```go
// Synchronous - blocks until handler completes
events.WithSync()

// Asynchronous - handler runs in goroutine, publisher continues immediately
events.WithAsync()
```

Use **sync** when:
- The operation must complete before the publisher continues
- The handler's result affects subsequent operations
- You need guaranteed execution order

Use **async** when:
- The operation is non-critical
- The handler might take a long time
- You don't want to block the publisher

### Priority

```go
// Higher priority = executes first (within sync/async group)
events.WithPriority(100)  // High priority
events.WithPriority(50)   // Medium priority
events.WithPriority(0)    // Default priority
```

Handlers are sorted by priority (descending) within their sync/async group. Sync handlers always execute before async handlers.

## Configuration

Configure the event bus via environment variables:

```bash
# Maximum retry attempts for failed handlers (default: 3)
GOLLUM_EVENTS_MAX_RETRIES=3

# Initial delay between retries in milliseconds (default: 100)
GOLLUM_EVENTS_RETRY_DELAY_MS=100

# Exponential backoff multiplier (default: 2)
GOLLUM_EVENTS_RETRY_BACKOFF=2
```

## Architecture

### Event Flow

```
ChangeDirectory Tool
    │
    ├─ publish(DirectoryChanged)
    │
    ├─── [SYNC, Priority 100] WorkspaceService
    │    └─ Updates current workspace path
    │
    └─── [SYNC, Priority 50] SkillService
         └─ Discovers new skills
         └─ publish(SkillsUpdated)
              │
              └─── [ASYNC] AgentRegistry
                   └─ Updates idle agents' system prompts
```

### Decoupling Principle

The event bus enforces strict decoupling:

1. **Publishers don't know subscribers**: A tool only publishes an event; it doesn't know or care who handles it
2. **Subscribers are self-contained**: Each service subscribes to its own events during initialization
3. **No direct service calls**: Tools never call service methods directly

This design allows:
- Adding new subscribers without modifying publishers
- Testing services in isolation
- Replacing implementations without affecting other components

## Best Practices

### Handler Design

1. **Keep handlers focused**: Each handler should do one thing
2. **Handle errors gracefully**: Return errors to trigger retry, but don't panic
3. **Use appropriate priority**: Higher priority for critical operations
4. **Avoid long-running sync handlers**: Use async for expensive operations

```go
// Good: Focused, fast handler
func (s *Service) handleEvent(ctx context.Context, p Payload) error {
    s.mu.Lock()
    s.state = p.NewValue
    s.mu.Unlock()
    return nil
}

// Avoid: Long-running sync handler
func (s *Service) handleEvent(ctx context.Context, p Payload) error {
    time.Sleep(10 * time.Second)  // Blocks all other handlers!
    return nil
}
```

### Event Design

1. **Use descriptive names**: `workspace.directory_changed` not `dir_change`
2. **Include all necessary data**: Payloads should be self-contained
3. **Don't over-publish**: Only publish when meaningful state changes occur

### Testing

Mock the event bus in tests:

```go
import "github.com/denkhaus/gollum/pkg/mocks"

func TestMyService(t *testing.T) {
    ctrl := gomock.NewController(t)
    mockBus := mocks.NewMockBus(ctrl)

    // Expect subscription
    mockBus.EXPECT().Subscribe(
        string(events.EventDirectoryChanged),
        gomock.Any(),
        gomock.Any(),
    ).Return("sub-id", nil)

    // Expect publication
    mockBus.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)

    // Test your service...
}
```

## Examples

### Adding a New Event Type

1. Define the event type constant in `pkg/events/types.go`:

```go
const (
    // ... existing events ...
    EventMyNewEvent EventType = "my_service.new_event"
)
```

2. Define the payload struct:

```go
type MyNewEventPayload struct {
    Field1 string
    Field2 int
}
```

3. Publish from your tool/service:

```go
events.PublishTyped(eventBus, ctx,
    string(events.EventMyNewEvent),
    "my_service",
    events.MyNewEventPayload{
        Field1: "value",
        Field2: 42,
    },
)
```

4. Subscribe in your service:

```go
events.SubscribeTyped[events.MyNewEventPayload](
    eventBus,
    string(events.EventMyNewEvent),
    handler,
    events.WithSync(),
)
```

### Event Chaining

Services can publish new events while handling others:

```go
func (s *Service) handleDirectoryChanged(ctx context.Context, p events.DirectoryChangedPayload) error {
    // Do work
    newData := s.processPath(p.NewPath)

    // Publish new event
    return events.PublishTyped(s.eventBus, ctx,
        string(events.EventMyNewEvent),
        "my_service",
        events.MyNewEventPayload{Data: newData},
    )
}
```

## Troubleshooting

### Handler Not Called

1. Check the event type string matches exactly
2. Verify the subscription was successful (check error return)
3. Ensure the bus instance is the same (DI singleton)

### Handler Blocks Publisher

Sync handlers block until complete. If this is a problem:
- Use `events.WithAsync()` instead
- Move expensive work to a goroutine

### Handler Fails Repeatedly

Check logs for retry attempts. The handler returns an error each time. After max retries, the event is dropped. Consider:
- Making the handler more resilient
- Increasing retry count via configuration
- Using async handlers for non-critical operations
