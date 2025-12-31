// Package hooks provides a middleware-style hook system for extensible
// lifecycle event handling throughout the application.
//
// Hook execution follows a middleware pattern where each hook can:
// - Inspect and modify the context
// - Decide whether to continue to the next hook
// - Return an error to stop the chain
package hooks

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// HookPoint identifies where in the application lifecycle a hook should be triggered.
type HookPoint string

const (
	// Session lifecycle hooks
	BeforeSessionStart HookPoint = "BeforeSessionStart"
	AfterSessionEnd   HookPoint = "AfterSessionEnd"

	// Agent lifecycle hooks
	BeforeAgentSpawn HookPoint = "BeforeAgentSpawn"
	AfterAgentSpawn  HookPoint = "AfterAgentSpawn"
	BeforeAgentRemove HookPoint = "BeforeAgentRemove"
	AfterAgentRemove  HookPoint = "AfterAgentRemove"
)

// String returns the string representation of the hook point.
func (h HookPoint) String() string {
	return string(h)
}

// HookContext carries contextual information for hook execution.
type HookContext struct {
	SessionID uuid.UUID // Optional: session identifier
	AgentID   uuid.UUID // Optional: agent identifier
	ToolName  string    // Optional: tool name for tool hooks
	Data      map[string]any
}

// Clone creates a shallow copy of the HookContext.
func (hc *HookContext) Clone() *HookContext {
	if hc == nil {
		return &HookContext{Data: make(map[string]any)}
	}
	cpy := &HookContext{
		SessionID: hc.SessionID,
		AgentID:   hc.AgentID,
		ToolName:  hc.ToolName,
		Data:      make(map[string]any, len(hc.Data)),
	}
	for k, v := range hc.Data {
		cpy.Data[k] = v
	}
	return cpy
}

// HookResult represents the result of a hook execution.
type HookResult struct {
	Stopped bool        // True if the hook chain was stopped
	Error   error       // Error from hook execution (if any)
	Data    map[string]any // Data returned by hooks
}

// HookFunc is the signature for hook functions.
//
// The function receives:
// - ctx: context for cancellation/control
// - hookCtx: contextual information about the event
// - next: function to call the next hook in the chain
//
// The function should:
// - Call next() to continue the chain (unless stopping intentionally)
// - Return an error to stop execution (if FatalError=true)
// - Modify hookCtx.Data to pass data to subsequent hooks
type HookFunc func(ctx context.Context, hookCtx *HookContext, next func() error) error

// HookMetadata contains configuration for a hook.
type HookMetadata struct {
	// Name uniquely identifies this hook registration
	Name string

	// Point determines when this hook is triggered
	Point HookPoint

	// Priority determines execution order (0=first, higher=later)
	Priority int

	// FatalError controls error handling:
	// - true: stop execution and return error
	// - false: log error and continue to next hook
	FatalError bool
}

// registration represents a registered hook with its metadata.
type registration struct {
	fn       HookFunc
	metadata HookMetadata
}

// hookRegistry manages hooks for a specific HookPoint.
type hookRegistry struct {
	mu     sync.RWMutex
	hooks  []*registration // Sorted by priority
	nextID int
}

// add registers a hook with the given metadata.
func (r *hookRegistry) add(fn HookFunc, meta HookMetadata) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	reg := &registration{
		fn:       fn,
		metadata: meta,
	}

	// Insert in priority order (lower priority first)
	inserted := false
	for i, h := range r.hooks {
		if meta.Priority < h.metadata.Priority {
			r.hooks = append(r.hooks[:i], append([]*registration{reg}, r.hooks[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		r.hooks = append(r.hooks, reg)
	}

	r.nextID++
	return r.nextID - 1
}

// remove removes a hook by name.
func (r *hookRegistry) remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, h := range r.hooks {
		if h.metadata.Name == name {
			r.hooks = append(r.hooks[:i], r.hooks[i+1:]...)
			return true
		}
	}
	return false
}

// get returns all hooks in priority order.
func (r *hookRegistry) get() []*registration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to avoid race conditions during iteration
	result := make([]*registration, len(r.hooks))
	copy(result, r.hooks)
	return result
}
