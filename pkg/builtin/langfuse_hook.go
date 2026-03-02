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

// cleanupAllTraceContexts removes all trace contexts from memory.
// This is called during shutdown to release resources.
func (h *LangfuseHook) cleanupAllTraceContexts() {
	h.traceCtxsMu.Lock()
	defer h.traceCtxsMu.Unlock()

	count := len(h.traceCtxs)
	if count > 0 {
		h.log.Debug("Cleaning up trace contexts",
			zap.Int("count", count))

		// Clear all trace contexts
		for sessionID, tc := range h.traceCtxs {
			// End any root spans that haven't been ended
			if tc.RootSpan != nil {
				if span, ok := tc.RootSpan.(interface{ End() }); ok {
					span.End()
				}
			}
			delete(h.traceCtxs, sessionID)
		}
	}
}

// Shutdown flushes all buffered traces and cleans up resources.
// This should be called during application shutdown.
// Flush errors are logged as warnings but don't prevent shutdown.
func (h *LangfuseHook) Shutdown() error {
	h.log.Info("Shutting down Langfuse hook")

	// Clean up all trace contexts first
	h.cleanupAllTraceContexts()

	// Flush buffered traces to Langfuse
	flushErr := h.flushTraces()
	if flushErr != nil {
		// Log warning but don't fail - shutdown should complete
		h.log.Warn("Failed to flush Langfuse traces during shutdown (non-fatal)",
			zap.Error(flushErr))
	} else {
		h.log.Info("Langfuse traces flushed successfully")
	}

	return nil
}

// LangfuseHookPriority defines the execution order for Langfuse hooks.
// Priority 500 runs after LoggingHook (1000) but before custom user hooks.
const LangfuseHookPriority = 500


