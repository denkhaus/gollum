package hooks

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testFilePath        = "/test/file.txt"
	testSuspiciousPath  = "/test/../etc/passwd"
	testContent         = "content"
	testModifiedContent = "modified"
	testOriginalContent = "original content"
	testLLMPrompt       = "test prompt"
	testLLMModel        = "claude-3-5-sonnet"
	testLLMResponse     = "LLM response"
)

// mockLogger is a simple mock logger for testing.
// Note: Using local mock instead of pkg/mocks to avoid import cycle:
// pkg/hooks → pkg/mocks → pkg/hooks (from mock_hook_manager.go)
type mockLogger struct{}

func (m *mockLogger) Debug(_ string, _ ...zap.Field) {}

func (m *mockLogger) Debugf(_ string, _ ...any) {}

func (m *mockLogger) Info(_ string, _ ...zap.Field) {}

func (m *mockLogger) Infof(_ string, _ ...any) {}

func (m *mockLogger) Warn(_ string, _ ...zap.Field) {}

func (m *mockLogger) Warnf(_ string, _ ...any) {}

func (m *mockLogger) Error(_ string, _ ...zap.Field) {}

func (m *mockLogger) Errorf(_ string, _ ...any) {}

func (m *mockLogger) GetLogger() *zap.Logger                       { return nil }
func (m *mockLogger) SetTUIWriter(_ io.Writer)                     {}
func (m *mockLogger) ResetToStdout()                               {}
func (m *mockLogger) GetLogs(_ logger.LogFilter) []logger.LogEntry { return nil }
func (m *mockLogger) GetLogStats() map[string]interface{}          { return nil }
func (m *mockLogger) IsTUIMode() bool                              { return false }
func (m *mockLogger) SetTUIMode(_ bool)                            {}

var _ logger.LoggerService = (*mockLogger)(nil)

// newTestHookManager creates a HookManager for testing with initialized registries.
func newTestHookManager() *hookManagerImpl {
	log := &mockLogger{}
	hm := &hookManagerImpl{
		log:        log,
		registries: make(map[HookPoint]*hookRegistry),
		names:      make(map[string]struct{}),
	}

	// Initialize registries for all known hook points
	for _, point := range []HookPoint{
		BeforeSessionStart,
		AfterSessionEnd,
		BeforeAgentSpawn,
		AfterAgentSpawn,
		BeforeAgentRemove,
		AfterAgentRemove,
		BeforeToolExecution,
		AfterToolExecution,
		OnToolError,
		BeforeFileRead,
		AfterFileRead,
		BeforeFileWrite,
		AfterFileWrite,
		BeforeFileDelete,
		AfterFileDelete,
		BeforeFileModify,
		AfterFileModify,
		BeforeLLMRequest,
		AfterLLMResponse,
		OnLLMError,
	} {
		hm.registries[point] = &hookRegistry{}
	}

	// Initialize typed registries
	hm.toolRegistry = NewTypedRegistry[ToolPayload]()
	hm.llmRegistry = NewTypedRegistry[LLMPayload]()
	hm.fileRegistry = NewTypedRegistry[FilePayload]()
	hm.sessionRegistry = NewTypedRegistry[SessionPayload]()
	hm.agentRegistry = NewTypedRegistry[AgentPayload]()
	hm.skillRegistry = NewTypedRegistry[SkillPayload]()

	return hm
}

