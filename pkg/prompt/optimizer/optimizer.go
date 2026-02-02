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

// OptimizerConfig contains configuration for the optimizer.
// Note: This is duplicated from types.go for reference in the factory.
// The actual type is in types.go.
//
//nolint:revive // This type is intentionally shadowed for documentation purposes
type _OptimizerConfigDoc struct {
	Kind                OptimizerStrategy
	MaxReflectionSteps  int
	MinReflectionSteps  int
	Provider            string
	GradientPrompt      string
	MetaPrompt          string
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

// gradientOptimizer implements the gradient strategy with reflection loop.
type gradientOptimizer struct {
	client gollem.LLMClient
	config *OptimizerConfig
}

// newGradientOptimizer creates a new gradient optimizer.
func newGradientOptimizer(client gollem.LLMClient, config *OptimizerConfig) (PromptOptimizer, error) {
	return &gradientOptimizer{
		client: client,
		config: config,
	}, nil
}

// Optimize runs the gradient optimization strategy.
// Phase 1: Reflection with think/critique tools
// Phase 2: Apply recommendations with metaprompt
func (o *gradientOptimizer) Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error) {
	// Stub implementation - will be completed in Plan 02
	return nil, fmt.Errorf("gradientOptimizer.Optimize: not yet implemented")
}

// metaPromptOptimizer implements the combined reflection and update strategy.
type metaPromptOptimizer struct {
	client gollem.LLMClient
	config *OptimizerConfig
}

// newMetaPromptOptimizer creates a new metaprompt optimizer.
func newMetaPromptOptimizer(client gollem.LLMClient, config *OptimizerConfig) (PromptOptimizer, error) {
	return &metaPromptOptimizer{
		client: client,
		config: config,
	}, nil
}

// Optimize runs the metaprompt optimization strategy (combined reflection + update).
func (o *metaPromptOptimizer) Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error) {
	// Stub implementation - will be completed in Plan 02
	return nil, fmt.Errorf("metaPromptOptimizer.Optimize: not yet implemented")
}

// promptMemoryOptimizer implements the single-shot optimization strategy.
type promptMemoryOptimizer struct {
	client gollem.LLMClient
	config *OptimizerConfig
}

// newPromptMemoryOptimizer creates a new prompt memory optimizer.
func newPromptMemoryOptimizer(client gollem.LLMClient, config *OptimizerConfig) (PromptOptimizer, error) {
	return &promptMemoryOptimizer{
		client: client,
		config: config,
	}, nil
}

// Optimize runs the prompt memory optimization strategy (single-shot).
func (o *promptMemoryOptimizer) Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error) {
	// Stub implementation - will be completed in Plan 02
	return nil, fmt.Errorf("promptMemoryOptimizer.Optimize: not yet implemented")
}
