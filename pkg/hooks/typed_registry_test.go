package hooks

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypedRegistry_Add(t *testing.T) {
	t.Run("adds hooks in priority order", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		var executionOrder []string

		// Add hooks with different priorities - using noOpTypedHook for simple cases
		require.NoError(t, r.Add(func(_ context.Context, _ *TypedHookContext[ToolPayload], next func() error) error {
			executionOrder = append(executionOrder, "priority-5")
			return next()
		}, TypedHookMetadata{Name: "hook-p5", Point: BeforeToolExecution, Priority: 5}))

		require.NoError(t, r.Add(func(_ context.Context, _ *TypedHookContext[ToolPayload], next func() error) error {
			executionOrder = append(executionOrder, "priority-1")
			return next()
		}, TypedHookMetadata{Name: "hook-p1", Point: BeforeToolExecution, Priority: 1}))

		require.NoError(t, r.Add(func(_ context.Context, _ *TypedHookContext[ToolPayload], next func() error) error {
			executionOrder = append(executionOrder, "priority-10")
			return next()
		}, TypedHookMetadata{Name: "hook-p10", Point: BeforeToolExecution, Priority: 10}))

		// Trigger to verify execution order (should be by priority: 1, 5, 10)
		hookCtx := &TypedHookContext[ToolPayload]{
			Payload: ToolPayload{Args: make(map[string]any)},
		}
		result := r.Trigger(context.Background(), BeforeToolExecution, hookCtx)
		assert.False(t, result.Stopped)

		// Verify execution order matches priority order
		require.Len(t, executionOrder, 3)
		assert.Equal(t, []string{"priority-1", "priority-5", "priority-10"}, executionOrder)
	})

	t.Run("rejects duplicate names", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		meta := TypedHookMetadata{Name: "duplicate", Point: BeforeToolExecution, Priority: 1}
		err := r.Add(noOpTypedHook[ToolPayload](), meta)
		require.NoError(t, err)

		// Try to add same name again
		err = r.Add(noOpTypedHook[ToolPayload](), meta)
		assert.Error(t, err)
		assert.Equal(t, ErrDuplicateHook, err)

		// Only one hook should be registered
		assert.Equal(t, 1, r.Count(BeforeToolExecution))
	})

	t.Run("rejects same name for different hook points", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		meta1 := TypedHookMetadata{Name: "myHook", Point: BeforeToolExecution, Priority: 1}
		meta2 := TypedHookMetadata{Name: "myHook", Point: AfterToolExecution, Priority: 1}

		err := r.Add(noOpTypedHook[ToolPayload](), meta1)
		require.NoError(t, err)

		// Same name should be rejected even for different hook point
		err = r.Add(noOpTypedHook[ToolPayload](), meta2)
		assert.Error(t, err)
		assert.Equal(t, ErrDuplicateHook, err)

		// Only first hook should be registered
		assert.Equal(t, 1, r.Count(BeforeToolExecution))
		assert.Equal(t, 0, r.Count(AfterToolExecution))
	})

	t.Run("handles empty registry", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		assert.False(t, r.HasHooks(BeforeToolExecution))
		assert.Equal(t, 0, r.Count(BeforeToolExecution))
	})
}

