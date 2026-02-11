# Requirements: Gollum Agent Framework

**Defined:** 2025-02-01
**Core Value:** Agent quality improves iteratively through automatic prompt optimization based on execution feedback.

## v1 Requirements

### Core Types

- [x] **TYPE-01**: Prompt struct with ID, Content, Version (SemVer), Aliases, Tags, Timestamps, IsBuiltin flag
- [x] **TYPE-02**: PromptContext for template rendering with Values, SubAgent, and Agent contexts
- [x] **TYPE-03**: Trajectory struct using Gollem Message format with Feedback field
- [x] **TYPE-04**: Feedback and EditFeedback structs for structured user input
- [x] **TYPE-05**: OptimizerInput with PromptID, Prompt, Trajectories, and NewVersionName
- [x] **TYPE-06**: OptimizerResult with OptimizedContent, NewPromptID, OldPromptID, Version, and Changes

### Store Layer

- [x] **STORE-01**: PromptStore interface with SaveNewVersion, Load, Delete, List, Exists, ListTags methods
- [x] **STORE-02**: ResolveAlias method to resolve shortcuts ("subagent" -> "subagent@latest")
- [x] **STORE-03**: ListVersions method to list all versions of a base prompt ID
- [x] **STORE-04**: SetLatestAlias method to manage @latest alias
- [x] **STORE-05**: InMemory store implementation for testing
- [x] **STORE-06**: File-based store implementation with JSON persistence
- [x] **STORE-07**: File locking with syscall.Flock for cross-process safety
- [x] **STORE-08**: Optional caching for file store
- [x] **STORE-09**: PromptStoreProvider for DI integration
- [x] **STORE-10**: Store returns nil (not error) when prompt not found for Load/Delete

### Prompt Manager

- [x] **MGR-01**: Extended PromptManager interface with GetPromptByID, GetPromptWithContext, SetPrompt, DeletePrompt, ListPrompts, RenderPrompt, GetStore methods
- [x] **MGR-02**: Lazy initialization pattern for built-in prompts (load from embedded FS on first access)
- [x] **MGR-03**: Bootstrap built-in prompts as version 1.0.0 with aliases
- [x] **MGR-04**: Template rendering using text/template with PromptContext variables
- [x] **MGR-05**: Thread-safe loading with sync.Once per built-in prompt
- [x] **MGR-06**: Backward compatibility with existing GetCompacterPrompt, GetSystemPrompt, GetSupervisorPrompt, GetSubagentPrompt methods
- [x] **MGR-07**: IsBuiltin flag prevents deletion of built-in prompts

### Configuration (Langfuse v1.1)

- [ ] **CFG-01**: LangfuseConfig struct with host, public key, secret key, enabled flag
- [ ] **CFG-02**: LangfuseConfig getter method on ConfigService (GetLangfuseConfig)
- [ ] **CFG-03**: Environment variable configuration for LANGFUSE_ENABLED, LANGFUSE_HOST, LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY
- [ ] **CFG-04**: Flush settings via LANGFUSE_FLUSH_INTERVAL and LANGFUSE_MAX_QUEUE_SIZE

### Client Initialization (Langfuse v1.1)

- [ ] **CLI-01**: Lazy Langfuse client initialization (no overhead when disabled)
- [ ] **CLI-02**: Client init failure handling (log warning, disable tracing for session)
- [ ] **CLI-03**: Thread-safe client initialization with sync.Mutex

### Hook Structure (Langfuse v1.1)

- [ ] **HOOK-01**: LangfuseHook struct following logging_hook.go pattern
- [ ] **HOOK-02**: NewLangfuseHook with DI injection (injector do.Injector)
- [ ] **HOOK-03**: NewLangfuseHookProvider returning hooks.HookFunc
- [ ] **HOOK-04**: RegisterLangfuseHooks function for all hook points
- [ ] **HOOK-05**: TraceContext struct with TraceID, RootSpan, Spans map, SessionID, CreatedAt

