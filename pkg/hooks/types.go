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

	// BeforeFileRead is triggered before a file is read.
	// Use case: Validate permissions, check file existence, implement caching.
	// Hooks can block the read operation by not calling next().
	BeforeFileRead HookPoint = "BeforeFileRead"
	// AfterFileRead is triggered after a file is successfully read.
	// Use case: Update access statistics, transform content, implement logging.
	// NOTE: Content modification via HookContext.FileContent requires use of
	// WithFileReadHooks to return modified content to caller.
	AfterFileRead HookPoint = "AfterFileRead"
	// BeforeFileWrite is triggered before a file is written.
	// Use case: Validate locks, check disk space, transform content, compute checksums.
	// Set HookContext.FileContent before work() to modify what gets written.
	BeforeFileWrite HookPoint = "BeforeFileWrite"
	// AfterFileWrite is triggered after a file is successfully written.
	// Use case: Update file state, invalidate caches, send notifications, audit logging.
	AfterFileWrite HookPoint = "AfterFileWrite"
	// BeforeFileDelete is triggered before a file is deleted.
	// Use case: Validate permissions, check dependencies, implement recycle bin.
	// Hooks can block the delete operation by not calling next().
	BeforeFileDelete HookPoint = "BeforeFileDelete"
	// AfterFileDelete is triggered after a file is successfully deleted.
	// Use case: Cleanup related state, update indexes, log deletion.
	AfterFileDelete HookPoint = "AfterFileDelete"
	// BeforeFileModify is triggered before a file is modified (edit operation).
	// Use case: Validate locks, capture old content, check file format.
	// Set HookContext.OldContent and HookContext.NewContent for tracking.
	BeforeFileModify HookPoint = "BeforeFileModify"
	// AfterFileModify is triggered after a file is successfully modified.
	// Use case: Track changes, update file state, trigger notifications.
	AfterFileModify HookPoint = "AfterFileModify"

	// BeforeLLMRequest is triggered before calling the LLM API.
	// Use case: Modify prompt, inject system messages, validate input, implement rate limiting.
	// Hooks can modify LLMInput or block the request by not calling next().
	BeforeLLMRequest HookPoint = "BeforeLLMRequest"
	// AfterLLMResponse is triggered after a successful LLM response.
	// Use case: Modify response, log interactions, implement caching, filter content.
	// Hooks can modify LLMResponse to change what's returned to the caller.
	AfterLLMResponse HookPoint = "AfterLLMResponse"
	// OnLLMError is triggered when an LLM call fails.
	// Use case: Implement retry logic, fallback to alternative models, log errors.
	// Hooks can recover by setting LLMResponse or allow error to propagate.
	OnLLMError HookPoint = "OnLLMError"

	// BeforeSkillInvoked is triggered before a skill is invoked.
	// Use case: Validate skill permissions, log skill usage, modify skill input.
	// Hooks can access skill name, context mode, and model via HookContext.Data.
	BeforeSkillInvoked HookPoint = "BeforeSkillInvoked"
	// AfterSkillInvoked is triggered after a skill completes successfully.
	// Use case: Log skill execution, update metrics, cache skill results.
	// Hooks can access skill name, context mode, model, and result via HookContext.Data.
	AfterSkillInvoked HookPoint = "AfterSkillInvoked"
	// OnSkillError is triggered when a skill invocation fails.
	// Use case: Implement fallback skills, log errors, notify administrators.
	// Hooks can recover by setting ToolResult or allow error to propagate.
	OnSkillError HookPoint = "OnSkillError"
)

// String returns the string representation of the hook point.
func (h HookPoint) String() string {
	return string(h)
}

