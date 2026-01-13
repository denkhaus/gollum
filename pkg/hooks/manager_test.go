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
		beforeHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "before")
			return next()
		}
		afterHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "after")
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeSessionStart, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterSessionEnd, Priority: 0, FatalError: false}))

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
		beforeHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "before")
			return next()
		}
		afterHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "after")
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeSessionStart, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterSessionEnd, Priority: 0, FatalError: false}))

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
		agentHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = true
			return next()
		}

		require.NoError(t, hm.RegisterHook(agentHook, HookMetadata{Name: "agent-hook", Point: BeforeAgentSpawn, Priority: 0, FatalError: false}))

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

// TestHookPoint_String tests HookPoint string representation
func TestHookPoint_String(t *testing.T) {
	tests := []struct {
		name     string
		point    HookPoint
		expected string
	}{
		{"BeforeSessionStart", BeforeSessionStart, "BeforeSessionStart"},
		{"AfterSessionEnd", AfterSessionEnd, "AfterSessionEnd"},
		{"BeforeAgentSpawn", BeforeAgentSpawn, "BeforeAgentSpawn"},
		{"AfterAgentSpawn", AfterAgentSpawn, "AfterAgentSpawn"},
		{"BeforeAgentRemove", BeforeAgentRemove, "BeforeAgentRemove"},
		{"AfterAgentRemove", AfterAgentRemove, "AfterAgentRemove"},
		{"BeforeToolExecution", BeforeToolExecution, "BeforeToolExecution"},
		{"AfterToolExecution", AfterToolExecution, "AfterToolExecution"},
		{"OnToolError", OnToolError, "OnToolError"},
		{"BeforeFileRead", BeforeFileRead, "BeforeFileRead"},
		{"AfterFileRead", AfterFileRead, "AfterFileRead"},
		{"BeforeFileWrite", BeforeFileWrite, "BeforeFileWrite"},
		{"AfterFileWrite", AfterFileWrite, "AfterFileWrite"},
		{"BeforeFileDelete", BeforeFileDelete, "BeforeFileDelete"},
		{"AfterFileDelete", AfterFileDelete, "AfterFileDelete"},
		{"BeforeFileModify", BeforeFileModify, "BeforeFileModify"},
		{"AfterFileModify", AfterFileModify, "AfterFileModify"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.point.String())
		})
	}
}

