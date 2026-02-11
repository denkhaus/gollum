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
	"github.com/git-hulk/langfuse-go"
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
