# Executor Hooks Guide

## Overview

The Flow Executor supports hooks at the step level through the comprehensive hook system in `pkg/hooks`. These hooks allow you to inspect, validate, and monitor flow execution.

## Available Hooks

### BeforeFlowStep

Triggered before any step in a flow executes.

**Use cases:**
- Logging step execution
- Validating step permissions
- Monitoring/debugging
- Blocking certain step types in production

**Payload:**
```go
type hooks.ExecutorPayload struct {
    FlowID       uuid.UUID
    FlowName     string
    SessionID    uuid.UUID
    CurrentState string
    StepType     string  // "llm", "shell", "func", "mcp"
    StateName    string
}
```

### AfterFlowStep

Triggered after any step in a flow completes.

**Use cases:**
- Logging step results
- Recording metrics
- Error tracking
- Performance monitoring

**Additional fields:**
```go
StepResult map[string]any
StepError  error
Duration   time.Duration
```

## Registration

```go
import "github.com/denkhaus/gollum/pkg/hooks"

// Get HookManager from DI
hm := do.MustInvoke[hooks.HookManager](injector)

// Register BeforeFlowStep hook
hm.RegisterExecutorHook(func(ctx *hooks.TypedHookContext[hooks.ExecutorPayload]) error {
    log.Info("Executing step",
        zap.String("flow", ctx.Payload.FlowName),
        zap.String("type", ctx.Payload.StepType),
        zap.String("state", ctx.Payload.StateName))
    return nil
}, hooks.TypedHookMetadata{
    Name:     "flow-logger",
    Point:    hooks.BeforeFlowStep,
    Priority: 50,
})

// Register AfterFlowStep hook for metrics
hm.RegisterExecutorHook(func(ctx *hooks.TypedHookContext[hooks.ExecutorPayload]) error {
    metrics.RecordStepDuration(
        ctx.Payload.StepType,
        ctx.Payload.Duration,
    )
    if ctx.Payload.StepError != nil {
        metrics.RecordStepError(ctx.Payload.StepType)
    }
    return nil
}, hooks.TypedHookMetadata{
    Name:  "metrics-collector",
    Point: hooks.AfterFlowStep,
})
```

## Blocking Execution

Hooks can block step execution by setting `ctx.Blocked = true`:

```go
hm.RegisterExecutorHook(func(ctx *hooks.TypedHookContext[hooks.ExecutorPayload]) error {
    // Block shell steps in production
    if ctx.Payload.StepType == "shell" && isProduction() {
        ctx.Blocked = true
        return nil
    }
    return nil
}, hooks.TypedHookMetadata{
    Name:     "prod-guard",
    Point:    hooks.BeforeFlowStep,
    Priority: 100,
})
```

## Hook Layering

The executor has hooks at multiple levels:

| Level | Hooks | Scope |
|-------|-------|-------|
| **Flow** | BeforeFlowStep / AfterFlowStep | All steps in flow |
| **Tool** | BeforeToolExecution / AfterToolExecution | Tool calls only |
| **LLM** | BeforeLLMRequest / AfterLLMResponse | LLM calls only |

This allows both general flow-level monitoring and specific tool/LLM behavior modification.

## Examples

### Logging Hook

```go
hm.RegisterExecutorHook(func(ctx *hooks.TypedHookContext[hooks.ExecutorPayload]) error {
    logger.Info("Flow step executing",
        zap.String("flow", ctx.Payload.FlowName),
        zap.String("step_type", ctx.Payload.StepType),
        zap.String("state", ctx.Payload.StateName))
    return nil
}, hooks.TypedHookMetadata{Name: "logger", Point: hooks.BeforeFlowStep})
```

### Monitoring Hook

```go
hm.RegisterExecutorHook(func(ctx *hooks.TypedHookContext[hooks.ExecutorPayload]) error {
    histogram.Observe(ctx.Payload.Duration.Seconds(),
        prometheus.Labels{
            "step_type": ctx.Payload.StepType,
            "flow": ctx.Payload.FlowName,
        })
    return nil
}, hooks.TypedHookMetadata{Name: "monitoring", Point: hooks.AfterFlowStep})
```

### Validation Hook

```go
hm.RegisterExecutorHook(func(ctx *hooks.TypedHookContext[hooks.ExecutorPayload]) error {
    if ctx.Payload.StepType == "shell" && isProduction() {
        return fmt.Errorf("shell steps not allowed in production")
    }
    return nil
}, hooks.TypedHookMetadata{
    Name:     "prod-validator",
    Point:    hooks.BeforeFlowStep,
    Priority: 100,
})
```
