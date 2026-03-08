# Flow Executor Consolidation Design

**Date:** 2025-03-08
**Status:** Design Approved
**Issue:** N/A

## Problem Statement

There are two implementations of the Flow executor:
- `pkg/flows/executor/executor.go` - Feature-complete but not DI-integrated
- `pkg/flows/executor/di_service.go` - DI-integrated but incomplete

This creates confusion and maintenance burden. We need a single, feature-complete executor following proper DI patterns.

## Analysis

| Feature | executor.go | di_service.go |
|---------|-------------|---------------|
| Extension service support | ✅ Full (with Scriggo runner) | ❌ Only builtin registry |
| Error handling | ✅ FuncError wrapper | ❌ Basic error return |
| Multiple constructors | ✅ 4 constructors | ❌ DI only |
| LLM step execution | ✅ Full implementation | ❌ Delegates to Executor |
| DI integration | ❌ Direct instantiation | ✅ do.Injector pattern |
| Tests use this | ✅ All test files | ❌ None |

**executor.go is more feature complete** but lacks DI integration.
**di_service.go** is DI-integrated but incomplete.

## Solution

Consolidate into a single, DI-integrated executor following the project's DI guidance (`guide.golang.di.md`).

### Architecture

```
executor.go (consolidated)
├── FlowExecutorService      // Public interface - service that creates instances
├── FlowExecutorInstance     // Public interface - instance that executes flows
├── flowExecutorServiceImpl  // Private - DI service
└── flowExecutorImpl         // Private - instance (execution logic)
```

### Public Interfaces

```go
// FlowExecutorService - DI service that creates executor instances
type FlowExecutorService interface {
    // New creates a new executor instance for a flow
    New(flow *flows.Flow) FlowExecutorInstance
}

// FlowExecutorInstance - Executor that executes a specific flow
type FlowExecutorInstance interface {
    SetInput(vals map[string]any)
    Validate() error
    Run() error
    GetContext() *Context
}

// FlowRegistry - DI service for looking up flows by reference
type FlowRegistry interface {
    GetFlow(ref string) (*flows.Flow, error)
}
```

### Private Implementation

```go
// flowExecutorServiceImpl - Private DI service implementation
type flowExecutorServiceImpl struct {
    bashToolProvider tools.BashToolProvider
    extService       extensions.ExtensionService
    flowRegistry     FlowRegistry
}

// flowExecutorImpl - Private instance implementation
type flowExecutorImpl struct {
    flow             *flows.Flow
    ctx              *Context
    currentState     string
    history          *ExecutionHistory
    startTime        time.Time
    bashToolProvider tools.BashToolProvider
    extService       extensions.ExtensionService
    flowRegistry     FlowRegistry
}
```

### DI Constructor

```go
// NewFlowExecutor creates the flow executor service (DI constructor)
func NewFlowExecutor(injector do.Injector) (FlowExecutorService, error) {
    bashToolProvider := do.MustInvoke[tools.BashToolProvider](injector)
    extService := do.MustInvoke[extensions.ExtensionService](injector)
    flowRegistry := do.MustInvoke[FlowRegistry](injector)

    return &flowExecutorServiceImpl{
        bashToolProvider: bashToolProvider,
        extService:       extService,
        flowRegistry:     flowRegistry,
    }, nil
}
```

## Implementation Plan

### Step 1: Create FlowRegistry Service

**File:** `pkg/flows/registry/service.go` (NEW)

```go
package registry

type FlowRegistryService interface {
    GetFlow(ref string) (*flows.Flow, error)
}

type flowRegistryServiceImpl struct {
    flows map[string]*flows.Flow
}

func NewFlowRegistryService(injector do.Injector) (FlowRegistryService, error) {
    // Load flows from configured directory
    return &flowRegistryServiceImpl{
        flows: make(map[string]*flows.Flow),
    }, nil
}
```

**Update:** `pkg/di/container.go`
```go
do.Provide(p.injector, registry.NewFlowRegistryService)
```

### Step 2: Consolidate executor.go

**Merge into:** `pkg/flows/executor/executor.go`

1. Add DI service interfaces
2. Add single DI constructor `NewFlowExecutor(injector)`
3. Merge feature-complete methods from executor.go:
   - executeLLMStep (full implementation)
   - executeFuncStep (with ExtensionService + Scriggo)
   - handleError
   - FuncError type
4. Remove all direct constructors (NewExecutor, NewExecutorWithProvider, etc.)
5. Use private implementations with `p` receivers
6. Remove duplicate helper functions

### Step 3: Update Tests

**Files:** `pkg/flows/executor/*_test.go`

- Create test injector: `injector := do.New()`
- Provide mocks via DI
- Update to use `FlowExecutorService` interface
- Remove direct constructor calls

### Step 4: Cleanup

- Delete `pkg/flows/executor/di_service.go`
- Delete `pkg/flows/executor/flow_registry.go` (SimpleFlowRegistry - test only)
- Run all tests: `go test ./pkg/flows/executor/...`

### Step 5: Verification

- All tests pass
- DI container registration correct
- No nil-pointer issues
- Call steps work with FlowRegistry

## Files Modified

```
pkg/flows/registry/service.go          (NEW)
pkg/flows/executor/executor.go         (CONSOLIDATE)
pkg/flows/executor/di_service.go       (DELETE)
pkg/flows/executor/flow_registry.go    (DELETE)
pkg/di/container.go                    (UPDATE)
pkg/flows/executor/*_test.go           (UPDATE)
```

## Success Criteria

1. Single executor implementation following DI patterns
2. All features from executor.go preserved
3. All sub-services injected via DI
4. All tests pass with DI pattern
5. No duplicate code
