// Package builtin provides production-ready built-in hooks for common use cases.
package builtin

import (
	"fmt"
	"sync"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/git-hulk/langfuse-go"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// LangfuseHook provides Langfuse tracing for LLM, tool, and agent operations.
// It creates observation spans at hook points and flushes traces to Langfuse.
type LangfuseHook struct {
	log      logger.LoggerService
	config   *config.LangfuseConfig
	client   *langfuse.Langfuse
	clientMu *sync.Mutex // Protects lazy client initialization
}

// NewLangfuseHook creates a new LangfuseHook instance.
// The Langfuse client is initialized lazily on first use.
func NewLangfuseHook(injector do.Injector) (*LangfuseHook, error) {
	log := do.MustInvoke[logger.LoggerService](injector)
	cfg := do.MustInvoke[config.ConfigService](injector)

	return &LangfuseHook{
		log:      log,
		config:   cfg.GetLangfuseConfig(),
		client:   nil, // Lazy initialized
		clientMu: &sync.Mutex{},
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
