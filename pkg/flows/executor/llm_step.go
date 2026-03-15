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
func (p *flowExecutorImpl) executeLLMStep(step *flows.Step, _ string) error {
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

	// Parse tools from step specification
	tools, err := p.parseStepTools(step.Tools)
	if err != nil {
		return fmt.Errorf("failed to parse tools: %w", err)
	}

	// Map flow Agent to shared.AgentConfig
	config := &shared.AgentConfig{
		ID:              uuid.New(),
		SystemPrompt:    agentConfig.Prompt,
		Role:            "flow-llm-step",
		Description:     fmt.Sprintf("LLM agent for flow %s, step %s", p.flow.Name, step.Name),
		LLMClientConfig: agentConfig.ToClientConfig(),
		OutputMode:      shared.OutputModeSilent, // Suppress output during flow execution
		Strategy:        simple.New(),
		Tools:           tools,
		ToolSets:        nil,
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

// parseStepTools parses the comma-separated tools string and creates gollem.Tool instances
func (p *flowExecutorImpl) parseStepTools(toolsStr string) ([]gollem.Tool, error) {
	if toolsStr == "" {
		return nil, nil
	}

	agentID := uuid.Nil // Flow tools don't have a specific agent ID
	var tools []gollem.Tool

	toolNames := strings.Split(toolsStr, ",")
	for _, toolName := range toolNames {
		toolName = strings.TrimSpace(toolName)
		if toolName == "" {
			continue
		}

		// Create tool using flow tools provider
		tool, err := p.flowToolsProvider.CreateTool(agentID, p, shared.ToolName(toolName))
		if err != nil {
			return nil, fmt.Errorf("failed to create tool '%s': %w", toolName, err)
		}

		tools = append(tools, tool)
	}

	return tools, nil
}
