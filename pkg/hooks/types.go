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
	"errors"
	"sync"

	"github.com/google/uuid"
)

// ErrDuplicateHook is returned when attempting to register a hook with a name that already exists.
var ErrDuplicateHook = errors.New("hook with this name already registered")

// HookPoint identifies where in the application lifecycle a hook should be triggered.
type HookPoint string

const (
	// BeforeSessionStart is triggered before a session starts.
	BeforeSessionStart HookPoint = "BeforeSessionStart"
	// AfterSessionEnd is triggered after a session ends.
	AfterSessionEnd HookPoint = "AfterSessionEnd"

	// BeforeAgentSpawn is triggered before an agent is spawned.
	BeforeAgentSpawn HookPoint = "BeforeAgentSpawn"
	// AfterAgentSpawn is triggered after an agent is spawned.
	AfterAgentSpawn HookPoint = "AfterAgentSpawn"
	// BeforeAgentRemove is triggered before an agent is removed.
	BeforeAgentRemove HookPoint = "BeforeAgentRemove"
	// AfterAgentRemove is triggered after an agent is removed.
	AfterAgentRemove HookPoint = "AfterAgentRemove"

	// BeforeToolExecution is triggered before a tool runs.
	// Hooks can modify args or block execution by not calling next().
	BeforeToolExecution HookPoint = "BeforeToolExecution"
	// AfterToolExecution is triggered after a tool completes successfully.
	// Hooks can modify the result before returning to the caller.
	AfterToolExecution HookPoint = "AfterToolExecution"
	// OnToolError is triggered when a tool execution fails.
	// Hooks can recover from errors or return fallback responses.
	OnToolError HookPoint = "OnToolError"
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
	ToolArgs  map[string]any   // Optional: tool arguments for BeforeToolExecution
	ToolResult map[string]any  // Optional: tool result for AfterToolExecution
	ToolError error           // Optional: tool error for OnToolError
	Data      map[string]any
}

// Clone creates a deep copy of the HookContext.
func (hc *HookContext) Clone() *HookContext {
	if hc == nil {
		return &HookContext{Data: make(map[string]any)}
	}
	cpy := &HookContext{
		SessionID: hc.SessionID,
		AgentID:   hc.AgentID,
		ToolName:  hc.ToolName,
		ToolError: hc.ToolError,
		Data:      make(map[string]any, len(hc.Data)),
	}

	// Deep copy ToolArgs
	if hc.ToolArgs != nil {
		cpy.ToolArgs = make(map[string]any, len(hc.ToolArgs))
		for k, v := range hc.ToolArgs {
			cpy.ToolArgs[k] = v
		}
	}

	// Deep copy ToolResult
	if hc.ToolResult != nil {
		cpy.ToolResult = make(map[string]any, len(hc.ToolResult))
		for k, v := range hc.ToolResult {
			cpy.ToolResult[k] = v
		}
	}

	// Deep copy Data
	for k, v := range hc.Data {
		cpy.Data[k] = v
	}
	return cpy
}

// HookResult represents the result of a hook execution.
type HookResult struct {
	Stopped bool           // True if the hook chain was stopped
	Error   error          // Error from hook execution (if any)
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
	hooks  []*registration     // Sorted by priority
	names  map[string]struct{} // O(1) name lookup for duplicate detection
	nextID int
}

// add registers a hook with the given metadata.
// Returns an error if a hook with the same name already exists.
func (r *hookRegistry) add(fn HookFunc, meta HookMetadata) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize names map if needed
	if r.names == nil {
		r.names = make(map[string]struct{})
	}

	// Check for duplicate name (O(1) lookup)
	if _, exists := r.names[meta.Name]; exists {
		return 0, ErrDuplicateHook
	}

	reg := &registration{
		fn:       fn,
		metadata: meta,
	}

	// Insert in priority order (lower priority first)
	inserted := false
	for i, h := range r.hooks {
		if meta.Priority < h.metadata.Priority {
			// Efficient insertion: grow slice by one, shift elements, insert new element
			r.hooks = append(r.hooks, nil)   // Grow slice by one
			copy(r.hooks[i+1:], r.hooks[i:]) // Shift elements to the right
			r.hooks[i] = reg                 // Insert new element at position i
			inserted = true
			break
		}
	}
	if !inserted {
		r.hooks = append(r.hooks, reg)
	}

	// Add name to map
	r.names[meta.Name] = struct{}{}

	r.nextID++
	return r.nextID - 1, nil
}

// remove removes a hook by name.
func (r *hookRegistry) remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, h := range r.hooks {
		if h.metadata.Name == name {
			r.hooks = append(r.hooks[:i], r.hooks[i+1:]...)
			// Remove from names map
			delete(r.names, name)
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
