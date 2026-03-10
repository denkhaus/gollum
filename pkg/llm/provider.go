// Package llm provides LLM client providers for Anthropic, OpenAI, and Gemini APIs.
package llm

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/claude"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/m-mizutani/gollem/llm/openai"
	"github.com/samber/do/v2"
)

type (
	clientProvider struct {
		logService    logger.LoggerService
		configService config.ConfigService
	}

	// ClientProvider provides LLM clients for different providers (Anthropic, OpenAI, Gemini)
	ClientProvider interface {
		GetClient(ctx context.Context, provider shared.LLMProvider) (gollem.LLMClient, error)
	}
)

// NewClientProvider creates a new LLM client provider
func NewClientProvider(injector do.Injector) (ClientProvider, error) {
	configService := do.MustInvoke[config.ConfigService](injector)
	logService := do.MustInvoke[logger.LoggerService](injector)

	prov := &clientProvider{
		configService: configService,
		logService:    logService,
	}

	return prov, nil
}

func (p *clientProvider) GetClient(ctx context.Context, provider shared.LLMProvider) (gollem.LLMClient, error) {
	switch provider {
	case shared.LLMProviderGemini:

		cfg := p.configService.GetGeminiConfig()
		client, err := gemini.New(ctx, cfg.ProjectID, cfg.Location, gemini.WithModel(cfg.Model))
		if err != nil {
			return nil, fmt.Errorf("failed to create gemini client: %v", err)
		}
		return client, nil

	case shared.LLMProviderAnthropic:

		cfg := p.configService.GetAnthropicConfig()
		client, err := claude.New(ctx, cfg.APIKey,
			claude.WithBaseURL(cfg.BaseURL),
			claude.WithModel(cfg.Model),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create antropic client: %v", err)
		}
		return client, nil

	case shared.LLMProviderOpenAI:

		cfg := p.configService.GetOpenAIConfig()
		client, err := openai.New(ctx, cfg.APIKey,
			openai.WithBaseURL(cfg.BaseURL),
			openai.WithModel(cfg.Model),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create openai client: %v", err)
		}
		return client, nil

	default:
		return nil, fmt.Errorf("failed to create llm client from provider %s", provider)
	}
}
