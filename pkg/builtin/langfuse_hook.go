// Package builtin provides production-ready built-in hooks for common use cases.
package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	langfuse "github.com/git-hulk/langfuse-go"
	"github.com/git-hulk/langfuse-go/pkg/traces"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// LangfuseHook provides Langfuse tracing for LLM, tool, and agent operations.
// It creates observation spans at hook points and flushes traces to Langfuse.
type LangfuseHook struct {
	log        logger.LoggerService
	config     *config.LangfuseConfig
	client     *langfuse.Langfuse
	clientMu   *sync.Mutex // Protects lazy client initialization
	traceCtxs  map[uuid.UUID]*TraceContext
	traceCtxsMu *sync.RWMutex // Protects traceCtxs map for concurrent access
}

// TraceContext holds Langfuse trace state for a single session.
// It manages spans hierarchy and trace lifecycle.
type TraceContext struct {
	// TraceID is the unique identifier for this trace in Langfuse.
	TraceID string
	// RootSpan is the top-level span for this trace.
	RootSpan interface{}
	// Spans holds all child spans created during this session.
	// Keys are span IDs for lookup during trace assembly.
	Spans map[string]interface{}
	// SessionID is the Gollum session identifier.
	SessionID uuid.UUID
	// CreatedAt is when this trace context was created.
	CreatedAt time.Time
}

// NewLangfuseHook creates a new LangfuseHook instance.
// The Langfuse client is initialized lazily on first use.
func NewLangfuseHook(injector do.Injector) (*LangfuseHook, error) {
	log := do.MustInvoke[logger.LoggerService](injector)
	cfg := do.MustInvoke[config.ConfigService](injector)

	return &LangfuseHook{
		log:         log,
		config:      cfg.GetLangfuseConfig(),
		client:      nil, // Lazy initialized
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[uuid.UUID]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}, nil
}

// NewLangfuseHookProvider creates a LangfuseHook provider for DI registration.
// This provider function registers LangfuseHook as a singleton in the DI container.
func NewLangfuseHookProvider(injector do.Injector) (hooks.HookFunc, error) {
	_, err := NewLangfuseHook(injector)
	if err != nil {
		return nil, err
	}

	// Return a HookFunc that wraps span creation/ending logic
	// For now, this is a no-op wrapper - actual span handling added in later phases
	return func(ctx context.Context, hookCtx *hooks.HookContext, next func() error) error {
		return next() // Just pass through - spans created in later phases
	}, nil
}

// NewLangfuseHooksProvider creates a LangfuseHook provider for builtin registration.
// This provider function returns the LangfuseHook instance directly (not a HookFunc)
// for use by NewBuiltinHooksProvider to register individual hooks via RegisterLangfuseHooks.
func NewLangfuseHooksProvider(injector do.Injector) (*LangfuseHook, error) {
	hook, err := NewLangfuseHook(injector)
	if err != nil {
		return nil, fmt.Errorf("failed to create Langfuse hook: %w", err)
	}

	// Return the hook instance directly
	// The RegisterLangfuseHooks will be called by NewBuiltinHooksProvider
	return hook, nil
}

// getClient lazily initializes and returns the Langfuse client.
// Returns error if credentials are not configured or client initialization fails.
func (h *LangfuseHook) getClient() (*langfuse.Langfuse, error) {
	h.clientMu.Lock()
	defer h.clientMu.Unlock()

	if h.client != nil {
		return h.client, nil
	}

	// Validate configuration
	if h.config.LangfusePublicKey == "" || h.config.LangfuseSecretKey == "" {
		return nil, fmt.Errorf("Langfuse credentials not configured (set LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY)")
	}

	// Create client with SDK
	client := langfuse.NewClient(
		h.config.LangfuseHost,
		h.config.LangfusePublicKey,
		h.config.LangfuseSecretKey,
	)

	h.client = client
	h.log.Info("Langfuse client initialized",
		zap.String("host", h.config.LangfuseHost))

	return h.client, nil
}

// getTraceContext retrieves the TraceContext for a given session ID.
// Returns nil if no trace context exists for this session.
// Uses RLock to allow concurrent reads.
func (h *LangfuseHook) getTraceContext(sessionID uuid.UUID) *TraceContext {
	h.traceCtxsMu.RLock()
	defer h.traceCtxsMu.RUnlock()

	return h.traceCtxs[sessionID]
}

