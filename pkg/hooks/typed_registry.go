// Package hooks provides a generic typed registry for type-safe hook storage.
// This enables compile-time type safety for hook functions and their payloads.
package hooks

import (
	"context"
	"sort"
	"sync"
)

// typedRegistration represents a registered typed hook with its metadata.
type typedRegistration[T any] struct {
	fn   TypedHookFunc[T]
	meta TypedHookMetadata
}

// NewTypedRegistry creates a new typed registry for hooks with payload type T.
func NewTypedRegistry[T any]() *TypedRegistry[T] {
	return &TypedRegistry[T]{
		hooks: make(map[HookPoint][]*typedRegistration[T]),
		names: make(map[string]struct{}),
	}
}

// TypedRegistry is a generic registry that stores hooks by type, enabling compile-time type safety.
// It manages hooks for a specific payload type T (e.g., ToolPayload, LLMPayload, FilePayload).
//
// Key features:
//   - Thread-safe with RWMutex for concurrent access
//   - Priority-based insertion (lower priority executes first)
//   - Duplicate name detection across all hooks in the registry
//   - Middleware-style chain execution with next() pattern
//
// Example usage:
//
//	registry := &TypedRegistry[ToolPayload]{}
//	err := registry.Add(myToolHook, TypedHookMetadata{
//	    Name:     "my-hook",
//	    Point:    BeforeToolExecution,
//	    Priority: 10,
//	})
type TypedRegistry[T any] struct {
	mu    sync.RWMutex
	hooks map[HookPoint][]*typedRegistration[T]
	names map[string]struct{}
}

// Add registers a typed hook function with the given metadata.
// Returns ErrDuplicateHook if a hook with the same name already exists.
//
// Hooks are inserted in priority order (lower priority first).
// Priority 0 executes first, higher numbers execute later.
//
// Example:
//
//	err := registry.Add(myHook, TypedHookMetadata{
//	    Name:     "validation-hook",
//	    Point:    BeforeToolExecution,
//	    Priority: 10,
//	    FatalError: true,
//	})
func (r *TypedRegistry[T]) Add(fn TypedHookFunc[T], meta TypedHookMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.names == nil {
		r.names = make(map[string]struct{})
	}

	if _, exists := r.names[meta.Name]; exists {
		return ErrDuplicateHook
	}

	reg := &typedRegistration[T]{fn: fn, meta: meta}

	if r.hooks == nil {
		r.hooks = make(map[HookPoint][]*typedRegistration[T])
	}

	// Insert sorted by priority
	hooks := r.hooks[meta.Point]
	idx := sort.Search(len(hooks), func(i int) bool {
		return hooks[i].meta.Priority >= meta.Priority
	})

	r.hooks[meta.Point] = append(hooks[:idx],
		append([]*typedRegistration[T]{reg}, hooks[idx:]...)...)
	r.names[meta.Name] = struct{}{}

	return nil
}

// Trigger executes all registered hooks for a given hook point.
// Hooks are executed in priority order (lowest first) using the middleware pattern.
//
// Each hook can:
//   - Modify the TypedHookContext.Payload to pass data to subsequent hooks
//   - Call next() to continue the chain
//   - Return an error to stop execution (if FatalError=true in metadata)
//   - Not call next() to stop the chain at that point
//
// Returns a TypedHookResult containing:
//   - Stopped: true if a hook stopped execution
//   - Error: the error that caused the stop (if any)
//   - Payload: the (potentially modified) payload after all hooks have run
func (r *TypedRegistry[T]) Trigger(ctx context.Context, point HookPoint,
	hookCtx *TypedHookContext[T]) TypedHookResult[T] {

	r.mu.RLock()
	hooks := make([]*typedRegistration[T], len(r.hooks[point]))
	copy(hooks, r.hooks[point])
	r.mu.RUnlock()

	result := TypedHookResult[T]{}
	if hookCtx != nil {
		result.Payload = hookCtx.Payload
	}

	if len(hooks) == 0 {
		return result
	}

	index := -1

	var next func() error
	next = func() error {
		index++
		if index >= len(hooks) {
			return nil
		}

		reg := hooks[index]
		err := reg.fn(ctx, hookCtx, next)

		if err != nil {
			if reg.meta.FatalError {
				result.Stopped = true
				result.Error = err
				return err
			}
			// Non-fatal error: continue to next hook
			return next()
		}

		return nil
	}

	_ = next()

	// If the chain was stopped prematurely (a hook didn't call next),
	// the index will not have reached len(hooks).
	if index < len(hooks) {
		result.Stopped = true
	}

	// Capture final payload state
	if hookCtx != nil {
		result.Payload = hookCtx.Payload
	}

	return result
}

// Remove removes a hook by name from all hook points in this registry.
// Returns true if at least one hook was removed.
func (r *TypedRegistry[T]) Remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	removed := false
	for point, hooks := range r.hooks {
		for i, h := range hooks {
			if h.meta.Name == name {
				r.hooks[point] = append(hooks[:i], hooks[i+1:]...)
				delete(r.names, name)
				removed = true
				break
			}
		}
		if removed {
			break
		}
	}
	return removed
}

// Get returns all hooks for a given hook point in priority order.
// Returns a copy to avoid race conditions during iteration.
func (r *TypedRegistry[T]) Get(point HookPoint) []TypedHookFunc[T] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hooks := r.hooks[point]
	if hooks == nil {
		return nil
	}

	result := make([]TypedHookFunc[T], len(hooks))
	for i, h := range hooks {
		result[i] = h.fn
	}
	return result
}

// HasHooks returns true if there are any hooks registered for the given point.
func (r *TypedRegistry[T]) HasHooks(point HookPoint) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.hooks[point]) > 0
}

// Count returns the number of hooks registered for a given hook point.
func (r *TypedRegistry[T]) Count(point HookPoint) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.hooks[point])
}

// CountAll returns the total number of hooks across all hook points.
func (r *TypedRegistry[T]) CountAll() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, hooks := range r.hooks {
		count += len(hooks)
	}
	return count
}