// RegisterLangfuseHooks registers all Langfuse tracing hooks with HookManager.
// It creates and manages trace contexts for session-based tracing.
// Hooks are registered at priority 500 (after logging, before custom hooks).
func RegisterLangfuseHooks(hm hooks.HookManager, hook *LangfuseHook) error {
	// Check if Langfuse tracing is enabled
	if !hook.config.LangfuseEnabled {
		return nil // Skip registration if tracing is disabled
	}

	priority := LangfuseHookPriority
	fatalError := false // Tracing failures should not stop execution

	// Helper to create TypedHookMetadata
	meta := func(name string, point hooks.HookPoint) hooks.TypedHookMetadata {
		return hooks.TypedHookMetadata{
			Name:       name,
			Point:      point,
			Priority:   priority,
			FatalError: fatalError,
		}
	}

	// Helper to register hook and log errors (non-fatal)
	registerTool := func(fn hooks.TypedHookFunc[hooks.ToolPayload], name string, point hooks.HookPoint) {
		if err := hm.RegisterToolHook(fn, meta(name, point)); err != nil {
			hook.log.Warn("Failed to register Langfuse hook",
				zap.String("hook_name", name),
				zap.String("hook_point", string(point)),
				zap.Error(err))
		}
	}

	registerLLM := func(fn hooks.TypedHookFunc[hooks.LLMPayload], name string, point hooks.HookPoint) {
		if err := hm.RegisterLLMHook(fn, meta(name, point)); err != nil {
			hook.log.Warn("Failed to register Langfuse hook",
				zap.String("hook_name", name),
				zap.String("hook_point", string(point)),
				zap.Error(err))
		}
	}

	registerFile := func(fn hooks.TypedHookFunc[hooks.FilePayload], name string, point hooks.HookPoint) {
		if err := hm.RegisterFileHook(fn, meta(name, point)); err != nil {
			hook.log.Warn("Failed to register Langfuse hook",
				zap.String("hook_name", name),
				zap.String("hook_point", string(point)),
				zap.Error(err))
		}
	}

	registerSession := func(fn hooks.TypedHookFunc[hooks.SessionPayload], name string, point hooks.HookPoint) {
		if err := hm.RegisterSessionHook(fn, meta(name, point)); err != nil {
			hook.log.Warn("Failed to register Langfuse hook",
				zap.String("hook_name", name),
				zap.String("hook_point", string(point)),
				zap.Error(err))
		}
	}

	registerAgent := func(fn hooks.TypedHookFunc[hooks.AgentPayload], name string, point hooks.HookPoint) {
		if err := hm.RegisterAgentHook(fn, meta(name, point)); err != nil {
			hook.log.Warn("Failed to register Langfuse hook",
				zap.String("hook_name", name),
				zap.String("hook_point", string(point)),
				zap.Error(err))
		}
	}

	// Session lifecycle hooks
	registerSession(hook.beforeSessionStartHook, "langfuse-before-session-start", hooks.BeforeSessionStart)
	registerSession(hook.afterSessionEndHook, "langfuse-after-session-end", hooks.AfterSessionEnd)

	// Agent lifecycle hooks
	registerAgent(hook.beforeAgentSpawnHook, "langfuse-before-agent-spawn", hooks.BeforeAgentSpawn)
	registerAgent(hook.afterAgentSpawnHook, "langfuse-after-agent-spawn", hooks.AfterAgentSpawn)
	registerAgent(hook.beforeAgentRemoveHook, "langfuse-before-agent-remove", hooks.BeforeAgentRemove)
	registerAgent(hook.afterAgentRemoveHook, "langfuse-after-agent-remove", hooks.AfterAgentRemove)

	// Tool execution hooks
	registerTool(hook.beforeToolExecutionHook, "langfuse-before-tool-execution", hooks.BeforeToolExecution)
	registerTool(hook.afterToolExecutionHook, "langfuse-after-tool-execution", hooks.AfterToolExecution)
	registerTool(hook.onToolErrorHook, "langfuse-on-tool-error", hooks.OnToolError)

	// File operation hooks
	registerFile(hook.beforeFileReadHook, "langfuse-before-file-read", hooks.BeforeFileRead)
	registerFile(hook.afterFileReadHook, "langfuse-after-file-read", hooks.AfterFileRead)
	registerFile(hook.beforeFileWriteHook, "langfuse-before-file-write", hooks.BeforeFileWrite)
	registerFile(hook.afterFileWriteHook, "langfuse-after-file-write", hooks.AfterFileWrite)
	registerFile(hook.beforeFileDeleteHook, "langfuse-before-file-delete", hooks.BeforeFileDelete)
	registerFile(hook.afterFileDeleteHook, "langfuse-after-file-delete", hooks.AfterFileDelete)
	registerFile(hook.beforeFileModifyHook, "langfuse-before-file-modify", hooks.BeforeFileModify)
	registerFile(hook.afterFileModifyHook, "langfuse-after-file-modify", hooks.AfterFileModify)

	// LLM hooks
	registerLLM(hook.beforeLLMRequestHook, "langfuse-before-llm-request", hooks.BeforeLLMRequest)
	registerLLM(hook.afterLLMResponseHook, "langfuse-after-llm-response", hooks.AfterLLMResponse)
	registerLLM(hook.onLLMErrorHook, "langfuse-on-llm-error", hooks.OnLLMError)

	return nil
}

// Session lifecycle hook methods - creates actual Langfuse SDK trace
func (h *LangfuseHook) beforeSessionStartHook(ctx context.Context, hookCtx *hooks.TypedHookContext[hooks.SessionPayload], next func() error) error {
	// Only create trace if Langfuse is enabled
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		return next()
	}

	// Get Langfuse client (lazy init)
	client, err := h.getClient()
	if err != nil {
		// Client not configured - log debug and continue without tracing
		h.log.Debug("Langfuse client not available, skipping session trace creation", zap.Error(err))
		return next()
	}

	// Create trace name from session ID
	traceName := "session-" + hookCtx.SessionID.String()[:8]

	// Create actual Langfuse trace using SDK
	trace := client.StartTrace(ctx, traceName)

	// Create root span for this session
	rootSpan := trace.StartSpan("session")

	// Create trace context with actual SDK objects
	h.traceCtxsMu.Lock()
	tc := &TraceContext{
		TraceID:   trace.ID,      // Use actual trace ID from SDK
		RootSpan:  rootSpan,      // Store actual SDK span
		Spans:     make(map[string]interface{}),
		SessionID: hookCtx.SessionID,
		CreatedAt: time.Now(),
	}
	h.traceCtxs[hookCtx.SessionID] = tc
	h.traceCtxsMu.Unlock()

	// Propagate trace ID to TypedHookContext.Tracing for child spans
	hookCtx.Tracing.TraceID = tc.TraceID

	h.log.Info("Langfuse session trace created",
		zap.String("trace_id", tc.TraceID),
		zap.String("session_id", hookCtx.SessionID.String()))

	return next()
}

