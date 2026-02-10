# Architecture Research: Langfuse Hook Integration

**Project:** Gollum Agent Framework
**Research Date:** 2026-02-10
**Milestone:** v1.1 Langfuse Integration
**Overall Confidence:** HIGH

## Executive Summary

This architecture research documents how Langfuse tracing integrates with Gollum's existing hook-based architecture. The hook system provides natural injection points for span creation at all lifecycle events (session, agent, LLM, tool operations). The LangfuseHook follows the established LoggingHook pattern, ensuring consistency with the codebase. Key architectural considerations include trace context propagation through HookContext.Data, thread-safe span management for concurrent agent execution, lazy client initialization to avoid overhead when disabled, and proper flush/shutdown handling for buffered traces.

## Key Findings

**Stack:** Hook-based middleware pattern with git-hulk/langfuse-go SDK for tracing
**Architecture:** LangfuseHook creates/ends spans at hook points; trace ID propagated via HookContext.Data; Langfuse client initialized lazily via DI
**Critical consideration:** Thread safety required for concurrent agent execution; use sync.RWMutex for trace context storage

## Components

### LangfuseHook (pkg/builtin/langfuse_hook.go)

**Purpose:** Captures telemetry data at hook points and creates Langfuse observation spans for tracing LLM interactions, tool executions, and agent lifecycle events.

**Dependencies:**
- `logger.LoggerService` - For logging hook execution and errors
- `config.ConfigService` - For Langfuse configuration (enabled flag, keys, host)
- `langfuse.Client` (from git-hulk/langfuse-go) - For creating and flushing observation spans

**Initialization via DI:**
```go
func NewLangfuseHook(injector do.Injector) (*LangfuseHook, error) {
    log := do.MustInvoke[logger.LoggerService](injector)
    cfg := do.MustInvoke[config.ConfigService](injector)
    return &LangfuseHook{
        log:       log,
        config:    cfg.GetLangfuseConfig(), // New config struct
        client:    nil,                     // Lazy init
        clientMu:  &sync.Mutex{},           // Thread-safe lazy init
        traceCtxs: make(map[uuid.UUID]*TraceContext), // Session -> trace mapping
        traceMu:   &sync.RWMutex{},         // Thread-safe access
    }, nil
}

func NewLangfuseHookProvider(injector do.Injector) (hooks.HookFunc, error) {
    hook, err := NewLangfuseHook(injector)
    // Returns HookFunc that wraps span creation/ending logic
    return func(ctx context.Context, hookCtx *hooks.HookContext, next func() error) error {
        return hook.withSpan(ctx, hookCtx, next)
    }, nil
}

func RegisterLangfuseHooks(hm hooks.HookManager, hook *LangfuseHook) error {
    if !hook.config.Enabled {
        return nil // Skip registration if disabled
    }

    priority := 500 // Medium priority (after logging at 1000)
    fatalError := false // Tracing errors should NOT stop execution

    // Register at all hook points (see Hook Point Mappings section below)
    // ... registration calls
    return nil
}
```

**Key Design Decisions:**
- **Lazy client init:** Langfuse client created on first use to avoid overhead when tracing disabled
- **Thread-safe context storage:** sync.RWMutex protects concurrent access to trace contexts map
- **Non-blocking errors:** FatalError=false ensures tracing failures don't break agent execution

### LangfuseConfig (pkg/config/service.go)

Add to existing `HooksConfig` struct (already has partial Langfuse config for prompt store):

```go
// In HooksConfig struct, add tracing-specific fields:
type HooksConfig struct {
    // ... existing fields (LoggingEnabled, SecurityMode, etc.)

    // Langfuse tracing configuration
    LangfuseEnabled bool   `envconfig:"LANGFUSE_ENABLED" default:"false"`
    LangfuseHost    string `envconfig:"LANGFUSE_HOST" default:"https://cloud.langfuse.com"`
    LangfusePublicKey string `envconfig:"LANGFUSE_PUBLIC_KEY"`
    LangfuseSecretKey string `envconfig:"LANGFUSE_SECRET_KEY"`

    // Langfuse flush settings
    LangfuseFlushInterval int `envconfig:"LANGFUSE_FLUSH_INTERVAL" default:"1000"` // milliseconds
    LangfuseMaxQueueSize  int  `envconfig:"LANGFUSE_MAX_QUEUE_SIZE" default:"100"`
}
```

**Rationale:** Extends existing hook configuration pattern; shares Langfuse keys with prompt store config (single source of truth).

