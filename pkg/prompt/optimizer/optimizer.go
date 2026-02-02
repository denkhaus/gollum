// Package optimizer provides prompt optimization strategies for improving agent prompts
// through trajectory analysis and feedback processing.
package optimizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-mizutani/gollem"
)

// thinkTool implements the "think" tool for recording reasoning during gradient optimization.
type thinkTool struct{}

// Spec returns the tool specification.
func (t *thinkTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "think",
		Description: "A reflection tool for reasoning over complexities and hypothesizing fixes.",
		Parameters: map[string]*gollem.Parameter{
			"thought": {
				Type:        gollem.TypeString,
				Description: "The thought or reasoning to process",
				Required:    []string{"thought"},
			},
		},
		Required: []string{"thought"},
	}
}

// Run executes the think tool.
func (t *thinkTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	_, ok := args["thought"].(string)
	if !ok {
		return nil, fmt.Errorf("thought must be a string")
	}
	return map[string]any{
		"status": "thought recorded",
	}, nil
}

// critiqueTool implements the "critique" tool for diagnosing flaws during gradient optimization.
type critiqueTool struct{}

// Spec returns the tool specification.
func (t *critiqueTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "critique",
		Description: "A critique tool for diagnosing flaws in reasoning.",
		Parameters: map[string]*gollem.Parameter{
			"criticism": {
				Type:        gollem.TypeString,
				Description: "The critique or criticism of the current reasoning",
				Required:    []string{"criticism"},
			},
		},
		Required: []string{"criticism"},
	}
}

// Run executes the critique tool.
func (t *critiqueTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	_, ok := args["criticism"].(string)
	if !ok {
		return nil, fmt.Errorf("criticism must be a string")
	}
	return map[string]any{
		"status": "critique recorded",
	}, nil
}

// recommendTool implements the "recommend" tool for providing optimization recommendations.
type recommendTool struct{}

// Spec returns the tool specification.
func (t *recommendTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "recommend",
		Description: "Decides whether a prompt should be adjusted and provides recommendations.",
		Parameters: map[string]*gollem.Parameter{
			"warrants_adjustment": {
				Type:        gollem.TypeBoolean,
				Description: "Whether the prompt warrants adjustment based on the analysis",
			},
			"hypotheses": {
				Type:        gollem.TypeString,
				Description: "Hypotheses about why the current prompt underperforms",
			},
			"full_recommendations": {
				Type:        gollem.TypeString,
				Description: "Detailed recommendations for prompt improvement",
			},
		},
		Required: []string{"warrants_adjustment"},
	}
}

// Run executes the recommend tool.
func (t *recommendTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	_, ok := args["warrants_adjustment"].(bool)
	if !ok {
		return nil, fmt.Errorf("warrants_adjustment must be a boolean")
	}
	return map[string]any{
		"status": "recommendations recorded",
	}, nil
}


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
	prompt := DEFAULT_GRADIENT_PROMPT

	// Replace placeholders
	prompt = strings.ReplaceAll(prompt, "{prompt}", input.Prompt)
	prompt = strings.ReplaceAll(prompt, "{trajectories}", FormatSessions(input.Trajectories))

	updateInstructions := input.UpdateInstructions
	if updateInstructions == "" {
		updateInstructions = "No specific instructions provided"
	}
	prompt = strings.ReplaceAll(prompt, "{update_instructions}", updateInstructions)

	return prompt
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

	// Build metaprompt
	metaprompt := DEFAULT_GRADIENT_METAPROMPT
	metaprompt = strings.ReplaceAll(metaprompt, "{current_prompt}", currentPrompt)
	metaprompt = strings.ReplaceAll(metaprompt, "{hypotheses}", hypotheses)
	metaprompt = strings.ReplaceAll(metaprompt, "{recommendations}", recommendations)

	// Create new agent (no tools needed for phase 2)
	agent := gollem.New(o.client)

	// Generate updated prompt
	resp, err := agent.Execute(ctx, gollem.Text(metaprompt))
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
	prompt := DEFAULT_METAPROMPT

	// Replace placeholders
	prompt = strings.ReplaceAll(prompt, "{prompt}", input.Prompt)
	prompt = strings.ReplaceAll(prompt, "{trajectories}", FormatSessions(input.Trajectories))

	updateInstructions := input.UpdateInstructions
	if updateInstructions == "" {
		updateInstructions = "No specific instructions provided"
	}
	prompt = strings.ReplaceAll(prompt, "{update_instructions}", updateInstructions)

	return prompt
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
	prompt := DEFAULT_PROMPT_MEMORY

	// Replace placeholders
	prompt = strings.ReplaceAll(prompt, "{current_prompt}", input.Prompt)

	// Use first trajectory for single-shot
	trajectory := ""
	if len(input.Trajectories) > 0 {
		trajectory = input.Trajectories[0].FormatForLLM()
	}
	prompt = strings.ReplaceAll(prompt, "{trajectory}", trajectory)

	feedback := input.Feedback
	if feedback == "" {
		feedback = "No feedback provided"
	}
	prompt = strings.ReplaceAll(prompt, "{feedback}", feedback)

	instructions := input.UpdateInstructions
	if instructions == "" {
		instructions = "No specific instructions provided"
	}
	prompt = strings.ReplaceAll(prompt, "{instructions}", instructions)

	return prompt
}

// isNoAdjustmentResponse checks if the response indicates no adjustment is needed.
func (o *promptMemoryOptimizer) isNoAdjustmentResponse(response string) bool {
	lower := strings.ToLower(response)
	noChangeIndicators := []string{
		"no adjustment",
		"warrants_adjustment = false",
		"warrants_adjustment=false",
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
	if strings.Contains(strings.ToLower(response), "warrants_adjustment") {
		// Find the prompt after the flag
		lines := strings.Split(response, "\n")
		var promptLines []string
		foundPrompt := false

		for _, line := range lines {
			if foundPrompt {
				promptLines = append(promptLines, line)
			} else if strings.Contains(strings.ToLower(line), "warrants_adjustment") {
				foundPrompt = true
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
