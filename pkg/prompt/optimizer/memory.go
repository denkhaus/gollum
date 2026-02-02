// Package optimizer provides the prompt memory optimization strategy.
package optimizer

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/m-mizutani/gollem"
)

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

// Optimize runs the prompt memory optimization strategy.
func (o *promptMemoryOptimizer) Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error) {
	// Build the prompt memory prompt
	prompt := o.buildPromptMemoryPrompt(input)

	// Create agent
	agent := gollem.New(o.client)

	// Generate content (single-shot)
	resp, err := agent.Execute(ctx, gollem.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("agent execute failed: %w", err)
	}

	if len(resp.Texts) == 0 {
		return nil, fmt.Errorf("no response from agent")
	}

	response := resp.Texts[0]

	// Check if response indicates no adjustment needed
	if o.isNoAdjustmentResponse(response) {
		return &OptimizerResult{
			NewPrompt:          input.Prompt,
			WarrantsAdjustment: false,
			ChangeDescription:  "No adjustment needed - prompt performs well",
		}, nil
	}

	// Extract the new prompt from the response
	newPrompt := o.extractNewPrompt(response, input.Prompt)

	return &OptimizerResult{
		NewPrompt:          newPrompt,
		WarrantsAdjustment: true,
		ChangeDescription:  "Prompt memory optimization applied",
	}, nil
}

// buildPromptMemoryPrompt constructs the prompt memory prompt.
func (o *promptMemoryOptimizer) buildPromptMemoryPrompt(input *OptimizerInput) string {
	// TODO: Load from PromptManager
	promptTemplate := `You are helping an AI agent improve. You can do this by changing their system prompt.

This is their current prompt:
<current_prompt>
{{.CurrentPrompt}}
</current_prompt>

Here was the agent's trajectory:
<trajectory>
{{.Trajectory}}
</trajectory>

Here is the user's feedback:

<feedback>
{{.Feedback}}
</feedback>

Here are instructions for updating the agent's prompt:

<instructions>
{{.Instructions}}
</instructions>


Based on this, return an updated prompt

You should return the full prompt, so if there's anything from before that you want to include, make sure to do that. Feel free to override or change anything that seems irrelevant. You do not need to update the prompt - if you don't want to, just return update_prompt = False and an empty string for new prompt.`

	// Render template with data
	tmpl, err := template.New("prompt_memory").Parse(promptTemplate)
	if err != nil {
		return fmt.Sprintf("ERROR: failed to parse template: %v", err)
	}

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
		instructions = "No specific instructions provided"
	}

	data := map[string]string{
		"CurrentPrompt": input.Prompt,
		"Trajectory":    trajectory,
		"Feedback":      feedback,
		"Instructions":  instructions,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("ERROR: failed to execute template: %v", err)
	}

	return buf.String()
}

// isNoAdjustmentResponse checks if the response indicates no adjustment is needed.
func (o *promptMemoryOptimizer) isNoAdjustmentResponse(response string) bool {
	lower := strings.ToLower(response)
	noChangeIndicators := []string{
		"no adjustment",
		"warrants_adjustment = false",
		"warrants_adjustment=false",
		"update_prompt = false",
		"update_prompt=false",
		"no update",
	}

	for _, indicator := range noChangeIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}

// extractNewPrompt extracts the new prompt from the response.
func (o *promptMemoryOptimizer) extractNewPrompt(response, originalPrompt string) string {
	// Check if response contains the warrants_adjustment flag
	if strings.Contains(strings.ToLower(response), "warrants_adjustment") ||
	   strings.Contains(strings.ToLower(response), "update_prompt") {
		// Find the prompt content after the flags
		lines := strings.Split(response, "\n")
		var promptLines []string
		capturing := false

		for _, line := range lines {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "warrants_adjustment") || strings.Contains(lower, "update_prompt") {
				// Skip this line and start capturing next
				capturing = true
				continue
			}
			if capturing && len(strings.TrimSpace(line)) > 0 {
				promptLines = append(promptLines, line)
			}
		}

		if len(promptLines) > 0 {
			extracted := strings.Join(promptLines, "\n")
			extracted = strings.TrimSpace(extracted)
			if extracted != "" {
				return extracted
			}
		}
	}

	// If no special format, return the response as-is
	return strings.TrimSpace(response)
}