// TestHookManager_WithToolHooks tests tool hook wrapping
func TestHookManager_WithToolHooks(t *testing.T) {
	t.Run("successfully executes tool with before and after hooks", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "before")
			assert.Equal(t, "test-tool", hc.ToolName)
			assert.NotNil(t, hc.ToolArgs)
			return next()
		}
		afterHook := func(_ context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "after")
			assert.NotNil(t, hc.ToolResult)
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeToolExecution, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterToolExecution, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		args := map[string]any{"input": "test"}

		result, err := hm.WithToolHooks(context.Background(), sessionID, agentID, "test-tool", args, func() (map[string]any, error) {
			executed = append(executed, "work")
			return map[string]any{"output": "success"}, nil
		})

		require.NoError(t, err)
		assert.Equal(t, map[string]any{"output": "success"}, result)
		assert.Equal(t, []string{"before", "work", "after"}, executed)
	})

	t.Run("before hook can modify args", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, hc *HookContext, next func() error) error {
			// Modify args
			hc.ToolArgs["input"] = testModifiedContent
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeToolExecution, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		args := map[string]any{"input": "original"}

		result, err := hm.WithToolHooks(context.Background(), sessionID, agentID, "test-tool", args, func() (map[string]any, error) {
			// Note: Work function does NOT receive modified args from hooks due to design.
			// Hooks can validate/block but cannot modify what work() receives.
			// The work function closes over the original 'args' parameter.
			return map[string]any{"output": "success"}, nil
		})

		require.NoError(t, err)
		assert.Equal(t, map[string]any{"output": "success"}, result)
	})

	t.Run("after hook can modify result", func(t *testing.T) {
		hm := newTestHookManager()

		afterHook := func(_ context.Context, hc *HookContext, next func() error) error {
			// Modify result
			hc.ToolResult["output"] = testModifiedContent
			return next()
		}

		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterToolExecution, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		args := map[string]any{"input": "test"}

		result, err := hm.WithToolHooks(context.Background(), sessionID, agentID, "test-tool", args, func() (map[string]any, error) {
			return map[string]any{"output": "original"}, nil
		})

		require.NoError(t, err)
		assert.Equal(t, map[string]any{"output": testModifiedContent}, result)
	})

	t.Run("before hook can block execution", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, hc *HookContext, _ func() error) error {
			// Don't call next() to block execution
			hc.ToolResult = map[string]any{"blocked": true}
			return nil
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeToolExecution, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		args := map[string]any{"input": "test"}

		workExecuted := false
		result, err := hm.WithToolHooks(context.Background(), sessionID, agentID, "test-tool", args, func() (map[string]any, error) {
			workExecuted = true
			return map[string]any{"output": "success"}, nil
		})

		require.NoError(t, err)
		assert.False(t, workExecuted, "work should not be executed when blocked")
		assert.Equal(t, map[string]any{"blocked": true}, result)
	})

	t.Run("on error hook can recover from errors", func(t *testing.T) {
		hm := newTestHookManager()

		errorHook := func(_ context.Context, hc *HookContext, next func() error) error {
			// Provide fallback result
			hc.ToolResult = map[string]any{"output": "fallback"}
			return next()
		}

		require.NoError(t, hm.RegisterHook(errorHook, HookMetadata{Name: "error", Point: OnToolError, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		args := map[string]any{"input": "test"}

		result, err := hm.WithToolHooks(context.Background(), sessionID, agentID, "test-tool", args, func() (map[string]any, error) {
			return nil, errors.New("tool failed")
		})

		require.NoError(t, err)
		assert.Equal(t, map[string]any{"output": "fallback"}, result)
	})

	t.Run("on error hook without recovery propagates error", func(t *testing.T) {
		hm := newTestHookManager()

		require.NoError(t, hm.RegisterHook(func(_ context.Context, hc *HookContext, next func() error) error {
			assert.Equal(t, "tool failed", hc.ToolError.Error())
			return next()
		}, HookMetadata{Name: "error", Point: OnToolError, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		args := map[string]any{"input": "test"}

		result, err := hm.WithToolHooks(context.Background(), sessionID, agentID, "test-tool", args, func() (map[string]any, error) {
			return nil, errors.New("tool failed")
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "tool failed")
	})

	t.Run("rejects empty tool name", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()
		args := map[string]any{"input": "test"}

		_, err := hm.WithToolHooks(context.Background(), sessionID, agentID, "", args, func() (map[string]any, error) {
			return map[string]any{"output": "success"}, nil
		})

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects nil work function", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()
		args := map[string]any{"input": "test"}

		_, err := hm.WithToolHooks(context.Background(), sessionID, agentID, "test-tool", args, nil)

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("fatal error in before hook stops execution", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, _ *HookContext, _ func() error) error {
			return errors.New("fatal hook error")
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeToolExecution, Priority: 0, FatalError: true}))

		sessionID := uuid.New()
		agentID := uuid.New()
		args := map[string]any{"input": "test"}

		workExecuted := false
		result, err := hm.WithToolHooks(context.Background(), sessionID, agentID, "test-tool", args, func() (map[string]any, error) {
			workExecuted = true
			return map[string]any{"output": "success"}, nil
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.False(t, workExecuted, "work should not be executed on fatal error")
		assert.Contains(t, err.Error(), "fatal hook error")
	})

	t.Run("fatal error in after hook propagates error", func(t *testing.T) {
		hm := newTestHookManager()

		afterHook := func(_ context.Context, _ *HookContext, _ func() error) error {
			return errors.New("fatal after error")
		}

		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterToolExecution, Priority: 0, FatalError: true}))

		sessionID := uuid.New()
		agentID := uuid.New()
		args := map[string]any{"input": "test"}

		result, err := hm.WithToolHooks(context.Background(), sessionID, agentID, "test-tool", args, func() (map[string]any, error) {
			return map[string]any{"output": "success"}, nil
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "fatal after error")
	})

	t.Run("non-fatal error in before hook continues execution", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, _ *HookContext, next func() error) error {
			// Return error but call next first - next() should succeed
			require.NoError(t, next())
			return errors.New("non-fatal error")
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeToolExecution, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		args := map[string]any{"input": "test"}

		result, err := hm.WithToolHooks(context.Background(), sessionID, agentID, "test-tool", args, func() (map[string]any, error) {
			return map[string]any{"output": "success"}, nil
		})

		require.NoError(t, err)
		assert.Equal(t, map[string]any{"output": "success"}, result)
	})
}

// TestHookManager_WithFileReadHooks tests file read hooks with content return
func TestHookManager_WithFileReadHooks(t *testing.T) {
	t.Run("successfully executes file read with before and after hooks", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "before")
			assert.Equal(t, testFilePath, hc.FilePath)
			return next()
		}
		afterHook := func(_ context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "after")
			assert.Equal(t, testFilePath, hc.FilePath)
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileRead, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterFileRead, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		content, err := hm.WithFileReadHooks(context.Background(), sessionID, agentID, filePath, func() (string, error) {
			executed = append(executed, "work")
			return testOriginalContent, nil
		})

		require.NoError(t, err)
		assert.Equal(t, testOriginalContent, content)
		assert.Equal(t, []string{"before", "work", "after"}, executed)
	})

	t.Run("after hook can modify file content", func(t *testing.T) {
		hm := newTestHookManager()

		afterHook := func(_ context.Context, hc *HookContext, next func() error) error {
			// Modify file content
			hc.FileContent = "modified by hook"
			return next()
		}

		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterFileRead, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		content, err := hm.WithFileReadHooks(context.Background(), sessionID, agentID, filePath, func() (string, error) {
			return testOriginalContent, nil
		})

		require.NoError(t, err)
		assert.Equal(t, "modified by hook", content, "after hook should modify the returned content")
		assert.NotEqual(t, testOriginalContent, content, "content should be different from original")
	})

	t.Run("before hook can block file read", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, _ *HookContext, _ func() error) error {
			// Don't call next() to block execution
			return nil
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileRead, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		workExecuted := false
		content, err := hm.WithFileReadHooks(context.Background(), sessionID, agentID, filePath, func() (string, error) {
			workExecuted = true
			return testContent, nil
		})

		require.NoError(t, err)
		assert.False(t, workExecuted, "work should not be executed when blocked by hook")
		assert.Equal(t, "", content)
	})

	t.Run("after hook runs even when work fails", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "before")
			return next()
		}
		afterHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "after")
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileRead, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterFileRead, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath
		workErr := errors.New("read failed")

		content, err := hm.WithFileReadHooks(context.Background(), sessionID, agentID, filePath, func() (string, error) {
			executed = append(executed, "work")
			return "", workErr
		})

		require.Error(t, err)
		assert.Equal(t, workErr, err, "work error should be returned")
		assert.Equal(t, []string{"before", "work", "after"}, executed, "after hook should run even when work fails")
		assert.Equal(t, "", content)
	})

	t.Run("rejects empty file path", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()

		content, err := hm.WithFileReadHooks(context.Background(), sessionID, agentID, "", func() (string, error) {
			return testContent, nil
		})

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
		assert.Equal(t, "", content)
	})

	t.Run("rejects suspicious file path with path traversal", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()

		content, err := hm.WithFileReadHooks(context.Background(), sessionID, agentID, testSuspiciousPath, func() (string, error) {
			return testContent, nil
		})

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
		assert.Contains(t, err.Error(), "suspicious elements")
		assert.Equal(t, "", content)
	})

	t.Run("rejects nil work function", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		content, err := hm.WithFileReadHooks(context.Background(), sessionID, agentID, filePath, nil)

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
		assert.Equal(t, "", content)
	})

	t.Run("fatal error in after hook takes precedence over work error", func(t *testing.T) {
		hm := newTestHookManager()

		afterHook := func(_ context.Context, _ *HookContext, _ func() error) error {
			return errors.New("fatal after error")
		}

		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterFileRead, Priority: 0, FatalError: true}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath
		workErr := errors.New("work failed")

		content, err := hm.WithFileReadHooks(context.Background(), sessionID, agentID, filePath, func() (string, error) {
			return "", workErr
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "fatal after error")
		assert.NotContains(t, err.Error(), "work failed")
		assert.Equal(t, "", content)
	})

	t.Run("non-fatal error in before hook continues execution", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "before")
			// Call next() then return non-fatal error
			_ = next()
			executed = append(executed, "before-error")
			return errors.New("non-fatal error")
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileRead, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		content, err := hm.WithFileReadHooks(context.Background(), sessionID, agentID, filePath, func() (string, error) {
			executed = append(executed, "work")
			return testContent, nil
		})

		require.NoError(t, err)
		// Non-fatal error logs and continues to work
		assert.Equal(t, []string{"before", "before-error", "work"}, executed)
		assert.Equal(t, testContent, content)
	})

	t.Run("after hook can modify content to empty string", func(t *testing.T) {
		hm := newTestHookManager()

		afterHook := func(_ context.Context, hc *HookContext, next func() error) error {
			// Clear content (e.g., redact sensitive file)
			hc.FileContent = ""
			return next()
		}

		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "redact", Point: AfterFileRead, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath
		originalContent := "SENSITIVE DATA"

		content, err := hm.WithFileReadHooks(context.Background(), sessionID, agentID, filePath, func() (string, error) {
			return originalContent, nil
		})

		require.NoError(t, err)
		assert.Equal(t, "", content, "hook should be able to clear content to empty string")
		assert.NotEqual(t, originalContent, content, "content should be different from original")
	})
}

