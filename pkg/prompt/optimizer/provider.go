// Package optimizer provides DI provider functions for the prompt optimizer
package optimizer

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/llm"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/samber/do/v2"
)

// NewOptimizerProvider creates a PromptOptimizer instance from DI container dependencies
func NewOptimizerProvider(injector do.Injector) (PromptOptimizer, error) {
	cfg := do.MustInvoke[config.ConfigService](injector)
	clientProvider := do.MustInvoke[llm.ClientProvider](injector)
	promptManager := do.MustInvoke[manager.PromptManager](injector)

	optimizerCfg := cfg.GetPromptOptimizerConfig()
	strategy := optimizerCfg.DefaultStrategy
	provider := optimizerCfg.DefaultProvider

	// Get LLM client (context.Background is used as client creation is one-time)
	client, err := clientProvider.GetClient(context.Background(), provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM client: %w", err)
	}

	// Create optimizer config
	optCfg := &OptimizerConfig{
		Kind:               strategy,
		Provider:           provider,
		MaxReflectionSteps: optimizerCfg.MaxReflectionSteps,
		MinReflectionSteps: optimizerCfg.MinReflectionSteps,
	}

	// Create optimizer with client
	return NewOptimizer(client, promptManager, optCfg)
}