// HookContext carries contextual information for hook execution.
//
// Deprecated: Use TypedHookContext[T] with typed payloads for better type safety.
// This type is maintained for backward compatibility during migration.
// See generics.go for the new typed implementation.
type HookContext struct {
	SessionID  uuid.UUID      // Optional: session identifier
	AgentID    uuid.UUID      // Optional: agent identifier
	ToolName   string         // Optional: tool name for tool hooks
	ToolArgs   map[string]any // Optional: tool arguments for BeforeToolExecution
	ToolResult map[string]any // Optional: tool result for AfterToolExecution
	ToolError  error          // Optional: tool error for OnToolError

	// File-related fields for file state hooks
	FilePath    string // Optional: file path for file hooks
	FileContent string // Optional: file content (hooks can modify before write)
	OldContent  string // Optional: old file content for modify operations
	NewContent  string // Optional: new file content for modify operations

	// LLM-related fields for LLM hooks
	LLMInput    string         // Optional: LLM input prompt for BeforeLLMRequest
	LLMResponse string         // Optional: LLM response text for AfterLLMResponse
	LLMModel    string         // Optional: LLM model identifier (e.g., "claude-3-5-sonnet")
	LLMError    error          // Optional: LLM error for OnLLMError
	LLMOptions  map[string]any // Optional: Additional LLM options/parameters

	Data map[string]any
}

