# Roadmap: Gollum Agent Framework

## Overview

Build a Go-based agent framework with comprehensive observability, prompt management, and optimization capabilities. Milestone v1.0 delivered the Prompt Optimizer with SemVer-based prompt store, lazy loading, and three optimization strategies. Milestone v1.1 adds Langfuse SDK integration for tracing LLM interactions, tool executions, and agent lifecycle events through the hook system.

## Phases

**Phase Numbering:**
- Integer phases (1-5): Milestone v1.0 Prompt Optimizer (COMPLETE)
- Integer phases (6-11): Milestone v1.1 Langfuse Integration (CURRENT)

- [x] **Phase 1: Core Types and Store Layer** - Foundation types and persistence abstraction
- [x] **Phase 2: Prompt Manager Extension** - ID-based prompt management with lazy loading
- [x] **Phase 3: Prompt Optimizer** - Three optimization strategies with LLM integration
- [x] **Phase 4: Configuration and DI Integration** - System integration and wiring
- [x] **Phase 5: Documentation and Knowledge Capture** - Guidance and documentation
- [ ] **Phase 6: Configuration and Client Initialization** - Langfuse config and lazy client init
- [ ] **Phase 7: LangfuseHook Struct and Basic Registration** - Hook structure, trace context, DI
- [ ] **Phase 8: LLM Tracing Implementation** - LLM span creation and lifecycle
- [ ] **Phase 9: Tool and Agent Lifecycle Tracing** - Tool and agent span hooks
- [ ] **Phase 10: Session Tracing and Flush/Shutdown** - Session lifecycle and trace flush
- [ ] **Phase 11: Testing and Documentation** - Test suite and knowledge base guidance

## Phase Details

### Phase 1: Core Types and Store Layer (COMPLETE)

**Goal**: Type system and persistence abstraction for versioned prompts

**Depends on**: Nothing (first phase)

**Requirements**: TYPE-01 through TYPE-06, STORE-01 through STORE-10, TEST-01, TEST-02, TEST-04, TEST-05, TEST-06, DCON-01, DCON-02, DCON-03, DCON-04

**Success Criteria**:
1. Prompt struct with SemVer versioning can be created and stored
2. InMemory store implementation persists and retrieves prompts correctly in tests
3. File store implementation persists prompts as JSON with file locking for concurrent safety
4. Alias resolution converts shortcuts like "subagent" to versioned IDs like "subagent@1.0.0"
5. Store returns nil (not error) when prompts are not found

**Plans**: 2 plans in 2 waves
- [x] 01-01-PLAN.md — Foundation: Core types, Store interface, InMemory store (Wave 1)
- [x] 01-02-PLAN.md — File persistence: File store, DI provider, Tests, Mocks (Wave 2)

### Phase 2: Prompt Manager Extension (COMPLETE)

**Goal**: Extended prompt management with lazy loading and template rendering

**Depends on**: Phase 1 (Store Layer)

**Requirements**: MGR-01 through MGR-07, TEST-03, DCON-01, DCON-04

**Success Criteria**:
1. Built-in prompts (system, supervisor, compacter, subagent) load lazily from embedded FS on first access
2. Prompts render with template variables using text/template syntax
3. GetPromptByID returns versioned prompts from store or bootstraps built-ins
4. Backward-compatible methods (GetSupervisorPrompt, GetSubagentPrompt, etc.) continue to work
5. Built-in prompts are protected from deletion by IsBuiltin flag

**Plans**: 3 plans in 3 waves (2 original + 1 gap closure)
- [x] 02-01-PLAN.md — Built-in prompt bootstrap and store integration (Wave 1)
- [x] 02-02-PLAN.md — Template rendering and backward compatibility (Wave 2)
- [x] 02-03-PLAN.md — Gap closure: IsBuiltin protection for built-in prompts (Wave 3)

### Phase 3: Prompt Optimizer (COMPLETE)

**Goal**: LLM-based prompt optimization with three strategies

**Depends on**: Phase 1 (Store Layer), Phase 2 (Prompt Manager)

**Requirements**: OPT-01 through OPT-10, TEST-07, DCON-01, DCON-02

**Success Criteria**:
1. Optimizer accepts trajectories (Gollem messages with optional feedback) and current prompt
2. Gradient strategy runs reflection loop with think/critique/recommend tools
3. Meta-prompt strategy combines reflection and update in single phase
4. Prompt memory strategy performs single-shot optimization
5. Optimizer returns new prompt version with incremented SemVer and change description

**Plans**: 3 plans in 3 waves
- [x] 03-01-PLAN.md — Core types, templates, and optimizer interface (Wave 1)
- [x] 03-02-PLAN.md — Three strategy implementations with TDD (Wave 2)
- [x] 03-03-PLAN.md — Integration helpers and mock registration (Wave 3)