func TestTypedRegistry_Trigger(t *testing.T) {
	t.Run("executes hooks in order and calls all", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		executionOrder := make([]string, 0)

		require.NoError(t, r.Add(func(_ context.Context, hc *TypedHookContext[ToolPayload], next func() error) error {
			executionOrder = append(executionOrder, "first")
			hc.Payload.Args["order"] = executionOrder
			return next()
		}, TypedHookMetadata{Name: "first", Point: BeforeToolExecution, Priority: 1}))

		require.NoError(t, r.Add(func(_ context.Context, hc *TypedHookContext[ToolPayload], next func() error) error {
			executionOrder = append(executionOrder, "second")
			hc.Payload.Args["order"] = executionOrder
			return next()
		}, TypedHookMetadata{Name: "second", Point: BeforeToolExecution, Priority: 5}))

		hookCtx := &TypedHookContext[ToolPayload]{
			BaseContext: BaseContext{SessionID: uuid.New(), AgentID: uuid.New()},
			Payload:     ToolPayload{Args: make(map[string]any)},
		}
		result := r.Trigger(context.Background(), BeforeToolExecution, hookCtx)

		assert.False(t, result.Stopped)
		assert.NoError(t, result.Error)

		order := result.Payload.Args["order"].([]string)
		require.Len(t, order, 2)
		assert.Equal(t, []string{"first", "second"}, order)
	})

	t.Run("stops on fatal error", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()

		require.NoError(t, r.Add(func(_ context.Context, _ *TypedHookContext[ToolPayload], _ func() error) error {
			return errors.New("fatal error")
		}, TypedHookMetadata{Name: "fatal", Point: BeforeToolExecution, Priority: 1, FatalError: true}))

		require.NoError(t, r.Add(func(_ context.Context, hc *TypedHookContext[ToolPayload], next func() error) error {
			hc.Payload.Args["secondCalled"] = true
			return next()
		}, TypedHookMetadata{Name: "second", Point: BeforeToolExecution, Priority: 5}))

		hookCtx := &TypedHookContext[ToolPayload]{
			BaseContext: BaseContext{SessionID: uuid.New(), AgentID: uuid.New()},
			Payload:     ToolPayload{Args: make(map[string]any)},
		}
		result := r.Trigger(context.Background(), BeforeToolExecution, hookCtx)

		assert.True(t, result.Stopped)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "fatal error")

		// Second hook should not have been called
		_, called := result.Payload.Args["secondCalled"]
		assert.False(t, called)
	})

	t.Run("continues on non-fatal error", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()

		require.NoError(t, r.Add(func(_ context.Context, _ *TypedHookContext[ToolPayload], _ func() error) error {
			// Return error but continue chain
			return errors.New("non-fatal error")
		}, TypedHookMetadata{Name: "nonfatal", Point: BeforeToolExecution, Priority: 1, FatalError: false}))

		require.NoError(t, r.Add(func(_ context.Context, hc *TypedHookContext[ToolPayload], next func() error) error {
			hc.Payload.Args["secondCalled"] = true
			return next()
		}, TypedHookMetadata{Name: "second", Point: BeforeToolExecution, Priority: 5}))

		hookCtx := &TypedHookContext[ToolPayload]{
			BaseContext: BaseContext{SessionID: uuid.New(), AgentID: uuid.New()},
			Payload:     ToolPayload{Args: make(map[string]any)},
		}
		result := r.Trigger(context.Background(), BeforeToolExecution, hookCtx)

		// Chain completes despite non-fatal error (all hooks called)
		assert.False(t, result.Stopped)
		assert.NoError(t, result.Error)

		// Second hook should have been called
		_, called := result.Payload.Args["secondCalled"]
		assert.True(t, called)
	})

	t.Run("stops when hook does not call next", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()

		require.NoError(t, r.Add(func(_ context.Context, _ *TypedHookContext[ToolPayload], _ func() error) error {
			// Don't call next - this stops the chain
			return nil
		}, TypedHookMetadata{Name: "stopper", Point: BeforeToolExecution, Priority: 1}))

		require.NoError(t, r.Add(func(_ context.Context, hc *TypedHookContext[ToolPayload], next func() error) error {
			hc.Payload.Args["secondCalled"] = true
			return next()
		}, TypedHookMetadata{Name: "second", Point: BeforeToolExecution, Priority: 5}))

		hookCtx := &TypedHookContext[ToolPayload]{
			BaseContext: BaseContext{SessionID: uuid.New(), AgentID: uuid.New()},
			Payload:     ToolPayload{Args: make(map[string]any)},
		}
		result := r.Trigger(context.Background(), BeforeToolExecution, hookCtx)

		assert.True(t, result.Stopped)

		// Second hook should not have been called
		_, called := result.Payload.Args["secondCalled"]
		assert.False(t, called)
	})

	t.Run("returns modified payload", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()

		require.NoError(t, r.Add(func(_ context.Context, hc *TypedHookContext[ToolPayload], next func() error) error {
			hc.Payload.Name = "modified"
			hc.Payload.Args["added"] = "value"
			return next()
		}, TypedHookMetadata{Name: "modifier", Point: BeforeToolExecution, Priority: 1}))

		hookCtx := &TypedHookContext[ToolPayload]{
			BaseContext: BaseContext{SessionID: uuid.New(), AgentID: uuid.New()},
			Payload:     ToolPayload{Name: "original", Args: make(map[string]any)},
		}
		result := r.Trigger(context.Background(), BeforeToolExecution, hookCtx)

		assert.False(t, result.Stopped)
		assert.NoError(t, result.Error)

		// Payload should be modified
		assert.Equal(t, shared.ToolName("modified"), result.Payload.Name)
		assert.Equal(t, "value", result.Payload.Args["added"])
	})

	t.Run("handles empty registry", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()

		hookCtx := &TypedHookContext[ToolPayload]{
			BaseContext: BaseContext{SessionID: uuid.New(), AgentID: uuid.New()},
			Payload:     ToolPayload{Name: "test"},
		}
		result := r.Trigger(context.Background(), BeforeToolExecution, hookCtx)

		assert.False(t, result.Stopped)
		assert.NoError(t, result.Error)
		assert.Equal(t, shared.ToolName("test"), result.Payload.Name)
	})
}

