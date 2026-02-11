---
phase: 07-langfusehook-struct-and-basic-registration
plan: 01b
type: execute
wave: 2
depends_on: ["07-01"]
files_modified:
  - pkg/builtin/langfuse_hook_test.go
autonomous: true

must_haves:
  truths:
    - "getTraceContext retrieves existing TraceContext by session UUID with RLock/RUnlock"
    - "createTraceContext creates new TraceContext with Lock/Unlock and stores in map"
    - "removeTraceContext deletes TraceContext from map with Lock/Unlock"
    - "Unit tests verify thread-safe CRUD operations with concurrent access patterns"
    - "TraceContext struct has TraceID, RootSpan, Spans, SessionID, CreatedAt fields"
  artifacts:
    - path: "pkg/builtin/langfuse_hook_test.go"
      provides: "Unit tests for TraceContext operations"
      contains: "TestLangfuseHook_TraceContextOperations, TestTraceContext_Struct with concurrent access tests"
      exports: []
  key_links:
    - from: "Unit tests"
      to: "traceCtxs map"
      via: "Direct map access validation"
      pattern: "traceCtxs\\[sessionID\\]"
---
<objective>
Write unit tests for TraceContext operations to verify thread-safe CRUD functionality.

Purpose: Ensure TraceContext management (get/create/remove) is thread-safe and handles all scenarios correctly.
Output: Comprehensive unit tests for TraceContext operations.
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
@.planning/phases/07-langfusehook-struct-and-basic-registration/07-01-PLAN.md

# Prior Phase Decisions (from Phase 6, 7-01)
- TraceContext: Holds TraceID, RootSpan, Spans map, SessionID, CreatedAt (from 7-01)
- Thread-safe operations: sync.RWMutex for map access (from ARCHITECTURE.md)
- CRUD methods: getTraceContext (RLock), createTraceContext (Lock), removeTraceContext (Lock) all implemented in 7-01
- CTX-04 (getTraceContext), CTX-05 (createTraceContext), CTX-06 (removeTraceContext) satisfied by 7-01

# Testing Requirements (from TESTING.md)
- Use testify/assert for assertions
- Use centralized mocks from pkg/mocks/
- Test concurrent access with goroutines
- Test both success and error cases
</context>

<tasks>

<task type="auto">
  <name>Task 1: Write unit tests for TraceContext CRUD operations</name>
  <files>pkg/builtin/langfuse_hook_test.go</files>
  <action>
    Add tests to pkg/builtin/langfuse_hook_test.go:

    ```go
    func TestLangfuseHook_TraceContextOperations(t *testing.T) {
        mockLog := mocks.NewMockLoggerService(t)
        cfg := &config.LangfuseConfig{
            LangfuseEnabled:   false,
            LangfusePublicKey: "pk-test",
            LangfuseSecretKey: "sk-test",
            LangfuseHost:      "https://cloud.langfuse.com",
        }

        hook := &LangfuseHook{
            log:        mockLog,
            config:     cfg,
            client:      nil,
            clientMu:    &sync.Mutex{},
            traceCtxs:   make(map[uuid.UUID]*TraceContext),
            traceCtxsMu: &sync.RWMutex{},
        }

        sessionID := uuid.New()

        t.Run("getTraceContext returns nil when not found", func(t *testing.T) {
            tc := hook.getTraceContext(sessionID)
            assert.Nil(t, tc, "getTraceContext should return nil for non-existent session")
        })

        t.Run("createTraceContext creates and stores context", func(t *testing.T) {
            tc := hook.createTraceContext(sessionID)

            assert.NotNil(t, tc, "createTraceContext should return non-nil TraceContext")
            assert.Equal(t, sessionID, tc.SessionID, "SessionID should match")
            assert.NotEmpty(t, tc.TraceID, "TraceID should be generated")
            assert.NotNil(t, tc.Spans, "Spans map should be initialized")
            assert.False(t, tc.CreatedAt.IsZero(), "CreatedAt should be set")
        })

        t.Run("getTraceContext returns created context", func(t *testing.T) {
            hook.createTraceContext(sessionID)
            tc := hook.getTraceContext(sessionID)

            assert.NotNil(t, tc, "getTraceContext should return created TraceContext")
            assert.Equal(t, sessionID, tc.SessionID)
        })

        t.Run("removeTraceContext deletes context", func(t *testing.T) {
            hook.createTraceContext(sessionID)
            hook.removeTraceContext(sessionID)

            // Context should be removed
            tc := hook.getTraceContext(sessionID)
            assert.Nil(t, tc, "getTraceContext should return nil after removal")
        })

        t.Run("concurrent access is thread-safe", func(t *testing.T) {
            // Create multiple session IDs
            session1 := uuid.New()
            session2 := uuid.New()
            session3 := uuid.New()

            // Run concurrent operations
            done := make(chan bool)
            go func() {
                hook.createTraceContext(session1)
                done <- true
            }()
            go func() {
                hook.createTraceContext(session2)
                done <- true
            }()
            go func() {
                hook.createTraceContext(session3)
                done <- true
            }()
            go func() {
                hook.getTraceContext(session1)
                done <- true
            }()

            // Wait for all goroutines
            for i := 0; i < 4; i++ {
                <-done
            }

            // Verify all contexts were created
            assert.NotNil(t, hook.getTraceContext(session1))
            assert.NotNil(t, hook.getTraceContext(session2))
            assert.NotNil(t, hook.getTraceContext(session3))
        })
    }
    ```

    Focus on:
    1. Nil returns for non-existent sessions
    2. Context creation with proper field initialization
    3. Context retrieval after creation
    4. Context removal
    5. Thread-safe concurrent access (no race conditions)
  </action>
  <verify>go test -v ./pkg/builtin/... -run TestLangfuseHook_TraceContextOperations passes without race detector</verify>
  <done>Unit tests verify getTraceContext, createTraceContext, removeTraceContext operations are thread-safe and handle all CRUD scenarios</done>
