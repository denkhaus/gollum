---
phase: 011-testing-documentation
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/builtin/langfuse_hook_test.go
autonomous: true

must_haves:
  truths:
    - Existing test file has ~2661 lines with 51.8% coverage (as of iteration 2)
    - Tests already cover LLM, tool, agent, and session tracing
    - Focus on filling specific coverage gaps identified by go test -coverprofile
    - Provider functions (NewLangfuseHook, NewLangfuseHookProvider, NewLangfuseHooksProvider) have 0% coverage
    - File operation hooks (8 stub functions) have 0% coverage - these are intentionally unimplemented
    - No new concurrent tests needed (TestLangfuseHook_ConcurrentAccess_Integration already exists)
    - Integration tests are deferred to Plan 02
  artifacts:
    - path: pkg/builtin/langfuse_hook_test.go
      provides: Extended unit tests for LangfuseHook
      min_lines: 2750 (current ~2661 + ~90 new)
      coverage_target: 60%
  key_links:
    - from: pkg/builtin/langfuse_hook_test.go
      to: pkg/mocks/
      via: Import statements
      pattern: mocks\\.NewMock
---

<objective>
Fill specific unit test coverage gaps for LangfuseHook using centralized mocks.

Purpose: Incrementally improve test coverage from 71% to 75% by targeting uncovered code paths identified by coverage profile.
Output: Targeted new tests for specific gaps without duplicating existing functionality.
</objective>

<execution_context>
@/home/denkhaus/.claude/get-shit-done/workflows/execute-plan.md
@/home/denkhaus/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@pkg/builtin/langfuse_hook.go
@pkg/builtin/langfuse_hook_test.go
@pkg/mocks/generate.go
</context>

<tasks>

<task type="manual">
  <name>Analyze coverage gaps using go test -coverprofile</name>
  <files>pkg/builtin/langfuse_hook_test.go</files>
  <action>
    Run coverage analysis to identify specific gaps:
    ```bash
    cd /home/denkhaus/dev/gomodules/gollum
    go test -v -coverprofile=coverage.out ./pkg/builtin/
    go tool cover -html=coverage.out -o coverage.html
    go tool cover -func=coverage.out | grep langfuse_hook
    ```

    Identify:
    1. Uncovered lines in langfuse_hook.go
    2. Edge cases in existing functions (nil checks, empty strings)
    3. Error paths not exercised
    4. Branch conditions not covered

    Document findings in comments at end of this task.
  </action>
  <verify>go tool cover -func=coverage.out | grep -E "langfuse_hook.*0.0%|langfuse_hook.*[0-9]\\.0%"</verify>
  <done>Coverage gaps documented with specific line numbers and functions</done>
</task>

<task type="auto">
  <name>Add tests for provider functions and uncovered error paths</name>
  <files>pkg/builtin/langfuse_hook_test.go</files>
  <action>
    Based on coverage analysis (51.8% current), add tests for uncovered functions:

    1. Provider function tests (lines 49, 65, 81 - currently 0%):
       - TestNewLangfuseHook_DI: Verify DI injector dependencies are resolved
       - TestNewLangfuseHookProvider_ReturnsHookFunc: Validate HookFunc signature
       - TestNewLangfuseHooksProvider_ReturnsHookInstance: Validate instance creation

    2. Error path tests:
       - getClient() edge cases: Empty host, zero flush interval, malformed URLs
       - Hook methods with disabled config: Verify no-ops when LangfuseEnabled=false
       - Missing span ID handling: afterLLMResponseHook, afterToolExecutionHook, afterAgentSpawnHook
       - Nil context handling: All hook methods with nil HookContext

    3. File operation hooks (lines 778-816 - 8 stub functions):
       These are intentionally unimplemented (just call propagateTraceID and return next()).
       Add a single test documenting this exclusion:
       ```go
       func TestLangfuseHook_FileOperationHooks_AreStubs(t *testing.T) {
           // File operation hooks are stubs that only propagate trace ID
           // They exist for future extensibility but currently have no behavior
           // This test documents this intentional coverage gap
       }
       ```

    Pattern for new tests:
    ```go
    func TestLangfuseHook_CoverageGaps(t *testing.T) {
        tests := []struct {
            name string
            setup func(*testing.T) *LangfuseHook
            test  func(*testing.T, *LangfuseHook)
        }{
            // provider function tests
            // error path tests
        }
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                hook := tt.setup(t)
                tt.test(t, hook)
            })
        }
    }
    ```

    Use centralized mocks from pkg/mocks for DI dependencies.
  </action>
  <verify>go test -v -run "TestLangfuseHook_CoverageGaps|TestNewLangfuseHook.*Provider" ./pkg/builtin/</verify>
  <done>New tests pass and coverage increases to at least 60% (from 51.8%)</done>
</task>

<task type="auto">
  <name>Verify no race conditions in existing tests</name>
  <files>pkg/builtin/langfuse_hook_test.go</files>
  <action>
    Run race detector to verify thread safety:
    ```bash
    cd /home/denkhaus/dev/gomodules/gollum
    go test -race -v ./pkg/builtin/
    ```

    If races detected:
    1. Identify the race location
    2. Add sync.RWMutex protection if missing in source
    3. Or update test to avoid race condition
    4. Re-run until clean

    Note: TestLangfuseHook_ConcurrentAccess_Integration (line 1903) already tests concurrent access.
    Do NOT add new concurrent tests unless coverage analysis reveals missing scenarios.
  </action>
  <verify>go test -race ./pkg/builtin/ 2>&1 | grep -E "(WARNING|DATA RACE)" || echo "No races detected"</verify>
  <done>All tests pass without race detector warnings</done>
</task>

</tasks>

<verification>
Overall phase checks:
- Coverage gaps identified using go test -coverprofile
- Tests added for specific uncovered code paths
- No duplication of existing concurrent access test
- go test -race passes without warnings
- Final coverage >= 75% (3% improvement from baseline 71%)
</verification>

<success_criteria>
1. Coverage analysis performed with documented gaps
2. Targeted tests added for uncovered paths including provider functions
3. Coverage increased from 51.8% to at least 60%
4. File operation hook stubs documented with exclusion test
5. No race conditions detected
6. No duplication of TestLangfuseHook_ConcurrentAccess_Integration
7. Integration tests deferred to Plan 02
</success_criteria>

<output>
After completion, create `.planning/phases/011-testing-documentation/011-testing-documentation-01-SUMMARY.md`
</output>
