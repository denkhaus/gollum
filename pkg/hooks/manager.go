package hooks

import (
	"context"
	"fmt"
	"sync"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// HookManager defines the interface for managing and triggering hooks.
type HookManager interface {
	// RegisterHook registers a hook function for a specific hook point.
	RegisterHook(fn HookFunc, meta HookMetadata) error

	// UnregisterHook removes a hook by name.
	UnregisterHook(name string) bool

	// TriggerHooks executes all registered hooks for a given hook point.
	// Returns a HookResult indicating whether execution was stopped and any errors.
	TriggerHooks(ctx context.Context, point HookPoint, hookCtx *HookContext) HookResult

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
		toolName string,
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

	// Initialize registries for all known hook points
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

// UnregisterHook removes a hook by name from all hook points.
// Returns true if at least one hook was removed.
func (p *hookManagerImpl) UnregisterHook(name string) bool {
	unregistered := false
	for point, registry := range p.registries {
		if registry.remove(name) {
			p.log.Debug("Hook unregistered",
				zap.String("name", name),
				zap.String("point", point.String()))
			unregistered = true
			// Only remove from global names once
			break
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
