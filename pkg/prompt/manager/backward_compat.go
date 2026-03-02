// Package manager provides backward-compatible wrapper methods for existing PromptManager API.
package manager

import (
	"context"

	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/shared"
)

// GetCompacterPrompt returns the compacter prompt with data values rendered
func (p *promptManager) GetCompacterPrompt(data any) (string, error) {
	ctx := context.Background()

	// Convert data to Values map
	var values map[string]interface{}
	switch v := data.(type) {
	case map[string]interface{}:
		// If already a map, use it directly
		values = v
	default:
		// For other types, wrap in a "data" key
		values = map[string]interface{}{
			"data": data,
		}
	}

	renderCtx := &prompt.RenderContext{
		Values: values,
	}
	return p.GetPromptWithContext(ctx, prompt.PromptIDCompacter, renderCtx)
}

// GetSystemPrompt returns the system prompt
func (p *promptManager) GetSystemPrompt() (string, error) {
	ctx := context.Background()
	return p.GetPromptWithContext(ctx, prompt.PromptIDSubagentSystem, nil)
}

// GetSupervisorPrompt returns the supervisor prompt
func (p *promptManager) GetSupervisorPrompt() (string, error) {
	ctx := context.Background()
	return p.GetPromptWithContext(ctx, prompt.PromptIDSupervisorSystem, nil)
}

// GetSubagentPrompt returns the subagent prompt with role and description
func (p *promptManager) GetSubagentPrompt(role, description string) (string, error) {
	ctx := context.Background()
	renderCtx := &prompt.RenderContext{
		SubAgent: &prompt.SubAgentContext{
			Role:            role,
			Description:     description,
			SpawnAgentTool:  shared.ToolNameSpawnAgent,
			RemoveAgentTool: shared.ToolNameRemoveAgent,
			ResumeAgentTool: shared.ToolNameResumeAgent,
			AgentOutputTool: shared.ToolNameAgentOutput,
			ListAgentsTool:  shared.ToolNameListAgents,
		},
	}
	return p.GetPromptWithContext(ctx, prompt.PromptIDSubagentTask, renderCtx)
}
