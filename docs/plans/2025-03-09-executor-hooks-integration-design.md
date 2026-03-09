# Executor Hooks Integration Design

## Overview

Integrate the comprehensive hook system (`pkg/hooks`) into the Flow Executor to replace the unimplemented hook definitions in `pkg/extensions/hooks.go`.

## Problem

The Flow Executor currently has hook points defined but not implemented:
- `agent_pre_execute` / `agent_post_execute`
- `flow_pre_execute` / `flow_post_execute`
- `tool_pre_execute` / `tool_post_execute`

These definitions in `pkg/extensions/hooks.go` are not type-safe and have no implementation.

## Solution

Add an **Executor** category to the existing `pkg/hooks` system and integrate it into the Flow Executor.

### Hook Reduction

Analysis shows significant overlap with existing hook systems:

| Executor Hook | Existing Equivalent | Status |
|---------------|---------------------|--------|
| `tool_pre_execute` | `BeforeToolExecution` | Already covered by Tool hooks |
| `tool_post_execute` | `AfterToolExecution` | Already covered by Tool hooks |
| `agent_pre_execute` | `BeforeAgentSpawn` | Covered by Agent hooks via AgentFactory |
| `agent_post_execute` | `AfterAgentSpawn` | Covered by Agent hooks via AgentFactory |
| `flow_pre_execute` | **NEW** | Requires Executor hooks |
| `flow_post_execute` | **NEW** | Requires Executor hooks |

**Result**: Only **2 new hooks** needed:
- `BeforeFlowStep` - Before any step in a flow executes
- `AfterFlowStep` - After any step in a flow completes

## Architecture

### ExecutorPayload Structure

```go
// ExecutorPayload contains flow execution context for step hooks
type ExecutorPayload struct {
    // Flow identification
    FlowID      uuid.UUID
    FlowName    string
    SessionID   uuid.UUID

    // Current execution state
    CurrentState string

    // Step information
    StepType     string  // "llm", "shell", "func", "mcp"
    StepIndex    int
    StateName    string

    // Execution metadata (AfterFlowStep only)
    StepResult   map[string]any
    StepError    error
    Duration     time.Duration
}
```

### Hook Points

```go
const (
    BeforeFlowStep HookPoint = "BeforeFlowStep"
    AfterFlowStep  HookPoint = "AfterFlowStep"
)
```

### HookManager Interface Extension

```go
type HookManager interface {
    // RegisterExecutorHook registers a typed hook for flow step execution
    RegisterExecutorHook(fn TypedHookFunc[ExecutorPayload], meta TypedHookMetadata) error

    // TriggerExecutorHooks executes executor hooks for a given hook point
    TriggerExecutorHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[ExecutorPayload]) TypedHookResult[ExecutorPayload]

    // WithFlowStepHooks wraps step execution with hooks
    WithFlowStepHooks(
        ctx context.Context,
        sessionID, flowID uuid.UUID,
        step *flows.Step,
        stateName string,
        work func() (map[string]any, error),
    ) (map[string]any, error)
}
```

## Implementation

### 1. HookManager Extension

Add to `pkg/hooks/manager.go`:

```go
type hookManagerImpl struct {
    // ... existing registries ...
    executorRegistry *TypedRegistry[ExecutorPayload]
}

func NewHookManager(injector do.Injector) (HookManager, error) {
    p := &hookManagerImpl{
        // ... existing initialization ...
        executorRegistry: NewTypedRegistry[ExecutorPayload](),
    }
    return p, nil
}
```

### 2. WithFlowStepHooks Implementation

```go
func (p *hookManagerImpl) WithFlowStepHooks(
    ctx context.Context,
    sessionID, flowID uuid.UUID,
    step *flows.Step,
    stateName string,
    work func() (map[string]any, error),
) (map[string]any, error) {
    startTime := time.Now()

    // BeforeFlowStep
    beforePayload := &ExecutorPayload{
        FlowID:       flowID,
        SessionID:    sessionID,
        CurrentState: stateName,
        StepType:     step.Type,
        StateName:    stateName,
    }

    beforeCtx := &TypedHookContext[ExecutorPayload]{
        SessionID: sessionID,
        Payload:   beforePayload,
    }

    beforeResult := p.executorRegistry.Trigger(ctx, BeforeFlowStep, beforeCtx)
    if beforeResult.Blocked {
        return nil, fmt.Errorf("step blocked by BeforeFlowStep hook")
    }

    // Execute step
    result, err := work()
    duration := time.Since(startTime)

    // AfterFlowStep (always execute, even on error)
    afterPayload := &ExecutorPayload{
        FlowID:       flowID,
        SessionID:    sessionID,
        CurrentState: stateName,
        StepType:     step.Type,
        StateName:    stateName,
        StepResult:   result,
        StepError:    err,
        Duration:     duration,
    }

    afterCtx := &TypedHookContext[ExecutorPayload]{
        SessionID: sessionID,
        Payload:   afterPayload,
    }

    p.executorRegistry.Trigger(ctx, AfterFlowStep, afterCtx)

    return result, err
}
```

### 3. Executor Integration

Modify `pkg/flows/executor/executor.go`:

