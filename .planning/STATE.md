# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2025-02-01)

**Core value:** Agent quality improves iteratively through automatic prompt optimization based on execution feedback.
**Current focus:** Phase 4 - Configuration and DI Integration

## Current Position

Phase: 4 of 5 (Configuration and DI Integration)
Plan: 3 of 3 in current phase - COMPLETE
Status: Phase 4 complete, ready for Phase 5
Last activity: 2026-02-02 — Completed 04-03 (PromptManager DI Integration)

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**
- Total plans completed: 13
- Average duration: 11 min
- Total execution time: 2h 23min

**By Phase:**

| Phase | Plans | Complete | Total | Avg/Plan |
|-------|-------|----------|-------|----------|
| 1     | 2     | 2        | 2     | 9 min    |
| 2     | 3     | 3        | 3     | 20 min   |
| 3     | 3     | 3        | 3     | 8 min    |
| 4     | 3     | 3        | 3     | 6 min    |

**Recent Trend:**
- Last 5 plans: 12min (02-03), 3min (03-01), 13min (03-02), 9min (03-03), 7min (04-03)
- Trend: 9 min average

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- File-based storage first: Simple, reliable, no external dependencies; learn patterns before complex backends
- Supervisor agent as initial target: Main user interaction point; high-value optimization target
- All three strategies from v1: Don't know which strategy works best until we have real data
- Lazy init for Prompt Store: Resources only allocated when needed; follows Go best practices
- **SemVer versioning embedded in ID**: Used for easy version identification (e.g., subagent@1.0.0)
- **Nil returns for not-found**: Load/Delete return nil (not error) to simplify caller error handling
- **Interface consistency**: All PromptStore methods accept context.Context for future async implementations
- **Type naming**: RenderContext (not PromptContext) to avoid stuttering in prompt package
- **PromptStoreConfig location**: Moved to pkg/config to avoid import cycle between packages
- **File store alias resolution**: Uses tag scanning instead of separate alias files for simpler design
- **Nil-safe field access**: Always check for nil slices before iteration in store operations
- **Local PromptStore interface copy**: Copied PromptStore interface to prompt package to avoid import cycle
- **Optional DI injection with recover**: Used recover pattern since do v2 doesn't have TryInvoke like do v1
- **ListFilter location**: Moved to prompt/types.go instead of store package to break import cycle
- **Template file mapping**: "system" -> subagent_system_prompt.md, "supervisor" -> supervisor_system_prompt.md, "compacter" -> compacter_prompt.md, "subagent" -> subagent_task_prompt.md
- **sync.Once per built-in**: Each built-in prompt has its own sync.Once for thread-safe lazy initialization
- **Package restructuring**: Moved PromptManager to pkg/prompt/manager/ to break dependency cycle, PromptStore now injected via constructor
- **Named template execution**: Use template.Lookup() for templates with define/end blocks (systemprompt, supervisorprompt, etc.)
- **Base ID extraction**: Extract base ID from versioned prompt IDs for template name mapping (system@1.0.0 -> system)
- **Template name mapping**: Hard-coded map of prompt IDs to template names for built-in prompts
- **GetCompacterPrompt map handling**: Accept map[string]interface{} directly for template variable access
- **SaveBuiltinVersion pattern**: Separate store method for built-in prompts to set IsBuiltin=true instead of modifying SaveNewVersion signature for backward compatibility
- **Gollem.Message reuse**: Use gollem.Message for Trajectory.Messages to avoid type duplication with core library
- **Feedback polymorphism**: Accept string, *Feedback, or *EditFeedback for flexible evaluation data
- **Template placeholder syntax**: Use {variable} syntax (not Go templates) for runtime string replacement
- **Strategy pattern with factory**: NewOptimizer validates config and creates strategy-specific optimizer instances
- **Single-package optimizer**: Keep all optimizer code in one package to avoid import cycles with strategies subpackage
- **Using gollem.Agent for tool handling**: Use gollem.New() instead of manual session management for cleaner tool integration
- **Separate test packages**: Use subdirectories (integration/, strategies/) for test packages to avoid import cycles with mocks
- **Integration helper pattern**: OptimizeAndSave orchestrates load→optimize→save workflow with automatic SemVer incrementing
- **Envconfig pattern**: Use `envconfig:"FIELD_NAME" default:"value"` tags for automatic environment variable loading
- **Interface extension**: Extend existing ConfigService interface rather than creating new service for backward compatibility
- **DI injection for PromptManager**: Use do.MustInvoke[store.PromptStore] in NewPromptManagerProvider for dependency injection
- **Config-driven store selection**: PromptStore type selected via GOLLUM_PROMPT_STORE_TYPE env var (memory/file) at DI layer
- **StrategyUnknown constant**: Added to OptimizerStrategy enum for invalid strategy handling

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Session Continuity

Last session: 2026-02-02T18:05:07Z
Stopped at: Completed 04-03 (PromptManager DI Integration)
Resume file: None
