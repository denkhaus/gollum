---
phase: 08-llm-tracing
plan: 02
subsystem: tracing
tags: [langfuse, llm, hooks, spans, error-handling]

# Dependency graph
requires:
  - phase: 08-llm-tracing
    plan: 01
    provides: [LLMSpanContext placeholder struct, beforeLLMRequestHook, afterLLMResponseHook, span correlation pattern]
provides:
  - onLLMErrorHook with ERROR level marking and error message propagation
  - Partial response preservation for streaming error scenarios
  - Unit tests for error marking with edge cases
  - Integration tests for full LLM span lifecycle (success and error paths)
affects: [09-tool-agent-tracing, 10-session-tracing]

# Tech tracking
tech-stack:
  added: []
  patterns: [error marking with Level/StatusMessage, partial response preservation, span lifecycle integration tests]

key-files:
  created: []
  modified: [pkg/builtin/langfuse_hook.go, pkg/builtin/langfuse_hook_test.go]

key-decisions:
  - "Partial response preservation: onLLMErrorHook preserves LLMResponse in Output field for streaming errors"
  - "Nil error handling: Use 'unknown error' message when LLMError is nil"
  - "Lock ordering consistency: Release traceCtxsMu before calling propagateTraceID to avoid deadlock"

patterns-established:
  - "LLM error lifecycle: Create span in before -> mark ERROR in onLLMError with status message"
  - "Graceful degradation: Skip error marking when span ID missing or Langfuse disabled"
  - "Thread-safe error marking: Lock/unlock around span state mutations"

# Metrics
duration: 8min
completed: 2026-02-11T11:44:06Z
---

# Phase 08 Plan 02: LLM Error Handling Summary

**OnLLMError hook marks LLMSpanContext as failed with ERROR level, propagates error message to StatusMessage, preserves partial response for streaming errors**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-11T11:36:06Z
- **Completed:** 2026-02-11T11:44:06Z
- **Tasks:** 3
- **Files modified:** 2

## Accomplishments

- Implemented onLLMErrorHook to mark LLMSpanContext as failed with ObservationLevel.ERROR
- Added error message propagation from LLMError.Error() to StatusMessage field
- Implemented partial response preservation for streaming error scenarios
- Added comprehensive unit tests covering error marking, disabled state, nil error, and missing span ID
- Added integration tests verifying full LLM span lifecycle for success and error paths

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement onLLMErrorHook to mark spans as failed** - `9b7f642` (feat)
2. **Fix: Preserve partial response in onLLMErrorHook** - `2728021` (fix)
3. **Task 2: Write unit tests for OnLLMError hook** - `be8d28e` (test)
4. **Task 3: Add integration tests for full LLM span lifecycle** - `ba2679f` (test)

**Plan metadata:** N/A (plan execution)

_Note: Task 1 was followed by a fix commit for partial response preservation_

## Files Created/Modified

- `pkg/builtin/langfuse_hook.go` - Added onLLMErrorHook with ERROR level marking, error message propagation, and partial response preservation
- `pkg/builtin/langfuse_hook_test.go` - Added TestLangfuseHook_OnLLMError (unit tests) and TestLangfuseHook_LLMSpanLifecycle_Integration (integration tests)

## Decisions Made

### Key Decisions

- **Partial response preservation**: When LLMError occurs, onLLMErrorHook preserves any partial response in LLMResponse field to the span's Output. This handles streaming errors where some response was received before failure.
- **Nil error handling**: When LLMError is nil but error hook is called, use "unknown error" as StatusMessage rather than empty string.
- **Lock ordering consistency**: Following pattern from Phase 08-01, release traceCtxsMu before calling propagateTraceID to avoid deadlock with getTraceContext's RLock.
- **Integration test credentials**: Added LangfusePublicKey and LangfuseSecretKey to test config to enable successful client initialization for span creation.

### Patterns Established

- **LLM error lifecycle**: Create span in before hook -> mark ERROR in onLLMError hook with status message and optional partial output
- **Graceful degradation**: Skip error marking when span ID is missing, trace context doesn't exist, or Langfuse is disabled
- **Thread-safe error marking**: Lock/unlock around span state mutations in TraceContext.Spans map

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed partial response preservation**
- **Found during:** Task 3 (Integration tests)
- **Issue:** Test for "LLM call with partial response then error" failed because onLLMErrorHook didn't preserve the partial response in Output field. The test expected partial response to be retained when streaming error occurs.
- **Fix:** Added code in onLLMErrorHook to set spanCtx.Output from hookCtx.LLMResponse if available (handles streaming error mid-response scenario)
- **Files modified:** pkg/builtin/langfuse_hook.go
- **Verification:** Integration test passes, partial response is preserved in span context
- **Committed in:** 2728021 (separate fix commit)

**2. [Rule 3 - Blocking] Added credentials to integration test config**
- **Found during:** Task 3 (Integration tests execution)
- **Issue:** Integration test failed because beforeLLMRequestHook calls getClient() which returns error when credentials not configured, preventing span creation
- **Fix:** Added LangfusePublicKey and LangfuseSecretKey to test config for successful client initialization
- **Files modified:** pkg/builtin/langfuse_hook_test.go
- **Verification:** Integration tests pass with valid credentials
- **Committed in:** ba2679f (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 bug fix for partial response, 1 blocking missing credentials)
**Impact on plan:** Both auto-fixes necessary for correctness and functionality. No scope creep.

## Issues Encountered

- **Integration test failure**: Initial integration test failed because beforeLLMRequestHook requires valid credentials to initialize client. Fixed by adding credentials to test config.
- **Partial response test failure**: Test expected partial response to be preserved when error occurs during streaming. Fixed by adding Output field preservation in onLLMErrorHook.

## User Setup Required

None - no external service configuration required for this phase. Langfuse credentials will be configured in later phases when actual SDK spans are created.

## Next Phase Readiness

**Phase 09 Readiness (Tool and Agent Tracing):**
- Error marking pattern can be applied to tool and agent error hooks
- Span lifecycle integration test pattern established for comprehensive testing
- Lock ordering and thread-safety patterns proven and ready for extension

**Phase 10 Readiness (Session Tracing):**
- LLMSpanContext placeholders include ERROR level marking ready for SDK span creation
- Partial response preservation ensures accurate span data for Langfuse submission

**Phase 8 Complete:** LLM tracing hooks (beforeLLMRequest, afterLLMResponse, onLLMError) fully implemented with comprehensive test coverage. Ready for Phase 9 (Tool and Agent Tracing).

---
*Phase: 08-llm-tracing*
*Completed: 2026-02-11*
