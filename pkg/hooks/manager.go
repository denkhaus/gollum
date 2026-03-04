package hooks

import (
	"context"
	"fmt"
	"sync"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// HookManager defines the interface for managing and triggering hooks.
type HookManager interface {
	// RegisterHook registers a hook function for a specific hook point.
	//
	// Deprecated: Use the typed Register*Hook methods (RegisterToolHook, RegisterLLMHook, etc.)
	// for better type safety. This method is maintained for backward compatibility.
	RegisterHook(fn HookFunc, meta HookMetadata) error

	// Type-safe registration methods per hook category.
	// These methods provide compile-time type safety for hook functions.

	// RegisterToolHook registers a typed hook for tool execution events.
	// Valid hook points: BeforeToolExecution, AfterToolExecution, OnToolError.
	RegisterToolHook(fn TypedHookFunc[ToolPayload], meta TypedHookMetadata) error

	// RegisterLLMHook registers a typed hook for LLM request/response events.
	// Valid hook points: BeforeLLMRequest, AfterLLMResponse, OnLLMError.
	RegisterLLMHook(fn TypedHookFunc[LLMPayload], meta TypedHookMetadata) error

	// RegisterFileHook registers a typed hook for file operation events.
	// Valid hook points: BeforeFileRead, AfterFileRead, BeforeFileWrite, AfterFileWrite,
	// BeforeFileDelete, AfterFileDelete, BeforeFileModify, AfterFileModify.
	RegisterFileHook(fn TypedHookFunc[FilePayload], meta TypedHookMetadata) error

	// RegisterSessionHook registers a typed hook for session lifecycle events.
	// Valid hook points: BeforeSessionStart, AfterSessionEnd.
	RegisterSessionHook(fn TypedHookFunc[SessionPayload], meta TypedHookMetadata) error

	// RegisterAgentHook registers a typed hook for agent lifecycle events.
	// Valid hook points: BeforeAgentSpawn, AfterAgentSpawn, BeforeAgentRemove, AfterAgentRemove.
	RegisterAgentHook(fn TypedHookFunc[AgentPayload], meta TypedHookMetadata) error

	// RegisterSkillHook registers a typed hook for skill invocation events.
	// Valid hook points: BeforeSkillInvoked, AfterSkillInvoked, OnSkillError.
	RegisterSkillHook(fn TypedHookFunc[SkillPayload], meta TypedHookMetadata) error

	// UnregisterHook removes a hook by name from all registries.
	// Returns true if at least one hook was removed.
	UnregisterHook(name string) bool

	// TriggerHooks executes all registered hooks for a given hook point.
	// Returns a HookResult indicating whether execution was stopped and any errors.
	//
	// Deprecated: Use the typed Trigger*Hooks methods (TriggerToolHooks, TriggerLLMHooks, etc.)
	// for better type safety. This method is maintained for backward compatibility.
	TriggerHooks(ctx context.Context, point HookPoint, hookCtx *HookContext) HookResult

	// Type-safe trigger methods per hook category.
	// These methods provide compile-time type safety for hook execution.

	// TriggerToolHooks executes all typed tool hooks for a given hook point.
	// Valid hook points: BeforeToolExecution, AfterToolExecution, OnToolError.
	TriggerToolHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[ToolPayload]) TypedHookResult[ToolPayload]

	// TriggerLLMHooks executes all typed LLM hooks for a given hook point.
	// Valid hook points: BeforeLLMRequest, AfterLLMResponse, OnLLMError.
	TriggerLLMHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[LLMPayload]) TypedHookResult[LLMPayload]

	// TriggerFileHooks executes all typed file hooks for a given hook point.
	// Valid hook points: BeforeFileRead, AfterFileRead, BeforeFileWrite, AfterFileWrite,
	// BeforeFileDelete, AfterFileDelete, BeforeFileModify, AfterFileModify.
	TriggerFileHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[FilePayload]) TypedHookResult[FilePayload]

	// TriggerSessionHooks executes all typed session hooks for a given hook point.
	// Valid hook points: BeforeSessionStart, AfterSessionEnd.
	TriggerSessionHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[SessionPayload]) TypedHookResult[SessionPayload]

	// TriggerAgentHooks executes all typed agent hooks for a given hook point.
	// Valid hook points: BeforeAgentSpawn, AfterAgentSpawn, BeforeAgentRemove, AfterAgentRemove.
	TriggerAgentHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[AgentPayload]) TypedHookResult[AgentPayload]

	// TriggerSkillHooks executes all typed skill hooks for a given hook point.
	// Valid hook points: BeforeSkillInvoked, AfterSkillInvoked, OnSkillError.
	TriggerSkillHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[SkillPayload]) TypedHookResult[SkillPayload]

	// WithSessionHooks wraps a function with session lifecycle hooks.
	WithSessionHooks(
		ctx context.Context,
		sessionID uuid.UUID,
		work func() error,
	) error

	// WithAgentHooks wraps a function with agent lifecycle hooks.
	WithAgentHooks(
		ctx context.Context,
		sessionID, agentID uuid.UUID,
		point HookPoint,
		work func() error,
	) error

	// WithToolHooks wraps a function with tool execution hooks.
	// BeforeToolExecution hooks can modify args before tool runs.
	// AfterToolExecution hooks can modify result after tool completes.
	// OnToolError hooks can handle errors or return fallback responses.
	WithToolHooks(
		ctx context.Context,
		sessionID, agentID uuid.UUID,
		toolName shared.ToolName,
		args map[string]any,
		work func() (map[string]any, error),
	) (map[string]any, error)

	// WithFileReadHooks wraps a file read operation with hooks.
	// Returns the file content, potentially modified by AfterFileRead hooks.
	//
	// Hook execution flow:
	// 1. BeforeFileRead hooks run - can validate and block reads
	// 2. File read work executes
	// 3. AfterFileRead hooks run - can transform content before returning
	//
	// The returned content is the final content after any modifications by AfterFileRead hooks.
	WithFileReadHooks(
		ctx context.Context,
		sessionID, agentID uuid.UUID,
		filePath string,
		work func() (string, error),
	) (string, error)

	// WithFileWriteHooks wraps a file write operation with hooks.
	// BeforeFileWrite hooks can transform the content before writing.
	// AfterFileWrite hooks can log/audit the write operation.
	//
	// Hook execution flow:
	// 1. BeforeFileWrite hooks run - can transform content via HookContext.FileContent
	// 2. File write work executes - receives the final content after hook transformations
	// 3. AfterFileWrite hooks run - can log/audit
	WithFileWriteHooks(
		ctx context.Context,
		sessionID, agentID uuid.UUID,
		filePath string,
		content string,
		work func(string) error,
	) error

	// WithFileHooks wraps a function with file operation hooks.
	// BeforeFileRead/BeforeFileWrite/BeforeFileDelete/BeforeFileModify hooks can block execution.
	// AfterFileRead/AfterFileWrite/AfterFileDelete/AfterFileModify hooks can log/audit operations.
	//
	// IMPORTANT: For file read operations that need content modification, use WithFileReadHooks instead.
	// This method cannot return modified content to the caller - hooks can log/audit only.
	//
	// The point parameter can be ANY of the 8 file hook points (BeforeFileRead, AfterFileRead,
	// BeforeFileWrite, AfterFileWrite, BeforeFileDelete, AfterFileDelete, BeforeFileModify,
	// AfterFileModify). The method will automatically determine the appropriate before/after
	// hook pair to execute. For example, passing either BeforeFileRead or AfterFileRead will
	// result in both BeforeFileRead and AfterFileRead hooks being executed in sequence.
	WithFileHooks(
		ctx context.Context,
		sessionID, agentID uuid.UUID,
		point HookPoint,
		filePath string,
		work func() error,
	) error

	// WithLLMHooks wraps an LLM API call with hooks for request, response, and error handling.
	// BeforeLLMRequest hooks can modify the prompt or block the request.
	// AfterLLMResponse hooks can modify the response text.
	// OnLLMError hooks can implement retry logic or provide fallback responses.
	//
	// Hook execution flow:
	// 1. BeforeLLMRequest hooks run - can modify prompt via HookContext.LLMInput
	// 2. LLM API call executes with potentially modified prompt
	// 3. If successful, AfterLLMResponse hooks run - can modify response via HookContext.LLMResponse
	// 4. If error occurs, OnLLMError hooks run - can recover by setting LLMResponse
	//
	// Returns the LLM response text, potentially modified by hooks.
	WithLLMHooks(
		ctx context.Context,
		sessionID, agentID uuid.UUID,
		prompt string,
		model string,
		work func(string) (string, error),
	) (string, error)
}

