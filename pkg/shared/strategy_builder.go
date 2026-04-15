package shared

import (
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/strategy/react"
	"github.com/samber/do/v2"
)

// strategyBuilderImpl implements the StrategyBuilder interface
type strategyBuilderImpl struct {
	// No dependencies needed - builder is stateless
}

// NewStrategyBuilder creates a new StrategyBuilder as a DI service
func NewStrategyBuilder(injector do.Injector) (StrategyBuilder, error) {
	return &strategyBuilderImpl{}, nil
}

// BuildReact creates a react strategy with the specified configuration
func (b *strategyBuilderImpl) BuildReact(cfg *StrategyConfig, client gollem.LLMClient) *react.Strategy {
	if cfg == nil {
		// Use defaults if config is nil
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

// BuildDefaultReact creates react strategy with default configuration
func (b *strategyBuilderImpl) BuildDefaultReact(client gollem.LLMClient) *react.Strategy {
	return react.New(client)
}
