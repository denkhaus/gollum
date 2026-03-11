# Event Bus Implementation Plan

## Overview

This plan outlines the implementation steps for the event bus system based on the [design document](./2026-03-02-event-bus-design.md).

## Phase 1: Core Package

### 1.1 Create Event Types and Payloads
**File:** `pkg/events/types.go`

- [ ] Define event type constants (namespaced strings)
- [ ] Define payload structs for each event type
- [ ] Add documentation for each event type

### 1.2 Create Event Interface
**File:** `pkg/events/event.go`

- [ ] Define `Event` interface
- [ ] Implement `TypedEvent[T]` struct
- [ ] Implement `NewEvent[T]` constructor function

### 1.3 Create Bus Interface
**File:** `pkg/events/bus.go`

- [ ] Define `Handler` type
- [ ] Define `SubscriptionOption` and `SubscriptionConfig`
- [ ] Implement `WithAsync()` and `WithPriority()` options
- [ ] Define `Bus` interface (Subscribe, Unsubscribe, Publish)

### 1.4 Create Generic Helpers
**File:** `pkg/events/generic.go`

- [ ] Define `TypedHandler[T]` type
- [ ] Implement `SubscribeTyped[T]` function
- [ ] Implement `PublishTyped[T]` function

### 1.5 Implement Memory Bus
**File:** `pkg/events/memory_bus.go`

- [ ] Implement `subscription` struct
- [ ] Implement `memoryBus` struct with config and logger
- [ ] Implement `Subscribe` method
- [ ] Implement `Unsubscribe` method
- [ ] Implement `Publish` method with priority sorting
- [ ] Implement `executeWithRetry` with exponential backoff
- [ ] Handle sync vs async handler execution

### 1.6 Create DI Provider
**File:** `pkg/events/provider.go`

- [ ] Implement `NewBusProvider` function
- [ ] Inject logger and config service

### 1.7 Add Unit Tests
**File:** `pkg/events/bus_test.go`

- [ ] Test Subscribe/Unsubscribe
- [ ] Test sync handler execution
- [ ] Test async handler execution
- [ ] Test priority ordering
- [ ] Test retry logic
- [ ] Test generic helpers

## Phase 2: Configuration

### 2.1 Add EventsConfig to Config Service
**File:** `pkg/config/service.go`

- [ ] Add `EventsConfig` struct with envconfig tags
- [ ] Add `Events` field to `serviceImpl`
- [ ] Add `GetEventsConfig()` method
- [ ] Add method to `ConfigService` interface

### 2.2 Add Unit Tests for Config
**File:** `pkg/config/service_test.go`

- [ ] Test default values
- [ ] Test environment variable parsing

## Phase 3: DI Integration

### 3.1 Register Bus in DI Container
**File:** `pkg/di/container.go`

- [ ] Add `events.NewBusProvider` to service registration
- [ ] Ensure correct registration order (before services that use it)

## Phase 4: Tool Integration

### 4.1 Update ChangeDirectory Tool
**File:** `pkg/tools/change_directory.go`

- [ ] Add `eventBus` field to struct
- [ ] Inject bus in provider function
- [ ] Publish `EventDirectoryChanged` after successful change
- [ ] Handle publish errors gracefully (log, don't fail)

### 4.2 Add Tests for Tool Integration
**File:** `pkg/tools/change_directory_test.go`

- [ ] Mock event bus
- [ ] Verify event is published with correct payload

## Phase 5: Service Integration

### 5.1 Update Workspace Service
**File:** `pkg/workspace/service.go`

- [ ] Define `SourceName` constant
- [ ] Add `eventBus` field to struct
- [ ] Inject bus in provider function
- [ ] Implement `subscribeToEvents()` method
- [ ] Subscribe to `EventDirectoryChanged` with sync + high priority

### 5.2 Update Skill Service
**File:** `pkg/skills/service.go`

- [ ] Define `SourceName` constant
- [ ] Add `eventBus` field to struct
- [ ] Inject bus in provider function
- [ ] Implement `subscribeToEvents()` method
- [ ] Subscribe to `EventDirectoryChanged` with sync
- [ ] Publish `EventSkillsUpdated` when new skills discovered

### 5.3 Update Agent Registry
**File:** `pkg/registry/service.go`

- [ ] Define `SourceName` constant
- [ ] Add `eventBus` field to struct
- [ ] Inject bus in provider function
- [ ] Implement `subscribeToEvents()` method
- [ ] Subscribe to `EventSkillsUpdated` with async
- [ ] Update idle agents' system prompts

### 5.4 Add Tests for Service Integration
**Files:** `pkg/*/service_test.go`

- [ ] Mock event bus in each service test
- [ ] Verify subscriptions are made during initialization
- [ ] Test event handlers

## Phase 6: Integration Testing

### 6.1 End-to-End Test
**File:** `pkg/events/integration_test.go`

- [ ] Test complete flow: Directory change → Skills update → Agent prompt update
- [ ] Verify event chaining works correctly
- [ ] Test async handlers don't block

## Phase 7: Documentation

### 7.1 Update CLAUDE.md
**File:** `CLAUDE.md`

- [ ] Add event bus section with usage examples
- [ ] Document environment variables

### 7.2 Add Package Documentation
**File:** `pkg/events/doc.go`

- [ ] Add package-level documentation

## Dependencies

```
Phase 1 (Core Package)
    ↓
Phase 2 (Configuration)
    ↓
Phase 3 (DI Integration)
    ↓
Phase 4 (Tool Integration) ← Phase 5 (Service Integration)
    ↓                              ↓
Phase 6 (Integration Testing)
    ↓
Phase 7 (Documentation)
```

## Estimated Effort

| Phase | Files | Complexity |
|-------|-------|------------|
| Phase 1 | 5 | Medium |
| Phase 2 | 1 | Low |
| Phase 3 | 1 | Low |
| Phase 4 | 1 | Low |
| Phase 5 | 3 | Medium |
| Phase 6 | 1 | Medium |
| Phase 7 | 2 | Low |

## Risks

1. **Registration Order**: Bus must be registered before services that use it
2. **Circular Dependencies**: Ensure services don't create circular event loops
3. **Goroutine Leaks**: Async handlers must properly handle context cancellation
