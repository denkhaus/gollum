package hooks

import (
	"context"
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

	// RegisterExecutorHook registers a typed hook for flow step execution events.
	// Valid hook points: BeforeFlowStep, AfterFlowStep.
	RegisterExecutorHook(fn TypedHookFunc[ExecutorPayload], meta TypedHookMetadata) error

	// UnregisterHook removes a hook by name from all registries.
	// Returns true if at least one hook was removed.
	UnregisterHook(name string) bool

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

	// TriggerExecutorHooks executes all typed executor hooks for a given hook point.
	// Valid hook points: BeforeFlowStep, AfterFlowStep.
	TriggerExecutorHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[ExecutorPayload]) TypedHookResult[ExecutorPayload]

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

	// WithFlowStepHooks wraps a function with flow step execution hooks.
	// BeforeFlowStep hooks can inspect/validate before execution.
	// AfterFlowStep hooks can log/audit after execution.
	WithFlowStepHooks(
		ctx context.Context,
		sessionID, flowID uuid.UUID,
		stepType, stateName string,
		work func() (map[string]any, error),
	) (map[string]any, error)
}

// hookManagerImpl is the private implementation of HookManager.
type hookManagerImpl struct {
	log   logger.LoggerService
	mu    sync.RWMutex        // Protects the names map
	names map[string]struct{} // Global name registry for uniqueness across all hook points

	// Typed registries for type-safe hook storage
	toolRegistry     *TypedRegistry[ToolPayload]
	llmRegistry      *TypedRegistry[LLMPayload]
	fileRegistry     *TypedRegistry[FilePayload]
	sessionRegistry  *TypedRegistry[SessionPayload]
	agentRegistry    *TypedRegistry[AgentPayload]
	skillRegistry    *TypedRegistry[SkillPayload]
	executorRegistry *TypedRegistry[ExecutorPayload]
}

// Ensure hookManagerImpl implements HookManager.
var _ HookManager = (*hookManagerImpl)(nil)

// NewHookManager creates a new HookManager service.
func NewHookManager(injector do.Injector) (HookManager, error) {
	log := do.MustInvoke[logger.LoggerService](injector)

	log.Debug("HookManager starting")

	p := &hookManagerImpl{
		log:   log,
		names: make(map[string]struct{}),
	}

	// Initialize typed registries for type-safe hook storage
	p.toolRegistry = NewTypedRegistry[ToolPayload]()
	p.llmRegistry = NewTypedRegistry[LLMPayload]()
	p.fileRegistry = NewTypedRegistry[FilePayload]()
	p.sessionRegistry = NewTypedRegistry[SessionPayload]()
	p.agentRegistry = NewTypedRegistry[AgentPayload]()
	p.skillRegistry = NewTypedRegistry[SkillPayload]()
	p.executorRegistry = NewTypedRegistry[ExecutorPayload]()

	return p, nil
}

// UnregisterHook removes a hook by name from all typed registries.
// Returns true if at least one hook was removed.
func (p *hookManagerImpl) UnregisterHook(name string) bool {
	unregistered := false

	// Check typed registries
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
	case p.executorRegistry.Remove(name):
		p.log.Debug("Hook unregistered from executor registry", zap.String("name", name))
		unregistered = true
	}

	// Remove from global names map if hook was found and removed
	if unregistered {
		p.mu.Lock()
		delete(p.names, name)
		p.mu.Unlock()
	}

	return unregistered
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

// RegisterExecutorHook registers a typed hook for flow step execution events.
func (p *hookManagerImpl) RegisterExecutorHook(fn TypedHookFunc[ExecutorPayload], meta TypedHookMetadata) error {
	if fn == nil {
		return errs.Validation("hook function cannot be nil")
	}
	if meta.Name == "" {
		return errs.Validation("hook name cannot be empty")
	}
	if err := p.validateExecutorHookPoint(meta.Point); err != nil {
		return err
	}

	// Check for global name uniqueness
	p.mu.Lock()
	if _, exists := p.names[meta.Name]; exists {
		p.mu.Unlock()
		return errs.Conflictf("hook with name '%s' is already registered globally", meta.Name)
	}

	if err := p.executorRegistry.Add(fn, meta); err != nil {
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

// TriggerExecutorHooks executes all typed executor hooks for a given hook point.
func (p *hookManagerImpl) TriggerExecutorHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[ExecutorPayload]) TypedHookResult[ExecutorPayload] {
	return p.executorRegistry.Trigger(ctx, point, hookCtx)
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

func (p *hookManagerImpl) validateExecutorHookPoint(point HookPoint) error {
	switch point {
	case BeforeFlowStep, AfterFlowStep:
		return nil
	default:
		return errs.Validationf("invalid executor hook point: %s (expected BeforeFlowStep or AfterFlowStep)", point)
	}
}
