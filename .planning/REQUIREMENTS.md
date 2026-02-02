# Requirements: Prompt Optimizer for Gollum

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

### Prompt Optimizer

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

### Config Integration

- [ ] **CFG-01**: PromptStoreConfig with Type (memory/file), FilePath, CacheEnabled fields
- [ ] **CFG-02**: Langfuse configuration options (deferred to v2, but config structure in place)
- [ ] **CFG-03**: PromptOptimizerConfig with DefaultStrategy, DefaultProvider, MaxReflectionSteps, MinReflectionSteps
- [ ] **CFG-04**: ConfigService extension methods: GetPromptStoreConfig, GetPromptOptimizerConfig

### DI Integration

- [ ] **DI-01**: Register PromptStoreProvider in DI container
- [ ] **DI-02**: Register PromptOptimizer in DI container
- [ ] **DI-03**: Update PromptManager registration to use PromptStoreProvider

### Testing

- [ ] **TEST-01**: Store interface tests using table-driven tests
- [ ] **TEST-02**: Mock generation via centralized pkg/mocks/generate.go
- [ ] **TEST-03**: Tests for lazy init pattern
- [ ] **TEST-04**: Tests for file locking behavior
- [ ] **TEST-05**: Tests for alias resolution
- [ ] **TEST-06**: Tests for version increment logic
- [ ] **TEST-07**: Optimizer strategy tests with mocked LLM client

### Documentation

- [ ] **DOC-01**: CLAUDE.md updated with Prompt Optimizer usage
- [ ] **DOC-02**: guide.golang.prompt-store.md created in knowledge base
- [ ] **DOC-03**: guide.golang.prompt-optimizer.md created in knowledge base
- [ ] **DOC-04**: guide.general.prompt-optimization.md created in knowledge base

### Design Constraints

- [ ] **DCON-01**: Follow Go guidance files in /home/denkhaus/dev/kb/guides/guide.golang.*.md for every phase
- [ ] **DCON-02**: Use existing LLM providers (Anthropic, OpenAI, Gemini) - no new provider dependencies
- [ ] **DCON-03**: Use centralized mocks from pkg/mocks/ (uber.org/mock)
- [ ] **DCON-04**: Follow existing Gollum package structure and DI patterns
- [ ] **DCON-05**: All guidance files must be general/universal (no project-specific paths in KB)

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

## Out of Scope

| Feature | Reason |
|---------|--------|
| Real-time optimization during execution | Optimization happens at session end to avoid performance impact |
| Multi-tenant prompt sharing | Single-tenant initially; sharing adds complexity |
| Prompt A/B testing framework | Focus on single-strategy optimization first |
| Automatic memory extraction | Separate component; manual trajectory capture sufficient for v1 |
| Semantic search | Requires vector database; file-based storage sufficient for v1 |
| Web UI for prompt management | CLI/API sufficient; UI can be added later |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
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
| CFG-01 | Phase 4 | Pending |
| CFG-02 | Phase 4 | Pending |
| CFG-03 | Phase 4 | Pending |
| CFG-04 | Phase 4 | Pending |
| DI-01 | Phase 4 | Pending |
| DI-02 | Phase 4 | Pending |
| DI-03 | Phase 4 | Pending |
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
- v1 requirements: 56 total
- Mapped to phases: 56
- Unmapped: 0

**Phase Distribution:**
- Phase 1: 22 requirements (Core Types + Store Layer)
- Phase 2: 7 requirements (Prompt Manager)
- Phase 3: 10 requirements (Prompt Optimizer)
- Phase 4: 7 requirements (Config + DI)
- Phase 5: 4 requirements (Documentation)
- All phases: 6 requirements (Testing + Design Constraints)

---
*Requirements defined: 2025-02-01*
*Last updated: 2025-02-01 after roadmap creation*
