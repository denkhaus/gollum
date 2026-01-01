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

// WithSessionHooks wraps a function with session lifecycle hooks.
func (p *hookManagerImpl) WithSessionHooks(
	ctx context.Context,
	sessionID uuid.UUID,
	work func() error,
) error {
	if sessionID == uuid.Nil {
		return errs.Validation("session ID cannot be nil")
	}

	// BeforeSessionStart hook
	hookCtx := &HookContext{
		SessionID: sessionID,
		Data:      make(map[string]any),
	}
	result := p.TriggerHooks(ctx, BeforeSessionStart, hookCtx)
	if result.Stopped {
		return result.Error
	}

	// Execute the work
	workErr := work()

	// AfterSessionEnd hook
	// Always trigger AfterSessionEnd, even if work failed, to ensure cleanup runs.
	result = p.TriggerHooks(ctx, AfterSessionEnd, hookCtx)

	// If a fatal 'after' hook failed, its error takes precedence.
	if result.Error != nil {
		if workErr != nil {
			p.log.Error("The original work function also returned an error, which is being superseded by the AfterSessionEnd hook error", zap.Error(workErr))
		}
		return result.Error
	}

	// Otherwise, return the error from the original work function (if any).
	return workErr
}

// WithAgentHooks wraps a function with agent lifecycle hooks.
func (p *hookManagerImpl) WithAgentHooks(
	ctx context.Context,
	sessionID, agentID uuid.UUID,
	point HookPoint,
	work func() error,
) error {
	if agentID == uuid.Nil {
		return errs.Validation("agent ID cannot be nil")
	}

	// Validate hook point
	switch point {
	case BeforeAgentSpawn, AfterAgentSpawn, BeforeAgentRemove, AfterAgentRemove:
		// valid agent hook points
	default:
		return errs.Validationf("invalid agent hook point: %s", point)
	}

	hookCtx := &HookContext{
		SessionID: sessionID,
		AgentID:   agentID,
		Data:      make(map[string]any),
	}

	result := p.TriggerHooks(ctx, point, hookCtx)
	if result.Stopped {
		return result.Error
	}

	// Execute work if provided
	if work != nil {
		return work()
	}

	return nil
}

// WithToolHooks wraps a function with tool execution hooks.
//
// The workflow is:
// 1. BeforeToolExecution hooks run with args in HookContext
//    - Hooks can modify args via HookContext.ToolArgs
//    - Hooks can block execution by not calling next()
// 2. Tool execution (work function) runs
// 3. If tool succeeds, AfterToolExecution hooks run with result
//    - Hooks can modify result via HookContext.ToolResult
// 4. If tool fails, OnToolError hooks run with error
//    - Hooks can recover by returning a new result
//    - Or hooks can allow the error to propagate
func (p *hookManagerImpl) WithToolHooks(
	ctx context.Context,
	sessionID, agentID uuid.UUID,
	toolName string,
	args map[string]any,
	work func() (map[string]any, error),
) (map[string]any, error) {
	if toolName == "" {
		return nil, errs.Validation("tool name cannot be empty")
	}

	// Make a copy of args to avoid modifying the original
	argsCopy := make(map[string]any, len(args))
	for k, v := range args {
		argsCopy[k] = v
	}

	// BeforeToolExecution hook
	hookCtx := &HookContext{
		SessionID: sessionID,
		AgentID:   agentID,
		ToolName:  toolName,
		ToolArgs:  argsCopy,
		Data:      make(map[string]any),
	}

	result := p.TriggerHooks(ctx, BeforeToolExecution, hookCtx)
	if result.Stopped {
		// Hook blocked execution
		if result.Error != nil {
			return nil, result.Error
		}
		// Hook stopped without error - return ToolResult if set, or empty result
		if hookCtx.ToolResult != nil {
			return hookCtx.ToolResult, nil
		}
		return map[string]any{"blocked": true}, nil
	}

	// Get potentially modified args from hook context
	argsCopy = hookCtx.ToolArgs
	if argsCopy == nil {
		argsCopy = make(map[string]any)
	}

	// Create a work wrapper that uses the modified args
	workFunc := work
	if workFunc == nil {
		return nil, errs.Validation("work function cannot be nil")
	}

	// Execute the tool work
	toolResult, workErr := workFunc()

	// Check if BeforeToolExecution hooks set a modified result
	if hookCtx.ToolResult != nil {
		toolResult = hookCtx.ToolResult
		workErr = nil
	}

	if workErr != nil {
		// OnToolError hook
		hookCtx.ToolError = workErr
		errorResult := p.TriggerHooks(ctx, OnToolError, hookCtx)

		// If hooks provided a fallback result, use it
		if hookCtx.ToolResult != nil {
			p.log.Debug("Tool error recovered by hook",
				zap.String("tool", toolName),
				zap.Error(workErr))
			return hookCtx.ToolResult, nil
		}

		// If error hook had fatal error, return that
		if errorResult.Error != nil {
			return nil, errorResult.Error
		}

		// Otherwise return the original tool error
		return nil, workErr
	}

	// AfterToolExecution hook
	if toolResult == nil {
		toolResult = make(map[string]any)
	}

	hookCtx.ToolResult = toolResult
	hookCtx.ToolError = nil

	afterResult := p.TriggerHooks(ctx, AfterToolExecution, hookCtx)
	if afterResult.Stopped && afterResult.Error != nil {
		return nil, afterResult.Error
	}

	// Return potentially modified result from hooks
	if hookCtx.ToolResult != nil {
		return hookCtx.ToolResult, nil
	}

	return toolResult, nil
}
