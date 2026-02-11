---
title: Golang Langfuse Tracing Guidelines
description: Langfuse SDK integration patterns for observability in Go agent frameworks
updated: 2026-02-11
created: 2026-02-11
tags:
  - golang
  - tracing
  - langfuse
  - observability
  - hooks
---

# Langfuse Tracing in Go

## What is Langfuse Tracing?

- Open-source observability platform for LLM applications
- Captures traces, spans, and metrics for agent operations
- Provides insights into prompt performance, token usage, and latency
- Enables debugging of multi-agent interactions and tool chains

## Core Concepts

### Trace Hierarchy

- Trace: Top-level container for a single session/operation
- Root Span: Session-level parent span
- Child Spans: LLM calls, tool executions, agent lifecycle events
- Span Types: LLM (generation), Tool (execution), Agent (lifecycle), Custom (user-defined)

### Hook-Based Integration

- Hooks provide natural injection points for span creation
- Non-invasive: No changes to core agent logic required
- Lazy initialization: Client created only when tracing enabled
- Thread-safe: Concurrent agent execution supported

## Implementation Patterns

### Hook Structure Pattern

Following logging_hook.go pattern:

```go
type LangfuseHook struct {
    log        logger.LoggerService
    config     *config.LangfuseConfig
    client     *langfuse.Langfuse
    clientMu   *sync.Mutex           // Protects lazy client init
    traceCtxs  map[uuid.UUID]*TraceContext
    traceCtxsMu *sync.RWMutex        // Protects traceCtxs map
}
```

Key principles:
- Lazy client initialization (getClient() with mutex)
- Thread-safe trace context access (RWMutex for reads, Mutex for writes)
- No-op hooks when tracing disabled
- Graceful degradation on client errors

### Trace Context Lifecycle

```go
type TraceContext struct {
    TraceID   string                    // Langfuse trace ID
    RootSpan  interface{}                // SDK root span
    Spans     map[string]interface{}     // Child spans by ID
    SessionID uuid.UUID                 // Session ID
    CreatedAt time.Time                  // Creation timestamp
}
```

Lifecycle:
1. Session start: Create TraceContext, generate TraceID
2. Operation: Create child spans, store in Spans map
3. Session end: End root span, flush to backend, remove context

### Span Creation Patterns

**LLM Spans:**
- Create in BeforeLLMRequest with model and input
- Update in AfterLLMResponse with output and usage
- Mark error in OnLLMError with error details

**Tool Spans:**
- Create in BeforeToolExecution with tool name and args
- Update in AfterToolExecution with result
- Mark error in OnToolError with error details

**Agent Spans:**
- Create in BeforeAgentSpawn/BeforeAgentRemove
- Update in AfterAgentSpawn/AfterAgentRemove
- Track hierarchy via ParentAgentID/NewAgentID

## Testing Patterns

### Mock Langfuse Client

For testing without actual Langfuse backend:

```go
// Create mock client that implements required methods
type mockLangfuseClient struct {
    t *testing.T
    traces []*traceRecord
}

func (m *mockLangfuseClient) StartTrace(ctx context.Context, name string) Trace {
    trace := &traceRecord{
        ID:   uuid.New().String(),
        Name:  name,
        Spans: []spanRecord{},
    }
    m.traces = append(m.traces, trace)
    return trace
}
```

### Test Concurrent Access

```go
func TestConcurrentTraceContext(t *testing.T) {
    hook := &LangfuseHook{
        traceCtxs:   make(map[uuid.UUID]*TraceContext),
        traceCtxsMu: &sync.RWMutex{},
    }

    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(sessionID uuid.UUID) {
            defer wg.Done()
            tc := hook.createTraceContext(sessionID)
            assert.NotNil(t, tc)
        }(uuid.New())
    }
    wg.Wait()
}
```

## Best Practices

### Configuration

- Default to disabled: Opt-in tracing avoids overhead
- Validate credentials on init: Fail fast on bad config
- Lazy client creation: No SDK calls when disabled

### Performance

- Batch flush operations: Send traces in groups
- Async flush when possible: Don't block agent execution
- Background cleanup: Remove old trace contexts periodically

### Error Handling

- Log but don't fail: Tracing errors are non-fatal
- Preserve partial data: Capture what you can on errors
- Clean up resources: Always remove trace contexts

### Thread Safety

- Use RWMutex for read-heavy maps: Allow concurrent reads
- Minimize lock duration: Copy data before releasing
- Document lock ordering: Avoid deadlocks with multiple locks

## Common Pitfalls

### Memory Leaks

- Always remove trace contexts on session end
- Clean up orphaned contexts during shutdown
- Use defer for cleanup in error paths

### Race Conditions

- Release traceCtxsMu before calling propagateTraceID
- Don't hold locks while calling external code
- Use unique span IDs (UUID, not counters)

### Missing Spans

- Create spans in before-hooks, update in after-hooks
- Store span ID in HookContext.Data for correlation
- Handle missing span IDs gracefully (return early, don't panic)

## Additional Resources

- Langfuse Go SDK: https://github.com/git-hulk/langfuse-go
- Langfuse Documentation: https://langfuse.com/docs
- Hook patterns: [[guide.golang.testing]]
- Prompt optimization: [[guide.general.prompt-optimization]]