// hookManagerImpl is the private implementation of HookManager.
type hookManagerImpl struct {
	log        logger.LoggerService
	registries map[HookPoint]*hookRegistry
	names      map[string]struct{} // Global name registry for uniqueness across all hook points
	mu         sync.RWMutex        // Protects the names map

	// Typed registries for type-safe hook storage
	toolRegistry    *TypedRegistry[ToolPayload]
	llmRegistry     *TypedRegistry[LLMPayload]
	fileRegistry    *TypedRegistry[FilePayload]
	sessionRegistry *TypedRegistry[SessionPayload]
	agentRegistry   *TypedRegistry[AgentPayload]
	skillRegistry   *TypedRegistry[SkillPayload]
}

// Ensure hookManagerImpl implements HookManager.
var _ HookManager = (*hookManagerImpl)(nil)

// NewHookManager creates a new HookManager service.
func NewHookManager(injector do.Injector) (HookManager, error) {
	log := do.MustInvoke[logger.LoggerService](injector)

	log.Debug("HookManager starting")

	p := &hookManagerImpl{
		log:        log,
		registries: make(map[HookPoint]*hookRegistry),
		names:      make(map[string]struct{}),
	}

	// Initialize legacy registries for all known hook points
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
		BeforeSkillInvoked,
		AfterSkillInvoked,
		OnSkillError,
	} {
		p.registries[point] = &hookRegistry{}
	}

	// Initialize typed registries for type-safe hook storage
	p.toolRegistry = NewTypedRegistry[ToolPayload]()
	p.llmRegistry = NewTypedRegistry[LLMPayload]()
	p.fileRegistry = NewTypedRegistry[FilePayload]()
	p.sessionRegistry = NewTypedRegistry[SessionPayload]()
	p.agentRegistry = NewTypedRegistry[AgentPayload]()
	p.skillRegistry = NewTypedRegistry[SkillPayload]()

	return p, nil
}

