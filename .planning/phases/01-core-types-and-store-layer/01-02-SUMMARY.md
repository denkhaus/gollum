---
phase: 01-core-types-and-store-layer
plan: 02
subsystem: prompt-management
tags: [file-storage, json-persistence, file-locking, di-provider, table-driven-tests, mock-generation]

# Dependency graph
requires:
  - phase: 01-core-types-and-store-layer
    plan: 01
    provides: [Prompt struct with SemVer versioning, PromptStore interface, InMemory store implementation]
provides:
  - File-based PromptStore implementation with JSON persistence and syscall.Flock for concurrent access
  - DI provider for PromptStore that creates configured instances (memory/file/langfuse)
  - Comprehensive table-driven tests for both store implementations including concurrent access tests
  - Centralized mock generation for PromptStore interface
affects: [01-03-prompt-manager, 01-04-rendering-engine]

# Tech tracking
tech-stack:
  added: []
  patterns: [file-based-persistence, file-locking-with-syscall, alias-resolution-via-tag-scanning, di-provider-pattern]

key-files:
  created:
    - pkg/prompt/store/file_store.go
    - pkg/prompt/store/provider.go
    - pkg/prompt/store/store_test.go
    - pkg/mocks/mock_store.go
  modified:
    - pkg/config/service.go (added PromptStoreConfig and GetPromptStoreConfig)
    - pkg/prompt/store/store.go (removed PromptStoreConfig to avoid import cycle)
    - pkg/prompt/store/memory_store.go (added nil-safe tag handling)
    - pkg/mocks/generate.go (added PromptStore mock generation directive)

key-decisions:
  - "PromptStoreConfig moved to pkg/config to avoid import cycle between packages"
  - "File store uses tag scanning for @latest resolution instead of separate alias files"
  - "Nil-safe tag handling added to prevent panics when loading prompts with missing Tags field"

patterns-established:
  - "File store pattern: JSON files with version-based naming (baseID@version.json)"
  - "File locking pattern: syscall.Flock with LOCK_EX for exclusive write access"
  - "Alias resolution pattern: Scan files for tag matching when alias files don't exist"
  - "Nil-safe field access: Always check for nil slices before iteration in store operations"

# Metrics
duration: 12min
completed: 2026-02-01
---

# Phase 1: Plan 2 - File Persistence and DI Summary

**File-based PromptStore with JSON persistence, syscall.Flock for concurrent access, DI provider, and comprehensive table-driven tests**

## Performance

- **Duration:** 12 min (731 seconds)
- **Started:** 2026-01-31T23:51:36Z
- **Completed:** 2026-02-01T00:03:47Z
- **Tasks:** 4
- **Files modified:** 8 created, 4 modified
- **Coverage:** 71.9% for store package

## Accomplishments

- **File store implementation**: JSON persistence with one file per versioned prompt, syscall.Flock for concurrent file access safety
- **DI provider**: PromptStoreProvider that creates configured store instances (memory/file/langfuse) based on configuration
- **Comprehensive tests**: Table-driven tests for both InMemory and File stores covering all operations including concurrent access
- **Mock generation**: Centralized mock generation for PromptStore interface in pkg/mocks

## Task Commits

Each task was committed atomically:

1. **Task 1: File store implementation** - `e3c23d1` (feat)
2. **Task 2: DI provider creation** - `e1815df` (feat)
3. **Task 3: Comprehensive tests** - `06d39d8` (feat)
4. **Task 4: Mock generation** - `455b63a` (feat)

**Plan metadata:** Not yet created (will be in final commit)

## Files Created/Modified

- `pkg/prompt/store/file_store.go` - File-based PromptStore with JSON persistence, syscall.Flock, tag-based alias resolution
- `pkg/prompt/store/provider.go` - DI provider for PromptStore with configurable backend selection
- `pkg/prompt/store/store_test.go` - Comprehensive table-driven tests for both store implementations
- `pkg/config/service.go` - Added PromptStoreConfig type and GetPromptStoreConfig() method
- `pkg/mocks/generate.go` - Added PromptStore mock generation directive
- `pkg/mocks/mock_store.go` - Generated mock for PromptStore interface
- `pkg/prompt/store/store.go` - Removed PromptStoreConfig to avoid import cycle
- `pkg/prompt/store/memory_store.go` - Added nil-safe tag handling

## Decisions Made

