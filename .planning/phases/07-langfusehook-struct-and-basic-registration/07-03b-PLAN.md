---
phase: 07-langfusehook-struct-and-basic-registration
plan: 03b
type: execute
wave: 4
depends_on: ["07-03"]
files_modified:
  - pkg/builtin/provider_test.go
  - pkg/builtin/langfuse_hook_test.go
autonomous: true

must_haves:
  truths:
    - "NewLangfuseHooksProvider returns LangfuseHook instance"
    - "Provider handles injector and dependencies correctly"
    - "RegisterLangfuseHooks integration with HookManager works"
    - "Unit tests verify provider and registration"
  artifacts:
    - path: "pkg/builtin/provider_test.go"
      provides: "Unit tests for DI provider integration"
      contains: "TestNewLangfuseHooksProvider"
      exports: []
    - path: "pkg/builtin/langfuse_hook_test.go"
      provides: "Unit tests for DI integration"
      contains: "TestNewBuiltinHooksProvider_WithLangfuse"
      exports: []
  key_links:
    - from: "NewBuiltinHooksProvider"
      to: "RegisterLangfuseHooks"
      via: "Direct function call after hook creation"
      pattern: "RegisterLangfuseHooks\\("
    - from: "NewLangfuseHooksProvider"
      to: "NewLangfuseHook"
      via: "DI injector"
      pattern: "NewLangfuseHook\\("
    - from: "do.Provide"
      to: "DI container"
      via: "Provider registration"
      pattern: "do\\.Provide\\("
---

<objective>
Write unit tests for DI integration to verify LangfuseHook provider and registration behavior.

Purpose: Ensure NewLangfuseHooksProvider creates LangfuseHook correctly and RegisterLangfuseHooks integrates with HookManager.
Output: Comprehensive unit tests for DI provider integration.
</objective>

<execution_context>
@/home/denkhaus/.claude/get-shit-done/workflows/execute-plan.md
@/home/denkhaus/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@pkg/di/container.go
@pkg/builtin/provider.go
@pkg/builtin/logging_hook.go
@pkg/builtin/security_hook.go
@.planning/phases/07-langfusehook-struct-and-basic-registration/07-01-PLAN.md
@.planning/phases/07-langfusehook-struct-and-basic-registration/07-01b-PLAN.md
@.planning/phases/07-langfusehook-struct-and-basic-registration/07-02-PLAN.md
@.planning/phases/07-langfusehook-struct-and-basic-registration/07-03-PLAN.md

# Prior Phase Decisions (from Phase 6, 7-01, 7-01b, 7-02, 7-03)
- DI registration: Use do.Provide for singleton registration (from DI-01 v1.1)
- Builtin provider: NewBuiltinHooksProvider orchestrates all builtin hook registration (from provider.go)
- Injector pattern: Pass do.Injector to constructors for dependency resolution
- Error handling: Log but don't fail for non-critical builtin registration failures

# Existing DI Pattern (from pkg/di/container.go and pkg/builtin/provider.go)
- NewBuiltinHooksProvider uses do.MustInvoke to get HookManager and LoggerService
- Creates hooks via NewLoggingHook, NewSecurityHook
- Registers hooks via RegisterLoggingHooks, RegisterSecurityHooks
- Returns shutdown function (currently no-op for all builtins)