// RegisterHook registers a hook function for a specific hook point.
func (p *hookManagerImpl) RegisterHook(fn HookFunc, meta HookMetadata) error {
	if fn == nil {
		return errs.Validation("hook function cannot be nil")
	}

	if meta.Name == "" {
		return errs.Validation("hook name cannot be empty")
	}

	// Validate hook point
	registry, exists := p.registries[meta.Point]
	if !exists {
		return errs.Validationf("unknown hook point: %s", meta.Point)
	}

	// Check for global name uniqueness
	p.mu.Lock()
	if _, exists := p.names[meta.Name]; exists {
		p.mu.Unlock()
		return errs.Conflictf("hook with name '%s' is already registered globally", meta.Name)
	}

	// add() performs atomic duplicate checking within the point-specific registry
	id, err := registry.add(fn, meta)
	if err != nil {
		p.mu.Unlock()
		if err == ErrDuplicateHook {
			return errs.Conflictf("hook with name '%s' already registered for point '%s'",
				meta.Name, meta.Point)
		}
		return err
	}

	// Add to global names map
	p.names[meta.Name] = struct{}{}
	p.mu.Unlock()

	p.log.Debug("Hook registered",
		zap.String("name", meta.Name),
		zap.String("point", meta.Point.String()),
		zap.Int("priority", meta.Priority),
		zap.Int("id", id))

	return nil
}

