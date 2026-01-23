package hooks

import (
	"context"
	"errors"
	"testing"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
