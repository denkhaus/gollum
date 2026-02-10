# Gollum Agent Framework

## What This Is

A Go-based agent framework with comprehensive observability, prompt management, and optimization capabilities. The framework provides extensible hooks for tracing, integrates with Langfuse for LLM observability, and includes automatic prompt optimization based on execution feedback.

## Core Value

**Full observability and prompt management enable understanding, debugging, and continuous improvement of agent behavior.**

Every operation is traceable, every prompt is versioned, and every session contributes to making agents better.

## Current Milestone: v1.1 Langfuse Integration

**Goal:** Integrate Langfuse SDK for comprehensive tracing and prompt management through the hook system.

**Target features:**
- LLM request/response tracing with token counts and latency
- Tool execution tracing with inputs, outputs, and errors
- Agent lifecycle tracking (spawn, remove, session events)
- Langfuse prompt library integration
- Hook-based implementation following the logging_hook.go pattern
- Configuration via environment variables
- Opt-in tracing with minimal performance overhead when disabled

## Requirements

### Validated

**v1.0 Prompt Optimizer** (Complete — 2026-02-02):
- ✓ Prompt Store with SemVer versioning (memory/file backends)
- ✓ Prompt Manager with lazy loading and template rendering
- ✓ Three optimization strategies (gradient, meta-prompt, prompt memory)
- ✓ Trajectory capture and storage
- ✓ LLM provider integration (Anthropic, OpenAI, Gemini)
- ✓ Configuration and DI integration
- ✓ Documentation and knowledge base guidance

**Existing Gollum Features**:
- ✓ Agent registry with parent-child relationships
- ✓ Tool system for agent capabilities
- ✓ Hook system for extensible lifecycle events
- ✓ MCP integration at localhost:8555/mcp
- ✓ Centralized testing patterns with uber.org/mock
- ✓ File watching and state management
- ✓ LoggingHook built-in implementation

### Active

- [ ] **LANG-01**: LangfuseHook implements hook pattern from logging_hook.go
- [ ] **LANG-02**: LLM tracing captures requests, responses, tokens, latency, models
- [ ] **LANG-03**: Tool execution tracing captures tool calls, inputs, outputs, errors
- [ ] **LANG-04**: Agent lifecycle tracking captures spawn/remove events
- [ ] **LANG-05**: Session tracing with Langfuse observation spans
- [ ] **LANG-06**: LangfuseConfig with host, public key, secret key, enabled flag
- [ ] **LANG-07**: Langfuse client initialization with git-hulk/langfuse-go SDK
- [ ] **LANG-08**: Flush/shutdown handling for buffered traces
- [ ] **LANG-09**: Thread-safe trace context management
- [ ] **LANG-10**: Prompt library integration for Langfuse-hosted prompts

### Out of Scope

- **Other observability backends** (OpenTelemetry, Prometheus) — Langfuse first, others in future milestones
- **Real-time streaming** — Tracing buffered and flushed on completion, not real-time
- **Langfuse prompt versioning** — Using existing Gollum prompt store for v1.1
- **Automatic instrumentation** — Manual hook registration, not compile-time injection
- **Custom span propagation** — Using Langfuse SDK's built-in context management
- **Advanced prompt features** — Prompt templates, A/B testing deferred to v1.2+

## Context

**Existing Gollum Codebase:**
- Go-based agent framework with dependency injection (samber/do/v2)
- LLM abstraction via m-mizutani/gollem (supports Anthropic, OpenAI, Gemini)
- Hook system with middleware-style execution pattern
- HookManager interface for registering custom hooks
- Built-in hooks: LoggingHook (reference implementation)
- Hook points: session, agent, tool, file, LLM operations

**Hook Pattern Reference** (`pkg/builtin/logging_hook.go`):
- Hook struct with dependencies (log service, config)
- `NewHook(injector do.Injector)` for DI injection
- `NewHookProvider(injector)` returns `hooks.HookFunc`
- `Register*Hooks(hm hooks.HookManager, hook *Hook)` registers at all hook points
- Priority-based execution order
- FatalError flag for error handling behavior

**Langfuse SDK** (https://github.com/git-hulk/langfuse-go):
- Go client for Langfuse tracing and prompt management
- Support for observation spans (LLM, tool, custom)
- Automatic token counting and latency tracking
- Buffered flushing for performance
- Configuration via host, public key, secret key

**Domain Knowledge:**
- OpenTelemetry-style span hierarchy for trace context
- Hooks provide natural injection points for span creation
- Lazy initialization avoids resource allocation when tracing disabled
- Thread-safe trace context required for concurrent agent execution

## Constraints

- **SDK Source**: Must use git-hulk/langfuse-go (fork of official SDK with Go idioms)
- **Hook Pattern**: MUST follow logging_hook.go pattern exactly for consistency
- **Go Guidance**: CRITICAL — `/home/denkhaus/dev/kb/guides/guide.golang.*.md` files MUST be read in every phase
- **Testing**: All code must include tests using centralized mocks from `pkg/mocks/`
- **Package Structure**: Follow Gollum conventions (pkg/ structure, DI patterns, naming)
- **Performance**: Tracing must have minimal overhead when disabled (lazy init, no-op hooks)
- **Breaking Changes**: Must not break existing hooks or hook users

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Hook-based tracing | Follows existing pattern; non-invasive; consistent with codebase | — Pending |
| Opt-in via config | Default disabled; users explicitly enable tracing | — Pending |
| Buffered flushing | Better performance; traces sent in batches | — Pending |
| git-hulk/langfuse-go | Maintained fork with better Go idioms | — Pending |
| Lazy client init | No overhead when tracing disabled | — Pending |

---
*Last updated: 2026-02-10 after milestone v1.1 initialization*