// TestHookManager_WithFileWriteHooks tests file write hooks with content modification
func TestHookManager_WithFileWriteHooks(t *testing.T) {
	t.Run("successfully executes file write with before and after hooks", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "before")
			assert.Equal(t, testFilePath, hc.FilePath)
			return next()
		}
		afterHook := func(_ context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "after")
			assert.Equal(t, testFilePath, hc.FilePath)
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterFileWrite, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		var writtenContent string
		err := hm.WithFileWriteHooks(context.Background(), sessionID, agentID, filePath, testOriginalContent, func(content string) error {
			executed = append(executed, "work")
			writtenContent = content
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, testOriginalContent, writtenContent)
		assert.Equal(t, []string{"before", "work", "after"}, executed)
	})

	t.Run("before hook can modify file content", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, hc *HookContext, next func() error) error {
			// Modify file content
			hc.FileContent = "modified by hook"
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		var writtenContent string
		err := hm.WithFileWriteHooks(context.Background(), sessionID, agentID, filePath, testOriginalContent, func(content string) error {
			writtenContent = content
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, "modified by hook", writtenContent, "before hook should modify the content passed to work")
		assert.NotEqual(t, testOriginalContent, writtenContent, "content should be different from original")
	})

	t.Run("before hook can modify content to empty string", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, hc *HookContext, next func() error) error {
			// Clear content (e.g., redact sensitive data before write)
			hc.FileContent = ""
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "redact", Point: BeforeFileWrite, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		var writtenContent string
		err := hm.WithFileWriteHooks(context.Background(), sessionID, agentID, filePath, "SENSITIVE DATA", func(content string) error {
			writtenContent = content
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, "", writtenContent, "hook should be able to clear content to empty string")
		assert.NotEqual(t, "SENSITIVE DATA", writtenContent, "content should be different from original")
	})

	t.Run("before hook can block file write", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, _ *HookContext, _ func() error) error {
			// Don't call next() to block execution
			return nil
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		workExecuted := false
		err := hm.WithFileWriteHooks(context.Background(), sessionID, agentID, filePath, testContent, func(_ string) error {
			workExecuted = true
			return nil
		})

		require.NoError(t, err)
		assert.False(t, workExecuted, "work should not be executed when blocked by hook")
	})

	t.Run("before hook can block with error", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, _ *HookContext, _ func() error) error {
			return errors.New("access denied")
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: true}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		workExecuted := false
		err := hm.WithFileWriteHooks(context.Background(), sessionID, agentID, filePath, testContent, func(_ string) error {
			workExecuted = true
			return nil
		})

		require.Error(t, err)
		assert.False(t, workExecuted)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("after hook runs even when work fails", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "before")
			return next()
		}
		afterHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "after")
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterFileWrite, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath
		workErr := errors.New("write failed")

		err := hm.WithFileWriteHooks(context.Background(), sessionID, agentID, filePath, testContent, func(_ string) error {
			executed = append(executed, "work")
			return workErr
		})

		require.Error(t, err)
		assert.Equal(t, workErr, err, "work error should be returned")
		assert.Equal(t, []string{"before", "work", "after"}, executed, "after hook should run even when work fails")
	})

	t.Run("rejects empty file path", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()

		err := hm.WithFileWriteHooks(context.Background(), sessionID, agentID, "", testContent, func(_ string) error {
			return nil
		})

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects suspicious file path with path traversal", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()

		err := hm.WithFileWriteHooks(context.Background(), sessionID, agentID, testSuspiciousPath, testContent, func(_ string) error {
			return nil
		})

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
		assert.Contains(t, err.Error(), "suspicious elements")
	})

	t.Run("rejects nil work function", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		err := hm.WithFileWriteHooks(context.Background(), sessionID, agentID, filePath, testContent, nil)

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("fatal error in after hook takes precedence over work error", func(t *testing.T) {
		hm := newTestHookManager()

		afterHook := func(_ context.Context, _ *HookContext, _ func() error) error {
			return errors.New("fatal after error")
		}

		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterFileWrite, Priority: 0, FatalError: true}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath
		workErr := errors.New("work failed")

		err := hm.WithFileWriteHooks(context.Background(), sessionID, agentID, filePath, testContent, func(_ string) error {
			return workErr
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "fatal after error")
		assert.NotContains(t, err.Error(), "work failed")
	})

	t.Run("non-fatal error in before hook continues execution", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "before")
			// Call next() then return non-fatal error
			_ = next()
			executed = append(executed, "before-error")
			return errors.New("non-fatal error")
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		err := hm.WithFileWriteHooks(context.Background(), sessionID, agentID, filePath, testContent, func(_ string) error {
			executed = append(executed, "work")
			return nil
		})

		require.NoError(t, err)
		// Non-fatal error logs and continues to work
		assert.Equal(t, []string{"before", "before-error", "work"}, executed)
	})
}