func (h *LangfuseHook) afterSessionEndHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.SessionPayload], next func() error) error {
	// Call next first to let session cleanup complete
	err := next()
	if err != nil {
		return err
	}

	// Only flush if Langfuse is enabled
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		return nil
	}

	// Get trace context
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		return nil
	}

	// End root span if it exists and has End method
	if tc.RootSpan != nil {
		// Type assert to span interface that has End() method
		if span, ok := tc.RootSpan.(interface{ End() }); ok {
			span.End()
			h.log.Debug("Root span ended",
				zap.String("trace_id", tc.TraceID),
				zap.String("session_id", hookCtx.SessionID.String()))
		}
	}

	// Flush traces to Langfuse backend
	flushErr := h.flushTraces()
	if flushErr != nil {
		// Log warning but don't fail - flush errors are non-fatal
		h.log.Warn("Failed to flush Langfuse traces (non-fatal)",
			zap.String("session_id", hookCtx.SessionID.String()),
			zap.Error(flushErr))
	}

	// Remove trace context after flush
	h.removeTraceContext(hookCtx.SessionID)

	h.log.Info("Langfuse session trace completed",
		zap.String("trace_id", tc.TraceID),
		zap.String("session_id", hookCtx.SessionID.String()))

	return nil
}

// flushTraces flushes buffered traces to Langfuse backend.
// Returns error if flush fails, but caller should treat as non-fatal.
func (h *LangfuseHook) flushTraces() error {
	h.clientMu.Lock()
	defer h.clientMu.Unlock()

	if h.client == nil {
		return nil // No client, nothing to flush
	}

	// Langfuse SDK Flush() is void, so we assume success
	h.client.Flush()
	return nil
}

// propagateTracingToContext sets trace ID in TypedHookContext.Tracing for span correlation.
func (h *LangfuseHook) propagateTracingToContext(sessionID uuid.UUID, tracing *hooks.TracingPayload) {
	if sessionID != uuid.Nil && tracing != nil {
		tc := h.getTraceContext(sessionID)
		if tc != nil {
			tracing.TraceID = tc.TraceID
		}
	}
}

// Agent lifecycle hook methods (span creation in Phase 9)
func (h *LangfuseHook) beforeAgentSpawnHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.AgentPayload], next func() error) error {
	// Only create spans if Langfuse is enabled and client is available
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return next()
	}

	// Get Langfuse client (lazy init)
	_, err := h.getClient()
	if err != nil {
		// Client not configured - log debug and continue
		h.log.Debug("Langfuse client not available, skipping agent span creation", zap.Error(err))
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return next()
	}

	// Get or create trace context for this session
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		// No trace context exists - create one for ad-hoc tracing
		tc = h.createTraceContext(hookCtx.SessionID)
	}

	// Generate span ID and store in Tracing for correlation
	spanID := uuid.New().String()
	hookCtx.Tracing.SpanID = spanID
	hookCtx.Tracing.StartTime = time.Now()

	// Store incomplete span in TraceContext
	h.traceCtxsMu.Lock()
	if tc.Spans == nil {
		tc.Spans = make(map[string]interface{})
	}
	// Create placeholder span object
	tc.Spans[spanID] = &AgentSpanContext{
		StartTime:     time.Now(),
		EventType:     "spawn",
		ParentAgentID: hookCtx.AgentID.String(),
	}
	h.traceCtxsMu.Unlock()

	h.log.Debug("Agent spawn span created",
		zap.String("span_id", spanID),
		zap.String("parent_agent_id", hookCtx.AgentID.String()))

	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

