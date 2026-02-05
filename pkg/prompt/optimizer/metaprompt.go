// Package optimizer provides the meta-prompt optimization strategy.
//
// # Meta-Prompt Strategy
//
// The meta-prompt optimizer provides a balanced approach to optimization,
// combining reflection and update into a single process with configurable
// iteration depth.
//
// ## When to Use
//
// Use this strategy when:
// - You need a balance between speed and quality
// - Moderate cost is acceptable (1-5 LLM calls)
// - Direct pattern learning from examples works well
// - You want better results than prompt memory without the cost of gradient
// - General-purpose optimization for most use cases
//
// ## How It Works
//
// Unlike the gradient strategy's two-phase approach, meta-prompt combines
// analysis and update into a single iterative process:
//   - Each iteration analyzes trajectories and generates an updated prompt
//   - Iterations continue from MinReflectionSteps to MaxReflectionSteps
//   - The final iteration produces the optimized prompt
//
// ## Performance
//
// - LLM Calls: 1-5 (configurable via Min/MaxReflectionSteps)
// - Cost: Medium
// - Speed: Medium
//
// ## Configuration
//
//	config := &optimizer.OptimizerConfig{
//		Kind:               shared.StrategyMetaPrompt,
//		MaxReflectionSteps: 5,  // Maximum iterations
//		MinReflectionSteps: 2,  // Minimum iterations
//	}
//
// ## Best Practices
//
// - Good default choice for most optimization scenarios
// - Use MinReflectionSteps of 1-2 for faster results
// - Use MaxReflectionSteps of 3-5 for better quality
// - Consider this strategy when unsure which to use
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

// metaPromptOptimizer implements the combined reflection and update strategy.
type metaPromptOptimizer struct {
	client        gollem.LLMClient
	promptManager manager.PromptManager
	config        *OptimizerConfig
}

// newMetaPromptOptimizer creates a new metaprompt optimizer.
func newMetaPromptOptimizer(client gollem.LLMClient, promptManager manager.PromptManager, config *OptimizerConfig) (PromptOptimizer, error) {
	return &metaPromptOptimizer{
		client:        client,
		promptManager: promptManager,
		config:        config,
	}, nil
}

// Optimize runs the metaprompt optimization strategy.
func (o *metaPromptOptimizer) Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error) {
	// Create response schema
	schema, err := gollem.ToSchema(MetaPromptResponse{})
	if err != nil {
		return nil, fmt.Errorf("failed to create response schema: %w", err)
	}

	// Build the metaprompt from PromptManager
	prompt, err := o.buildMetaPrompt(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to build metaprompt: %w", err)
	}

	// Create session with JSON schema for structured output
	session, err := o.client.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(schema),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Run reflection iterations (min to max steps)
	minSteps := o.config.MinReflectionSteps
	maxSteps := o.config.MaxReflectionSteps

	var lastResponse *MetaPromptResponse

	for step := 0; step < maxSteps; step++ {
		// Generate structured response
		resp, err := session.GenerateContent(ctx, gollem.Text(prompt))
		if err != nil {
			return nil, fmt.Errorf("generate content failed: %w", err)
		}

		// Parse JSON response
		var response MetaPromptResponse
		if err := json.Unmarshal([]byte(resp.Texts[0]), &response); err != nil {
			return nil, fmt.Errorf("failed to parse metaprompt response: %w", err)
		}

		lastResponse = &response

		// If we have a clear decision and met min steps, we can stop
		if step >= minSteps-1 {
			break
		}
	}

	if lastResponse == nil {
		return nil, fmt.Errorf("no valid metaprompt response received")
	}

	// If no adjustment warranted, return original
	if !lastResponse.WarrantsAdjustment {
		return &OptimizerResult{
			NewPrompt:          input.Prompt,
			WarrantsAdjustment: false,
			ChangeDescription:  "No adjustment needed - prompt performs well",
		}, nil
	}

	return &OptimizerResult{
		NewPrompt:          lastResponse.UpdatedPrompt,
		WarrantsAdjustment: true,
		ChangeDescription:  fmt.Sprintf("Metaprompt optimization applied: %s", lastResponse.Reasoning),
	}, nil
}

// buildMetaPrompt constructs the metaprompt using PromptManager.
func (o *metaPromptOptimizer) buildMetaPrompt(ctx context.Context, input *OptimizerInput) (string, error) {
	updateInstructions := input.UpdateInstructions
	if updateInstructions == "" {
		updateInstructions = defaultUpdateInstructions
	}

	renderCtx := &prompt.RenderContext{
		Values: map[string]interface{}{
			"Prompt":             input.Prompt,
			"Trajectories":       FormatSessions(input.Trajectories),
			"UpdateInstructions": updateInstructions,
		},
	}

	// Use PromptManager's GetPromptWithContext
	return o.promptManager.GetPromptWithContext(ctx, prompt.PromptIDOptimizerMeta, renderCtx)
}
