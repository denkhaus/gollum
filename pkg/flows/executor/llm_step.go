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
func (p *flowExecutorImpl) executeLLMStep(ctx context.Context, step *flows.Step, _ string) error {
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

	// Parse tools from step.Tools (comma-separated string like "bash,read_file")
	allowedTools := p.parseToolNames(step.Tools)

	// Map flow Agent to shared.AgentConfig
	config := &shared.AgentConfig{
		ID:              uuid.New(),
		SystemPrompt:    agentConfig.Prompt,
		Role:            "flow-llm-step",
		Description:     fmt.Sprintf("LLM agent for flow %s, step %s", p.flow.Name, step.Name),
		LLMClientConfig: agentConfig.ToClientConfig(),
		OutputMode:      shared.OutputModeSilent, // Suppress output during flow execution
		Strategy:        simple.New(),
		AllowedTools:    allowedTools,
	}

	// Create agent using factory
	agent, err := p.agentFactory.CreateAgent(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Execute with user prompt (pass ctx for enriched logging in tools)
	response, err := agent.Execute(ctx, gollem.Text(prompt))
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
			if err := p.ctx.SetOutputField(fieldName, responseText); err != nil {
				return fmt.Errorf("failed to set output field '%s': %w", fieldName, err)
			}
		}
		// Handle path-based outputs
		for _, path := range step.Output.Paths {
			fieldName := extractFieldName(path.Assign)
			var value string
			switch path.Path {
			case "text", "content", "response":
				value = responseText
			case "finish_reason":
				// Note: ExecuteResponse doesn't have FinishReason
				// The response was successful if we got here
				value = "success"
			default:
				continue
			}
			if err := p.ctx.SetOutputField(fieldName, value); err != nil {
				return fmt.Errorf("failed to set output field '%s': %w", fieldName, err)
			}
		}
	}

	return nil
}

// parseToolNames parses the comma-separated tools string and returns tool names
func (p *flowExecutorImpl) parseToolNames(toolsStr string) []string {
	if toolsStr == "" {
		return nil
	}

	var toolNames []string
	names := strings.Split(toolsStr, ",")
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			toolNames = append(toolNames, name)
		}
	}

	return toolNames
}
