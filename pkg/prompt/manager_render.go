// Package prompt provides template rendering for prompts using text/template.
package prompt

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
)

// RenderPrompt renders a prompt template with the given render context.
// Uses text/template syntax for variable substitution.
func (p *promptManager) RenderPrompt(ctx context.Context, prompt *Prompt, renderCtx *RenderContext) (string, error) {
	if prompt == nil {
		return "", fmt.Errorf("prompt cannot be nil")
	}

	// Create new template from prompt content
	tmpl, err := template.New(prompt.ID).Parse(prompt.Content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template for prompt %s: %w", prompt.ID, err)
	}

	// Build template data map from renderCtx
	data := p.buildTemplateData(renderCtx)

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render prompt %s: %w", prompt.ID, err)
	}

	return buf.String(), nil
}

// buildTemplateData constructs the template data map from RenderContext.
// Data is built in layers: Values -> SubAgent -> Agent.
// Later layers can override earlier ones if there are key conflicts.
func (p *promptManager) buildTemplateData(renderCtx *RenderContext) map[string]interface{} {
	data := make(map[string]interface{})

	if renderCtx == nil {
		return data
	}

	// Add top-level values first
	if renderCtx.Values != nil {
		for k, v := range renderCtx.Values {
			data[k] = v
		}
	}

	// Add SubAgent context fields
	if renderCtx.SubAgent != nil {
		data["Role"] = renderCtx.SubAgent.Role
		data["Description"] = renderCtx.SubAgent.Description
		data["SpawnAgentTool"] = renderCtx.SubAgent.SpawnAgentTool
		data["RemoveAgentTool"] = renderCtx.SubAgent.RemoveAgentTool
		data["ResumeAgentTool"] = renderCtx.SubAgent.ResumeAgentTool
		data["AgentOutputTool"] = renderCtx.SubAgent.AgentOutputTool
		data["ListAgentsTool"] = renderCtx.SubAgent.ListAgentsTool
	}

	// Add Agent context fields
	if renderCtx.Agent != nil {
		data["AgentID"] = renderCtx.Agent.AgentID
		data["Task"] = renderCtx.Agent.Task
	}

	return data
}

// GetPromptWithContext retrieves a prompt by ID and renders it with the given context.
// This is a convenience method that combines GetPromptByID and RenderPrompt.
func (p *promptManager) GetPromptWithContext(ctx context.Context, id string, renderCtx *RenderContext) (string, error) {
	// Get the prompt
	prompt, err := p.GetPromptByID(ctx, id)
	if err != nil {
		return "", err
	}

	if prompt == nil {
		return "", ErrPromptNotFound
	}

	// Render the prompt
	rendered, err := p.RenderPrompt(ctx, prompt, renderCtx)
	if err != nil {
		return "", fmt.Errorf("failed to render prompt %s: %w", id, err)
	}

	return rendered, nil
}