func (h *LangfuseHook) afterAgentSpawnHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.AgentPayload], next func() error) error {
	// Call next first to let the spawn complete
	err := next()
	if err != nil {
		return err
	}

	// Only update spans if Langfuse is enabled
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get span ID from Tracing
	spanID := hookCtx.Tracing.SpanID
	if spanID == "" {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get trace context
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	h.traceCtxsMu.Lock()
	// Get span from TraceContext
	spanCtx, ok := tc.Spans[spanID].(*AgentSpanContext)
	if !ok {
		h.traceCtxsMu.Unlock()
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Update span with new agent ID from payload
	spanCtx.NewAgentID = hookCtx.Payload.NewAgentID.String()

	// Calculate latency
	latency := time.Since(spanCtx.StartTime)

	// Set completion status
	spanCtx.Level = traces.ObservationLevelDefault
	spanCtx.StatusMessage = "success"
	h.traceCtxsMu.Unlock()

	// Log span completion
	h.log.Debug("Agent spawn span completed",
		zap.String("span_id", spanID),
		zap.String("parent_agent_id", spanCtx.ParentAgentID),
		zap.String("new_agent_id", spanCtx.NewAgentID),
		zap.Duration("latency", latency))

	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return nil
}

func (h *LangfuseHook) beforeAgentRemoveHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.AgentPayload], next func() error) error {
	// Only create spans if Langfuse is enabled and client is available
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return next()
	}

	// Get Langfuse client (lazy init)
	_, err := h.getClient()
	if err != nil {
		h.log.Debug("Langfuse client not available, skipping agent removal span creation", zap.Error(err))
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return next()
	}

	// Get or create trace context for this session
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		tc = h.createTraceContext(hookCtx.SessionID)
	}

	// Generate span ID and store in Tracing for correlation
	spanID := uuid.New().String()
	hookCtx.Tracing.SpanID = spanID
	hookCtx.Tracing.StartTime = time.Now()

	// Store incomplete span in TraceContext
	h.traceCtxsMu.Lock()
	if tc.Spans == nil {
		tc.Spans = make(map[string]interface{})
	}
	tc.Spans[spanID] = &AgentSpanContext{
		StartTime: time.Now(),
		EventType: "remove",
		AgentID:   hookCtx.AgentID.String(),
	}
	h.traceCtxsMu.Unlock()

	h.log.Debug("Agent remove span created",
		zap.String("span_id", spanID),
		zap.String("agent_id", hookCtx.AgentID.String()))

	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

func (h *LangfuseHook) afterAgentRemoveHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.AgentPayload], next func() error) error {
	// Call next first to let the removal complete
	err := next()
	if err != nil {
		return err
	}

	// Only update spans if Langfuse is enabled
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get span ID from Tracing
	spanID := hookCtx.Tracing.SpanID
	if spanID == "" {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get trace context
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	h.traceCtxsMu.Lock()
	// Get span from TraceContext
	spanCtx, ok := tc.Spans[spanID].(*AgentSpanContext)
	if !ok {
		h.traceCtxsMu.Unlock()
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Calculate latency
	latency := time.Since(spanCtx.StartTime)

	// Set completion status
	spanCtx.Level = traces.ObservationLevelDefault
	spanCtx.StatusMessage = "success"
	h.traceCtxsMu.Unlock()

	// Log span completion
	h.log.Debug("Agent remove span completed",
		zap.String("span_id", spanID),
		zap.String("agent_id", spanCtx.AgentID),
		zap.Duration("latency", latency))

	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return nil
}

// Tool execution hook methods
func (h *LangfuseHook) beforeToolExecutionHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.ToolPayload], next func() error) error {
	// Only create spans if Langfuse is enabled and client is available
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return next()
	}

	// Get Langfuse client (lazy init)
	_, err := h.getClient()
	if err != nil {
		// Client not configured - log debug and continue
		h.log.Debug("Langfuse client not available, skipping tool span creation", zap.Error(err))
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return next()
	}

	// Get or create trace context for this session
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		// No trace context exists - create one for ad-hoc tracing
		tc = h.createTraceContext(hookCtx.SessionID)
	}

	// Generate span ID and store in Tracing for correlation
	spanID := uuid.New().String()
	hookCtx.Tracing.SpanID = spanID
	hookCtx.Tracing.StartTime = time.Now()

	// Store incomplete span in TraceContext
	h.traceCtxsMu.Lock()
	if tc.Spans == nil {
		tc.Spans = make(map[string]interface{})
	}
	// Create placeholder span object - access tool data via Payload
	tc.Spans[spanID] = &ToolSpanContext{
		StartTime: time.Now(),
		ToolName:  hookCtx.Payload.Name,
		Input:     hookCtx.Payload.Args,
	}
	h.traceCtxsMu.Unlock()

	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

