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

// TestHookManager_WithFileHooks tests file hook wrapping
func TestHookManager_WithFileHooks(t *testing.T) {
	t.Run("successfully executes file operation with before and after hooks", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}
		beforeHook := func(_ context.Context, hc *TypedHookContext[FilePayload], next func() error) error {
			executed = append(executed, "before")
			assert.Equal(t, testFilePath, hc.Payload.Path)
			return next()
		}
		afterHook := func(_ context.Context, hc *TypedHookContext[FilePayload], next func() error) error {
			executed = append(executed, "after")
			assert.Equal(t, testFilePath, hc.Payload.Path)
			return next()
		}

		require.NoError(t, hm.RegisterFileHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterFileHook(afterHook, TypedHookMetadata{Name: "after", Point: AfterFileWrite, Priority: 0, FatalError: false}))

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

		beforeHook := func(_ context.Context, _ *TypedHookContext[FilePayload], _ func() error) error {
			// Don't call next() to block execution
			return nil
		}

		require.NoError(t, hm.RegisterFileHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeFileDelete, Priority: 0, FatalError: false}))

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

		beforeHook := func(_ context.Context, _ *TypedHookContext[FilePayload], _ func() error) error {
			return errors.New("access denied")
		}

		require.NoError(t, hm.RegisterFileHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: true}))

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
		beforeHook := func(_ context.Context, _ *TypedHookContext[FilePayload], next func() error) error {
			executed = append(executed, "before")
			return next()
		}
		afterHook := func(_ context.Context, _ *TypedHookContext[FilePayload], next func() error) error {
			executed = append(executed, "after")
			return next()
		}

		require.NoError(t, hm.RegisterFileHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeFileRead, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterFileHook(afterHook, TypedHookMetadata{Name: "after", Point: AfterFileRead, Priority: 0, FatalError: false}))

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

		afterHook := func(_ context.Context, _ *TypedHookContext[FilePayload], _ func() error) error {
			return errors.New("fatal after error")
		}

		require.NoError(t, hm.RegisterFileHook(afterHook, TypedHookMetadata{Name: "after", Point: AfterFileWrite, Priority: 0, FatalError: true}))

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

		beforeHook := func(_ context.Context, hc *TypedHookContext[FilePayload], next func() error) error {
			// Set old/new content for the hook to use
			hc.Payload.OldContent = "old content"
			hc.Payload.NewContent = "new content"
			return next()
		}

		require.NoError(t, hm.RegisterFileHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeFileModify, Priority: 0, FatalError: false}))

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
		beforeHook := func(_ context.Context, _ *TypedHookContext[FilePayload], next func() error) error {
			executed = append(executed, "before")
			// Call next() then return non-fatal error
			_ = next()
			executed = append(executed, "before-error")
			return errors.New("non-fatal error")
		}

		require.NoError(t, hm.RegisterFileHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeFileWrite, Priority: 0, FatalError: false}))

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