### TraceContext (internal struct in langfuse_hook.go)

```go
// TraceContext holds Langfuse trace data for a session
type TraceContext struct {
    TraceID   string                 // Langfuse trace ID
    RootSpan  *langfuse.Span         // Root observation span for session
    Spans     map[string]*langfuse.Span // Active child spans (agent_name -> span)
    SessionID uuid.UUID              // Gollum session ID
    CreatedAt time.Time
}
```

**Purpose:** Maintains mapping between Gollum session IDs and Langfuse trace contexts for span hierarchy.

### Langfuse Client Lifecycle

**Initialization (lazy):**
```go
func (h *LangfuseHook) getClient() (*langfuse.Client, error) {
    h.clientMu.Lock()
    defer h.clientMu.Unlock()

    if h.client != nil {
        return h.client, nil
    }

    // Validate config
    if h.config.LangfusePublicKey == "" || h.config.LangfuseSecretKey == "" {
        return nil, fmt.Errorf("Langfuse credentials not configured")
    }

    // Create client with SDK
    client := langfuse.NewClient(
        langfuse.WithPublicKey(h.config.LangfusePublicKey),
        langfuse.WithSecretKey(h.config.LangfuseSecretKey),
        langfuse.WithHost(h.config.LangfuseHost),
    )

    h.client = client
    return h.client, nil
}
```

**Shutdown (flush):**
```go
func (h *LangfuseHook) Shutdown() error {
    h.clientMu.Lock()
    defer h.clientMu.Unlock()

    if h.client != nil {
        // Flush any buffered traces before shutdown
        if err := h.client.Flush(); err != nil {
            h.log.Error("Failed to flush Langfuse traces", zap.Error(err))
            return err
        }
        h.log.Info("Langfuse traces flushed successfully")
    }
    return nil
}
```

**Integration point:** Call Shutdown() in AfterSessionEnd hook or container shutdown.

## Data Flow

### Session Lifecycle Tracing

```
1. BeforeSessionStart hook triggered
   - Check if tracing enabled
   - Lazy-init Langfuse client if needed
   - Create Langfuse trace: client.NewTrace(sessionID)
   - Create root span for session
   - Store TraceContext in map: traceMu.Lock(); traceCtxs[sessionID] = ctx; traceMu.Unlock()
   - Store trace ID in HookContext.Data["langfuse_trace_id"] for propagation

2. Agent operations (spawn, tool calls, LLM requests)
   - Each hook reads trace ID from HookContext.Data or looks up via SessionID
   - Creates child spans under root session span
   - Spans stored in TraceContext.Spans map for hierarchy

3. AfterSessionEnd hook triggered
   - End all active child spans
   - End root session span
   - Remove TraceContext from map
   - Flush traces (if immediate flush configured) or let SDK buffer
```

### LLM Request Flow Tracing

```
1. BeforeLLMRequest hook triggered
   - Get or create TraceContext for session
   - Create LLM span: langfuse.NewLLMSpan(
       parent: rootSpan,
       model: hookCtx.LLMModel,
       input: hookCtx.LLMInput,
     )
   - Start timing: span.Start()
   - Store span reference in TraceContext.Spans["llm_request"]

2. LLM API call executes (via work() function in WithLLMHooks)
   - Hook does NOT modify prompt (that's for other hooks)
   - Only observes the request/response

3. AfterLLMResponse hook triggered
   - Retrieve LLM span from TraceContext.Spans["llm_request"]
   - Update span with response data: span.End(
       output: hookCtx.LLMResponse,
       usage: completion.Usage, // tokens from response
       latency: time.Since(startTime),
     )

4. OnLLMError hook triggered (if LLM call fails)
   - Retrieve LLM span
   - Mark span as failed: span.SetError(hookCtx.LLMError)
   - End span with error status
```

### Tool Execution Flow Tracing

```
1. BeforeToolExecution hook triggered
   - Get TraceContext for session
   - Create tool span: langfuse.NewSpan(
       parent: rootSpan,
       name: hookCtx.ToolName,
       input: hookCtx.ToolArgs,
     )
   - Store span in TraceContext.Spans[toolName]

2. Tool executes (via work() in WithToolHooks)

3. AfterToolExecution hook triggered
   - Retrieve tool span
   - Update with result: span.End(output: hookCtx.ToolResult)

4. OnToolError hook triggered
   - Mark tool span as failed: span.SetError(hookCtx.ToolError)
```

