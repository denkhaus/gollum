// Package optimizer provides the gradient optimization strategy.
package optimizer

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/m-mizutani/gollem"
)

// gradientOptimizer implements the gradient strategy with reflection loop.
type gradientOptimizer struct {
	client         gollem.LLMClient
	promptManager  manager.PromptManager
	config         *OptimizerConfig
}

// newGradientOptimizer creates a new gradient optimizer.
func newGradientOptimizer(client gollem.LLMClient, config *OptimizerConfig) (PromptOptimizer, error) {
	// TODO: Add PromptManager as dependency
	return &gradientOptimizer{
		client: client,
		config: config,
	}, nil
}

// Optimize runs the gradient optimization strategy.
func (o *gradientOptimizer) Optimize(ctx context.Context, input *OptimizerInput) (*OptimizerResult, error) {
	// Build reflection prompt
	reflectionPrompt := o.buildReflectionPrompt(input)

	// Create agent with tools
	agent := gollem.New(
		o.client,
		gollem.WithTools(&thinkTool{}),
	)

	// Phase 1: Run reflection loop
	response, err := o.runReflectionLoop(ctx, agent, reflectionPrompt)
	if err != nil {
		return nil, fmt.Errorf("reflection loop failed: %w", err)
	}

	// Check if adjustment is warranted from the response
	if !o.warrantsAdjustment(response) {
		return &OptimizerResult{
			NewPrompt:          input.Prompt,
			WarrantsAdjustment: false,
			ChangeDescription:  "No adjustment needed - prompt performs well",
		}, nil
	}

	// Phase 2: Apply recommendations
	newPrompt, err := o.applyRecommendations(ctx, input.Prompt, response)
	if err != nil {
		return nil, fmt.Errorf("failed to apply recommendations: %w", err)
	}

	return &OptimizerResult{
		NewPrompt:          newPrompt,
		WarrantsAdjustment: true,
		ChangeDescription:  "Gradient optimization applied based on reflection",
	}, nil
}

