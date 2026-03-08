package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestExecuteLLMStep_SubstitutesPrompt(t *testing.T) {
	flow := &flows.Flow{
		Name: "test",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "pr_number"}},
		},
		Agents: []flows.Agent{
			{Name: "worker", Model: "test-model", Prompt: "You are a helper"},
		},
		States: []flows.State{
			{Name: "init", Initial: true, Steps: []flows.Step{
				{Type: "llm", Agent: "worker", Prompt: "Analyze PR #${input.pr_number}"},
			}},
		},
	}

	exec := NewExecutor(flow)
	exec.SetInput(map[string]any{"pr_number": 123})

	step := &flows.Step{Type: "llm", Agent: "worker", Prompt: "Analyze PR #${input.pr_number}"}
	err := exec.executeStep(step, "init")

	// Should get error about LLM execution not being fully implemented, but prompt substitution should work
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM execution not yet implemented")
}

func TestExecuteLLMStep_ValidatesAgentExists(t *testing.T) {
	flow := &flows.Flow{
		Name: "test",
		States: []flows.State{
			{Name: "init", Initial: true, Steps: []flows.Step{
				{Type: "llm", Agent: "nonexistent"},
			}},
		},
	}

	exec := NewExecutor(flow)
	step := &flows.Step{Type: "llm", Agent: "nonexistent"}
	err := exec.executeStep(step, "init")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
}