// UnregisterHook removes a hook by name from all registries (both legacy and typed).
// Returns true if at least one hook was removed.
func (p *hookManagerImpl) UnregisterHook(name string) bool {
	unregistered := false

	// Check legacy registries
	for point, registry := range p.registries {
		if registry.remove(name) {
			p.log.Debug("Hook unregistered from legacy registry",
				zap.String("name", name),
				zap.String("point", point.String()))
			unregistered = true
			break
		}
	}

	// Check typed registries if not found in legacy
	if !unregistered {
		switch {
		case p.toolRegistry.Remove(name):
			p.log.Debug("Hook unregistered from tool registry", zap.String("name", name))
			unregistered = true
		case p.llmRegistry.Remove(name):
			p.log.Debug("Hook unregistered from LLM registry", zap.String("name", name))
			unregistered = true
		case p.fileRegistry.Remove(name):
			p.log.Debug("Hook unregistered from file registry", zap.String("name", name))
			unregistered = true
		case p.sessionRegistry.Remove(name):
			p.log.Debug("Hook unregistered from session registry", zap.String("name", name))
			unregistered = true
		case p.agentRegistry.Remove(name):
			p.log.Debug("Hook unregistered from agent registry", zap.String("name", name))
			unregistered = true
		case p.skillRegistry.Remove(name):
			p.log.Debug("Hook unregistered from skill registry", zap.String("name", name))
			unregistered = true
		}
	}

	// Remove from global names map if hook was found and removed
	if unregistered {
		p.mu.Lock()
		delete(p.names, name)
		p.mu.Unlock()
	}

	return unregistered
}

// TriggerHooks executes all registered hooks for a given hook point.
//
// Hooks are executed in priority order (lowest first). Each hook can:
// - Modify the HookContext to pass data to subsequent hooks
// - Call next() to continue the chain (and optionally wrap it for before/after logic)
// - Return an error to stop execution (if FatalError=true)
// - Not call next() to stop the chain at that point
//
// Returns a HookResult with:
// - Stopped: true if a hook stopped execution
// - Error: the error that caused the stop (if any)
// - Data: accumulated data from all hooks
func (p *hookManagerImpl) TriggerHooks(ctx context.Context, point HookPoint, hookCtx *HookContext) HookResult {
	result := HookResult{
		Stopped: false,
		Error:   nil,
		Data:    make(map[string]any),
	}

	// Initialize hookCtx if nil
	if hookCtx == nil {
		hookCtx = &HookContext{Data: make(map[string]any)}
	}
	if hookCtx.Data == nil {
		hookCtx.Data = make(map[string]any)
	}

	// Initialize tool-related fields for tool hook points (defensive)
	if point == BeforeToolExecution && hookCtx.ToolArgs == nil {
		hookCtx.ToolArgs = make(map[string]any)
	}
	if point == AfterToolExecution && hookCtx.ToolResult == nil {
		hookCtx.ToolResult = make(map[string]any)
	}

	// Get registry for this point
	registry, exists := p.registries[point]
	if !exists {
		p.log.Warn("No hooks registered for point", zap.String("point", point.String()))
		return result
	}

	// Get hooks in priority order
	hooks := registry.get()
	if len(hooks) == 0 {
		return result
	}

	p.log.Debug("Triggering hooks",
		zap.String("point", point.String()),
		zap.Int("count", len(hooks)))

	// Index-based dispatcher for middleware pattern
	// This allows hooks to:
	// 1. Wrap next() to run code before and after subsequent hooks
	// 2. Stop the chain by not calling next()
	index := -1
	var next func() error

	next = func() error {
		// Check if chain was already stopped by a fatal error
		if result.Stopped {
			return result.Error
		}

		index++
		if index >= len(hooks) {
			// Reached the end of the chain
			return nil
		}

		reg := hooks[index]
		meta := reg.metadata

		// Execute the current hook
		err := reg.fn(ctx, hookCtx, next)

		if err != nil {
			if meta.FatalError {
				result.Stopped = true
				result.Error = errs.Wrap(err, errs.TypeInternal,
					fmt.Sprintf("hook '%s' at point '%s' failed", meta.Name, meta.Point))
				p.log.Error("Hook execution failed (fatal)",
					zap.String("name", meta.Name),
					zap.String("point", meta.Point.String()),
					zap.Error(err))
				// Stop execution by not calling next() and returning the error
				return result.Error
			}
			// Non-fatal: log and continue to the next hook in the chain
			p.log.Warn("Hook execution failed (non-fatal, continuing)",
				zap.String("name", meta.Name),
				zap.String("point", meta.Point.String()),
				zap.Error(err))
			return next()
		}

		// If the hook returned nil, the chain either continued via a call to next()
		// or was intentionally stopped by the hook not calling next().
		// The recursive nature of the calls handles this correctly.
		return nil
	}

	_ = next()

	// If the chain was stopped prematurely (a hook didn't call next),
	// the index will not have reached len(hooks).
	// A complete chain would have index == len(hooks) after all hooks complete.
	if index < len(hooks) && result.Error == nil {
		result.Stopped = true
	}

	// Accumulate data from hook context
	for k, v := range hookCtx.Data {
		result.Data[k] = v
	}

	return result
}

