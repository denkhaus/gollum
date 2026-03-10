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
		GetClient(ctx context.Context, cnf *shared.LLMClientConfig) (gollem.LLMClient, error)
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

func (p *clientProvider) GetClient(ctx context.Context, cnf *shared.LLMClientConfig) (gollem.LLMClient, error) {
	// Parse provider from model string (e.g., "anthropic/claude-3-5-sonnet-20241022")
	provider, err := cnf.Provider()
	if err != nil {
		return nil, err
	}

	// Parse model name from model string
	modelName, err := cnf.ModelName()
	if err != nil {
		return nil, err
	}

	switch provider {
	case shared.LLMProviderGemini:
		cfg := p.configService.GetGeminiConfig()
		if cfg.ProjectID == "" || cfg.Location == "" {
			return nil, fmt.Errorf("gemini project_id and location must be configured")
		}

		// Use config values as defaults, override with cnf values if provided
		temp := float32(cfg.Temperature)
		if cnf.Temperature != nil {
			temp = float32(*cnf.Temperature)
		}

		maxTokens := int32(cfg.MaxTokens)
		if cnf.MaxTokens != nil {
			maxTokens = int32(*cnf.MaxTokens)
		}

		topP := float32(cfg.TopP)
		if cnf.TopP != nil {
			topP = float32(*cnf.TopP)
		}

		client, err := gemini.New(ctx, cfg.ProjectID, cfg.Location,
			gemini.WithModel(modelName),
			gemini.WithTemperature(temp),
			gemini.WithMaxTokens(maxTokens),
			gemini.WithTopP(topP),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create gemini client: %v", err)
		}
		return client, nil

	case shared.LLMProviderAnthropic:
		cfg := p.configService.GetAnthropicConfig()
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("anthropic api_key must be configured")
		}

		// Use config values as defaults, override with cnf values if provided
		temp := cfg.Temperature
		if cnf.Temperature != nil {
			temp = *cnf.Temperature
		}

		maxTokens := int64(cfg.MaxTokens)
		if cnf.MaxTokens != nil {
			maxTokens = int64(*cnf.MaxTokens)
		}

		topP := cfg.TopP
		if cnf.TopP != nil {
			topP = *cnf.TopP
		}

		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}

		client, err := claude.New(ctx, cfg.APIKey,
			claude.WithTemperature(temp),
			claude.WithMaxTokens(maxTokens),
			claude.WithTopP(topP),
			claude.WithBaseURL(baseURL),
			claude.WithModel(modelName),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create anthropic client: %v", err)
		}
		return client, nil

	case shared.LLMProviderOpenAI:
		cfg := p.configService.GetOpenAIConfig()
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("openai api_key must be configured")
		}

		// Use config values as defaults, override with cnf values if provided
		temp := float32(cfg.Temperature)
		if cnf.Temperature != nil {
			temp = float32(*cnf.Temperature)
		}

		maxTokens := cfg.MaxTokens
		if cnf.MaxTokens != nil {
			maxTokens = *cnf.MaxTokens
		}

		topP := float32(cfg.TopP)
		if cnf.TopP != nil {
			topP = float32(*cnf.TopP)
		}

		client, err := openai.New(ctx, cfg.APIKey,
			openai.WithBaseURL(cfg.BaseURL),
			openai.WithModel(modelName),
			openai.WithMaxTokens(maxTokens),
			openai.WithTemperature(temp),
			openai.WithTopP(topP),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create openai client: %v", err)
		}
		return client, nil

	default:
		return nil, fmt.Errorf("failed to create llm client from provider %s", provider)
	}
}