func (h *LangfuseHook) afterToolExecutionHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.ToolPayload], next func() error) error {
	// Call next first to get the actual tool result
	err := next()
	if err != nil {
		return err
	}

	// Only update spans if Langfuse is enabled
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get span ID from Tracing
	spanID := hookCtx.Tracing.SpanID
	if spanID == "" {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get trace context
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	h.traceCtxsMu.Lock()
	// Get span from TraceContext
	spanCtx, ok := tc.Spans[spanID].(*ToolSpanContext)
	if !ok {
		h.traceCtxsMu.Unlock()
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Update span with result data - access via Payload
	spanCtx.Output = hookCtx.Payload.Result

	// Calculate latency
	latency := time.Since(spanCtx.StartTime)

	// Set completion status
	spanCtx.Level = traces.ObservationLevelDefault
	spanCtx.StatusMessage = "success"
	h.traceCtxsMu.Unlock()

	// Log span completion
	h.log.Debug("Tool span completed",
		zap.String("span_id", spanID),
		zap.String("tool_name", spanCtx.ToolName),
		zap.Duration("latency", latency))

	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return nil
}

func (h *LangfuseHook) onToolErrorHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.ToolPayload], next func() error) error {
	// Call next first to ensure error chain continues
	err := next()
	if err != nil {
		return err
	}

	// Only update spans if Langfuse is enabled
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get span ID from Tracing
	spanID := hookCtx.Tracing.SpanID
	if spanID == "" {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get trace context
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	h.traceCtxsMu.Lock()
	// Get span from TraceContext
	spanCtx, ok := tc.Spans[spanID].(*ToolSpanContext)
	if !ok {
		h.traceCtxsMu.Unlock()
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Mark span as failed
	spanCtx.Level = traces.ObservationLevelError

	// Set error message from Payload.Error
	if hookCtx.Payload.Error != nil {
		spanCtx.StatusMessage = hookCtx.Payload.Error.Error()
	} else {
		spanCtx.StatusMessage = "unknown error"
	}

	// Calculate latency (time from start to error)
	latency := time.Since(spanCtx.StartTime)

	// Log span error
	h.log.Debug("Tool span failed",
		zap.String("span_id", spanID),
		zap.String("tool_name", spanCtx.ToolName),
		zap.Duration("latency", latency),
		zap.Error(hookCtx.Payload.Error))
	h.traceCtxsMu.Unlock()

	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return nil
}

// File operation hook methods
func (h *LangfuseHook) beforeFileReadHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

func (h *LangfuseHook) afterFileReadHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

func (h *LangfuseHook) beforeFileWriteHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

func (h *LangfuseHook) afterFileWriteHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

func (h *LangfuseHook) beforeFileDeleteHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

func (h *LangfuseHook) afterFileDeleteHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

func (h *LangfuseHook) beforeFileModifyHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

func (h *LangfuseHook) afterFileModifyHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.FilePayload], next func() error) error {
	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

// LLMSpanContext holds LLM span data for correlation between before/after hooks.
type LLMSpanContext struct {
	StartTime     time.Time
	Model         string
	Input         string
	Output        string
	Usage         *traces.Usage
	Level         traces.ObservationLevel
	StatusMessage string
}

// ToolSpanContext holds tool span data for correlation between before/after/error hooks.
type ToolSpanContext struct {
	StartTime     time.Time
	ToolName      string
	Input         map[string]any
	Output        map[string]any
	Level         traces.ObservationLevel
	StatusMessage string
}

// AgentSpanContext holds agent span data for correlation between before/after hooks.
type AgentSpanContext struct {
	StartTime     time.Time
	EventType     string // "spawn" or "remove"
	ParentAgentID string // For spawn events, the parent agent ID
	AgentID       string // The agent being affected
	NewAgentID    string // For spawn events, the new child agent ID (set in after hook)
	Level         traces.ObservationLevel
	StatusMessage string
}

// LLM hook methods
func (h *LangfuseHook) beforeLLMRequestHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.LLMPayload], next func() error) error {
	// Only create spans if Langfuse is enabled and client is available
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return next()
	}

	// Get Langfuse client (lazy init)
	_, err := h.getClient()
	if err != nil {
		// Client not configured - log debug and continue
		h.log.Debug("Langfuse client not available, skipping LLM span creation", zap.Error(err))
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return next()
	}

	// Get or create trace context for this session
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		// No trace context exists - create one for ad-hoc tracing
		tc = h.createTraceContext(hookCtx.SessionID)
	}

	// Generate span ID and store in Tracing for correlation
	spanID := uuid.New().String()
	hookCtx.Tracing.SpanID = spanID
	hookCtx.Tracing.StartTime = time.Now()

	// Store incomplete span in TraceContext
	h.traceCtxsMu.Lock()
	if tc.Spans == nil {
		tc.Spans = make(map[string]interface{})
	}
	// Create placeholder span object - access LLM data via Payload
	tc.Spans[spanID] = &LLMSpanContext{
		StartTime: time.Now(),
		Model:     hookCtx.Payload.Model,
		Input:     hookCtx.Payload.Input,
	}
	h.traceCtxsMu.Unlock()

	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return next()
}