- **PromptStoreConfig location**: Moved from pkg/prompt/store to pkg/config to avoid import cycle (config service imports prompt store, so store cannot import config)
- **File store alias resolution**: Uses tag scanning (findLatestWithTag) instead of separate alias files - simpler design, fewer files to manage
- **Nil-safe field access**: Added nil checks for Tags slices throughout store code to prevent panics when loading from JSON files

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Resolved import cycle between packages**
- **Found during:** Task 2 (DI provider creation)
- **Issue:** pkg/prompt/store/provider.go imported pkg/config, but pkg/config/service.go imported pkg/prompt/store for PromptStoreConfig
- **Fix:** Moved PromptStoreConfig from store.go to config/service.go, updated provider import
- **Files modified:** pkg/config/service.go, pkg/prompt/store/store.go, pkg/prompt/store/provider.go
- **Verification:** Build passes without import cycle errors
- **Committed in:** e1815df (Task 2 commit)

**2. [Rule 1 - Bug] Fixed nil pointer panic in SetLatestAlias**
- **Found during:** Task 3 (testing file store SetLatestAlias)
- **Issue:** v.Tags could be nil when loaded from JSON, causing panic when iterating range over nil slice
- **Fix:** Added nil check before iterating over Tags in SetLatestAlias, and nil check before appending in both stores
- **Files modified:** pkg/prompt/store/file_store.go, pkg/prompt/store/memory_store.go
- **Verification:** TestFileStore_SetLatestAlias passes, no panics
- **Committed in:** 06d39d8 (Task 3 commit)

**3. [Rule 1 - Bug] Fixed ResolveAlias not finding @latest after SetLatestAlias**
- **Found during:** Task 3 (testing ResolveAlias after SetLatestAlias)
- **Issue:** ResolveAlias("setlatest@latest") tried to load "setlatest@latest.json" which doesn't exist, and fell through without scanning for the tag
- **Fix:** Added explicit check for @latest suffix to trigger findLatestWithTag even when Load returns nil
- **Files modified:** pkg/prompt/store/file_store.go
- **Verification:** TestFileStore_ResolveAlias passes, SetLatestAlias then ResolveAlias works correctly
- **Committed in:** 06d39d8 (Task 3 commit)

**4. [Rule 1 - Bug] Fixed SaveNewVersion not removing @latest from old version files**
- **Found during:** Task 3 (testing alias migration between versions)
- **Issue:** removeAliasFromFile tried to load alias files that don't exist (e.g., "setlatest.json"), so old versions never had their @latest tags removed
- **Fix:** Replaced removeAliasFromFile with direct update of latestPrompt object before writing new version
- **Files modified:** pkg/prompt/store/file_store.go
- **Verification:** SetLatestAlias correctly moves @latest tag between versions
- **Committed in:** 06d39d8 (Task 3 commit)

**5. [Rule 1 - Bug] Fixed nil pointer dereference in test assertions**
- **Found during:** Task 3 (running tests)
- **Issue:** Test asserted p.ID without checking p != nil first, causing panic when ResolveAlias returned nil
- **Fix:** Added assert.NotNil(t, p, "ResolveAlias returned nil") before accessing p.ID
- **Files modified:** pkg/prompt/store/store_test.go
- **Verification:** Tests fail with clear error message instead of panic
- **Committed in:** 06d39d8 (Task 3 commit)

---

**Total deviations:** 5 auto-fixed (1 blocking, 4 bugs)
**Impact on plan:** All auto-fixes necessary for correctness. Import cycle was blocker; nil pointer bugs would cause production panics; alias management bugs would break core functionality. No scope creep.

## Issues Encountered

- **Import cycle**: Initial design had PromptStoreConfig in store package, creating circular dependency when DI provider needed config. Resolved by moving config type to config package.
- **Nil pointer panics**: JSON unmarshaling doesn't initialize empty slices, so Tags could be nil. Added nil checks throughout.
- **Alias file assumption**: Initially assumed alias files would exist, but file store only creates versioned files. Switched to tag-based scanning for alias resolution.

## User Setup Required

None - no external service configuration required. File store uses local filesystem with configurable path.

## Next Phase Readiness

- File persistence layer complete with JSON format and concurrent access safety
- DI provider configured to create store instances based on environment variables
- Comprehensive test coverage with concurrent access tests validates thread safety
- MockPromptStore available for testing prompt manager and rendering engine
- PromptStore interface fully implemented and ready for use by prompt manager

**No blockers or concerns** - ready to proceed with prompt manager implementation in Plan 01-03.

---
*Phase: 01-core-types-and-store-layer*
*Completed: 2026-02-01*
