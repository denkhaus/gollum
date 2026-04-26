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

	"github.com/denkhaus/gollum/pkg/shared"
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

// NoOpHookManager is a no-op implementation of HookManager for testing.
type NoOpHookManager struct{}

// Ensure NoOpHookManager implements HookManager.
var _ HookManager = (*NoOpHookManager)(nil)

// NewNoOpHookManager creates a new no-op hook manager.
func NewNoOpHookManager() *NoOpHookManager {
	return &NoOpHookManager{}
}

// UnregisterHook is a no-op implementation of HookManager.UnregisterHook.
func (n *NoOpHookManager) UnregisterHook(_ string) bool { return false }

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

// RegisterExecutorHook is a no-op implementation of HookManager.RegisterExecutorHook.
func (n *NoOpHookManager) RegisterExecutorHook(_ TypedHookFunc[ExecutorPayload], _ TypedHookMetadata) error {
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

// TriggerExecutorHooks is a no-op implementation of HookManager.TriggerExecutorHooks.
func (n *NoOpHookManager) TriggerExecutorHooks(_ context.Context, _ HookPoint, hookCtx *TypedHookContext[ExecutorPayload]) TypedHookResult[ExecutorPayload] {
	return TypedHookResult[ExecutorPayload]{Payload: hookCtx.Payload}
}

// WithSessionHooks is a no-op implementation of HookManager.WithSessionHooks.
func (n *NoOpHookManager) WithSessionHooks(_ context.Context, _ shared.SessionContext, work func() error) error {
	return work()
}

// WithAgentHooks is a no-op implementation of HookManager.WithAgentHooks.
func (n *NoOpHookManager) WithAgentHooks(_ context.Context, _ shared.SessionContext, _ HookPoint, work func() error) error {
	return work()
}

// WithToolHooks is a no-op implementation of HookManager.WithToolHooks.
func (n *NoOpHookManager) WithToolHooks(_ context.Context, _ shared.SessionContext, _ shared.ToolName, _ map[string]any, work func() (map[string]any, error)) (map[string]any, error) {
	return work()
}

// WithFileReadHooks is a no-op implementation of HookManager.WithFileReadHooks.
func (n *NoOpHookManager) WithFileReadHooks(_ context.Context, _ shared.SessionContext, _ string, work func() (string, error)) (string, error) {
	return work()
}

// WithFileWriteHooks is a no-op implementation of HookManager.WithFileWriteHooks.
func (n *NoOpHookManager) WithFileWriteHooks(_ context.Context, _ shared.SessionContext, _, _ string, work func(string) error) error {
	return work("")
}

// WithFileHooks is a no-op implementation of HookManager.WithFileHooks.
func (n *NoOpHookManager) WithFileHooks(_ context.Context, _ shared.SessionContext, _ HookPoint, _ string, work func() error) error {
	return work()
}

// WithLLMHooks is a no-op implementation of HookManager.WithLLMHooks.
func (n *NoOpHookManager) WithLLMHooks(_ context.Context, _ shared.SessionContext, _, _ string, work func(string) (string, error)) (string, error) {
	return work("")
}

// WithFlowStepHooks is a no-op implementation of HookManager.WithFlowStepHooks.
func (n *NoOpHookManager) WithFlowStepHooks(_ context.Context, _, _ uuid.UUID, _, _, _ string, work func() (map[string]any, error)) (map[string]any, error) {
	return work()
}

// FlowStepContext holds metadata about the current flow and step execution.
// This is stored in context.Context to provide logging context for tool calls.
type FlowStepContext struct {
	FlowName  string // Name of the flow being executed
	StateName string // Name of the current state
	StepType  string // Type of step (llm, shell, func, mcp)
}

// flowStepContextKey is the key type used for storing FlowStepContext in context.
// Using a private struct type prevents key collisions.
type flowStepContextKey struct{}

// GetFlowStepContext retrieves FlowStepContext from the context.
// Returns nil if no flow/step context is set in the context.
func GetFlowStepContext(ctx context.Context) *FlowStepContext {
	if fc, ok := ctx.Value(flowStepContextKey{}).(*FlowStepContext); ok {
		return fc
	}
	return nil
}

// WithFlowStepContext stores FlowStepContext in a new context derived from the parent.
func WithFlowStepContext(ctx context.Context, fc *FlowStepContext) context.Context {
	return context.WithValue(ctx, flowStepContextKey{}, fc)
}
