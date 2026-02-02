// Package optimizer provides prompt optimization strategies for improving agent prompts
// through trajectory analysis and feedback processing.
package optimizer

import (
	"context"
	"fmt"

	"github.com/m-mizutani/gollem"
)

// PromptOptimizer defines the interface for prompt optimization strategies.
type PromptOptimizer interface {
	// Optimize analyzes trajectories and returns an optimized prompt.
	Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error)
}

// NewOptimizer creates a new PromptOptimizer based on the specified strategy.
// Validates config and sets defaults (MaxReflectionSteps: 5, MinReflectionSteps: 1).
func NewOptimizer(client gollem.LLMClient, config *OptimizerConfig) (PromptOptimizer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if client == nil {
		return nil, fmt.Errorf("LLM client cannot be nil")
	}

	// Set defaults
	cfg := *config // Copy to avoid modifying original
	if cfg.MaxReflectionSteps == 0 {
		cfg.MaxReflectionSteps = 5
	}
	if cfg.MinReflectionSteps == 0 {
		cfg.MinReflectionSteps = 1
	}

	// Validate reflection steps
	if cfg.MaxReflectionSteps < cfg.MinReflectionSteps {
		return nil, fmt.Errorf("max_reflection_steps (%d) must be >= min_reflection_steps (%d)",
			cfg.MaxReflectionSteps, cfg.MinReflectionSteps)
	}

	switch cfg.Kind {
	case StrategyGradient:
		return newGradientOptimizer(client, &cfg)
	case StrategyMetaPrompt:
		return newMetaPromptOptimizer(client, &cfg)
	case StrategyPromptMemory:
		return newPromptMemoryOptimizer(client, &cfg)
	default:
		return nil, fmt.Errorf("unknown optimizer strategy: %s", cfg.Kind)
	}
}