// createTraceContext creates a new TraceContext for a given session ID.
// Generates a unique TraceID and initializes all fields.
// Uses Lock to prevent concurrent writes.
func (h *LangfuseHook) createTraceContext(sessionID uuid.UUID) *TraceContext {
	h.traceCtxsMu.Lock()
	defer h.traceCtxsMu.Unlock()

	tc := &TraceContext{
		TraceID:   uuid.New().String(), // Generate unique trace ID for Langfuse
		RootSpan:  nil,                  // Will be set in Phase 10 when root span created
		Spans:     make(map[string]interface{}), // Initialize empty spans map
		SessionID: sessionID,
		CreatedAt: time.Now(),
	}

	h.traceCtxs[sessionID] = tc
	return tc
}

// removeTraceContext deletes the TraceContext for a given session ID.
// Uses Lock to prevent concurrent writes.
func (h *LangfuseHook) removeTraceContext(sessionID uuid.UUID) {
	h.traceCtxsMu.Lock()
	defer h.traceCtxsMu.Unlock()

	delete(h.traceCtxs, sessionID)
}

// Shutdown flushes any buffered traces and closes the Langfuse client.
// This should be called during application shutdown or session end.
func (h *LangfuseHook) Shutdown() error {
	h.clientMu.Lock()
	defer h.clientMu.Unlock()

	if h.client != nil {
		h.client.Flush()
		h.log.Info("Langfuse traces flushed successfully")
	}
	return nil
}

// LangfuseHookPriority defines the execution order for Langfuse hooks.
// Priority 500 runs after LoggingHook (1000) but before custom user hooks.
const LangfuseHookPriority = 500

// hookRegisterer is a minimal interface for hook registration.
// This allows RegisterLangfuseHooks to work with any type that implements RegisterHook.
type hookRegisterer interface {
	RegisterHook(fn hooks.HookFunc, meta hooks.HookMetadata) error
}