</task>

<task type="auto">
  <name>Task 2: Write unit tests for TraceContext struct</name>
  <files>pkg/builtin/langfuse_hook_test.go</files>
  <action>
    Add struct validation tests to pkg/builtin/langfuse_hook_test.go:

    ```go
    func TestTraceContext_Struct(t *testing.T) {
        t.Run("TraceContext has all required fields", func(t *testing.T) {
            tc := &TraceContext{
                TraceID:   "test-trace-123",
                RootSpan:  nil,
                Spans:     make(map[string]interface{}),
                SessionID:  uuid.New(),
                CreatedAt:  time.Now(),
            }

            assert.Equal(t, "test-trace-123", tc.TraceID)
            assert.Nil(t, tc.RootSpan)
            assert.NotNil(t, tc.Spans)
            assert.NotEqual(t, uuid.Nil, tc.SessionID)
            assert.False(t, tc.CreatedAt.IsZero())
        })

        t.Run("TraceContext initializes with zero values", func(t *testing.T) {
            tc := &TraceContext{}

            assert.Empty(t, tc.TraceID)
            assert.Nil(t, tc.RootSpan)
            assert.Nil(t, tc.Spans)
            assert.Equal(t, uuid.Nil, tc.SessionID)
            assert.True(t, tc.CreatedAt.IsZero())
        })
    }
    ```

    Focus on TraceContext struct field validation.
  </action>
  <verify>go test -v ./pkg/builtin/... -run TestTraceContext_Struct passes</verify>
  <done>Unit tests verify TraceContext struct fields are correctly defined and initialized</done>
</task>

</tasks>

<verification>
1. Build passes: `go build ./...`
2. Tests pass: `go test ./pkg/builtin/...`
3. TestLangfuseHook_TraceContextOperations covers get/create/remove operations
4. TestTraceContext_Struct validates struct fields
5. Concurrent access test passes without race detector
6. All existing tests still pass
</verification>

<success_criteria>
1. Unit tests for TraceContext CRUD operations created
2. Unit tests for TraceContext struct validation created
3. Thread-safe concurrent access verified
4. All tests pass without race detector
5. All existing tests still pass
</success_criteria>

<output>
After completion, create `.planning/phases/07-langfusehook-struct-and-basic-registration/07-01b-SUMMARY.md`
</output>