// buildReflectionPrompt constructs the prompt for the reflection phase.
func (o *gradientOptimizer) buildReflectionPrompt(input *OptimizerInput) string {
	// TODO: Load from PromptManager
	// For now, use placeholder
	prompt := `You are reviewing the performance of an AI assistant in a given interaction.

## Instructions

The current prompt that was used for the session is provided below.

<current_prompt>
{{.Prompt}}
</current_prompt>

The developer provided the following instructions around when and how to update the prompt:

<update_instructions>
{{.UpdateInstructions}}
</update_instructions>

## Session data

Analyze the following trajectories (and any associated user feedback) (either conversations with a user or other work that was performed by the assistant):

<trajectories>
{{.Trajectories}}
</trajectories>

## Task

Analyze the conversation, including the user's request and the assistant's response, and evaluate:
1. How effectively the assistant fulfilled the user's intent.
2. Where the assistant might have deviated from user expectations or the desired outcome.
3. Specific areas (correctness, completeness, style, tone, alignment, etc.) that need improvement.

If the prompt seems to do well, then no further action is needed. We ONLY recommend updates if there is evidence of failures.
When failures occur, we want to recommend the minimal required changes to fix the problem.

Focus on actionable changes and be concrete.

1. Summarize the key successes and failures in the assistant's response.
2. Identify which failure mode(s) best describe the issues (examples: style mismatch, unclear or incomplete instructions, flawed logic or reasoning, hallucination, etc.).
3. Based on these failure modes, recommend the most suitable edit strategy. For example, consider:
   - Use synthetic few-shot examples for style or clarifying decision boundaries.
   - Use explicit instruction updates for conditionals, rules, or logic fixes.
   - Provide step-by-step reasoning guidelines for multi-step logic problems.
4. Provide detailed, concrete suggestions for how to update the prompt accordingly.

But remember, the final updated prompt should only be changed if there is evidence of poor performance, and our recommendations should be minimally invasive.
Do not recommend generic changes that aren't clearly linked to failure modes.

First think through the conversation and critique the current behavior.
If you believe the prompt needs to further adapt to the target context, provide precise recommendations.
Otherwise, mark warrants_adjustment as False and respond with 'No recommendations.'`

	// Render template with data
	tmpl, err := template.New("gradient").Parse(prompt)
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

// runReflectionLoop executes the reflection loop with min/max bounds.
func (o *gradientOptimizer) runReflectionLoop(ctx context.Context, agent *gollem.Agent, prompt string) (string, error) {
	minSteps := o.config.MinReflectionSteps
	maxSteps := o.config.MaxReflectionSteps

	var lastResponse string

	for step := 0; step < maxSteps; step++ {
		// Execute agent
		resp, err := agent.Execute(ctx, gollem.Text(prompt))
		if err != nil {
			return "", fmt.Errorf("agent execute failed: %w", err)
		}

		if len(resp.Texts) == 0 {
			continue
		}

		// Extract text response
		lastResponse = resp.Texts[0]

		// Check if we got a definitive recommendation
		if o.hasRecommendation(lastResponse) {
			// We have a decision
			if step >= minSteps-1 {
				// Met minimum steps, can stop
				break
			}
		}
	}

	return lastResponse, nil
}

// warrantsAdjustment checks if the response indicates adjustment is needed.
func (o *gradientOptimizer) warrantsAdjustment(response string) bool {
	if response == "" {
		return false
	}

	lower := strings.ToLower(response)

	// Check for explicit no-adjustment signals
	noAdjustSignals := []string{
		"no adjustment",
		"warrants_adjustment = false",
		"warrants_adjustment=false",
		"no recommendations",
		"no changes needed",
	}

	for _, signal := range noAdjustSignals {
		if strings.Contains(lower, signal) {
			return false
		}
	}

	// If response has content and doesn't explicitly say no, assume adjustment
	return len(response) > 10
}

// hasRecommendation checks if the response contains a recommendation.
func (o *gradientOptimizer) hasRecommendation(response string) bool {
	if response == "" {
		return false
	}

	lower := strings.ToLower(response)
	indicators := []string{
		"recommend",
		"warrants_adjustment",
		"hypothes",
		"adjustment",
	}

	for _, indicator := range indicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}

// applyRecommendations applies the recommendations using the metaprompt.
func (o *gradientOptimizer) applyRecommendations(ctx context.Context, currentPrompt string, reflectionResponse string) (string, error) {
	// Extract hypotheses and recommendations from reflection
	hypotheses := o.extractHypotheses(reflectionResponse)
	recommendations := o.extractRecommendations(reflectionResponse)

	// TODO: Load from PromptManager
	metaprompt := `You are optimizing a prompt to handle its target task more effectively.

<current_prompt>
{{.CurrentPrompt}}
</current_prompt>

We hypothesize the current prompt underperforms for these reasons:

<hypotheses>
{{.Hypotheses}}
</hypotheses>

Based on these hypotheses, we recommend the following adjustments:

<recommendations>
{{.Recommendations}}
</recommendations>

Respond with the updated prompt. Remember to ONLY make changes that are clearly necessary. Aim to be minimally invasive.`

	// Render template
	tmpl, err := template.New("gradient_meta").Parse(metaprompt)
	if err != nil {
		return "", fmt.Errorf("failed to parse metaprompt template: %w", err)
	}

	data := map[string]string{
		"CurrentPrompt":    currentPrompt,
		"Hypotheses":       hypotheses,
		"Recommendations":  recommendations,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute metaprompt template: %w", err)
	}

	// Create new agent (no tools needed for phase 2)
	agent := gollem.New(o.client)

	// Generate updated prompt
	resp, err := agent.Execute(ctx, gollem.Text(buf.String()))
	if err != nil {
		return "", fmt.Errorf("agent execute failed: %w", err)
	}

	if len(resp.Texts) == 0 {
		return "", fmt.Errorf("no response from agent")
	}

	return resp.Texts[0], nil
}

// extractHypotheses extracts hypotheses from the reflection response.
func (o *gradientOptimizer) extractHypotheses(response string) string {
	// Look for hypotheses section
	lines := strings.Split(response, "\n")
	var hypotheses []string
	capturing := false

	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "hypothes") {
			capturing = true
		}
		if capturing && len(strings.TrimSpace(line)) > 0 {
			hypotheses = append(hypotheses, line)
		}
		// Stop at recommendations
		if strings.Contains(lower, "recommend") {
			break
		}
	}

	if len(hypotheses) > 0 {
		return strings.Join(hypotheses, "\n")
	}

	return response
}