func TestTypedRegistry_Remove(t *testing.T) {
	t.Run("removes existing hook", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		meta := TypedHookMetadata{Name: "toRemove", Point: BeforeToolExecution, Priority: 1}
		err := r.Add(noOpTypedHook[ToolPayload](), meta)
		require.NoError(t, err)
		assert.True(t, r.HasHooks(BeforeToolExecution))

		removed := r.Remove("toRemove")
		assert.True(t, removed)
		assert.False(t, r.HasHooks(BeforeToolExecution))
	})

	t.Run("returns false for non-existent hook", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		removed := r.Remove("nonexistent")
		assert.False(t, removed)
	})

	t.Run("removes only the specified hook", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		meta1 := TypedHookMetadata{Name: "hook1", Point: BeforeToolExecution, Priority: 1}
		meta2 := TypedHookMetadata{Name: "hook2", Point: BeforeToolExecution, Priority: 2}

		err := r.Add(noOpTypedHook[ToolPayload](), meta1)
		require.NoError(t, err)
		err = r.Add(noOpTypedHook[ToolPayload](), meta2)
		require.NoError(t, err)

		removed := r.Remove("hook1")
		assert.True(t, removed)
		assert.Equal(t, 1, r.Count(BeforeToolExecution))
	})

	t.Run("removes hook from correct point", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		meta1 := TypedHookMetadata{Name: "hook1", Point: BeforeToolExecution, Priority: 1}
		meta2 := TypedHookMetadata{Name: "hook2", Point: AfterToolExecution, Priority: 1}

		err := r.Add(noOpTypedHook[ToolPayload](), meta1)
		require.NoError(t, err)
		err = r.Add(noOpTypedHook[ToolPayload](), meta2)
		require.NoError(t, err)

		// Remove hook1 from BeforeToolExecution
		removed := r.Remove("hook1")
		assert.True(t, removed)
		assert.False(t, r.HasHooks(BeforeToolExecution))
		assert.True(t, r.HasHooks(AfterToolExecution))

		// Remove hook2 from AfterToolExecution
		removed = r.Remove("hook2")
		assert.True(t, removed)
		assert.False(t, r.HasHooks(AfterToolExecution))
	})
}

func TestTypedRegistry_HasHooks(t *testing.T) {
	t.Run("returns true when hooks exist", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		meta := TypedHookMetadata{Name: "test", Point: BeforeToolExecution, Priority: 1}
		err := r.Add(noOpTypedHook[ToolPayload](), meta)
		require.NoError(t, err)

		assert.True(t, r.HasHooks(BeforeToolExecution))
		assert.False(t, r.HasHooks(AfterToolExecution))
	})

	t.Run("returns false when no hooks", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		assert.False(t, r.HasHooks(BeforeToolExecution))
	})
}

func TestTypedRegistry_Count(t *testing.T) {
	t.Run("counts hooks for specific point", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		meta1 := TypedHookMetadata{Name: "hook1", Point: BeforeToolExecution, Priority: 1}
		meta2 := TypedHookMetadata{Name: "hook2", Point: BeforeToolExecution, Priority: 2}
		meta3 := TypedHookMetadata{Name: "hook3", Point: AfterToolExecution, Priority: 1}

		err := r.Add(noOpTypedHook[ToolPayload](), meta1)
		require.NoError(t, err)
		err = r.Add(noOpTypedHook[ToolPayload](), meta2)
		require.NoError(t, err)
		err = r.Add(noOpTypedHook[ToolPayload](), meta3)
		require.NoError(t, err)

		assert.Equal(t, 2, r.Count(BeforeToolExecution))
		assert.Equal(t, 1, r.Count(AfterToolExecution))
		assert.Equal(t, 0, r.Count(OnToolError))
	})
}