### Phase 4: Configuration and DI Integration (COMPLETE)

**Goal**: System integration through configuration and dependency injection

**Depends on**: Phase 1 (Store Layer), Phase 2 (Prompt Manager), Phase 3 (Prompt Optimizer)

**Requirements**: CFG-01 through CFG-04 (v1.0), DI-01 through DI-03, DCON-01, DCON-04

**Success Criteria**:
1. PromptStore config selects store type (memory/file) via environment variables
2. PromptOptimizer config specifies default strategy, LLM provider, reflection steps
3. PromptStoreProvider registers in DI container and provides configured store
4. PromptOptimizer registers in DI container with LLM client dependency
5. PromptManager uses PromptStoreProvider from DI container

**Plans**: 3 plans in 2 waves
- [x] 04-01-PLAN.md — Config: PromptOptimizerConfig, ConfigService extension (Wave 1)
- [x] 04-02-PLAN.md — DI Optimizer: PromptOptimizer provider and registration (Wave 2)
- [x] 04-03-PLAN.md — DI Manager: PromptManager uses PromptStore from DI (Wave 2)

### Phase 5: Documentation and Knowledge Capture (COMPLETE)

**Goal**: Universal knowledge captured in guidance documentation

**Depends on**: Phase 1, Phase 2, Phase 3, Phase 4 (all phases complete)

**Requirements**: DOC-01 through DOC-04, DCON-01, DCON-05

**Success Criteria**:
1. CLAUDE.md updated with Prompt Optimizer usage examples and configuration
2. guide.golang.prompt-store.md created in knowledge base with store patterns
3. guide.golang.prompt-optimizer.md created in knowledge base with optimization patterns
4. guide.general.prompt-optimization.md created with universal optimization concepts
5. All guidance files are general/universal (no project-specific paths or details)

**Plans**: 4 plans in 1 wave
- [x] 05-01-PLAN.md — Go guidance: Prompt store patterns (Wave 1)
- [x] 05-02-PLAN.md — Go guidance: Prompt optimizer patterns (Wave 1)
- [x] 05-03-PLAN.md — General guidance: Prompt optimization concepts (Wave 1)
- [x] 05-04-PLAN.md — Project documentation: CLAUDE.md update (Wave 1)

### Phase 6: Configuration and Client Initialization

**Goal**: Langfuse configuration and lazy client initialization

**Depends on**: Nothing (first phase of v1.1)

**Requirements**: CFG-01 through CFG-04 (v1.1), CLI-01 through CLI-03

**Success Criteria**:
1. LangfuseConfig struct with host, public key, secret key, enabled flag added to HooksConfig
2. Environment variables LANGFUSE_ENABLED, LANGFUSE_HOST, LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY configure tracing
3. GetLangfuseConfig() method on ConfigService returns LangfuseConfig
4. Flush settings LANGFUSE_FLUSH_INTERVAL and LANGFUSE_MAX_QUEUE_SIZE configurable
5. Lazy Langfuse client initialization with sync.Mutex for thread safety
6. Client init failure logs warning and disables tracing gracefully

**Plans**: 2 plans in 2 waves
- [x] 06-01-PLAN.md — LangfuseConfig structure and environment variable loading
- [x] 06-02-PLAN.md — Lazy client initialization with thread safety

### Phase 7: LangfuseHook Struct and Basic Registration

**Goal**: Hook structure following logging_hook.go pattern with trace context management

**Depends on**: Phase 6 (Configuration and Client Initialization)

**Requirements**: HOOK-01 through HOOK-05, CTX-01 through CTX-06, DI-01 through DI-02

**Success Criteria**:
1. LangfuseHook struct created with logger, config, client, traceCtxs map, mutexes
2. NewLangfuseHook(injector) constructor with DI injection
3. NewLangfuseHookProvider(injector) returns hooks.HookFunc
4. RegisterLangfuseHooks(hm, hook) registers at all hook points with priority 500
5. TraceContext struct holds TraceID, RootSpan, Spans map, SessionID, CreatedAt
6. Thread-safe trace context operations (get, create, remove) with RWMutex
7. LangfuseHook registered in DI container via provider
8. Exported in pkg/builtin/provider.go

**Plans**: 3 plans in 2 waves
- [x] 07-01-PLAN.md — LangfuseHook struct with TraceContext and thread-safe operations (Wave 1)
- [x] 07-02-PLAN.md — Hook registration at all hook points (Wave 2)
- [x] 07-03-PLAN.md — DI integration and export via provider (Wave 2)

### Phase 8: LLM Tracing Implementation

**Goal**: LLM span creation and lifecycle hooks

**Depends on**: Phase 7 (LangfuseHook Struct and Basic Registration)

**Requirements**: LLM-01 through LLM-03