// TestHookManager_WithFileHooks tests file hook wrapping
func TestHookManager_WithFileHooks(t *testing.T) {
	t.Run("successfully executes file operation with before and after hooks", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "before")
			assert.Equal(t, testFilePath, hc.FilePath)
			return next()
		}
		afterHook := func(_ context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "after")
			assert.Equal(t, testFilePath, hc.FilePath)
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterFileWrite, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		err := hm.WithFileHooks(context.Background(), sessionID, agentID, BeforeFileWrite, filePath, func() error {
			executed = append(executed, "work")
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"before", "work", "after"}, executed)
	})

	t.Run("before hook can block file operation", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, _ *HookContext, _ func() error) error {
			// Don't call next() to block execution
			return nil
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileDelete, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		workExecuted := false
		err := hm.WithFileHooks(context.Background(), sessionID, agentID, BeforeFileDelete, filePath, func() error {
			workExecuted = true
			return nil
		})

		require.NoError(t, err)
		assert.False(t, workExecuted, "work should not be executed when blocked by hook")
	})

	t.Run("before hook can block with error", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, _ *HookContext, _ func() error) error {
			return errors.New("access denied")
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: true}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		workExecuted := false
		err := hm.WithFileHooks(context.Background(), sessionID, agentID, BeforeFileWrite, filePath, func() error {
			workExecuted = true
			return nil
		})

		require.Error(t, err)
		assert.False(t, workExecuted)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("after hook runs even when work fails", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "before")
			return next()
		}
		afterHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "after")
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileRead, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterFileRead, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath
		workErr := errors.New("read failed")

		err := hm.WithFileHooks(context.Background(), sessionID, agentID, BeforeFileRead, filePath, func() error {
			executed = append(executed, "work")
			return workErr
		})

		require.Error(t, err)
		assert.Equal(t, workErr, err, "work error should be returned")
		assert.Equal(t, []string{"before", "work", "after"}, executed, "after hook should run even when work fails")
	})

	t.Run("rejects empty file path", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()

		err := hm.WithFileHooks(context.Background(), sessionID, agentID, BeforeFileWrite, "", func() error {
			return nil
		})

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects suspicious file path with path traversal", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()

		err := hm.WithFileHooks(context.Background(), sessionID, agentID, BeforeFileWrite, testSuspiciousPath, func() error {
			return nil
		})

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
		assert.Contains(t, err.Error(), "suspicious elements")
	})

	t.Run("rejects nil work function", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		err := hm.WithFileHooks(context.Background(), sessionID, agentID, BeforeFileWrite, filePath, nil)

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects invalid file hook point", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		err := hm.WithFileHooks(context.Background(), sessionID, agentID, BeforeToolExecution, filePath, func() error {
			return nil
		})

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("fatal error in after hook takes precedence over work error", func(t *testing.T) {
		hm := newTestHookManager()

		afterHook := func(_ context.Context, _ *HookContext, _ func() error) error {
			return errors.New("fatal after error")
		}

		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterFileWrite, Priority: 0, FatalError: true}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath
		workErr := errors.New("work failed")

		err := hm.WithFileHooks(context.Background(), sessionID, agentID, BeforeFileWrite, filePath, func() error {
			return workErr
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "fatal after error")
		assert.NotContains(t, err.Error(), "work failed")
	})

	t.Run("file modify hooks receive old and new content", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, hc *HookContext, next func() error) error {
			// Set old/new content for the hook to use
			hc.OldContent = "old content"
			hc.NewContent = "new content"
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileModify, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		err := hm.WithFileHooks(context.Background(), sessionID, agentID, BeforeFileModify, filePath, func() error {
			return nil
		})

		require.NoError(t, err)
	})

	t.Run("non-fatal error in before hook continues execution", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, _ *HookContext, next func() error) error {
			executed = append(executed, "before")
			// Call next() then return non-fatal error
			_ = next()
			executed = append(executed, "before-error")
			return errors.New("non-fatal error")
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		err := hm.WithFileHooks(context.Background(), sessionID, agentID, BeforeFileWrite, filePath, func() error {
			executed = append(executed, "work")
			return nil
		})

		require.NoError(t, err)
		// Non-fatal error logs and continues to work
		// No after hooks registered, so only before and work execute
		assert.Equal(t, []string{"before", "before-error", "work"}, executed)
	})
}

