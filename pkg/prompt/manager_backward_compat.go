// Package prompt provides backward-compatible wrapper methods for the PromptManager.
// These methods delegate to the new store-based API while maintaining existing signatures.
package prompt

import (
	"context"

	"github.com/denkhaus/gollum/pkg/shared"
)

// GetCompacterPrompt returns the compacter prompt with optional data.
// Backward-compatible wrapper that uses the new store-based API.
func (p *promptManager) GetCompacterPrompt(data any) (string, error) {
	ctx := context.Background()

	// Build RenderContext with data in Values
	renderCtx := &RenderContext{
		Values: make(map[string]interface{}),
	}

	// If data is a map, copy it to Values
	if m, ok := data.(map[string]interface{}); ok {
		for k, v := range m {
			renderCtx.Values[k] = v
		}
	} else if data != nil {
		// For non-map data, put it under "Data" key
		renderCtx.Values["Data"] = data
	}

	// Use new API to get and render prompt
	return p.GetPromptWithContext(ctx, PromptIDCompacter, renderCtx)
}

// GetSystemPrompt returns the system prompt.
// Backward-compatible wrapper that uses the new store-based API.
func (p *promptManager) GetSystemPrompt() (string, error) {
	ctx := context.Background()
	// System prompt doesn't need context variables
	return p.GetPromptWithContext(ctx, PromptIDSystem, nil)
}

// GetSupervisorPrompt returns the supervisor prompt.
// Backward-compatible wrapper that uses the new store-based API.
func (p *promptManager) GetSupervisorPrompt() (string, error) {
	ctx := context.Background()
	// Supervisor prompt doesn't need context variables
	return p.GetPromptWithContext(ctx, PromptIDSupervisor, nil)
}

// GetSubagentPrompt returns a subagent prompt with the given role and description.
// Backward-compatible wrapper that uses the new store-based API.
func (p *promptManager) GetSubagentPrompt(role, description string) (string, error) {
	ctx := context.Background()

	// Build RenderContext with SubAgentContext
	renderCtx := &RenderContext{
		SubAgent: &SubAgentContext{
			Role:            role,
			Description:     description,
			SpawnAgentTool:  shared.ToolNameSpawnAgent,
			RemoveAgentTool: shared.ToolNameRemoveAgent,
			ResumeAgentTool: shared.ToolNameResumeAgent,
			AgentOutputTool: shared.ToolNameAgentOutput,
			ListAgentsTool:  shared.ToolNameListAgents,
		},
	}

	// Use new API to get and render prompt
	return p.GetPromptWithContext(ctx, PromptIDSubagent, renderCtx)
}