```go
type flowExecutorImpl struct {
    // ... existing fields ...
    hookManager hooks.HookManager
}

type flowExecutorServiceImpl struct {
    // ... existing fields ...
    hookManager hooks.HookManager
}

func NewFlowExecutor(injector do.Injector) (FlowExecutorService, error) {
    // ... existing injections ...
    hookManager := do.MustInvoke[hooks.HookManager](injector)

    return &flowExecutorServiceImpl{
        // ... existing fields ...
        hookManager: hookManager,
    }, nil
}

func (p *flowExecutorServiceImpl) New(flow *flows.Flow) FlowExecutorInstance {
    return &flowExecutorImpl{
        // ... existing fields ...
        hookManager: p.hookManager,
    }
}

func (p *flowExecutorImpl) executeStep(step *flows.Step, stateName string) error {
    result, err := p.hookManager.WithFlowStepHooks(
        context.Background(),
        uuid.Nil, // SessionID - will be available later
        p.flow.ID,
        step,
        stateName,
        func() (map[string]any, error) {
            return p.executeStepLogic(step, stateName)
        },
    )

    _ = result // Result handling depends on step type
    return err
}

func (p *flowExecutorImpl) executeStepLogic(step *flows.Step, stateName string) (map[string]any, error) {
    switch step.Type {
    case "llm":
        err := p.executeLLMStep(step, stateName)
        return nil, err
    case "shell":
        err := p.executeShellStep(step, stateName)
        return nil, err
    case "func":
        err := p.executeFuncStep(step, stateName)
        return nil, err
    case "mcp":
        err := p.executeMCPStep(step, stateName)
        return nil, err
    default:
        return nil, fmt.Errorf("unknown step type: %s", step.Type)
    }
}
```

### 4. DI Container Registration

Add to `pkg/di/container.go`:

```go
do.Provide(p.injector, hooks.NewHookManager)
```

Note: This should already be registered if hooks are used elsewhere.

## Hook Layering

The design allows **double hooks** for certain step types, which is intentional:

| Step Type | Flow-Level Hook | Specific Hook |
|-----------|-----------------|---------------|
| **shell** | BeforeFlowStep / AfterFlowStep | BeforeToolExecution / AfterToolExecution |
| **mcp** | BeforeFlowStep / AfterFlowStep | BeforeToolExecution / AfterToolExecution |
| **llm** | BeforeFlowStep / AfterFlowStep | BeforeLLMRequest / AfterLLMResponse |
| **func** | BeforeFlowStep / AfterFlowStep | (none - Scriggo has no hooks) |

This separation allows:
- **Flow-level hooks** for general logging, monitoring, debugging
- **Specific hooks** for detailed tool/LLM behavior modification

## Use Cases

### Logging Hook

```go
hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
    log.Info("Executing step",
        zap.String("flow", ctx.Payload.FlowName),
        zap.String("type", ctx.Payload.StepType),
        zap.String("state", ctx.Payload.StateName))
    return nil
}, TypedHookMetadata{Name: "flow-logger", Point: BeforeFlowStep})
```

### Monitoring Hook

```go
hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
    metrics.RecordStepDuration(
        ctx.Payload.StepType,
        ctx.Payload.Duration,
    )
    return nil
}, TypedHookMetadata{Name: "metrics", Point: AfterFlowStep})
```

### Validation Hook

```go
hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
    if ctx.Payload.StepType == "shell" && isProduction() {
        return fmt.Errorf("shell steps not allowed in production")
    }
    return nil
}, TypedHookMetadata{Name: "prod-guard", Point: BeforeFlowStep, Priority: 100})
```

## Testing

### Unit Tests

```go
func TestWithFlowStepHooks(t *testing.T) {
    hm := NewHookManager(injector)

    var beforeCalled, afterCalled bool
    var capturedPayload *ExecutorPayload

    hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
        beforeCalled = true
        capturedPayload = ctx.Payload
        return nil
    }, TypedHookMetadata{Name: "test-before", Point: BeforeFlowStep})

    hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
        afterCalled = true
        return nil
    }, TypedHookMetadata{Name: "test-after", Point: AfterFlowStep})

    step := &flows.Step{Type: "llm"}
    result, err := hm.WithFlowStepHooks(
        context.Background(),
        uuid.New(),
        uuid.New(),
        step,
        "test-state",
        func() (map[string]any, error) {
            return map[string]any{"output": "test"}, nil
        },
    )

    assert.NoError(t, err)
    assert.True(t, beforeCalled)
    assert.True(t, afterCalled)
    assert.Equal(t, "llm", capturedPayload.StepType)
    assert.Equal(t, "test-state", capturedPayload.StateName)
}
```

### Integration Tests

```go
func TestFlowExecutor_WithHooks(t *testing.T) {
    // Setup executor with hook manager
    injector := do.New()
    do.Provide(injector, hooks.NewHookManager)
    do.Provide(injector, NewFlowExecutor)

    // Register monitoring hook
    hm := do.MustInvoke[hooks.HookManager](injector)
    hm.RegisterExecutorHook(monitoringHook, ...)

    // Execute flow and verify hooks were called
    executor := do.MustInvoke[FlowExecutorService](injector)
    instance := executor.New(testFlow)
    err := instance.Run()

    assert.NoError(t, err)
    // Assert hook was called with correct payload
}
```

## Migration Path

1. **Phase 1**: Add ExecutorPayload and hook points to `pkg/hooks`
2. **Phase 2**: Implement HookManager extension methods
3. **Phase 3**: Integrate into Flow Executor
4. **Phase 4**: Remove unused `pkg/extensions/hooks.go` definitions
5. **Phase 5**: Update tests and documentation

## Benefits

- **Type-safe** - Compile-time type checking for hook functions
- **Consistent** - Same pattern as existing Tool/LLM/Agent hooks
- **Reduced** - Only 2 hooks instead of 6 originally defined
- **Extensible** - Easy to add new hook points if needed
- **Testable** - Mock-friendly with NoOpHookManager pattern
