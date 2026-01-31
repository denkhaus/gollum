# Phase 1: Core Types and Store Layer - Context

**Gathered:** 2025-02-01
**Status:** Ready for planning

## Phase Boundary

Type system and persistence abstraction for versioned prompts. Define core data structures (Prompt, Trajectory, Feedback), implement PromptStore interface with InMemory and File backends, and establish SemVer versioning with alias resolution. This is foundational infrastructure — all subsequent phases depend on these types and storage patterns.

## Implementation Decisions

### Version handling
- SemVer versioning with version embedded in ID (e.g., `subagent@1.0.0`)
- Auto-increment patch version in SaveNewVersion (1.0.0 → 1.0.1)
- Aliases resolve shortcuts: "subagent" → "subagent@latest" → versioned ID
- Only latest version has aliases; older versions have none
- `github.com/Masterminds/semver/v3` for version parsing

### File store design
- JSON storage: one file per prompt (`{id}.json`)
- Directory structure: `{baseDir}/{promptID}.json`
- `syscall.Flock` for cross-process file locking (Unix-only)
- Optional in-memory caching for file store
- Lazy init: built-in prompts loaded from embedded FS on first access

### Error behavior
- Load returns nil (not error) when prompt not found
- Delete returns nil (not error) when prompt not found
- IsBuiltin prompts return error on delete attempt
- Defined error types: ErrPromptNotFound, ErrPromptIsBuiltin, ErrInvalidPromptID, ErrCASFailed

### Test coverage depth
- Table-driven tests for store operations
- Mock generation via centralized pkg/mocks/generate.go
- Tests for: CRUD operations, alias resolution, version increment, file locking
- Concurrent access tests for file store locking behavior

### Package structure
- `pkg/prompt/types.go` — Core types (Prompt, PromptContext, etc.)
- `pkg/prompt/store/` — Store interface and implementations
  - `store.go` — PromptStore interface
  - `memory_store.go` — In-memory implementation
  - `file_store.go` — File-based implementation
  - `provider.go` — DI provider

### Claude's Discretion
- Exact error message wording
- Test case organization and naming
- Helper function extraction in tests

## Specific Ideas

- Follow patterns from existing Gollum codebase (see `.planning/codebase/`)
- Lazy init pattern matches Go best practices for embedded resources
- Use `github.com/Masterminds/semver/v3` for version handling

## Deferred Ideas

None — discussion stayed within phase scope.

---

*Phase: 01-core-types-and-store-layer*
*Context gathered: 2025-02-01*
