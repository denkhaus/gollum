package executor

import (
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
)

// executeLLMStep executes an LLM step
func (p *flowExecutorImpl) executeLLMStep(step *flows.Step, _ string) error {
	// Find agent
	var agent *flows.Agent
	for i := range p.flow.Agents {
		if p.flow.Agents[i].Name == step.Agent {
			agent = &p.flow.Agents[i]
			break
		}
	}

	if agent == nil {
		return fmt.Errorf("agent not found: %s", step.Agent)
	}

	// Substitute variables in prompt
	prompt := SubstituteTemplate(p.ctx, step.Prompt)

	// Parse tools
	tools := p.parseTools(step.Tools)

	// TODO: Integrate with actual LLM execution
	// For now, this is a placeholder
	fmt.Printf("[LLM] Agent: %s, Model: %s\n", agent.Name, agent.Model)
	fmt.Printf("[LLM] Prompt: %s\n", prompt)
	fmt.Printf("[LLM] Tools: %s\n", strings.Join(tools, ", "))

	return fmt.Errorf("LLM execution not yet implemented")
}

// parseTools parses tools attribute into tool names
func (p *flowExecutorImpl) parseTools(toolsStr string) []string {
	if toolsStr == "" {
		return nil
	}

	return strings.Split(toolsStr, ",")
}
