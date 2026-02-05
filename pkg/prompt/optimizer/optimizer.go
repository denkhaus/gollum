// Package optimizer provides prompt optimization strategies for improving agent prompts
// through trajectory analysis and feedback processing.
//
// # Optimization Strategies
//
// This package implements three optimization strategies adapted from LangMEM:
// https://github.com/langchain-ai/langmem
//
// ## Strategy Selection Guide
//
// Choose the appropriate strategy based on your requirements:
//
// 1. Prompt Memory (StrategyPromptMemory / "prompt_memory")
//    - LLM Calls: 1 total (fastest)
//    - Use Case: Simple adjustments, quick iterations
//    - Cost: Lowest
//    - Speed: Fastest
//    - Trade-off: Limited ability to learn from complex patterns
//    - Best for: Rapid prototyping, simple prompt tweaks, cost-sensitive applications
//
// 2. Meta-Prompt (StrategyMetaPrompt / "metaprompt")
//    - LLM Calls: 1-5 (configurable via Min/MaxReflectionSteps)
//    - Use Case: Balanced optimization between speed and quality
//    - Cost: Medium
//    - Speed: Medium
//    - Best for: General-purpose optimization, moderate complexity improvements
//
// 3. Gradient (StrategyGradient / "gradient")
//    - LLM Calls: 2-10 (configurable via Min/MaxReflectionSteps)
//    - Use Case: Complex improvements requiring thorough analysis
//    - Cost: Highest (each reflection step requires multiple LLM calls)
//    - Speed: Slowest
//    - Best for: Critical prompts, complex reasoning improvements, production-quality optimization
//
// # Performance Characteristics
//
// | Strategy   | Min LLM Calls | Max LLM Calls | Phases      | Structured Output |
// |------------|--------------|---------------|-------------|-------------------|
// | Memory     | 1            | 1             | Single      | Yes               |
// | Metaprompt | 1            | 5             | Combined    | Yes               |
// | Gradient   | 2            | 10            | Reflection+Update | Yes       |
//
// # Usage Example
//
//	import "github.com/denkhaus/gollum/pkg/prompt/optimizer"
//
//	// Create optimizer with default configuration
//	config := &optimizer.OptimizerConfig{
//		Kind:               shared.StrategyGradient,
//		MaxReflectionSteps: 5,
//		MinReflectionSteps: 2,
//		Provider:           shared.ProviderAnthropic,
//	}
//	opt, err := optimizer.NewOptimizer(llmClient, promptManager, config)
//
//	// Optimize a prompt based on conversation trajectories
//	result, err := opt.Optimize(ctx, &optimizer.OptimizerInput{
//		Prompt:             "You are a helpful assistant...",
//		Trajectories:       trajectories,
//		UpdateInstructions: "Improve clarity and add examples",
//	})
package optimizer

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/m-mizutani/gollem"
)

// PromptOptimizer defines the interface for prompt optimization strategies.
type PromptOptimizer interface {
	// Optimize analyzes trajectories and returns an optimized prompt.
	Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error)
}

// NewOptimizer creates a new PromptOptimizer based on the specified strategy.
// Validates config and sets defaults (MaxReflectionSteps: 5, MinReflectionSteps: 1).
func NewOptimizer(client gollem.LLMClient, promptManager manager.PromptManager, config *OptimizerConfig) (PromptOptimizer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if client == nil {
		return nil, fmt.Errorf("LLM client cannot be nil")
	}

	if promptManager == nil {
		return nil, fmt.Errorf("promptManager cannot be nil")
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
	case shared.StrategyGradient:
		return newGradientOptimizer(client, promptManager, &cfg)
	case shared.StrategyMetaPrompt:
		return newMetaPromptOptimizer(client, promptManager, &cfg)
	case shared.StrategyPromptMemory:
		return newPromptMemoryOptimizer(client, promptManager, &cfg)
	default:
		return nil, fmt.Errorf("unknown optimizer strategy: %s", cfg.Kind)
	}
}
