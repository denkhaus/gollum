package strategy

import (
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/strategy/react"
	"github.com/samber/do/v2"
)

// builderImpl implements the Builder interface
type builderImpl struct {
	configService config.ConfigService
}

// NewBuilder creates a new Builder as a DI service
func NewBuilder(injector do.Injector) (Builder, error) {
	configService := do.MustInvoke[config.ConfigService](injector)
	return &builderImpl{
		configService: configService,
	}, nil
}

// BuildReact creates a react strategy with the specified configuration
func (b *builderImpl) BuildReact(cfg *config.StrategyConfig, client gollem.LLMClient) gollem.Strategy {
	if cfg == nil {
		// Use react's defaults if config is nil
		return react.New(client)
	}

	// Build options from config
	var opts []react.Option

	// Only add options if values are non-zero to respect react defaults
	if cfg.MaxIterations > 0 {
		opts = append(opts, react.WithMaxIterations(cfg.MaxIterations))
	}

	if cfg.MaxRepeatedActions > 0 {
		opts = append(opts, react.WithMaxRepeatedActions(cfg.MaxRepeatedActions))
	}

	return react.New(client, opts...)
}

// BuildDefaultReact creates react strategy with default configuration from config service
func (b *builderImpl) BuildDefaultReact(client gollem.LLMClient) gollem.Strategy {
	subAgentCfg := b.configService.GetSubAgentConfig()
	return b.BuildReact(&subAgentCfg.Strategy, client)
}
