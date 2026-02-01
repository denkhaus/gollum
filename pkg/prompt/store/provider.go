// Package store provides DI provider for PromptStore implementations.
package store

import (
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

type (
	// PromptStoreProvider provides PromptStore instances.
	PromptStoreProvider interface {
		GetStore() PromptStore
	}

	promptStoreProvider struct {
		logService    logger.LoggerService
		configService config.ConfigService
		store         PromptStore
	}
)

// NewPromptStoreProvider creates a new PromptStore provider.
func NewPromptStoreProvider(injector do.Injector) (PromptStoreProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	configService := do.MustInvoke[config.ConfigService](injector)

	cfg := configService.GetPromptStoreConfig()

	var s PromptStore
	switch cfg.Type {
	case config.PromptStoreTypeFile:
		s = NewFileStore(cfg.FilePath, cfg.CacheEnabled)
		logService.Info("initialized file store",
			zap.String("type", string(cfg.Type)),
			zap.String("path", cfg.FilePath),
		)
	case config.PromptStoreTypeLangfuse:
		// TODO: Implement Langfuse store
		logService.Warn("langfuse store not yet implemented, falling back to memory store",
			zap.String("host", cfg.LangfuseHost),
		)
		s = NewMemoryStore()
	case config.PromptStoreTypeMemory, "":
		s = NewMemoryStore()
		logService.Info("initialized memory store",
			zap.String("type", string(cfg.Type)),
		)
	default:
		logService.Warn("unknown prompt store type, falling back to memory store",
			zap.String("type", string(cfg.Type)),
		)
		s = NewMemoryStore()
	}

	return &promptStoreProvider{
		logService:    logService,
		configService: configService,
		store:         s,
	}, nil
}

// GetStore returns the PromptStore instance.
func (p *promptStoreProvider) GetStore() PromptStore {
	return p.store
}

// NewPromptStore creates a new PromptStore and returns it directly.
// This provider is used by the prompt manager for DI injection.
func NewPromptStore(injector do.Injector) (PromptStore, error) {
	provider, err := NewPromptStoreProvider(injector)
	if err != nil {
		return nil, err
	}
	return provider.GetStore(), nil
}