// TestHookManager_RegisterHook tests hook registration
func TestHookManager_RegisterHook(t *testing.T) {
	t.Run("successfully registers a hook", func(t *testing.T) {
		hm := newTestHookManager()

		fn := func(_ context.Context, _ *HookContext, next func() error) error {
			return next()
		}
		meta := HookMetadata{
			Name:       "test-hook",
			Point:      BeforeSessionStart,
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(fn, meta)
		require.NoError(t, err)
	})

	t.Run("rejects nil hook function", func(t *testing.T) {
		hm := newTestHookManager()

		meta := HookMetadata{
			Name:       "test-hook",
			Point:      BeforeSessionStart,
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(nil, meta)
		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects empty hook name", func(t *testing.T) {
		hm := newTestHookManager()

		fn := func(_ context.Context, _ *HookContext, next func() error) error {
			return next()
		}
		meta := HookMetadata{
			Name:       "",
			Point:      BeforeSessionStart,
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(fn, meta)
		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects unknown hook point", func(t *testing.T) {
		hm := newTestHookManager()

		fn := func(_ context.Context, _ *HookContext, next func() error) error {
			return next()
		}
		meta := HookMetadata{
			Name:       "test-hook",
			Point:      HookPoint("UnknownPoint"),
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(fn, meta)
		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects duplicate hook name", func(t *testing.T) {
		hm := newTestHookManager()

		fn := func(_ context.Context, _ *HookContext, next func() error) error {
			return next()
		}
		meta := HookMetadata{
			Name:       "test-hook",
			Point:      BeforeSessionStart,
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(fn, meta)
		require.NoError(t, err)

		err = hm.RegisterHook(fn, meta)
		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeConflict))
	})

	t.Run("rejects duplicate hook name across different hook points", func(t *testing.T) {
		hm := newTestHookManager()

		fn := func(_ context.Context, _ *HookContext, next func() error) error {
			return next()
		}

		// Register hook for BeforeSessionStart
		meta1 := HookMetadata{
			Name:       "global-hook",
			Point:      BeforeSessionStart,
			Priority:   0,
			FatalError: false,
		}
		err := hm.RegisterHook(fn, meta1)
		require.NoError(t, err)

		// Try to register hook with same name for AfterSessionEnd - should fail
		meta2 := HookMetadata{
			Name:       "global-hook",
			Point:      AfterSessionEnd,
			Priority:   0,
			FatalError: false,
		}
		err = hm.RegisterHook(fn, meta2)
		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeConflict))
		assert.Contains(t, err.Error(), "already registered globally")
	})
}

// TestHookManager_UnregisterHook tests hook unregistration
func TestHookManager_UnregisterHook(t *testing.T) {
	t.Run("successfully unregisters a hook", func(t *testing.T) {
		hm := newTestHookManager()

		fn := func(_ context.Context, _ *HookContext, next func() error) error {
			return next()
		}
		meta := HookMetadata{
			Name:       "test-hook",
			Point:      BeforeSessionStart,
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(fn, meta)
		require.NoError(t, err)

		unregistered := hm.UnregisterHook("test-hook")
		assert.True(t, unregistered)

		// Should not be able to unregister again
		unregistered = hm.UnregisterHook("test-hook")
		assert.False(t, unregistered)
	})

	t.Run("returns false for non-existent hook", func(t *testing.T) {
		hm := newTestHookManager()

		unregistered := hm.UnregisterHook("non-existent")
		assert.False(t, unregistered)
	})
}

// TestHookManager_TriggerHooks_PriorityOrdering tests priority-based execution
func TestHookManager_TriggerHooks_PriorityOrdering(t *testing.T) {
	t.Run("executes hooks in priority order (lowest first)", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		fnP0 := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "priority-0")
			return next()
		}
		fnP5 := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "priority-5")
			return next()
		}
		fnP10 := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "priority-10")
			return next()
		}

		// Register in random order to test sorting
		require.NoError(t, hm.RegisterHook(fnP10, HookMetadata{Name: "h_p10", Point: BeforeSessionStart, Priority: 10, FatalError: false}))
		require.NoError(t, hm.RegisterHook(fnP0, HookMetadata{Name: "h_p0", Point: BeforeSessionStart, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(fnP5, HookMetadata{Name: "h_p5", Point: BeforeSessionStart, Priority: 5, FatalError: false}))

		result := hm.TriggerHooks(context.Background(), BeforeSessionStart, &HookContext{})

		assert.False(t, result.Stopped)
		assert.NoError(t, result.Error)
		// Hooks should execute in priority order: 0, 5, 10
		assert.Equal(t, []string{"priority-0", "priority-5", "priority-10"}, executed)
	})
}

// TestHookManager_TriggerHooks_ErrorHandling tests error handling
func TestHookManager_TriggerHooks_ErrorHandling(t *testing.T) {
	t.Run("fatal error stops execution", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		fn1 := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "hook1")
			return next()
		}
		fn2 := func(_ context.Context, _ *HookContext, _ func() error) error {
			executed = append(executed, "hook2")
			return errors.New("fatal error")
		}
		fn3 := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "hook3")
			return next()
		}

		require.NoError(t, hm.RegisterHook(fn1, HookMetadata{Name: "h1", Point: BeforeSessionStart, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(fn2, HookMetadata{Name: "h2", Point: BeforeSessionStart, Priority: 1, FatalError: true}))
		require.NoError(t, hm.RegisterHook(fn3, HookMetadata{Name: "h3", Point: BeforeSessionStart, Priority: 2, FatalError: false}))

		result := hm.TriggerHooks(context.Background(), BeforeSessionStart, &HookContext{})

		assert.True(t, result.Stopped)
		assert.Error(t, result.Error)
		assert.Equal(t, []string{"hook1", "hook2"}, executed)
	})

	t.Run("non-fatal error continues execution", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		fn1 := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "hook1")
			return next()
		}
		fn2 := func(_ context.Context, _ *HookContext, _ func() error) error {
			executed = append(executed, "hook2")
			return errors.New("non-fatal error")
		}
		fn3 := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "hook3")
			return next()
		}

		require.NoError(t, hm.RegisterHook(fn1, HookMetadata{Name: "h1", Point: BeforeSessionStart, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(fn2, HookMetadata{Name: "h2", Point: BeforeSessionStart, Priority: 1, FatalError: false}))
		require.NoError(t, hm.RegisterHook(fn3, HookMetadata{Name: "h3", Point: BeforeSessionStart, Priority: 2, FatalError: false}))

		result := hm.TriggerHooks(context.Background(), BeforeSessionStart, &HookContext{})

		assert.False(t, result.Stopped)
		assert.NoError(t, result.Error)
		assert.Equal(t, []string{"hook1", "hook2", "hook3"}, executed)
	})
}