### Agent Lifecycle Flow Tracing

```
1. BeforeAgentSpawn / AfterAgentSpawn
   - Create agent span: langfuse.NewSpan(
       parent: rootSpan,
       name: "agent:" + agentID.String(),
       metadata: map[string]any{
         "parent_agent": hookCtx.AgentID,
         "session": hookCtx.SessionID,
       },
     )

2. BeforeAgentRemove / AfterAgentRemove
   - End agent span with status
```

## Hook Point Mappings

| Hook Point | Langfuse Action | Span Operation | Trace Context Usage |
|------------|----------------|----------------|---------------------|
| **Session Hooks** |
| BeforeSessionStart | Create trace + root span | `client.NewTrace()` + `trace.NewSpan()` | New TraceContext created, stored in map |
| AfterSessionEnd | End root span + flush | `span.End()` + `client.Flush()` | TraceContext removed from map |
| **LLM Hooks** |
| BeforeLLMRequest | Create LLM child span | `trace.NewLLMSpan(model, input)` | Span stored in TraceContext.Spans["llm"] |
| AfterLLMResponse | Finalize LLM span | `span.End(output, usage, latency)` | Retrieve span, update with response data |
| OnLLMError | Mark LLM error | `span.SetError(err)` + `span.End()` | Mark error before ending |
| **Tool Hooks** |
| BeforeToolExecution | Create tool span | `trace.NewSpan(name, input)` | Span stored in TraceContext.Spans[toolName] |
| AfterToolExecution | Finalize tool span | `span.End(output)` | Retrieve and end span |
| OnToolError | Mark tool error | `span.SetError(err)` + `span.End()` | Error handling |
| **Agent Hooks** |
| BeforeAgentSpawn | Create agent span | `trace.NewSpan("agent_spawn")` | Track parent-child relationships |
| AfterAgentSpawn | Finalize spawn span | `span.End(metadata)` | Record new agent ID |
| BeforeAgentRemove | Create remove span | `trace.NewSpan("agent_remove")` | Track agent removal |
| AfterAgentRemove | Finalize remove span | `span.End()` | Record removal success |

## Build Order

Based on component dependencies, suggested build order:

### Phase 1: Configuration and Client Initialization
- Add Langfuse fields to HooksConfig in pkg/config/service.go
- Create LangfuseConfig getter method on ConfigService
- Implement lazy client initialization in LangfuseHook
- Unit tests for config loading and client creation

### Phase 2: LangfuseHook Struct and Basic Registration
- Create pkg/builtin/langfuse_hook.go with struct
- Implement NewLangfuseHook, NewLangfuseHookProvider, RegisterLangfuseHooks
- Add trace context storage (TraceContext struct, map, mutex)
- Register LangfuseHook in pkg/builtin/provider.go
- DI registration in pkg/di/container.go

### Phase 3: LLM Tracing Implementation
- Implement BeforeLLMRequest hook (create LLM span, start timing)
- Implement AfterLLMResponse hook (end span with response, tokens, latency)
- Implement OnLLMError hook (mark error status)
- Test with mock LLM client

### Phase 4: Tool and Agent Lifecycle Tracing
- Implement BeforeToolExecution / AfterToolExecution / OnToolError hooks
- Implement agent spawn/remove hooks (BeforeAgentSpawn, AfterAgentSpawn, etc.)
- Test span hierarchy (root -> agent -> tool/LLM children)

### Phase 5: Flush/Shutdown and Thread Safety
- Implement Shutdown() method for client flush
- Add AfterSessionEnd hook to flush traces on session end
- Concurrent access testing (multiple agents, sessions)
- Integration tests with real Langfuse instance (or mock server)

### Phase 6: Testing and Documentation
- Comprehensive test suite using centralized mocks
- Update CLAUDE.md with Langfuse configuration examples
- Knowledge base guidance: guide.golang.langfuse-tracing.md

## Thread Safety Considerations

### Concurrent Access Patterns

**Scenario 1: Multiple agents in same session**
- Agents share same TraceContext (same session ID)
- Span creation/ending must be synchronized
- Use separate locks per span or use Langfuse SDK's built-in thread safety

**Scenario 2: Concurrent sessions**
- Different sessions have different TraceContext entries
- map access requires RWMutex for safe reads/writes
- Lock contention minimal as sessions are independent

**Scenario 3: Client lazy initialization**
- Multiple hooks may call getClient() simultaneously
- Use sync.Mutex for client initialization (Mutex, not RWMutex - write pattern)