// Clone creates a deep copy of the HookContext.
func (hc *HookContext) Clone() *HookContext {
	if hc == nil {
		return &HookContext{Data: make(map[string]any)}
	}
	cpy := &HookContext{
		SessionID:   hc.SessionID,
		AgentID:     hc.AgentID,
		ToolName:    hc.ToolName,
		ToolError:   hc.ToolError,
		FilePath:    hc.FilePath,
		FileContent: hc.FileContent,
		OldContent:  hc.OldContent,
		NewContent:  hc.NewContent,
		LLMInput:    hc.LLMInput,
		LLMResponse: hc.LLMResponse,
		LLMModel:    hc.LLMModel,
		LLMError:    hc.LLMError,
		Data:        make(map[string]any, len(hc.Data)),
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

	// Deep copy LLMOptions
	if hc.LLMOptions != nil {
		cpy.LLMOptions = make(map[string]any, len(hc.LLMOptions))
		for k, v := range hc.LLMOptions {
			cpy.LLMOptions[k] = v
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
// Deprecated: Use TypedHookFunc[T] with typed payloads for better type safety.
// This type is maintained for backward compatibility during migration.
// See generics.go for the new typed implementation.
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

// NoOpHookManager is a no-op implementation of HookManager for testing.
type NoOpHookManager struct{}

// Ensure NoOpHookManager implements HookManager.
var _ HookManager = (*NoOpHookManager)(nil)

// NewNoOpHookManager creates a new no-op hook manager.
func NewNoOpHookManager() *NoOpHookManager {
	return &NoOpHookManager{}
}

// RegisterHook is a no-op implementation of HookManager.RegisterHook.
func (n *NoOpHookManager) RegisterHook(_ HookFunc, _ HookMetadata) error { return nil }

// UnregisterHook is a no-op implementation of HookManager.UnregisterHook.
func (n *NoOpHookManager) UnregisterHook(_ string) bool { return false }

// TriggerHooks is a no-op implementation of HookManager.TriggerHooks.
func (n *NoOpHookManager) TriggerHooks(_ context.Context, _ HookPoint, _ *HookContext) HookResult {
	return HookResult{}
}

// RegisterToolHook is a no-op implementation of HookManager.RegisterToolHook.
func (n *NoOpHookManager) RegisterToolHook(_ TypedHookFunc[ToolPayload], _ TypedHookMetadata) error {
	return nil
}

// RegisterLLMHook is a no-op implementation of HookManager.RegisterLLMHook.
func (n *NoOpHookManager) RegisterLLMHook(_ TypedHookFunc[LLMPayload], _ TypedHookMetadata) error {
	return nil
}

// RegisterFileHook is a no-op implementation of HookManager.RegisterFileHook.
func (n *NoOpHookManager) RegisterFileHook(_ TypedHookFunc[FilePayload], _ TypedHookMetadata) error {
	return nil
}

// RegisterSessionHook is a no-op implementation of HookManager.RegisterSessionHook.
func (n *NoOpHookManager) RegisterSessionHook(_ TypedHookFunc[SessionPayload], _ TypedHookMetadata) error {
	return nil
}

// RegisterAgentHook is a no-op implementation of HookManager.RegisterAgentHook.
func (n *NoOpHookManager) RegisterAgentHook(_ TypedHookFunc[AgentPayload], _ TypedHookMetadata) error {
	return nil
}

// RegisterSkillHook is a no-op implementation of HookManager.RegisterSkillHook.
func (n *NoOpHookManager) RegisterSkillHook(_ TypedHookFunc[SkillPayload], _ TypedHookMetadata) error {
	return nil
}

// TriggerToolHooks is a no-op implementation of HookManager.TriggerToolHooks.
func (n *NoOpHookManager) TriggerToolHooks(_ context.Context, _ HookPoint, hookCtx *TypedHookContext[ToolPayload]) TypedHookResult[ToolPayload] {
	return TypedHookResult[ToolPayload]{Payload: hookCtx.Payload}
}

// TriggerLLMHooks is a no-op implementation of HookManager.TriggerLLMHooks.
func (n *NoOpHookManager) TriggerLLMHooks(_ context.Context, _ HookPoint, hookCtx *TypedHookContext[LLMPayload]) TypedHookResult[LLMPayload] {
	return TypedHookResult[LLMPayload]{Payload: hookCtx.Payload}
}

// TriggerFileHooks is a no-op implementation of HookManager.TriggerFileHooks.
func (n *NoOpHookManager) TriggerFileHooks(_ context.Context, _ HookPoint, hookCtx *TypedHookContext[FilePayload]) TypedHookResult[FilePayload] {
	return TypedHookResult[FilePayload]{Payload: hookCtx.Payload}
}

// TriggerSessionHooks is a no-op implementation of HookManager.TriggerSessionHooks.
func (n *NoOpHookManager) TriggerSessionHooks(_ context.Context, _ HookPoint, hookCtx *TypedHookContext[SessionPayload]) TypedHookResult[SessionPayload] {
	return TypedHookResult[SessionPayload]{Payload: hookCtx.Payload}
}

// TriggerAgentHooks is a no-op implementation of HookManager.TriggerAgentHooks.
func (n *NoOpHookManager) TriggerAgentHooks(_ context.Context, _ HookPoint, hookCtx *TypedHookContext[AgentPayload]) TypedHookResult[AgentPayload] {
	return TypedHookResult[AgentPayload]{Payload: hookCtx.Payload}
}

// TriggerSkillHooks is a no-op implementation of HookManager.TriggerSkillHooks.
func (n *NoOpHookManager) TriggerSkillHooks(_ context.Context, _ HookPoint, hookCtx *TypedHookContext[SkillPayload]) TypedHookResult[SkillPayload] {
	return TypedHookResult[SkillPayload]{Payload: hookCtx.Payload}
}

// WithSessionHooks is a no-op implementation of HookManager.WithSessionHooks.
func (n *NoOpHookManager) WithSessionHooks(_ context.Context, _ uuid.UUID, work func() error) error {
	return work()
}

// WithAgentHooks is a no-op implementation of HookManager.WithAgentHooks.
func (n *NoOpHookManager) WithAgentHooks(_ context.Context, _, _ uuid.UUID, _ HookPoint, work func() error) error {
	return work()
}

// WithToolHooks is a no-op implementation of HookManager.WithToolHooks.
func (n *NoOpHookManager) WithToolHooks(_ context.Context, _, _ uuid.UUID, _ string, _ map[string]any, work func() (map[string]any, error)) (map[string]any, error) {
	return work()
}

// WithFileReadHooks is a no-op implementation of HookManager.WithFileReadHooks.
func (n *NoOpHookManager) WithFileReadHooks(_ context.Context, _, _ uuid.UUID, _ string, work func() (string, error)) (string, error) {
	return work()
}

// WithFileWriteHooks is a no-op implementation of HookManager.WithFileWriteHooks.
func (n *NoOpHookManager) WithFileWriteHooks(_ context.Context, _, _ uuid.UUID, _, _ string, work func(string) error) error {
	return work("")
}

// WithFileHooks is a no-op implementation of HookManager.WithFileHooks.
func (n *NoOpHookManager) WithFileHooks(_ context.Context, _, _ uuid.UUID, _ HookPoint, _ string, work func() error) error {
	return work()
}

// WithLLMHooks is a no-op implementation of HookManager.WithLLMHooks.
func (n *NoOpHookManager) WithLLMHooks(_ context.Context, _, _ uuid.UUID, _, _ string, work func(string) (string, error)) (string, error) {
	return work("")
}

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
