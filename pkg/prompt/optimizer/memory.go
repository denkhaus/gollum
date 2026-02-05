// Package optimizer provides the prompt memory optimization strategy.
//
// # Prompt Memory Strategy
//
// The prompt memory optimizer is the fastest optimization strategy, requiring only
// a single LLM call. It analyzes the first trajectory and provides direct feedback
// on prompt improvements.
//
// ## When to Use
//
// Use this strategy when:
// - You need fast, single-shot optimization (1 LLM call)
// - Simple adjustments are sufficient
// - Cost/latency is a concern
// - You're optimizing for speed over thoroughness
//
// ## Limitations
//
// - Limited ability to learn from complex patterns
// - Only analyzes the first trajectory provided
// - No iterative refinement
//
// ## Performance
//
// - LLM Calls: 1 total (fastest)
// - Cost: Lowest
// - Speed: Fastest
//
// ## Configuration
//
// This strategy does not use reflection steps configuration (MinReflectionSteps,
// MaxReflectionSteps) as it completes in a single LLM call.
//
// # Reference
//
// Adapted from LangMEM: https://github.com/langchain-ai/langmem
package optimizer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/m-mizutani/gollem"
)

// promptMemoryOptimizer implements the single-shot optimization strategy.
type promptMemoryOptimizer struct {
	client        gollem.LLMClient
	promptManager manager.PromptManager
	config        *OptimizerConfig
}

// newPromptMemoryOptimizer creates a new prompt memory optimizer.
func newPromptMemoryOptimizer(client gollem.LLMClient, promptManager manager.PromptManager, config *OptimizerConfig) (PromptOptimizer, error) {
	return &promptMemoryOptimizer{
		client:        client,
		promptManager: promptManager,
		config:        config,
	}, nil
}

// Optimize runs the prompt memory optimization strategy.
func (o *promptMemoryOptimizer) Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error) {
	// Create response schema
	schema, err := gollem.ToSchema(PromptMemoryResponse{})
	if err != nil {
		return nil, fmt.Errorf("failed to create response schema: %w", err)
	}

	// Build the prompt memory prompt from PromptManager
	prompt, err := o.buildPromptMemoryPrompt(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt memory prompt: %w", err)
	}

	// Create session with JSON schema for structured output
	session, err := o.client.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(schema),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate structured response (single-shot)
	resp, err := session.GenerateContent(ctx, gollem.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("generate content failed: %w", err)
	}

	// Parse JSON response
	var response PromptMemoryResponse
	if err := json.Unmarshal([]byte(resp.Texts[0]), &response); err != nil {
		return nil, fmt.Errorf("failed to parse prompt memory response: %w", err)
	}

	// If no adjustment warranted, return original
	if !response.WarrantsAdjustment {
		return &OptimizerResult{
			NewPrompt:          input.Prompt,
			WarrantsAdjustment: false,
			ChangeDescription:  "No adjustment needed - prompt performs well",
		}, nil
	}

	return &OptimizerResult{
		NewPrompt:          response.UpdatedPrompt,
		WarrantsAdjustment: true,
		ChangeDescription:  fmt.Sprintf("Prompt memory optimization applied: %s", response.Reasoning),
	}, nil
}

// buildPromptMemoryPrompt constructs the prompt memory prompt using PromptManager.
func (o *promptMemoryOptimizer) buildPromptMemoryPrompt(ctx context.Context, input *OptimizerInput) (string, error) {
	// Use first trajectory for single-shot
	trajectory := ""
	if len(input.Trajectories) > 0 {
		trajectory = input.Trajectories[0].FormatForLLM()
	}

	feedback := input.Feedback
	if feedback == "" {
		feedback = "No feedback provided"
	}

	instructions := input.UpdateInstructions
	if instructions == "" {
		instructions = defaultUpdateInstructions
	}

	renderCtx := &prompt.RenderContext{
		Values: map[string]interface{}{
			"CurrentPrompt": input.Prompt,
			"Trajectory":    trajectory,
			"Feedback":      feedback,
			"Instructions":  instructions,
		},
	}

	// Use PromptManager's GetPromptWithContext
	return o.promptManager.GetPromptWithContext(ctx, prompt.PromptIDOptimizerMemory, renderCtx)
}
