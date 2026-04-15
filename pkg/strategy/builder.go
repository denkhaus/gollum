package strategy

import (
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/strategy/react"
	"github.com/m-mizutani/gollem/strategy/simple"
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

// BuildForSupervisor creates a strategy for supervisor agents
func (b *builderImpl) BuildForSupervisor(client gollem.LLMClient, strategyType StrategyType) gollem.Strategy {
	supervisorCfg := b.configService.GetSupervisorConfig()

	// Handle both "react" and "default" (which maps to react)
	if strategyType == StrategyTypeSimple {
		return b.BuildSimple(client)
	}
	// Default to react for everything else (StrategyTypeReact, StrategyTypeDefault, unknown)
	return b.BuildReact(&supervisorCfg.Strategy, client)
}

// BuildForSubAgent creates a strategy for subagent creation
func (b *builderImpl) BuildForSubAgent(client gollem.LLMClient, strategyType StrategyType) gollem.Strategy {
	subAgentCfg := b.configService.GetSubAgentConfig()

	// Handle both "react" and "default" (which maps to react)
	if strategyType == StrategyTypeSimple {
		return b.BuildSimple(client)
	}
	// Default to react for everything else (StrategyTypeReact, StrategyTypeDefault, unknown)
	return b.BuildReact(&subAgentCfg.Strategy, client)
}

// BuildForLLMStep creates a strategy for flow LLM steps
// Uses subagent defaults for LLM steps
func (b *builderImpl) BuildForLLMStep(client gollem.LLMClient, strategyType StrategyType) gollem.Strategy {
	return b.BuildForSubAgent(client, strategyType)
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

// BuildSimple creates a simple strategy
func (b *builderImpl) BuildSimple(client gollem.LLMClient) gollem.Strategy {
	return simple.New()
}