func (h *LangfuseHook) afterLLMResponseHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.LLMPayload], next func() error) error {
	// Call next first to get the actual LLM response
	err := next()
	if err != nil {
		return err
	}

	// Only update spans if Langfuse is enabled
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get span ID from Tracing
	spanID := hookCtx.Tracing.SpanID
	if spanID == "" {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get trace context
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	h.traceCtxsMu.Lock()
	// Get span from TraceContext
	spanCtx, ok := tc.Spans[spanID].(*LLMSpanContext)
	if !ok {
		h.traceCtxsMu.Unlock()
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Update span with response data - access via Payload
	spanCtx.Output = hookCtx.Payload.Response

	// Calculate latency
	latency := time.Since(spanCtx.StartTime)

	// Create Usage struct (token counts not available yet)
	spanCtx.Usage = &traces.Usage{
		Input:  0, // Will be populated from actual LLM response
		Output: 0,
		Total:  0,
		Unit:   "TOKENS",
	}

	// Set completion status
	spanCtx.Level = traces.ObservationLevelDefault
	spanCtx.StatusMessage = "success"
	h.traceCtxsMu.Unlock()

	// Log span completion
	h.log.Debug("LLM span completed",
		zap.String("span_id", spanID),
		zap.String("model", spanCtx.Model),
		zap.Duration("latency", latency),
		zap.Int("input_length", len(spanCtx.Input)),
		zap.Int("output_length", len(spanCtx.Output)))

	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return nil
}

func (h *LangfuseHook) onLLMErrorHook(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.LLMPayload], next func() error) error {
	// Call next first to ensure error chain continues
	err := next()
	if err != nil {
		return err
	}

	// Only update spans if Langfuse is enabled
	if !h.config.LangfuseEnabled || hookCtx.SessionID == uuid.Nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get span ID from Tracing
	spanID := hookCtx.Tracing.SpanID
	if spanID == "" {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Get trace context
	tc := h.getTraceContext(hookCtx.SessionID)
	if tc == nil {
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	h.traceCtxsMu.Lock()
	// Get span from TraceContext
	spanCtx, ok := tc.Spans[spanID].(*LLMSpanContext)
	if !ok {
		h.traceCtxsMu.Unlock()
		h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
		return nil
	}

	// Mark span as failed
	spanCtx.Level = traces.ObservationLevelError

	// Set error message from Payload.Error
	if hookCtx.Payload.Error != nil {
		spanCtx.StatusMessage = hookCtx.Payload.Error.Error()
	} else {
		spanCtx.StatusMessage = "unknown error"
	}

	// Preserve partial response if available
	if hookCtx.Payload.Response != "" {
		spanCtx.Output = hookCtx.Payload.Response
	}

	// Calculate latency (time from start to error)
	latency := time.Since(spanCtx.StartTime)

	// Log span error
	h.log.Debug("LLM span failed",
		zap.String("span_id", spanID),
		zap.String("model", spanCtx.Model),
		zap.Duration("latency", latency),
		zap.Error(hookCtx.Payload.Error))
	h.traceCtxsMu.Unlock()

	h.propagateTracingToContext(hookCtx.SessionID, &hookCtx.Tracing)
	return nil
}
