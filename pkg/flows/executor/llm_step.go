package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/strategy/simple"
)

// executeLLMStep executes an LLM step
func (p *flowExecutorImpl) executeLLMStep(step *flows.Step, stateName string) error {
	// Find agent configuration
	var agentConfig *flows.Agent
	for i := range p.flow.Agents {
		if p.flow.Agents[i].Name == step.Agent {
			agentConfig = &p.flow.Agents[i]
			break
		}
	}

	if agentConfig == nil {
		return fmt.Errorf("agent not found: %s", step.Agent)
	}

	// Substitute variables in prompt
	prompt := SubstituteTemplate(p.ctx, step.Prompt)

	// Map flow Agent to shared.AgentConfig
	config := &shared.AgentConfig{
		ID:           uuid.New(),
		SystemPrompt: agentConfig.Prompt,
		Role:         "flow-llm-step",
		Description:  fmt.Sprintf("LLM agent for flow %s, step %s", p.flow.Name, step.Name),
		LLMProvider:  p.inferLLMProvider(agentConfig.Model),
		OutputMode:   shared.OutputModeSilent, // Suppress output during flow execution
		Strategy:     simple.New(),
		Tools:        nil, // Tools can be added later if needed
		ToolSets:     nil,
	}

	// Create agent using factory
	agent, err := p.agentFactory.CreateAgent(context.Background(), config)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Execute with user prompt
	response, err := agent.Execute(context.Background(), gollem.Text(prompt))
	if err != nil {
		return fmt.Errorf("LLM execution failed: %w", err)
	}

	// Extract response text
	var responseText string
	if len(response.Texts) > 0 {
		responseText = response.Texts[0]
	}

	// Map result to output fields
	if step.Output != nil {
		// Handle simple assign
		if step.Output.Assign != "" {
			fieldName := extractFieldName(step.Output.Assign)
			// For LLM steps, we assign the response text
			p.ctx.SetOutputField(fieldName, responseText)
		}
		// Handle path-based outputs
		for _, path := range step.Output.Paths {
			fieldName := extractFieldName(path.Assign)
			switch path.Path {
			case "text", "content", "response":
				p.ctx.SetOutputField(fieldName, responseText)
			case "finish_reason":
				// Note: ExecuteResponse doesn't have FinishReason
				// The response was successful if we got here
				p.ctx.SetOutputField(fieldName, "success")
			}
		}
	}

	return nil
}

// inferLLMProvider infers the LLM provider from the model name
// Default to Anthropic if unknown
func (p *flowExecutorImpl) inferLLMProvider(model string) shared.LLMProvider {
	// Check model name patterns
	modelLower := strings.ToLower(model)

	// OpenAI models
	if strings.HasPrefix(modelLower, "gpt-") ||
		strings.HasPrefix(modelLower, "o1-") ||
		strings.Contains(modelLower, "openai") {
		return shared.LLMProviderOpenAI
	}

	// Gemini models
	if strings.HasPrefix(modelLower, "gemini-") ||
		strings.Contains(modelLower, "google") {
		return shared.LLMProviderGemini
	}

	// Claude models (default)
	// claude-3, claude-3.5, claude-3.7, claude-4, etc.
	if strings.HasPrefix(modelLower, "claude-") ||
		strings.HasPrefix(modelLower, "anthropic") {
		return shared.LLMProviderAnthropic
	}

	// Default to Anthropic
	return shared.LLMProviderAnthropic
}

// parseTools parses tools attribute into tool names
// Note: This is kept for compatibility but tools are not yet implemented for LLM steps
func (p *flowExecutorImpl) parseTools(toolsStr string) []string {
	if toolsStr == "" {
		return nil
	}

	return strings.Split(toolsStr, ",")
}