// RegisterLangfuseHooks registers all Langfuse tracing hooks with HookManager.
// It creates and manages trace contexts for session-based tracing.
// Hooks are registered at priority 500 (after logging, before custom hooks).
func RegisterLangfuseHooks(hm hookRegisterer, hook *LangfuseHook) error {
	// Check if Langfuse tracing is enabled
	if !hook.config.LangfuseEnabled {
		return nil // Skip registration if tracing is disabled
	}

	priority := LangfuseHookPriority
	fatalError := false // Tracing failures should not stop execution

	// Helper to register hook and log errors (non-fatal)
	register := func(fn hooks.HookFunc, meta hooks.HookMetadata) {
		if err := hm.RegisterHook(fn, meta); err != nil {
			// Log but don't fail - tracing hooks are non-critical
			hook.log.Warn("Failed to register Langfuse hook",
				zap.String("hook_name", meta.Name),
				zap.String("hook_point", string(meta.Point)),
				zap.Error(err))
		}
	}

	// Session lifecycle hooks
	register(hook.beforeSessionStartHook,
		hooks.HookMetadata{Name: "langfuse-before-session-start", Point: hooks.BeforeSessionStart, Priority: priority, FatalError: fatalError})
	register(hook.afterSessionEndHook,
		hooks.HookMetadata{Name: "langfuse-after-session-end", Point: hooks.AfterSessionEnd, Priority: priority, FatalError: fatalError})

	// Agent lifecycle hooks
	register(hook.beforeAgentSpawnHook,
		hooks.HookMetadata{Name: "langfuse-before-agent-spawn", Point: hooks.BeforeAgentSpawn, Priority: priority, FatalError: fatalError})
	register(hook.afterAgentSpawnHook,
		hooks.HookMetadata{Name: "langfuse-after-agent-spawn", Point: hooks.AfterAgentSpawn, Priority: priority, FatalError: fatalError})
	register(hook.beforeAgentRemoveHook,
		hooks.HookMetadata{Name: "langfuse-before-agent-remove", Point: hooks.BeforeAgentRemove, Priority: priority, FatalError: fatalError})
	register(hook.afterAgentRemoveHook,
		hooks.HookMetadata{Name: "langfuse-after-agent-remove", Point: hooks.AfterAgentRemove, Priority: priority, FatalError: fatalError})

	// Tool execution hooks
	register(hook.beforeToolExecutionHook,
		hooks.HookMetadata{Name: "langfuse-before-tool-execution", Point: hooks.BeforeToolExecution, Priority: priority, FatalError: fatalError})
	register(hook.afterToolExecutionHook,
		hooks.HookMetadata{Name: "langfuse-after-tool-execution", Point: hooks.AfterToolExecution, Priority: priority, FatalError: fatalError})
	register(hook.onToolErrorHook,
		hooks.HookMetadata{Name: "langfuse-on-tool-error", Point: hooks.OnToolError, Priority: priority, FatalError: fatalError})

	// File operation hooks
	register(hook.beforeFileReadHook,
		hooks.HookMetadata{Name: "langfuse-before-file-read", Point: hooks.BeforeFileRead, Priority: priority, FatalError: fatalError})
	register(hook.afterFileReadHook,
		hooks.HookMetadata{Name: "langfuse-after-file-read", Point: hooks.AfterFileRead, Priority: priority, FatalError: fatalError})
	register(hook.beforeFileWriteHook,
		hooks.HookMetadata{Name: "langfuse-before-file-write", Point: hooks.BeforeFileWrite, Priority: priority, FatalError: fatalError})
	register(hook.afterFileWriteHook,
		hooks.HookMetadata{Name: "langfuse-after-file-write", Point: hooks.AfterFileWrite, Priority: priority, FatalError: fatalError})
	register(hook.beforeFileDeleteHook,
		hooks.HookMetadata{Name: "langfuse-before-file-delete", Point: hooks.BeforeFileDelete, Priority: priority, FatalError: fatalError})
	register(hook.afterFileDeleteHook,
		hooks.HookMetadata{Name: "langfuse-after-file-delete", Point: hooks.AfterFileDelete, Priority: priority, FatalError: fatalError})
	register(hook.beforeFileModifyHook,
		hooks.HookMetadata{Name: "langfuse-before-file-modify", Point: hooks.BeforeFileModify, Priority: priority, FatalError: fatalError})
	register(hook.afterFileModifyHook,
		hooks.HookMetadata{Name: "langfuse-after-file-modify", Point: hooks.AfterFileModify, Priority: priority, FatalError: fatalError})

	// LLM hooks
	register(hook.beforeLLMRequestHook,
		hooks.HookMetadata{Name: "langfuse-before-llm-request", Point: hooks.BeforeLLMRequest, Priority: priority, FatalError: fatalError})
	register(hook.afterLLMResponseHook,
		hooks.HookMetadata{Name: "langfuse-after-llm-response", Point: hooks.AfterLLMResponse, Priority: priority, FatalError: fatalError})
	register(hook.onLLMErrorHook,
		hooks.HookMetadata{Name: "langfuse-on-llm-error", Point: hooks.OnLLMError, Priority: priority, FatalError: fatalError})

	return nil
}

// Session lifecycle hook methods (actual span creation in Phase 10)
func (h *LangfuseHook) beforeSessionStartHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	// Create trace context on session start
	if hookCtx.SessionID != uuid.Nil {
		h.createTraceContext(hookCtx.SessionID)
	}
	return next()
}

func (h *LangfuseHook) afterSessionEndHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	// Remove trace context on session end
	if hookCtx.SessionID != uuid.Nil {
		h.removeTraceContext(hookCtx.SessionID)
	}
	return next()
}

// Agent lifecycle hook methods (span creation in Phase 9)
func (h *LangfuseHook) beforeAgentSpawnHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) afterAgentSpawnHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) beforeAgentRemoveHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) afterAgentRemoveHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

// Tool execution hook methods (span creation in Phase 9)
func (h *LangfuseHook) beforeToolExecutionHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) afterToolExecutionHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) onToolErrorHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

// File operation hook methods (span creation in Phase 9)
func (h *LangfuseHook) beforeFileReadHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) afterFileReadHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) beforeFileWriteHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) afterFileWriteHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) beforeFileDeleteHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) afterFileDeleteHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) beforeFileModifyHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) afterFileModifyHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

// LLMSpanContext holds LLM span data for correlation between before/after hooks.
// This placeholder struct stores span data until actual Langfuse SDK spans are created in Phase 10.
type LLMSpanContext struct {
	StartTime     time.Time
	Model         string
	Input         string
	Output        string
	Usage         *traces.Usage
	Level         traces.ObservationLevel
	StatusMessage string
}

