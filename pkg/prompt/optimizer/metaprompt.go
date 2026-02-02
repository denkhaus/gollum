// Package optimizer provides the meta-prompt optimization strategy.
package optimizer

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/m-mizutani/gollem"
)

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

// Optimize runs the metaprompt optimization strategy.
func (o *metaPromptOptimizer) Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error) {
	// Build the metaprompt
	prompt := o.buildMetaPrompt(input)

	// Create agent
	agent := gollem.New(o.client)

	// Run reflection loop (min to max steps)
	minSteps := o.config.MinReflectionSteps
	maxSteps := o.config.MaxReflectionSteps

	var lastResponse string
	var warrantsAdjustment bool

	for step := 0; step < maxSteps; step++ {
		// Execute agent
		resp, err := agent.Execute(ctx, gollem.Text(prompt))
		if err != nil {
			return nil, fmt.Errorf("agent execute failed: %w", err)
		}

		if len(resp.Texts) == 0 {
			continue
		}

		// Extract text response
		lastResponse = resp.Texts[0]

		// Check if response indicates no adjustment needed
		if o.isNoAdjustmentResponse(lastResponse) {
			warrantsAdjustment = false
			// Still need to meet min steps
			if step >= minSteps-1 {
				break
			}
		} else {
			warrantsAdjustment = true
			// Can stop once we have a valid adjustment
			if step >= minSteps-1 {
				break
			}
		}
	}

	// If no adjustment warranted, return original
	if !warrantsAdjustment {
		return &OptimizerResult{
			NewPrompt:          input.Prompt,
			WarrantsAdjustment: false,
			ChangeDescription:  "No adjustment needed - prompt performs well",
		}, nil
	}

	return &OptimizerResult{
		NewPrompt:          lastResponse,
		WarrantsAdjustment: true,
		ChangeDescription:  "Metaprompt optimization applied",
	}, nil
}

// buildMetaPrompt constructs the metaprompt.
func (o *metaPromptOptimizer) buildMetaPrompt(input *OptimizerInput) string {
	// TODO: Load from PromptManager
	promptTemplate := `You are helping an AI assistant learn by optimizing its prompt.

## Background

Below is the current prompt:

<current_prompt>
{{.Prompt}}
</current_prompt>

The developer provided these instructions regarding when/how to update:

<update_instructions>
{{.UpdateInstructions}}
</update_instructions>

## Session Data
Analyze the session(s) (and any user feedback) below:

<trajectories>
{{.Trajectories}}
</trajectories>

## Instructions

1. Reflect on the agent's performance on the given session(s) and identify any real failure modes (e.g., style mismatch, unclear or incomplete instructions, flawed reasoning, etc.).
2. Recommend the minimal changes necessary to address any real failures. If the prompt performs perfectly, simply respond with the original prompt without making any changes.
3. Retain any f-string variables in the existing prompt exactly as they are (e.g. {{.VariableName}}).

IFF changes are warranted, focus on actionable edits. Be concrete. Edits should be appropriate for the identified failure modes. For example, consider synthetic few-shot examples for style or clarifying decision boundaries, or adding or modifying explicit instructions for conditionals, rules, or logic fixes; or provide step-by-step reasoning guidelines for multi-step logic problems if the model is failing to reason appropriately.`

	// Render template with data
	tmpl, err := template.New("metaprompt").Parse(promptTemplate)
	if err != nil {
		return fmt.Sprintf("ERROR: failed to parse template: %v", err)
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
		return fmt.Sprintf("ERROR: failed to execute template: %v", err)
	}

	return buf.String()
}

// isNoAdjustmentResponse checks if the response indicates no adjustment is needed.
func (o *metaPromptOptimizer) isNoAdjustmentResponse(response string) bool {
	lower := strings.ToLower(response)
	noChangeIndicators := []string{
		"no adjustment",
		"no changes",
		"warrants_adjustment = false",
		"warrants_adjustment=false",
		"no recommendations",
	}

	for _, indicator := range noChangeIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}
