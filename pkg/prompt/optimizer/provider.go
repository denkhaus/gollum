// Package optimizer provides DI provider functions for the prompt optimizer
package optimizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/llm"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
)

// NewOptimizerProvider creates a PromptOptimizer instance from DI container dependencies
func NewOptimizerProvider(injector do.Injector) (PromptOptimizer, error) {
	cfg := do.MustInvoke[config.ConfigService](injector)
	clientProvider := do.MustInvoke[llm.ClientProvider](injector)
	promptManager := do.MustInvoke[manager.PromptManager](injector)

	optimizerCfg := cfg.GetPromptOptimizerConfig()

	// Map config string to OptimizerStrategy
	strategy := mapStrategy(optimizerCfg.DefaultStrategy)
	if strategy == StrategyUnknown {
		return nil, fmt.Errorf("unknown optimizer strategy: %s", optimizerCfg.DefaultStrategy)
	}

	// Map config string to LLMProvider
	provider := mapProvider(optimizerCfg.DefaultProvider)
	if provider == shared.LLMProvider("") {
		return nil, fmt.Errorf("unknown LLM provider: %s", optimizerCfg.DefaultProvider)
	}

	// Get LLM client (context.Background is used as client creation is one-time)
	client, err := clientProvider.GetClient(context.Background(), provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM client: %w", err)
	}

	// Create optimizer config
	optCfg := &OptimizerConfig{
		Kind:               strategy,
		Provider:           string(provider),
		MaxReflectionSteps: optimizerCfg.MaxReflectionSteps,
		MinReflectionSteps: optimizerCfg.MinReflectionSteps,
	}

	// Create optimizer with client
	return NewOptimizer(client, promptManager, optCfg)
}

// mapStrategy converts config string to OptimizerStrategy enum
// Supports: gradient, metaprompt/meta-prompt, prompt_memory/prompt-memory
// Returns StrategyUnknown for invalid values
func mapStrategy(s string) OptimizerStrategy {
	switch strings.ToLower(strings.ReplaceAll(s, "-", "_")) {
	case "gradient":
		return StrategyGradient
	case "metaprompt", "meta_prompt":
		return StrategyMetaPrompt
	case "promptmemory", "prompt_memory":
		return StrategyPromptMemory
	default:
		return StrategyUnknown
	}
}

// mapProvider converts config string to LLMProvider enum
// Supports: anthropic, openai, gemini
// Returns empty string for invalid values
func mapProvider(p string) shared.LLMProvider {
	switch strings.ToLower(p) {
	case "anthropic":
		return shared.LLMProviderAnthropic
	case "openai":
		return shared.LLMProviderOpenAI
	case "gemini":
		return shared.LLMProviderGemini
	default:
		return shared.LLMProvider("")
	}
}
