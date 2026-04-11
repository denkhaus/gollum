package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/agents"
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
	prompt := p.ctx.SubstituteTemplate(step.Prompt)

	// Parse tools from step.Tools
	// Separate flow tools from built-in/MCP tools
	toolNames := p.parseToolNames(step.Tools)
	flowToolNames, allowedTools := p.separateFlowTools(toolNames)

	// Map flow Agent to shared.AgentConfig
	config := &shared.AgentConfig{
		ID:              uuid.New(),
		SystemPrompt:    agentConfig.Prompt,
		Role:            "flow-llm-step",
		Description:     fmt.Sprintf("LLM agent for flow %s, step %s", p.flow.Name, step.Name),
		LLMClientConfig: agentConfig.ToClientConfig(),
		OutputMode:      p.getOutputModeForStep(step), // Use verbose flag to control output
		Strategy:        simple.New(),
		AllowedTools:    allowedTools,
	}

	// Create agent using factory
	agent, err := p.agentFactory.CreateAgent(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Create and add flow tools (these are always available in flow execution)
	if len(flowToolNames) > 0 {
		if err := p.addFlowToolsToAgent(ctx, agent, flowToolNames); err != nil {
			return fmt.Errorf("failed to add flow tools: %w", err)
		}
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
	if step.Result != nil {
		// Handle simple assign
		if step.Result.AssignTo != "" {
			scope, fieldName, err := p.parseAssignTarget(step.Result.AssignTo)
			if err != nil {
				return fmt.Errorf("invalid assignTo: %w", err)
			}
			// For LLM steps, we assign the response text
			if scope == flows.FlowVariableScopeContext {
				if err := p.ctx.SetContextField(fieldName, responseText); err != nil {
					return fmt.Errorf("failed to set context field '%s': %w", fieldName, err)
				}
			} else {
				if err := p.ctx.SetOutputField(fieldName, responseText); err != nil {
					return fmt.Errorf("failed to set output field '%s': %w", fieldName, err)
				}
			}
		}
		// Handle path-based outputs
		for _, path := range step.Result.Paths {
			scope, fieldName, err := p.parseAssignTarget(path.AssignTo)
			if err != nil {
				return fmt.Errorf("invalid path assignTo: %w", err)
			}
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
			if scope == flows.FlowVariableScopeContext {
				if err := p.ctx.SetContextField(fieldName, value); err != nil {
					return fmt.Errorf("failed to set context field '%s': %w", fieldName, err)
				}
			} else {
				if err := p.ctx.SetOutputField(fieldName, value); err != nil {
					return fmt.Errorf("failed to set output field '%s': %w", fieldName, err)
				}
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

// separateFlowTools separates flow tools from built-in/MCP tools
// Flow tools are: set_context_field, set_output_field, get_context, emit_log, transition_to
func (p *flowExecutorImpl) separateFlowTools(toolNames []string) (flowTools []string, allowedTools []string) {
	flowToolSet := map[string]bool{
		"set_context_field": true,
		"set_output_field":  true,
		"get_context":       true,
		"emit_log":          true,
		"transition_to":     true,
	}

	for _, name := range toolNames {
		if flowToolSet[name] {
			flowTools = append(flowTools, name)
		} else {
			allowedTools = append(allowedTools, name)
		}
	}

	return flowTools, allowedTools
}

// addFlowToolsToAgent creates flow tools and adds them to the agent
// This is a workaround because flow tools require special handling (FlowContext)
func (p *flowExecutorImpl) addFlowToolsToAgent(ctx context.Context, agent shared.Agent, toolNames []string) error {
	var flowTools []gollem.Tool
	for _, toolName := range toolNames {
		tool, err := p.flowToolsProvider.CreateTool(agent, p, shared.ToolName(toolName))
		if err != nil {
			return fmt.Errorf("failed to create flow tool '%s': %w", toolName, err)
		}
		flowTools = append(flowTools, tool)
	}

	// Type assert to access internal tools field
	// This is necessary because flow tools need special handling
	if defAgent, ok := agent.(*agents.DefaultAgent); ok {
		defAgent.AddTools(flowTools)
	}

	return nil
}

// getOutputModeForStep determines the output mode based on step's verbose flag
func (p *flowExecutorImpl) getOutputModeForStep(step *flows.Step) shared.OutputMode {
	if step.Verbose {
		return shared.OutputModeFull // Show LLM response in logs
	}
	return shared.OutputModeSilent // Suppress LLM response (default)
}