### Mutex Usage Strategy

```go
type LangfuseHook struct {
    // ... other fields

    client   *langfuse.Client
    clientMu *sync.Mutex       // Protects lazy client init (single write, many reads)

    traceCtxs map[uuid.UUID]*TraceContext
    traceMu   *sync.RWMutex    // Protects traceCtxs map (many reads/writes)
}
```

**Lock hierarchy:** Always acquire clientMu BEFORE traceMu to prevent deadlocks.

### Langfuse SDK Thread Safety

**Assumption:** git-hulk/langfuse-go SDK client is thread-safe for concurrent span creation/ending.

**Verification needed:** Check SDK documentation or source code for:
- Whether `NewTrace()` and span methods are goroutine-safe
- Whether `Flush()` is safe to call while spans are being created
- Recommended patterns for concurrent tracing

**If SDK is NOT thread-safe:**
- Add mutex wrapping all Langfuse client calls
- Consider per-goroutine client instances (if SDK supports)
- Buffer span operations and serialize to single goroutine

### Data Race Prevention

**Reading TraceContext:**
```go
func (h *LangfuseHook) getTraceContext(sessionID uuid.UUID) (*TraceContext, bool) {
    h.traceMu.RLock()
    defer h.traceMu.RUnlock()
    ctx, ok := h.traceCtxs[sessionID]
    return ctx, ok
}
```

**Writing TraceContext:**
```go
func (h *LangfuseHook) createTraceContext(sessionID uuid.UUID, traceID string) *TraceContext {
    h.traceMu.Lock()
    defer h.traceMu.Unlock()

    ctx := &TraceContext{
        TraceID:   traceID,
        RootSpan:  nil, // Created later
        Spans:     make(map[string]*langfuse.Span),
        SessionID: sessionID,
        CreatedAt: time.Now(),
    }
    h.traceCtxs[sessionID] = ctx
    return ctx
}
```

**Deleting TraceContext:**
```go
func (h *LangfuseHook) removeTraceContext(sessionID uuid.UUID) {
    h.traceMu.Lock()
    defer h.traceMu.Unlock()
    delete(h.traceCtxs, sessionID)
}
```

## Error Handling Strategy

### Non-Fatal Tracing Errors (FatalError=false)

**Rationale:** Tracing failures should NOT break agent execution.

**Error types:**
1. **Client init failure:** Log warning, disable tracing for session, return nil from hooks
2. **Span creation failure:** Log error, continue without span
3. **Flush failure:** Log error, traces may be lost but session succeeds

**Pattern:**
```go
func (h *LangfuseHook) withSpan(ctx context.Context, hookCtx *hooks.HookContext, next func() error) error {
    // Always call next() to ensure execution continues
    err := next()

    // Tracing is best-effort
    if traceErr := h.captureTrace(hookCtx); traceErr != nil {
        h.log.Warn("Tracing failed (non-fatal)", zap.Error(traceErr))
    }

    return err // Return original execution error, not tracing error
}
```

### Hook Chain Error Propagation

**From HookManager design:**
- If FatalError=false: Error logged, chain continues to next hook
- If FatalError=true: Error stops chain, returned to caller

**LangfuseHook uses FatalError=false** for all registrations:
```go
register(hook.createHookFunc(hooks.BeforeLLMRequest, "before_llm"),
    hooks.HookMetadata{
        Name:       "langfuse-before-llm",
        Point:      hooks.BeforeLLMRequest,
        Priority:   500,
        FatalError: false, // Non-blocking
    })
```

## Quality Gate

- [x] Components clearly defined with boundaries
- [x] Data flow direction explicit (session -> agent -> tool/LLM)
- [x] Build order implications noted (phases 1-6)
- [x] Hook point mappings specified (11 hook points)
- [x] Thread safety considerations addressed (RWMutex strategy)

## Sources

- Gollum codebase: /home/denkhaus/dev/gomodules/gollum/pkg/hooks/ (hook system architecture)
- Gollum codebase: /home/denkhaus/dev/gomodules/gollum/pkg/builtin/logging_hook.go (reference hook pattern)
- Gollum codebase: /home/denkhaus/dev/gomodules/gollum/pkg/config/service.go (configuration pattern)
- git-hulk/langfuse-go GitHub: https://github.com/git-hulk/langfuse-go (Langfuse Go SDK)

---

*Research completed: 2026-02-10*
*Next step: ROADMAP.md phase structure informed by this architecture*