// extractRecommendations extracts recommendations from the reflection response.
func (o *gradientOptimizer) extractRecommendations(response string) string {
	// Look for recommendations section
	lines := strings.Split(response, "\n")
	var recommendations []string
	capturing := false

	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "recommend") {
			capturing = true
		}
		if capturing && len(strings.TrimSpace(line)) > 0 {
			recommendations = append(recommendations, line)
		}
	}

	if len(recommendations) > 0 {
		return strings.Join(recommendations, "\n")
	}

	return response
}

// thinkTool implements the "think" tool for recording reasoning during gradient optimization.
type thinkTool struct{}

// Spec returns the tool specification.
func (t *thinkTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "think",
		Description: "A reflection tool for reasoning over complexities and hypothesizing fixes. Use this to explore ideas and analyze the conversation deeply.",
		Parameters: map[string]*gollem.Parameter{
			"thought": {
				Type:        gollem.TypeString,
				Description: "Your thought, reasoning, or hypothesis about the conversation",
				Required:    []string{"thought"},
			},
		},
		Required: []string{"thought"},
	}
}

// Run executes the think tool and provides encouraging guidance.
func (t *thinkTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	thought, ok := args["thought"].(string)
	if !ok {
		return nil, fmt.Errorf("thought must be a string")
	}
	// Provide acknowledgment that encourages continued reflection
	return map[string]any{
		"acknowledged": "Your thought has been recorded. Continue analyzing or proceed to critique when ready.",
		"thought":      thought, // Echo back for context
	}, nil
}

// critiqueTool implements the "critique" tool for diagnosing flaws during gradient optimization.
type critiqueTool struct{}

// Spec returns the tool specification.
func (t *critiqueTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "critique",
		Description: "A critique tool for examining your previous reasoning and identifying flaws or gaps. Use this to challenge your own hypotheses.",
		Parameters: map[string]*gollem.Parameter{
			"criticism": {
				Type:        gollem.TypeString,
				Description: "Your critique or counter-argument to previous reasoning",
				Required:    []string{"criticism"},
			},
		},
		Required: []string{"criticism"},
	}
}

// Run executes the critique tool and provides structured guidance.
func (t *critiqueTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	criticism, ok := args["criticism"].(string)
	if !ok {
		return nil, fmt.Errorf("criticism must be a string")
	}
	// Provide guidance for next steps
	return map[string]any{
		"acknowledged": "Your critique has been recorded. Use this to refine your hypotheses or proceed to recommendations.",
		"criticism":    criticism, // Echo back for context
	}, nil
}

// recommendTool implements the "recommend" tool for providing optimization recommendations.
type recommendTool struct{}

// Spec returns the tool specification.
func (t *recommendTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "recommend",
		Description: "Use this tool to finalize your analysis. Specify whether adjustment is warranted and provide hypotheses and recommendations if needed.",
		Parameters: map[string]*gollem.Parameter{
			"warrants_adjustment": {
				Type:        gollem.TypeBoolean,
				Description: "Whether the prompt warrants adjustment based on your analysis (true/false)",
			},
			"hypotheses": {
				Type:        gollem.TypeString,
				Description: "Hypotheses about why the current prompt underperforms (if adjustment warranted)",
			},
			"full_recommendations": {
				Type:        gollem.TypeString,
				Description: "Detailed recommendations for prompt improvement (if adjustment warranted)",
			},
		},
		Required: []string{"warrants_adjustment"},
	}
}

// Run executes the recommend tool and captures the final decision.
func (t *recommendTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	warrantsAdjustment, ok := args["warrants_adjustment"].(bool)
	if !ok {
		return nil, fmt.Errorf("warrants_adjustment must be a boolean")
	}

	result := map[string]any{
		"warrants_adjustment": warrantsAdjustment,
		"decision_made":      "true",
	}

	if !warrantsAdjustment {
		result["message"] = "No adjustment needed. The prompt performs well."
	} else {
		// Capture hypotheses and recommendations if provided
		if hypotheses, ok := args["hypotheses"].(string); ok && hypotheses != "" {
			result["hypotheses"] = hypotheses
		}
		if recommendations, ok := args["full_recommendations"].(string); ok && recommendations != "" {
			result["recommendations"] = recommendations
		}
		result["message"] = "Recommendations captured. Proceeding to apply changes."
	}

	return result, nil
}
