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

// TestHookManager_WithFileReadHooks tests file read hooks with content return
func TestHookManager_WithFileReadHooks(t *testing.T) {
	t.Run("successfully executes file read with before and after hooks", func(t *testing.T) {
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

		require.NoError(t, hm.RegisterFileHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeFileRead, Priority: 0, FatalError: false}))
		require.NoError(t, hm.RegisterFileHook(afterHook, TypedHookMetadata{Name: "after", Point: AfterFileRead, Priority: 0, FatalError: false}))

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

		afterHook := func(_ context.Context, hc *TypedHookContext[FilePayload], next func() error) error {
			// Modify file content
			hc.Payload.Content = testModifiedContent
			return next()
		}

		require.NoError(t, hm.RegisterFileHook(afterHook, TypedHookMetadata{Name: "after", Point: AfterFileRead, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		filePath := testFilePath

		content, err := hm.WithFileReadHooks(context.Background(), sessionID, agentID, filePath, func() (string, error) {
			return testOriginalContent, nil
		})

		require.NoError(t, err)
		assert.Equal(t, testModifiedContent, content, "after hook should modify the returned content")
		assert.NotEqual(t, testOriginalContent, content, "content should be different from original")
	})

	t.Run("before hook can block file read", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, _ *TypedHookContext[FilePayload], _ func() error) error {
			// Don't call next() to block execution
			return nil
		}

		require.NoError(t, hm.RegisterFileHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeFileRead, Priority: 0, FatalError: false}))

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

		afterHook := func(_ context.Context, _ *TypedHookContext[FilePayload], _ func() error) error {
			return errors.New("fatal after error")
		}

		require.NoError(t, hm.RegisterFileHook(afterHook, TypedHookMetadata{Name: "after", Point: AfterFileRead, Priority: 0, FatalError: true}))

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
		beforeHook := func(_ context.Context, _ *TypedHookContext[FilePayload], next func() error) error {
			executed = append(executed, "before")
			// Call next() then return non-fatal error
			_ = next()
			executed = append(executed, "before-error")
			return errors.New("non-fatal error")
		}

		require.NoError(t, hm.RegisterFileHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeFileRead, Priority: 0, FatalError: false}))

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

		afterHook := func(_ context.Context, hc *TypedHookContext[FilePayload], next func() error) error {
			// Clear content (e.g., redact sensitive file)
			hc.Payload.Content = ""
			return next()
		}

		require.NoError(t, hm.RegisterFileHook(afterHook, TypedHookMetadata{Name: "redact", Point: AfterFileRead, Priority: 0, FatalError: false}))

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