// ============================================================================
// Typed Registration Methods
// ============================================================================

// RegisterToolHook registers a typed hook for tool execution events.
func (p *hookManagerImpl) RegisterToolHook(fn TypedHookFunc[ToolPayload], meta TypedHookMetadata) error {
	if fn == nil {
		return errs.Validation("hook function cannot be nil")
	}
	if meta.Name == "" {
		return errs.Validation("hook name cannot be empty")
	}
	if err := p.validateToolHookPoint(meta.Point); err != nil {
		return err
	}

	// Check for global name uniqueness
	p.mu.Lock()
	if _, exists := p.names[meta.Name]; exists {
		p.mu.Unlock()
		return errs.Conflictf("hook with name '%s' is already registered globally", meta.Name)
	}

	if err := p.toolRegistry.Add(fn, meta); err != nil {
		p.mu.Unlock()
		if err == ErrDuplicateHook {
			return errs.Conflictf("hook with name '%s' already registered", meta.Name)
		}
		return err
	}

	p.names[meta.Name] = struct{}{}
	p.mu.Unlock()

	p.log.Debug("Typed hook registered",
		zap.String("name", meta.Name),
		zap.String("point", meta.Point.String()),
		zap.Int("priority", meta.Priority))

	return nil
}

// RegisterLLMHook registers a typed hook for LLM request/response events.
func (p *hookManagerImpl) RegisterLLMHook(fn TypedHookFunc[LLMPayload], meta TypedHookMetadata) error {
	if fn == nil {
		return errs.Validation("hook function cannot be nil")
	}
	if meta.Name == "" {
		return errs.Validation("hook name cannot be empty")
	}
	if err := p.validateLLMHookPoint(meta.Point); err != nil {
		return err
	}

	// Check for global name uniqueness
	p.mu.Lock()
	if _, exists := p.names[meta.Name]; exists {
		p.mu.Unlock()
		return errs.Conflictf("hook with name '%s' is already registered globally", meta.Name)
	}

	if err := p.llmRegistry.Add(fn, meta); err != nil {
		p.mu.Unlock()
		if err == ErrDuplicateHook {
			return errs.Conflictf("hook with name '%s' already registered", meta.Name)
		}
		return err
	}

	p.names[meta.Name] = struct{}{}
	p.mu.Unlock()

	p.log.Debug("Typed hook registered",
		zap.String("name", meta.Name),
		zap.String("point", meta.Point.String()),
		zap.Int("priority", meta.Priority))

	return nil
}

// RegisterFileHook registers a typed hook for file operation events.
func (p *hookManagerImpl) RegisterFileHook(fn TypedHookFunc[FilePayload], meta TypedHookMetadata) error {
	if fn == nil {
		return errs.Validation("hook function cannot be nil")
	}
	if meta.Name == "" {
		return errs.Validation("hook name cannot be empty")
	}
	if err := p.validateFileHookPoint(meta.Point); err != nil {
		return err
	}

	// Check for global name uniqueness
	p.mu.Lock()
	if _, exists := p.names[meta.Name]; exists {
		p.mu.Unlock()
		return errs.Conflictf("hook with name '%s' is already registered globally", meta.Name)
	}

	if err := p.fileRegistry.Add(fn, meta); err != nil {
		p.mu.Unlock()
		if err == ErrDuplicateHook {
			return errs.Conflictf("hook with name '%s' already registered", meta.Name)
		}
		return err
	}

	p.names[meta.Name] = struct{}{}
	p.mu.Unlock()

	p.log.Debug("Typed hook registered",
		zap.String("name", meta.Name),
		zap.String("point", meta.Point.String()),
		zap.Int("priority", meta.Priority))

	return nil
}

