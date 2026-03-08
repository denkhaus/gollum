package executor

import (
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
)

// executeLLMStep executes an LLM step
func (e *Executor) executeLLMStep(step *flows.Step, stateName string) error {
	// Find agent
	var agent *flows.Agent
	for i := range e.flow.Agents {
		if e.flow.Agents[i].Name == step.Agent {
			agent = &e.flow.Agents[i]
			break
		}
	}

	if agent == nil {
		return fmt.Errorf("agent not found: %s", step.Agent)
	}

	// Substitute variables in prompt
	prompt := SubstituteTemplate(e.ctx, step.Prompt)

	// Parse tools
	tools := e.parseTools(step.Tools)

	// TODO: Integrate with actual LLM execution
	// For now, this is a placeholder
	fmt.Printf("[LLM] Agent: %s, Model: %s\n", agent.Name, agent.Model)
	fmt.Printf("[LLM] Prompt: %s\n", prompt)
	fmt.Printf("[LLM] Tools: %s\n", strings.Join(tools, ", "))

	return fmt.Errorf("LLM execution not yet implemented")
}

// parseTools parses tools attribute into tool names
func (e *Executor) parseTools(toolsStr string) []string {
	if toolsStr == "" {
		return nil
	}

	return strings.Split(toolsStr, ",")
}
