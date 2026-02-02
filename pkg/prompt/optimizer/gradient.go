// Package optimizer provides the gradient optimization strategy.
package optimizer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/m-mizutani/gollem"
)

// gradientOptimizer implements the gradient strategy with reflection loop.
type gradientOptimizer struct {
	client        gollem.LLMClient
	promptManager manager.PromptManager
	config        *OptimizerConfig
}

// newGradientOptimizer creates a new gradient optimizer.
func newGradientOptimizer(client gollem.LLMClient, promptManager manager.PromptManager, config *OptimizerConfig) (PromptOptimizer, error) {
	return &gradientOptimizer{
		client:        client,
		promptManager: promptManager,
		config:        config,
	}, nil
}

// Optimize runs the gradient optimization strategy.
func (o *gradientOptimizer) Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error) {
	// Phase 1: Run reflection loop with structured output
	reflection, err := o.runReflectionLoop(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("reflection loop failed: %w", err)
	}

	// If no adjustment is warranted, return original
	if !reflection.WarrantsAdjustment {
		return &OptimizerResult{
			NewPrompt:          input.Prompt,
			WarrantsAdjustment: false,
			ChangeDescription:  "No adjustment needed - prompt performs well",
		}, nil
	}

	// Phase 2: Apply recommendations
	newPrompt, err := o.applyRecommendations(ctx, input.Prompt, reflection)
	if err != nil {
		return nil, fmt.Errorf("failed to apply recommendations: %w", err)
	}

	return &OptimizerResult{
		NewPrompt:          newPrompt,
		WarrantsAdjustment: true,
		ChangeDescription:  fmt.Sprintf("Gradient optimization applied: %s", reflection.Reasoning),
	}, nil
}

// runReflectionLoop executes the reflection loop with structured output.
func (o *gradientOptimizer) runReflectionLoop(ctx context.Context, input *OptimizerInput) (*GradientReflectionResponse, error) {
	// Create response schema
	schema, err := gollem.ToSchema(GradientReflectionResponse{})
	if err != nil {
		return nil, fmt.Errorf("failed to create response schema: %w", err)
	}

	// Build reflection prompt from PromptManager
	prompt, err := o.buildReflectionPrompt(input)
	if err != nil {
		return nil, fmt.Errorf("failed to build reflection prompt: %w", err)
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

	var lastResponse *GradientReflectionResponse

	for step := 0; step < maxSteps; step++ {
		// Generate structured response
		resp, err := session.GenerateContent(ctx, gollem.Text(prompt))
		if err != nil {
			return nil, fmt.Errorf("generate content failed: %w", err)
		}

		// Parse JSON response
		var reflection GradientReflectionResponse
		if err := json.Unmarshal([]byte(resp.Texts[0]), &reflection); err != nil {
			return nil, fmt.Errorf("failed to parse reflection response: %w", err)
		}

		lastResponse = &reflection

		// If we have a clear decision and met min steps, we can stop
		if step >= minSteps-1 {
			break
		}
	}

	if lastResponse == nil {
		return nil, fmt.Errorf("no valid reflection response received")
	}

	return lastResponse, nil
}

// buildReflectionPrompt constructs the prompt for the reflection phase using PromptManager.
func (o *gradientOptimizer) buildReflectionPrompt(input *OptimizerInput) (string, error) {
	// Load template from PromptManager
	templateContent, err := o.promptManager.GetOptimizerGradientPrompt()
	if err != nil {
		return "", fmt.Errorf("failed to load gradient prompt template: %w", err)
	}

	// Parse template
	tmpl, err := template.New("optimizergradientprompt").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	updateInstructions := input.UpdateInstructions
	if updateInstructions == "" {
		updateInstructions = "No specific instructions provided"
	}

	data := map[string]string{
		"Prompt":             input.Prompt,
		"Trajectories":       FormatSessions(input.Trajectories),
		"UpdateInstructions": updateInstructions,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// applyRecommendations applies the recommendations using the metaprompt.
func (o *gradientOptimizer) applyRecommendations(ctx context.Context, currentPrompt string, reflection *GradientReflectionResponse) (string, error) {
	// Create response schema for the updated prompt
	schema, err := gollem.ToSchema(struct {
		UpdatedPrompt string `json:"updated_prompt"`
	}{})
	if err != nil {
		return "", fmt.Errorf("failed to create response schema: %w", err)
	}

	// Build metaprompt with hypotheses and recommendations from PromptManager
	metaprompt, err := o.buildMetaprompt(currentPrompt, reflection.Hypotheses, reflection.Recommendations)
	if err != nil {
		return "", fmt.Errorf("failed to build metaprompt: %w", err)
	}

	// Create session for phase 2
	session, err := o.client.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(schema),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	// Generate updated prompt
	resp, err := session.GenerateContent(ctx, gollem.Text(metaprompt))
	if err != nil {
		return "", fmt.Errorf("generate content failed: %w", err)
	}

	// Parse response
	var result struct {
		UpdatedPrompt string `json:"updated_prompt"`
	}
	if err := json.Unmarshal([]byte(resp.Texts[0]), &result); err != nil {
		return "", fmt.Errorf("failed to parse metaprompt response: %w", err)
	}

	return result.UpdatedPrompt, nil
}

// buildMetaprompt constructs the metaprompt for phase 2 using PromptManager.
func (o *gradientOptimizer) buildMetaprompt(currentPrompt, hypotheses, recommendations string) (string, error) {
	// Load template from PromptManager
	templateContent, err := o.promptManager.GetOptimizerGradientMetaprompt()
	if err != nil {
		return "", fmt.Errorf("failed to load gradient metaprompt template: %w", err)
	}

	// Parse template
	tmpl, err := template.New("optimizergradientmetaprompt").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := map[string]string{
		"CurrentPrompt":   currentPrompt,
		"Hypotheses":      hypotheses,
		"Recommendations": recommendations,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