// RegisterSessionHook registers a typed hook for session lifecycle events.
func (p *hookManagerImpl) RegisterSessionHook(fn TypedHookFunc[SessionPayload], meta TypedHookMetadata) error {
	if fn == nil {
		return errs.Validation("hook function cannot be nil")
	}
	if meta.Name == "" {
		return errs.Validation("hook name cannot be empty")
	}
	if err := p.validateSessionHookPoint(meta.Point); err != nil {
		return err
	}

	// Check for global name uniqueness
	p.mu.Lock()
	if _, exists := p.names[meta.Name]; exists {
		p.mu.Unlock()
		return errs.Conflictf("hook with name '%s' is already registered globally", meta.Name)
	}

	if err := p.sessionRegistry.Add(fn, meta); err != nil {
		p.mu.Unlock()
		if err == ErrDuplicateHook {
			return errs.Conflictf("hook with name '%s' already registered", meta.Name)
		}
		return err
	}

	p.names[meta.Name] = struct{}{}
	p.mu.Unlock()

	p.log.Debug("Typed hook registered",
		zap.String("name", meta.Name),
		zap.String("point", meta.Point.String()),
		zap.Int("priority", meta.Priority))

	return nil
}

// RegisterAgentHook registers a typed hook for agent lifecycle events.
func (p *hookManagerImpl) RegisterAgentHook(fn TypedHookFunc[AgentPayload], meta TypedHookMetadata) error {
	if fn == nil {
		return errs.Validation("hook function cannot be nil")
	}
	if meta.Name == "" {
		return errs.Validation("hook name cannot be empty")
	}
	if err := p.validateAgentHookPoint(meta.Point); err != nil {
		return err
	}

	// Check for global name uniqueness
	p.mu.Lock()
	if _, exists := p.names[meta.Name]; exists {
		p.mu.Unlock()
		return errs.Conflictf("hook with name '%s' is already registered globally", meta.Name)
	}

	if err := p.agentRegistry.Add(fn, meta); err != nil {
		p.mu.Unlock()
		if err == ErrDuplicateHook {
			return errs.Conflictf("hook with name '%s' already registered", meta.Name)
		}
		return err
	}

	p.names[meta.Name] = struct{}{}
	p.mu.Unlock()

	p.log.Debug("Typed hook registered",
		zap.String("name", meta.Name),
		zap.String("point", meta.Point.String()),
		zap.Int("priority", meta.Priority))

	return nil
}

// RegisterSkillHook registers a typed hook for skill invocation events.
func (p *hookManagerImpl) RegisterSkillHook(fn TypedHookFunc[SkillPayload], meta TypedHookMetadata) error {
	if fn == nil {
		return errs.Validation("hook function cannot be nil")
	}
	if meta.Name == "" {
		return errs.Validation("hook name cannot be empty")
	}
	if err := p.validateSkillHookPoint(meta.Point); err != nil {
		return err
	}

	// Check for global name uniqueness
	p.mu.Lock()
	if _, exists := p.names[meta.Name]; exists {
		p.mu.Unlock()
		return errs.Conflictf("hook with name '%s' is already registered globally", meta.Name)
	}

	if err := p.skillRegistry.Add(fn, meta); err != nil {
		p.mu.Unlock()
		if err == ErrDuplicateHook {
			return errs.Conflictf("hook with name '%s' already registered", meta.Name)
		}
		return err
	}

	p.names[meta.Name] = struct{}{}
	p.mu.Unlock()

	p.log.Debug("Typed hook registered",
		zap.String("name", meta.Name),
		zap.String("point", meta.Point.String()),
		zap.Int("priority", meta.Priority))

	return nil
}

