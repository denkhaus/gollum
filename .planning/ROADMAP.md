# Roadmap: Prompt Optimizer for Gollum

## Overview

Build a prompt optimization system for the Gollum agent framework that automatically improves agent prompts using execution trajectory feedback. Starting with foundational types and storage layer, then extending the prompt manager, implementing the optimizer with three strategies, integrating with configuration/DI, and capturing knowledge in documentation.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Core Types and Store Layer** - Foundation types and persistence abstraction
- [ ] **Phase 2: Prompt Manager Extension** - ID-based prompt management with lazy loading
- [ ] **Phase 3: Prompt Optimizer** - Three optimization strategies with LLM integration
- [ ] **Phase 4: Configuration and DI Integration** - System integration and wiring
- [ ] **Phase 5: Documentation and Knowledge Capture** - Guidance and documentation

## Phase Details

### Phase 1: Core Types and Store Layer

**Goal**: Type system and persistence abstraction for versioned prompts

**Depends on**: Nothing (first phase)

**Requirements**: TYPE-01, TYPE-02, TYPE-03, TYPE-04, TYPE-05, TYPE-06, STORE-01, STORE-02, STORE-03, STORE-04, STORE-05, STORE-06, STORE-07, STORE-08, STORE-09, STORE-10, TEST-01, TEST-02, TEST-04, TEST-05, TEST-06, DCON-01, DCON-02, DCON-03, DCON-04

**Success Criteria** (what must be TRUE):
1. Prompt struct with SemVer versioning can be created and stored
2. InMemory store implementation persists and retrieves prompts correctly in tests
3. File store implementation persists prompts as JSON with file locking for concurrent safety
4. Alias resolution converts shortcuts like "subagent" to versioned IDs like "subagent@1.0.0"
5. Store returns nil (not error) when prompts are not found

**Plans**: TBD

### Phase 2: Prompt Manager Extension

**Goal**: Extended prompt management with lazy loading and template rendering

**Depends on**: Phase 1 (Store Layer)

**Requirements**: MGR-01, MGR-02, MGR-03, MGR-04, MGR-05, MGR-06, MGR-07, TEST-03, DCON-01, DCON-04

**Success Criteria** (what must be TRUE):
1. Built-in prompts (system, supervisor, compacter, subagent) load lazily from embedded FS on first access
2. Prompts render with template variables using text/template syntax
3. GetPromptByID returns versioned prompts from store or bootstraps built-ins
4. Backward-compatible methods (GetSupervisorPrompt, GetSubagentPrompt, etc.) continue to work
5. Built-in prompts are protected from deletion by IsBuiltin flag

**Plans**: TBD

### Phase 3: Prompt Optimizer

**Goal**: LLM-based prompt optimization with three strategies

**Depends on**: Phase 1 (Store Layer), Phase 2 (Prompt Manager)

**Requirements**: OPT-01, OPT-02, OPT-03, OPT-04, OPT-05, OPT-06, OPT-07, OPT-08, OPT-09, OPT-10, TEST-07, DCON-01, DCON-02

**Success Criteria** (what must be TRUE):
1. Optimizer accepts trajectories (Gollem messages with optional feedback) and current prompt
2. Gradient strategy runs reflection loop with think/critique/recommend tools
3. Meta-prompt strategy combines reflection and update in single phase
4. Prompt memory strategy performs single-shot optimization
5. Optimizer returns new prompt version with incremented SemVer and change description

**Plans**: TBD

### Phase 4: Configuration and DI Integration

**Goal**: System integration through configuration and dependency injection

**Depends on**: Phase 1 (Store Layer), Phase 2 (Prompt Manager), Phase 3 (Prompt Optimizer)

**Requirements**: CFG-01, CFG-02, CFG-03, CFG-04, DI-01, DI-02, DI-03, DCON-01, DCON-04

**Success Criteria** (what must be TRUE):
1. PromptStore config selects store type (memory/file) via environment variables
2. PromptOptimizer config specifies default strategy, LLM provider, reflection steps
3. PromptStoreProvider registers in DI container and provides configured store
4. PromptOptimizer registers in DI container with LLM client dependency
5. PromptManager uses PromptStoreProvider from DI container

**Plans**: TBD

### Phase 5: Documentation and Knowledge Capture

**Goal**: Universal knowledge captured in guidance documentation

**Depends on**: Phase 1, Phase 2, Phase 3, Phase 4 (all phases complete)

**Requirements**: DOC-01, DOC-02, DOC-03, DOC-04, DCON-01, DCON-05

**Success Criteria** (what must be TRUE):
1. CLAUDE.md updated with Prompt Optimizer usage examples and configuration
2. guide.golang.prompt-store.md created in knowledge base with store patterns
3. guide.golang.prompt-optimizer.md created in knowledge base with optimization patterns
4. guide.general.prompt-optimization.md created with universal optimization concepts
5. All guidance files are general/universal (no project-specific paths or details)

**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Core Types and Store Layer | 0/TBD | Not started | - |
| 2. Prompt Manager Extension | 0/TBD | Not started | - |
| 3. Prompt Optimizer | 0/TBD | Not started | - |
| 4. Configuration and DI Integration | 0/TBD | Not started | - |
| 5. Documentation and Knowledge Capture | 0/TBD | Not started | - |