// TestHookManager_WithSessionHooks tests session hook wrapping
func TestHookManager_WithSessionHooks(t *testing.T) {
	t.Run("successfully wraps work with session hooks", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, _ *TypedHookContext[SessionPayload], next func() error) error {
			executed = append(executed, "before")
			return next()
		}
		afterHook := func(_ context.Context, _ *TypedHookContext[SessionPayload], next func() error) error {
			executed = append(executed, "after")
			return next()
		}

		require.NoError(t, hm.RegisterSessionHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeSessionStart, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterSessionHook(afterHook, TypedHookMetadata{Name: "after", Point: AfterSessionEnd, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		workExecuted := false
		err := hm.WithSessionHooks(context.Background(), sessionID, func() error {
			executed = append(executed, "work")
			workExecuted = true
			return nil
		})

		require.NoError(t, err)
		assert.True(t, workExecuted)
		assert.Equal(t, []string{"before", "work", "after"}, executed)
	})

	t.Run("after hook runs even when work fails", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, _ *TypedHookContext[SessionPayload], next func() error) error {
			executed = append(executed, "before")
			return next()
		}
		afterHook := func(_ context.Context, _ *TypedHookContext[SessionPayload], next func() error) error {
			executed = append(executed, "after")
			return next()
		}

		require.NoError(t, hm.RegisterSessionHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeSessionStart, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterSessionHook(afterHook, TypedHookMetadata{Name: "after", Point: AfterSessionEnd, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		workErr := errors.New("work failed")
		err := hm.WithSessionHooks(context.Background(), sessionID, func() error {
			executed = append(executed, "work")
			return workErr
		})

		require.Error(t, err)
		assert.Equal(t, workErr, err, "work error should be returned")
		assert.Equal(t, []string{"before", "work", "after"}, executed, "after hook should run even when work fails")
	})

	t.Run("rejects nil session ID", func(t *testing.T) {
		hm := newTestHookManager()

		err := hm.WithSessionHooks(context.Background(), uuid.Nil, func() error {
			return nil
		})

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})
}

// TestHookManager_WithAgentHooks tests agent hook wrapping
func TestHookManager_WithAgentHooks(t *testing.T) {
	t.Run("successfully triggers agent hooks", func(t *testing.T) {
		hm := newTestHookManager()

		executed := false
		agentHook := func(_ context.Context, _ *TypedHookContext[AgentPayload], next func() error) error {
			executed = true
			return next()
		}

		require.NoError(t, hm.RegisterAgentHook(agentHook, TypedHookMetadata{Name: "agent-hook", Point: BeforeAgentSpawn, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		err := hm.WithAgentHooks(context.Background(), sessionID, agentID, BeforeAgentSpawn, func() error {
			return nil
		})

		require.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("rejects nil agent ID", func(t *testing.T) {
		hm := newTestHookManager()

		err := hm.WithAgentHooks(context.Background(), uuid.New(), uuid.Nil, BeforeAgentSpawn, nil)

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects invalid agent hook point", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()
		err := hm.WithAgentHooks(context.Background(), sessionID, agentID, BeforeSessionStart, nil)

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})
}

// TestHookContext_Clone tests HookContext cloning
func TestHookContext_Clone(t *testing.T) {
	t.Run("creates a deep copy of HookContext maps", func(t *testing.T) {
		original := &HookContext{
			SessionID: uuid.New(),
			AgentID:   uuid.New(),
			ToolName:  "test-tool",
			Data:      map[string]any{"key1": "value1", "key2": 42},
		}

		cloned := original.Clone()

		assert.Equal(t, original.SessionID, cloned.SessionID)
		assert.Equal(t, original.AgentID, cloned.AgentID)
		assert.Equal(t, original.ToolName, cloned.ToolName)
		assert.Equal(t, original.Data, cloned.Data)

		// Modify cloned data and verify it doesn't affect original
		cloned.Data["key1"] = testModifiedContent
		assert.Equal(t, "value1", original.Data["key1"], "Modifying a value in the cloned map should not affect the original map")
		assert.Equal(t, testModifiedContent, cloned.Data["key1"])

		// Verify shallow copy of reference types inside the map
		original.Data["ref"] = []int{10}
		clonedWithRef := original.Clone()
		// Modify the content of the slice in the clone
		clonedWithRef.Data["ref"].([]int)[0] = 20
		// Since it's a shallow copy, modifying the slice content affects the original
		assert.Equal(t, 20, original.Data["ref"].([]int)[0], "Modifying content of a reference type in clone's map should affect original (shallow copy)")
	})

	t.Run("handles nil HookContext", func(t *testing.T) {
		cloned := (*HookContext)(nil).Clone()
		assert.NotNil(t, cloned)
		assert.NotNil(t, cloned.Data)
	})
}