func TestTypedRegistry_CountAll(t *testing.T) {
	t.Run("counts all hooks", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		meta1 := TypedHookMetadata{Name: "hook1", Point: BeforeToolExecution, Priority: 1}
		meta2 := TypedHookMetadata{Name: "hook2", Point: AfterToolExecution, Priority: 1}
		meta3 := TypedHookMetadata{Name: "hook3", Point: OnToolError, Priority: 1}

		err := r.Add(noOpTypedHook[ToolPayload](), meta1)
		require.NoError(t, err)
		err = r.Add(noOpTypedHook[ToolPayload](), meta2)
		require.NoError(t, err)
		err = r.Add(noOpTypedHook[ToolPayload](), meta3)
		require.NoError(t, err)

		assert.Equal(t, 3, r.CountAll())
	})

	t.Run("returns zero for empty registry", func(t *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		assert.Equal(t, 0, r.CountAll())
	})
}

func TestTypedRegistry_Concurrency(t *testing.T) {
	t.Run("concurrent add and trigger", func(_ *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		var wg sync.WaitGroup

		// Concurrently add hooks
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				meta := TypedHookMetadata{
					Name:     "hook" + string(rune('0'+i%10)),
					Point:    BeforeToolExecution,
					Priority: i,
				}
				_ = r.Add(noOpTypedHook[ToolPayload](), meta)
			}(i)
		}

		// Concurrently trigger hooks
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx := context.Background()
				hookCtx := &TypedHookContext[ToolPayload]{
					Payload: ToolPayload{Args: make(map[string]any)},
				}
				_ = r.Trigger(ctx, BeforeToolExecution, hookCtx)
			}()
		}

		wg.Wait()
		// If we get here without race conditions, the test passes
	})

	t.Run("concurrent add and remove", func(_ *testing.T) {
		r := NewTypedRegistry[ToolPayload]()
		var wg sync.WaitGroup

		// Add hooks
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				meta := TypedHookMetadata{
					Name:     "hook" + string(rune('0'+i%10)),
					Point:    BeforeToolExecution,
					Priority: i,
				}
				_ = r.Add(noOpTypedHook[ToolPayload](), meta)
			}(i)
		}

		// Remove hooks
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_ = r.Remove("hook" + string(rune('0'+i%10)))
			}(i)
		}

		wg.Wait()
		// If we get here without race conditions, the test passes
	})
}

func TestTypedRegistry_DifferentPayloadTypes(t *testing.T) {
	t.Run("works with LLMPayload", func(t *testing.T) {
		r := NewTypedRegistry[LLMPayload]()
		meta := TypedHookMetadata{Name: "llmHook", Point: BeforeLLMRequest, Priority: 1}
		err := r.Add(noOpTypedHook[LLMPayload](), meta)
		require.NoError(t, err)

		ctx := context.Background()
		hookCtx := &TypedHookContext[LLMPayload]{
			Payload: LLMPayload{Input: "test prompt", Model: "claude-3-5-sonnet"},
		}
		result := r.Trigger(ctx, BeforeLLMRequest, hookCtx)
		assert.False(t, result.Stopped)
	})

	t.Run("works with FilePayload", func(t *testing.T) {
		r := NewTypedRegistry[FilePayload]()
		meta := TypedHookMetadata{Name: "fileHook", Point: BeforeFileWrite, Priority: 1}
		err := r.Add(noOpTypedHook[FilePayload](), meta)
		require.NoError(t, err)

		ctx := context.Background()
		hookCtx := &TypedHookContext[FilePayload]{
			Payload: FilePayload{Path: "/test/path", Content: "content"},
		}
		result := r.Trigger(ctx, BeforeFileWrite, hookCtx)
		assert.False(t, result.Stopped)
	})
}

// Helper function to create a no-op typed hook
func noOpTypedHook[T any]() TypedHookFunc[T] {
	return func(_ context.Context, _ *TypedHookContext[T], next func() error) error {
		return next()
	}
}