### Trace Context Management (Langfuse v1.1)

- [ ] **CTX-01**: Thread-safe trace context storage with sync.RWMutex
- [ ] **CTX-02**: Session to trace mapping with UUID keys (map[uuid.UUID]*TraceContext)
- [ ] **CTX-03**: Trace ID propagation via HookContext.Data["langfuse_trace_id"]
- [ ] **CTX-04**: getTraceContext method with RLock/RUnlock
- [ ] **CTX-05**: createTraceContext method with Lock/Unlock
- [ ] **CTX-06**: removeTraceContext method with Lock/Unlock

### LLM Tracing (Langfuse v1.1)

- [ ] **LLM-01**: BeforeLLMRequest hook creates LLM span with model and input
- [ ] **LLM-02**: AfterLLMResponse hook updates span with output, tokens, latency
- [ ] **LLM-03**: OnLLMError hook marks span as failed with error details

### Tool Tracing (Langfuse v1.1)

- [ ] **TOOL-01**: BeforeToolExecution hook creates tool span with name and input
- [ ] **TOOL-02**: AfterToolExecution hook finalizes tool span with output
- [ ] **TOOL-03**: OnToolError hook marks tool span as failed

### Agent Lifecycle Tracing (Langfuse v1.1)

- [ ] **AGT-01**: BeforeAgentSpawn hook creates agent span with parent metadata
- [ ] **AGT-02**: AfterAgentSpawn hook finalizes spawn span with new agent ID
- [ ] **AGT-03**: BeforeAgentRemove hook creates removal span
- [ ] **AGT-04**: AfterAgentRemove hook finalizes removal span

### Session Tracing (Langfuse v1.1)

- [ ] **SES-01**: BeforeSessionStart hook creates Langfuse trace and root span
- [ ] **SES-02**: AfterSessionEnd hook ends root span and flushes traces
- [ ] **SES-03**: TraceContext stored in map on session start
- [ ] **SES-04**: TraceContext removed from map on session end

### Flush and Shutdown (Langfuse v1.1)

- [ ] **FLUSH-01**: Shutdown() method flushes buffered traces before exit
- [ ] **FLUSH-02**: Flush error handling with logging (non-fatal)

### Integration (Langfuse v1.1)

- [ ] **DI-01**: LangfuseHook registration in DI container (pkg/di/container.go)
- [ ] **DI-02**: Exported NewLangfuseHookProvider in pkg/builtin/provider.go

### Testing (Langfuse v1.1)

- [ ] **TEST-01**: Unit tests with centralized mocks (pkg/mocks/)
- [ ] **TEST-02**: Concurrent access tests for trace context map
- [ ] **TEST-03**: Integration tests with mock Langfuse server

### Documentation (Langfuse v1.1)

- [ ] **DOC-01**: CLAUDE.md update with Langfuse configuration examples
- [ ] **DOC-02**: Knowledge base guidance: guide.golang.langfuse-tracing.md

### Prompt Optimizer (v1.0 - Deferred to v2)

- [ ] **OPT-01**: PromptOptimizer interface with Optimize method taking OptimizerInput and returning OptimizerResult
- [ ] **OPT-02**: Three optimization strategies: gradient, metaprompt, prompt_memory
- [ ] **OPT-03**: Gradient strategy with reflection loop (think/critique tools) and recommend decision
- [ ] **OPT-04**: Meta-prompt strategy combining reflection and update in single phase
- [ ] **OPT-05**: Prompt memory strategy with single-shot optimization
- [ ] **OPT-06**: Min-max reflection steps (configurable, default 2-5)
- [ ] **OPT-07**: Early exit when recommend indicates no adjustment needed
- [ ] **OPT-08**: Separate strategy files in strategies/ subdirectory
- [ ] **OPT-09**: Tool registration (think, critique, recommend) for gradient strategy
- [ ] **OPT-10**: Structured output using Gollem ResponseSchema