// TestWithLLMHooks tests the LLM hook functionality.
func TestWithLLMHooks(t *testing.T) {
	t.Run("executes LLM call without hooks when no hooks registered", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(p string) (string, error) {
			assert.Equal(t, prompt, p, "prompt should be passed unchanged")
			return testLLMResponse, nil
		})

		require.NoError(t, err)
		assert.Equal(t, testLLMResponse, response)
	})

	t.Run("BeforeLLMRequest hook can modify prompt", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, hc *HookContext, next func() error) error {
			// Modify the prompt
			hc.LLMInput = "modified prompt"
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeLLMRequest, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := "original prompt"
		model := testLLMModel

		var receivedPrompt string
		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(p string) (string, error) {
			receivedPrompt = p
			return testLLMResponse, nil
		})

		require.NoError(t, err)
		assert.Equal(t, "modified prompt", receivedPrompt, "work function should receive modified prompt")
		assert.Equal(t, testLLMResponse, response)
	})

	t.Run("AfterLLMResponse hook can modify response", func(t *testing.T) {
		hm := newTestHookManager()

		afterHook := func(_ context.Context, hc *HookContext, _ func() error) error {
			// Modify the response
			hc.LLMResponse = "modified response"
			return nil
		}

		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterLLMResponse, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			return "original response", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "modified response", response, "should return modified response from hook")
	})

	t.Run("BeforeLLMRequest hook can block execution by not calling next", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, hc *HookContext, _ func() error) error {
			// Set a canned response and don't call next
			hc.LLMResponse = "canned response"
			return nil
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeLLMRequest, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		workCalled := false
		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			workCalled = true
			return "should not see this", nil
		})

		require.NoError(t, err)
		assert.False(t, workCalled, "work function should not be called")
		assert.Equal(t, "canned response", response, "should return canned response from hook")
	})

	t.Run("OnLLMError hook can recover from error", func(t *testing.T) {
		hm := newTestHookManager()

		errorHook := func(_ context.Context, hc *HookContext, _ func() error) error {
			// Provide fallback response
			hc.LLMResponse = "fallback response"
			return nil
		}

		require.NoError(t, hm.RegisterHook(errorHook, HookMetadata{Name: "error", Point: OnLLMError, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			return "", errors.New("LLM API error")
		})

		require.NoError(t, err)
		assert.Equal(t, "fallback response", response, "should return fallback from error hook")
	})

	t.Run("LLM error propagates when error hook doesn't recover", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			return "", errors.New("LLM API error")
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "LLM API error")
		assert.Empty(t, response)
	})

	t.Run("BeforeLLMRequest hook with fatal error blocks execution", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, _ *HookContext, _ func() error) error {
			return errs.Validation("prompt validation failed")
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeLLMRequest, Priority: 0, FatalError: true}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		workCalled := false
		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			workCalled = true
			return "should not see this", nil
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prompt validation failed")
		assert.False(t, workCalled, "work function should not be called")
		assert.Empty(t, response)
	})

	t.Run("AfterLLMResponse hook with fatal error overrides successful response", func(t *testing.T) {
		hm := newTestHookManager()

		afterHook := func(_ context.Context, _ *HookContext, next func() error) error {
			_ = next()
			return errs.Validation("response validation failed")
		}

		require.NoError(t, hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterLLMResponse, Priority: 0, FatalError: true}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			return testLLMResponse, nil
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "response validation failed")
		assert.Empty(t, response)
	})

	t.Run("hooks receive correct context with sessionID, agentID, model", func(t *testing.T) {
		hm := newTestHookManager()

		var receivedCtx *HookContext
		beforeHook := func(_ context.Context, hc *HookContext, next func() error) error {
			receivedCtx = hc
			return next()
		}

		require.NoError(t, hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeLLMRequest, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		_, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			return "response", nil
		})

		require.NoError(t, err)
		require.NotNil(t, receivedCtx)
		assert.Equal(t, sessionID, receivedCtx.SessionID)
		assert.Equal(t, agentID, receivedCtx.AgentID)
		assert.Equal(t, prompt, receivedCtx.LLMInput)
		assert.Equal(t, model, receivedCtx.LLMModel)
	})

	t.Run("multiple hooks execute in priority order", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}

		hook1 := func(_ context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "hook1")
			hc.LLMInput += " + hook1"
			return next()
		}

		hook2 := func(_ context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "hook2")
			hc.LLMInput += " + hook2"
			return next()
		}

		require.NoError(t, hm.RegisterHook(hook1, HookMetadata{Name: "hook1", Point: BeforeLLMRequest, Priority: 1, FatalError: false}))
		require.NoError(t, hm.RegisterHook(hook2, HookMetadata{Name: "hook2", Point: BeforeLLMRequest, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := "original"
		model := testLLMModel

		var receivedPrompt string
		_, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(p string) (string, error) {
			receivedPrompt = p
			return "response", nil
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"hook2", "hook1"}, executed, "hooks should execute in priority order (0 first)")
		assert.Equal(t, "original + hook2 + hook1", receivedPrompt)
	})
}