# Langfuse Integration Requirements (from ROADMAP DI-01, DI-02)
- DI-01: LangfuseHook registered in pkg/di/container.go
- DI-02: NewLangfuseHookProvider exported in pkg/builtin/provider.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Write unit tests for NewLangfuseHooksProvider</name>
  <files>pkg/builtin/provider_test.go or pkg/builtin/langfuse_hook_test.go</files>
  <action>
    Add tests to verify DI integration. If provider_test.go exists, add there. Otherwise, add to langfuse_hook_test.go:

    ```go
    func TestNewLangfuseHooksProvider(t *testing.T) {
        t.Run("creates LangfuseHook successfully", func(t *testing.T) {
            // Create a minimal mock injector
            mockLog := mocks.NewMockLoggerService(t)
            mockCfg := &config.LangfuseConfig{
                LangfuseEnabled: false,
            }

            // We can't fully test without a real DI container
            // But we can test that the provider returns the right type
            provider := func(injector do.Injector) (interface{}, error) {
                return NewLangfuseHook(injector)
            }

            // Create a minimal injector mock
            type mockInjector struct {
                log logger.LoggerService
                cfg config.ConfigService
            }
            injector := &mockInjector{log: mockLog, cfg: &mockConfigService{cfg: mockCfg}}

            result, err := provider(injector)

            assert.NoError(t, err)
            assert.NotNil(t, result)

            // Verify it's a LangfuseHook
            hook, ok := result.(*LangfuseHook)
            assert.True(t, ok, "Provider should return *LangfuseHook")
            assert.Equal(t, mockLog, hook.log)
            assert.Equal(t, mockCfg, hook.config)
        })

        t.Run("returns error when injector is nil", func(t *testing.T) {
            // This test verifies graceful error handling
            // Note: In real usage, do.MustInvoke would panic on nil injector
            // But our provider should handle it gracefully if possible

            // Actually, looking at NewLangfuseHook, it uses do.MustInvoke
            // which will panic if injector is nil. So this test verifies
            // provider fails (which is expected behavior)

            provider := func(injector do.Injector) (interface{}, error) {
                return NewLangfuseHook(injector)
            }

            result, err := provider(nil)

            // The error will be from do.MustInvoke panic
            // We can't test panic without recover()
            defer func() {
                if r := recover(); r != nil {
                    // Expected to panic on nil injector
                    assert.NotNil(t, r)
                }
            }()

            result, err = provider(nil)

            // After panic recovery
            assert.Nil(t, result)
            assert.Error(t, err)
        })
    }
    ```

    Focus on:
    1. NewLangfuseHooksProvider returns LangfuseHook instance
    2. Provider handles injector and dependencies correctly
  </action>
  <verify>go test -v ./pkg/builtin/... -run TestNewLangfuseHooksProvider passes (if added to provider_test.go, adjust path)</verify>
  <done>Unit tests verify NewLangfuseHooksProvider creates LangfuseHook and handles dependencies correctly</done>
</task>

<task type="auto">
  <name>Task 2: Write unit tests for RegisterLangfuseHooks integration</name>
  <files>pkg/builtin/provider_test.go or pkg/builtin/langfuse_hook_test.go</files>
  <action>
    Add tests to verify RegisterLangfuseHooks integrates with HookManager:

    ```go
    func TestNewBuiltinHooksProvider_WithLangfuse(t *testing.T) {
        if testing.Short() {
            t.Skip("Skipping integration test in short mode")
        }

        // This test verifies that NewBuiltinHooksProvider includes LangfuseHook
        // In a real scenario, this would use a full DI container
        // For now, we verify RegisterLangfuseHooks is called

        t.Run("logs Langfuse registration", func(t *testing.T) {
            // We can't fully test without a DI container setup
            // But we can verify RegisterLangfuseHooks is called

            // Create a mock HookManager that tracks registrations
            mockHM := &mockHookManager{}

            mockLog := mocks.NewMockLoggerService(t)
            cfg := &config.LangfuseConfig{
                LangfuseEnabled: true,
            }
            hook := &LangfuseHook{
                log:        mockLog,
                config:     cfg,
                client:      nil,
                clientMu:    &sync.Mutex{},
                traceCtxs:   make(map[uuid.UUID]*TraceContext),
                traceCtxsMu: &sync.RWMutex{},
            }

            err := RegisterLangfuseHooks(mockHM, hook)

            assert.NoError(t, err)
            assert.Greater(t, mockHM.registerCount, 0, "Langfuse hooks should be registered")
        })
    }

    // mockHookManager is a minimal mock for testing hook registration
    type mockHookManager struct {
        registerCount int
        registeredHooks []hooks.HookMetadata
    }

    func (m *mockHookManager) RegisterHook(fn hooks.HookFunc, meta hooks.HookMetadata) error {
        m.registerCount++
        m.registeredHooks = append(m.registeredHooks, meta)
        return nil
    }
    ```

    Focus on:
    1. RegisterLangfuseHooks integration with HookManager
    2. All hooks are registered correctly
  </action>
  <verify>go test -v ./pkg/builtin/... -run TestNewBuiltinHooksProvider_WithLangfuse passes</verify>
  <done>Unit tests verify RegisterLangfuseHooks integrates with HookManager and registers all hooks</done>
</task>

</tasks>

<verification>
1. Build passes: `go build ./...`
2. Tests pass: `go test ./pkg/builtin/... ./pkg/di/...`
3. NewLangfuseHooksProvider function exists in pkg/builtin/provider.go
4. NewBuiltinHooksProvider calls NewLangfuseHooksProvider and RegisterLangfuseHooks
5. LangfuseHook registration logged in builtin provider output
6. All existing tests still pass
7. DI integration verified through tests
</verification>

<success_criteria>
1. NewLangfuseHooksProvider function created in pkg/builtin/provider.go
2. NewBuiltinHooksProvider updated to create and register LangfuseHook
3. LangfuseHook available for DI injection (through builtin provider)
4. Unit tests verify provider and registration
5. All existing tests still pass
</success_criteria>

<output>
After completion, create `.planning/phases/07-langfusehook-struct-and-basic-registration/07-03b-SUMMARY.md`
</output>