// LLM hook methods (span creation in Phase 8)
func (h *LangfuseHook) beforeLLMRequestHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	// Only create spans if Langfuse is enabled and client is available
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTraceID(hookCtx)
		return next()
	}

	// Get Langfuse client (lazy init)
	_, err := h.getClient()
	if err != nil {
		// Client not configured - log debug and continue
		h.log.Debug("Langfuse client not available, skipping LLM span creation", zap.Error(err))
		h.propagateTraceID(hookCtx)
		return next()
	}

	// Get or create trace context for this session
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		// No trace context exists - create one for ad-hoc tracing
		tc = h.createTraceContext(hookCtx.SessionID)
	}

	// Create LLM Generation span using Langfuse SDK
	// StartGeneration creates a child observation for LLM interactions
	spanName := "llm-" + hookCtx.LLMModel
	if spanName == "llm-" {
		spanName = "llm-request" // Fallback if model is empty
	}

	// Store span metadata in HookContext for AfterLLMResponse to complete
	hookCtx.Data["langfuse_llm_start"] = time.Now()
	hookCtx.Data["langfuse_llm_model"] = hookCtx.LLMModel
	hookCtx.Data["langfuse_llm_input"] = hookCtx.LLMInput

	// For Phase 8, we'll create the actual span when session tracing is ready
	// Store span reference in TraceContext for AfterLLMResponse
	spanID := uuid.New().String()
	hookCtx.Data["langfuse_span_id"] = spanID

	// Store incomplete span in TraceContext
	h.traceCtxsMu.Lock()
	if tc.Spans == nil {
		tc.Spans = make(map[string]interface{})
	}
	// Create placeholder span object (will be replaced with actual SDK span in Phase 10)
	tc.Spans[spanID] = &LLMSpanContext{
		StartTime: time.Now(),
		Model:     hookCtx.LLMModel,
		Input:     hookCtx.LLMInput,
	}
	h.traceCtxsMu.Unlock()

	h.propagateTraceID(hookCtx)
	return next()
}

func (h *LangfuseHook) afterLLMResponseHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	// Call next first to get the actual LLM response
	err := next()
	if err != nil {
		return err
	}

	// Only update spans if Langfuse is enabled
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTraceID(hookCtx)
		return nil
	}

	// Get span ID from HookContext.Data
	spanID, ok := hookCtx.Data["langfuse_span_id"].(string)
	if !ok || spanID == "" {
		h.propagateTraceID(hookCtx)
		return nil
	}

	// Get trace context
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		h.propagateTraceID(hookCtx)
		return nil
	}

	h.traceCtxsMu.Lock()
	// Get span from TraceContext
	spanCtx, ok := tc.Spans[spanID].(*LLMSpanContext)
	if !ok {
		h.traceCtxsMu.Unlock()
		h.propagateTraceID(hookCtx)
		return nil
	}

	// Update span with response data
	spanCtx.Output = hookCtx.LLMResponse

	// Calculate latency
	latency := time.Since(spanCtx.StartTime)
	endTime := spanCtx.StartTime.Add(latency)

	// Create Usage struct (token counts not available in HookContext yet)
	// In Phase 9/10, we'll extract token counts from actual LLM responses
	spanCtx.Usage = &traces.Usage{
		Input:  0, // Will be populated from actual LLM response
		Output: 0, // Will be populated from actual LLM response
		Total:  0,
		Unit:   "TOKENS",
	}

	// Set completion status
	spanCtx.Level = traces.ObservationLevelDefault
	spanCtx.StatusMessage = "success"
	h.traceCtxsMu.Unlock()

	// Log span completion (actual span submission to Langfuse in Phase 10)
	_ = endTime // Used for latency calculation
	h.log.Debug("LLM span completed",
		zap.String("span_id", spanID),
		zap.String("model", spanCtx.Model),
		zap.Duration("latency", latency),
		zap.Int("input_length", len(spanCtx.Input)),
		zap.Int("output_length", len(spanCtx.Output)))

	h.propagateTraceID(hookCtx)
	return nil
}

func (h *LangfuseHook) onLLMErrorHook(_ context.Context, hookCtx *hooks.HookContext, next func() error) error {
	h.propagateTraceID(hookCtx)
	return next()
}

// propagateTraceID sets the langfuse_trace_id in HookContext.Data for span correlation.
func (h *LangfuseHook) propagateTraceID(hookCtx *hooks.HookContext) {
	if hookCtx.SessionID != uuid.Nil && hookCtx.Data != nil {
		tc := h.getTraceContext(hookCtx.SessionID)
		if tc != nil {
			hookCtx.Data["langfuse_trace_id"] = tc.TraceID
		}
	}
}