**Success Criteria**:
1. BeforeLLMRequest hook creates LLM span with model and input from HookContext
2. AfterLLMResponse hook updates span with output, usage tokens, and latency
3. OnLLMError hook marks span as failed with error details
4. LLM span stored in TraceContext.Spans for retrieval
5. Hooks tested with mock LLM client

**Plans**: TBD
- [ ] 08-01-PLAN.md — BeforeLLMRequest and AfterLLMResponse hooks
- [ ] 08-02-PLAN.md — OnLLMError hook and LLM span testing

### Phase 9: Tool and Agent Lifecycle Tracing

**Goal**: Tool execution and agent spawn/remove tracing

**Depends on**: Phase 7 (LangfuseHook Struct and Basic Registration)

**Requirements**: TOOL-01 through TOOL-03, AGT-01 through AGT-04

**Success Criteria**:
1. BeforeToolExecution hook creates tool span with name and input
2. AfterToolExecution hook finalizes tool span with output
3. OnToolError hook marks tool span as failed
4. BeforeAgentSpawn / AfterAgentSpawn hooks create and finalize agent span
5. BeforeAgentRemove / AfterAgentRemove hooks track agent removal
6. Agent spans include parent-child metadata
7. Span hierarchy tested (root -> agent -> tool/LLM children)

**Plans**: TBD
- [ ] 09-01-PLAN.md — Tool execution hooks (BeforeToolExecution, AfterToolExecution, OnToolError)
- [ ] 09-02-PLAN.md — Agent lifecycle hooks (spawn and remove)
- [ ] 09-03-PLAN.md — Span hierarchy testing

### Phase 10: Session Tracing and Flush/Shutdown

**Goal**: Session lifecycle tracing and trace flush on completion

**Depends on**: Phase 7 (LangfuseHook Struct and Basic Registration), Phase 8 (LLM Tracing), Phase 9 (Tool/Agent Tracing)

**Requirements**: SES-01 through SES-04, FLUSH-01 through FLUSH-02

**Success Criteria**:
1. BeforeSessionStart hook creates Langfuse trace and root span
2. AfterSessionEnd hook ends root span and flushes traces
3. TraceContext stored in map on session start, removed on session end
4. Trace ID propagated via HookContext.Data["langfuse_trace_id"]
5. Shutdown() method flushes buffered traces
6. Flush failures logged as non-fatal errors
7. Session end triggers trace cleanup and flush

**Plans**: TBD
- [ ] 10-01-PLAN.md — Session lifecycle hooks (BeforeSessionStart, AfterSessionEnd)
- [ ] 10-02-PLAN.md — Shutdown method and flush handling

### Phase 11: Testing and Documentation

**Goal**: Comprehensive test suite and knowledge base guidance

**Depends on**: Phase 6 through Phase 10 (all Langfuse integration phases)

**Requirements**: TEST-01 through TEST-03, DOC-01 through DOC-02

**Success Criteria**:
1. Unit tests using centralized mocks from pkg/mocks/
2. Concurrent access tests for trace context map
3. Integration tests with mock Langfuse server
4. CLAUDE.md updated with Langfuse configuration examples
5. guide.golang.langfuse-tracing.md created in knowledge base

**Plans**: TBD
- [ ] 11-01-PLAN.md — Unit tests with centralized mocks
- [ ] 11-02-PLAN.md — Concurrent access and integration tests
- [ ] 11-03-PLAN.md — CLAUDE.md and knowledge base documentation

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Core Types and Store Layer | 2/2 | Complete | 2025-02-01 |
| 2. Prompt Manager Extension | 3/3 | Complete | 2026-02-01 |
| 3. Prompt Optimizer | 3/3 | Complete | 2026-02-02 |
| 4. Configuration and DI Integration | 3/3 | Complete | 2026-02-02 |
| 5. Documentation and Knowledge Capture | 4/4 | Complete | 2026-02-02 |
| 6. Configuration and Client Initialization | 2/2 | Complete | 2026-02-11 |
| 7. LangfuseHook Struct and Basic Registration | 3/3 | Pending | — |
| 8. LLM Tracing Implementation | 0/2 | Pending | — |
| 9. Tool and Agent Lifecycle Tracing | 0/3 | Pending | — |
| 10. Session Tracing and Flush/Shutdown | 0/2 | Pending | — |
| 11. Testing and Documentation | 0/3 | Pending | — |

**Milestone Status:**
- **Milestone v1.0 — Prompt Optimizer: COMPLETE**
  - All 5 phases finished with 16 plans completed
- **Milestone v1.1 — Langfuse Integration: IN PROGRESS**
  - 6 phases planned with 15 plans estimated
  - 39 requirements to implement
  - Phase 6 complete (2/2 plans) ✓