## v2 Requirements

Deferred to future release. Acknowledged but not in current roadmap.

### Extended Storage

- **STORE-V2-01**: Langfuse store implementation for remote prompt management
- **STORE-V2-02**: Database store (PostgreSQL/MySQL) for multi-writer scenarios
- **STORE-V2-03**: Vector-based semantic search for similar trajectories

### Advanced Features

- **OPT-V2-01**: Multi-prompt optimization with credit assignment
- **OPT-V2-02**: Background asynchronous optimization
- **OPT-V2-03**: Metrics collection (latency, token count, cost)
- **OPT-V2-04**: Prompt rollback UI for version comparison

### Memory Integration

- **MEM-V2-01**: Memory Manager for automatic memory extraction
- **MEM-V2-02**: Vector database integration for semantic memory retrieval

### Advanced Langfuse Features (v1.2+)

- **LANG-V2-01**: Langfuse prompt library integration for hosted prompts
- **LANG-V2-02**: Prompt versioning via Langfuse
- **LANG-V2-03**: Real-time streaming traces
- **LANG-V2-04**: Custom span propagation

## Out of Scope

| Feature | Reason |
|---------|--------|
| Real-time optimization during execution | Optimization happens at session end to avoid performance impact |
| Multi-tenant prompt sharing | Single-tenant initially; sharing adds complexity |
| Prompt A/B testing framework | Focus on single-strategy optimization first |
| Automatic memory extraction | Separate component; manual trajectory capture sufficient for v1 |
| Semantic search | Requires vector database; file-based storage sufficient for v1 |
| Web UI for prompt management | CLI/API sufficient; UI can be added later |
| Other observability backends (OpenTelemetry, Prometheus) | Langfuse first, others in future milestones |
| Real-time streaming traces | Tracing buffered and flushed on completion, not real-time |
| Automatic instrumentation | Manual hook registration, not compile-time injection |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CFG-01 | Phase 6 | Pending |
| CFG-02 | Phase 6 | Pending |
| CFG-03 | Phase 6 | Pending |
| CFG-04 | Phase 6 | Pending |
| CLI-01 | Phase 6 | Pending |
| CLI-02 | Phase 6 | Pending |
| CLI-03 | Phase 6 | Pending |
| HOOK-01 | Phase 7 | Pending |
| HOOK-02 | Phase 7 | Pending |
| HOOK-03 | Phase 7 | Pending |
| HOOK-04 | Phase 7 | Pending |
| HOOK-05 | Phase 7 | Pending |
| CTX-01 | Phase 7 | Pending |
| CTX-02 | Phase 7 | Pending |
| CTX-03 | Phase 7 | Pending |
| CTX-04 | Phase 7 | Pending |
| CTX-05 | Phase 7 | Pending |
| CTX-06 | Phase 7 | Pending |
| LLM-01 | Phase 8 | Pending |
| LLM-02 | Phase 8 | Pending |
| LLM-03 | Phase 8 | Pending |
| TOOL-01 | Phase 9 | Pending |
| TOOL-02 | Phase 9 | Pending |
| TOOL-03 | Phase 9 | Pending |
| AGT-01 | Phase 9 | Pending |
| AGT-02 | Phase 9 | Pending |
| AGT-03 | Phase 9 | Pending |
| AGT-04 | Phase 9 | Pending |
| SES-01 | Phase 10 | Pending |
| SES-02 | Phase 10 | Pending |
| SES-03 | Phase 10 | Pending |
| SES-04 | Phase 10 | Pending |
| FLUSH-01 | Phase 10 | Pending |
| FLUSH-02 | Phase 10 | Pending |
| DI-01 | Phase 7 | Pending |
| DI-02 | Phase 7 | Pending |
| TEST-01 | Phase 11 | Pending |
| TEST-02 | Phase 11 | Pending |
| TEST-03 | Phase 11 | Pending |
| DOC-01 | Phase 11 | Pending |
| DOC-02 | Phase 11 | Pending |
| TYPE-01 | Phase 1 | Complete |
| TYPE-02 | Phase 1 | Complete |
| TYPE-03 | Phase 1 | Complete |
| TYPE-04 | Phase 1 | Complete |
| TYPE-05 | Phase 1 | Complete |
| TYPE-06 | Phase 1 | Complete |
| STORE-01 | Phase 1 | Complete |
| STORE-02 | Phase 1 | Complete |
| STORE-03 | Phase 1 | Complete |
| STORE-04 | Phase 1 | Complete |
| STORE-05 | Phase 1 | Complete |
| STORE-06 | Phase 1 | Complete |
| STORE-07 | Phase 1 | Complete |
| STORE-08 | Phase 1 | Complete |
| STORE-09 | Phase 1 | Complete |
| STORE-10 | Phase 1 | Complete |
| TEST-01 | All | Pending |
| TEST-02 | All | Pending |
| TEST-04 | All | Pending |
| TEST-05 | All | Pending |
| TEST-06 | All | Pending |
| TEST-03 | Phase 2 | Complete |
| TEST-07 | All | Pending |
| MGR-01 | Phase 2 | Complete |
| MGR-02 | Phase 2 | Complete |
| MGR-03 | Phase 2 | Complete |
| MGR-04 | Phase 2 | Complete |
| MGR-05 | Phase 2 | Complete |
| MGR-06 | Phase 2 | Complete |
| MGR-07 | Phase 2 | Complete |
| OPT-01 | Phase 3 | Pending |
| OPT-02 | Phase 3 | Pending |
| OPT-03 | Phase 3 | Pending |
| OPT-04 | Phase 3 | Pending |
| OPT-05 | Phase 3 | Pending |
| OPT-06 | Phase 3 | Pending |
| OPT-07 | Phase 3 | Pending |
| OPT-08 | Phase 3 | Pending |
| OPT-09 | Phase 3 | Pending |
| OPT-10 | Phase 3 | Pending |
| CFG-01 | Phase 4 | Complete |
| CFG-02 | Phase 4 | Complete |
| CFG-03 | Phase 4 | Complete |
| CFG-04 | Phase 4 | Complete |
| DI-01 | Phase 4 | Complete |
| DI-02 | Phase 4 | Complete |
| DI-03 | Phase 4 | Complete |
| DOC-01 | Phase 5 | Pending |
| DOC-02 | Phase 5 | Pending |
| DOC-03 | Phase 5 | Pending |
| DOC-04 | Phase 5 | Pending |
| DCON-01 | All | Pending |
| DCON-02 | All | Pending |
| DCON-03 | All | Pending |
| DCON-04 | All | Pending |
| DCON-05 | All | Pending |

**Coverage:**
- v1.0 requirements: 56 total (Complete)
- v1.1 requirements: 39 total (Langfuse integration)
- Mapped to phases: 95
- Unmapped: 0 ✓

**Phase Distribution:**
- Phase 1: 22 requirements (Core Types + Store Layer) - Complete
- Phase 2: 7 requirements (Prompt Manager) - Complete
- Phase 3: 10 requirements (Prompt Optimizer) - Complete
- Phase 4: 7 requirements (Config + DI) - Complete
- Phase 5: 4 requirements (Documentation) - Complete
- Phase 6: 7 requirements (Langfuse Configuration + Client Init)
- Phase 7: 13 requirements (Langfuse Hook Structure + Trace Context + DI)
- Phase 8: 3 requirements (LLM Tracing)
- Phase 9: 7 requirements (Tool + Agent Lifecycle Tracing)
- Phase 10: 6 requirements (Session Tracing + Flush/Shutdown)
- Phase 11: 5 requirements (Testing + Documentation)

---
*Requirements defined: 2025-02-01*
*Last updated: 2026-02-11 for milestone v1.1*
