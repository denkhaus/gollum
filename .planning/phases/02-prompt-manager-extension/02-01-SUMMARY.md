---
phase: 02-prompt-manager-extension
plan: 01
subsystem: prompt-management
tags: [lazy-init, sync-once, semver, di-injection, store-integration]

# Dependency graph
requires:
  - phase: 01-core-types-and-store-layer
    provides: [Prompt type, PromptStore interface, MemoryStore/FileStore implementations, ListFilter type]
provides:
  - Extended PromptManager interface with 7 new methods for ID-based prompt access
  - Lazy built-in prompt bootstrapping from embedded FS using sync.Once
  - PromptStore integration for persistent prompt management
  - DI provider for optional PromptStore injection
affects: [02-prompt-manager-extension/02-02, 02-prompt-manager-extension/02-03]

# Tech tracking
tech-stack:
  added: [sync.Once, semver v3, do v2 optional injection pattern]
  patterns: [lazy initialization, bootstrap pattern, local interface copy to avoid import cycles]

key-files:
  created: [pkg/prompt/manager_bootstrap.go, pkg/prompt/manager_store.go, pkg/prompt/provider.go, pkg/prompt/manager_test.go]
  modified: [pkg/prompt/manager.go, pkg/prompt/types.go, pkg/di/container.go, pkg/prompt/store/provider.go]

key-decisions:
  - "Local PromptStore interface copy in prompt package to avoid import cycle"
  - "Optional DI injection using recover pattern (do v2 lacks TryInvoke)"
  - "sync.Once per built-in prompt for thread-safe lazy initialization"
  - "ListFilter moved to prompt/types.go to break import cycle"

patterns-established:
  - "Lazy Bootstrap Pattern: Use sync.Once for single initialization of expensive resources"
  - "Import Cycle Avoidance: Copy interface types locally instead of importing across package boundaries"
  - "Optional DI Pattern: Use recover() with do.MustInvoke for optional dependencies"

# Metrics
duration: 22min
completed: 2026-02-01
---

# Phase 2: Prompt Manager Extension Summary

**Lazy built-in prompt bootstrapping from embedded FS with sync.Once-based thread-safe initialization, PromptStore integration for persistent versioned prompts, and extended PromptManager interface with 7 new ID-based methods**

## Performance

- **Duration:** 22 min
- **Started:** 2026-02-01T12:38:06Z
- **Completed:** 2026-02-01T13:00:00Z (estimated)
- **Tasks:** 4
- **Files modified:** 8

## Accomplishments

- Extended PromptManager interface with 7 new methods (GetPromptByID, GetPromptWithContext, SetPrompt, DeletePrompt, ListPrompts, RenderPrompt, GetStore)
- Implemented lazy built-in prompt bootstrapping with sync.Once for thread-safe single initialization
- Integrated PromptStore for persistent prompt management with ID-based retrieval
- Added DI provider for optional PromptStore injection using recover pattern
- Created comprehensive test suite with 11 test cases including race detection

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend PromptManager interface** - `cb8dd66` (feat)
2. **Task 2: Implement lazy built-in prompt bootstrapping** - `ac1ea70` (feat)
3. **Task 3: Implement store integration and ID-based retrieval** - `a968e07` (feat)
4. **Task 4: Add tests for lazy bootstrapping** - `b222b4f` (test)

## Files Created/Modified

- `pkg/prompt/manager.go` - Extended PromptManager interface with new methods, added sync.Once fields to struct
- `pkg/prompt/manager_bootstrap.go` - Lazy initialization with sync.Once for each built-in prompt
- `pkg/prompt/manager_store.go` - PromptStore integration with ID-based retrieval, delete protection
- `pkg/prompt/provider.go` - DI provider with optional PromptStore injection
- `pkg/prompt/manager_test.go` - Test suite with 11 cases including concurrent operations
- `pkg/prompt/types.go` - Added ListFilter type to break import cycle
- `pkg/di/container.go` - Added PromptStore registration before PromptManager
- `pkg/prompt/store/provider.go` - Added NewPromptStore function for direct DI injection

## Decisions Made

- **Local PromptStore interface**: Copied PromptStore interface to prompt package to avoid import cycle (prompt imports store which imports prompt)
- **Optional DI injection with recover**: Used recover pattern since do v2 doesn't have TryInvoke like do v1
- **ListFilter location**: Moved to prompt/types.go instead of store package to break import cycle
- **Template mapping**: "system" -> subagent_system_prompt.md, "supervisor" -> supervisor_system_prompt.md, "compacter" -> compacter_prompt.md, "subagent" -> subagent_task_prompt.md

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] ToolName type mismatch in SubAgentContext**
- **Found during:** Task 1 (Extend PromptManager interface)
- **Issue:** SubAgentContext fields used shared.ToolName constants but template required string type
- **Fix:** Initially changed ToolName to typed constant, but this broke 20+ downstream files in tools package
- **Final fix:** Reverted ToolName to string constants, updated SubAgentContext in types.go to use string fields
- **Files modified:** pkg/shared/tool_names.go, pkg/prompt/types.go
- **Verification:** All packages build successfully
- **Committed in:** cb8dd66, a968e07 (part of Task 1 and Task 3 commits)

---

**Total deviations:** 1 auto-fixed (1 blocking issue)
**Impact on plan:** Fix necessary for compilation. No scope creep.

## Issues Encountered

- **Import cycle between prompt and store packages**: Resolved by creating local copy of PromptStore interface in prompt package
- **do v2 lacks TryInvoke**: Resolved by using recover pattern with do.MustInvoke for optional injection
- **Test package import cycle**: Resolved by using separate package name (prompt_test) to allow importing both prompt and store packages

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- PromptManager interface ready for template rendering implementation (02-02)
- Store integration complete and tested
- Lazy bootstrap pattern established for all built-in prompts
- DI container properly configured with PromptStore injection
- Ready for GetPromptWithContext and RenderPrompt implementation in next plan

---
*Phase: 02-prompt-manager-extension*
*Plan: 01*
*Completed: 2026-02-01*