// ============================================================================
// Typed Trigger Methods
// ============================================================================

// TriggerToolHooks executes all typed tool hooks for a given hook point.
func (p *hookManagerImpl) TriggerToolHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[ToolPayload]) TypedHookResult[ToolPayload] {
	return p.toolRegistry.Trigger(ctx, point, hookCtx)
}

// TriggerLLMHooks executes all typed LLM hooks for a given hook point.
func (p *hookManagerImpl) TriggerLLMHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[LLMPayload]) TypedHookResult[LLMPayload] {
	return p.llmRegistry.Trigger(ctx, point, hookCtx)
}

// TriggerFileHooks executes all typed file hooks for a given hook point.
func (p *hookManagerImpl) TriggerFileHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[FilePayload]) TypedHookResult[FilePayload] {
	return p.fileRegistry.Trigger(ctx, point, hookCtx)
}

// TriggerSessionHooks executes all typed session hooks for a given hook point.
func (p *hookManagerImpl) TriggerSessionHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[SessionPayload]) TypedHookResult[SessionPayload] {
	return p.sessionRegistry.Trigger(ctx, point, hookCtx)
}

// TriggerAgentHooks executes all typed agent hooks for a given hook point.
func (p *hookManagerImpl) TriggerAgentHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[AgentPayload]) TypedHookResult[AgentPayload] {
	return p.agentRegistry.Trigger(ctx, point, hookCtx)
}

// TriggerSkillHooks executes all typed skill hooks for a given hook point.
func (p *hookManagerImpl) TriggerSkillHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[SkillPayload]) TypedHookResult[SkillPayload] {
	return p.skillRegistry.Trigger(ctx, point, hookCtx)
}

// ============================================================================
// Hook Point Validation
// ============================================================================

func (p *hookManagerImpl) validateToolHookPoint(point HookPoint) error {
	switch point {
	case BeforeToolExecution, AfterToolExecution, OnToolError:
		return nil
	default:
		return errs.Validationf("invalid tool hook point: %s (expected BeforeToolExecution, AfterToolExecution, or OnToolError)", point)
	}
}

func (p *hookManagerImpl) validateLLMHookPoint(point HookPoint) error {
	switch point {
	case BeforeLLMRequest, AfterLLMResponse, OnLLMError:
		return nil
	default:
		return errs.Validationf("invalid LLM hook point: %s (expected BeforeLLMRequest, AfterLLMResponse, or OnLLMError)", point)
	}
}

func (p *hookManagerImpl) validateFileHookPoint(point HookPoint) error {
	switch point {
	case BeforeFileRead, AfterFileRead, BeforeFileWrite, AfterFileWrite,
		BeforeFileDelete, AfterFileDelete, BeforeFileModify, AfterFileModify:
		return nil
	default:
		return errs.Validationf("invalid file hook point: %s", point)
	}
}

func (p *hookManagerImpl) validateSessionHookPoint(point HookPoint) error {
	switch point {
	case BeforeSessionStart, AfterSessionEnd:
		return nil
	default:
		return errs.Validationf("invalid session hook point: %s (expected BeforeSessionStart or AfterSessionEnd)", point)
	}
}

func (p *hookManagerImpl) validateAgentHookPoint(point HookPoint) error {
	switch point {
	case BeforeAgentSpawn, AfterAgentSpawn, BeforeAgentRemove, AfterAgentRemove:
		return nil
	default:
		return errs.Validationf("invalid agent hook point: %s (expected BeforeAgentSpawn, AfterAgentSpawn, BeforeAgentRemove, or AfterAgentRemove)", point)
	}
}

func (p *hookManagerImpl) validateSkillHookPoint(point HookPoint) error {
	switch point {
	case BeforeSkillInvoked, AfterSkillInvoked, OnSkillError:
		return nil
	default:
		return errs.Validationf("invalid skill hook point: %s (expected BeforeSkillInvoked, AfterSkillInvoked, or OnSkillError)", point)
	}
}
