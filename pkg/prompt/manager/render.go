// Package manager provides template rendering for prompts.
package manager

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/denkhaus/gollum/pkg/prompt"
	promptstore "github.com/denkhaus/gollum/pkg/prompt/store"
)

// templateNameMap maps prompt IDs to their template names
var templateNameMap = map[string]string{
	prompt.PromptIDSystem:     "systemprompt",
	prompt.PromptIDSupervisor: "supervisorprompt",
	prompt.PromptIDCompacter:  "compacter",
	prompt.PromptIDSubagent:   "subagenttaskprompt",
	// Optimizer templates (not loaded from store, embedded only)
	"optimizer_gradient_prompt":     "optimizergradientprompt",
	"optimizer_gradient_metaprompt": "optimizergradientmetaprompt",
	"optimizer_metaprompt":          "optimizermetapromptprompt",
	"optimizer_prompt_memory":       "optimizerpromptmemory",
}

// extractBaseID extracts the base prompt ID from a versioned ID
// e.g., "system@1.0.0" -> "system", "system" -> "system"
func extractBaseID(id string) string {
	if idx := strings.Index(id, "@"); idx != -1 {
		return id[:idx]
	}
	return id
}

// RenderPrompt renders a prompt template with the given context
func (p *promptManager) RenderPrompt(ctx context.Context, prompt *prompt.Prompt, renderCtx *prompt.RenderContext) (string, error) {
	_ = ctx // Reserved for future use (logging, tracing, cancellation)
	// Check for nil prompt
	if prompt == nil {
		return "", fmt.Errorf("prompt cannot be nil")
	}

	// Parse template
	tmpl, err := template.New(prompt.ID).Parse(prompt.Content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template for prompt %s: %w", prompt.ID, err)
	}

	// Build template data map from renderCtx
	data := p.buildTemplateData(renderCtx)

	// Execute template
	var buf strings.Builder

	// Check if this is a named template (has define blocks)
	// Try to execute the specific named template if it exists
	// Extract base ID from versioned ID (e.g., "system@1.0.0" -> "system")
	baseID := extractBaseID(prompt.ID)
	if templateName, ok := templateNameMap[baseID]; ok {
		// Look up the named template and execute it
		namedTmpl := tmpl.Lookup(templateName)
		if namedTmpl != nil {
			if err := namedTmpl.Execute(&buf, data); err != nil {
				return "", fmt.Errorf("failed to execute named template %q for prompt %s: %w", templateName, prompt.ID, err)
			}
			return buf.String(), nil
		}
	}

	// Fall back to executing the root template
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template for prompt %s: %w", prompt.ID, err)
	}

	return buf.String(), nil
}

// GetPromptWithContext retrieves a prompt by ID and renders it with context
func (p *promptManager) GetPromptWithContext(ctx context.Context, id string, renderCtx *prompt.RenderContext) (string, error) {
	// Get prompt
	loadedPrompt, err := p.GetPromptByID(ctx, id)
	if err != nil {
		return "", err
	}
	if loadedPrompt == nil {
		return "", promptstore.ErrPromptNotFound
	}

	// Render prompt
	return p.RenderPrompt(ctx, loadedPrompt, renderCtx)
}

// buildTemplateData builds a template data map from RenderContext
func (p *promptManager) buildTemplateData(renderCtx *prompt.RenderContext) map[string]interface{} {
	data := make(map[string]interface{})

	// Handle nil context
	if renderCtx == nil {
		return data
	}

	// Top-level values
	if renderCtx.Values != nil {
		for k, v := range renderCtx.Values {
			data[k] = v
		}
	}

	// SubAgent context
	if renderCtx.SubAgent != nil {
		data["Role"] = renderCtx.SubAgent.Role
		data["Description"] = renderCtx.SubAgent.Description
		data["SpawnAgentTool"] = renderCtx.SubAgent.SpawnAgentTool
		data["RemoveAgentTool"] = renderCtx.SubAgent.RemoveAgentTool
		data["ResumeAgentTool"] = renderCtx.SubAgent.ResumeAgentTool
		data["AgentOutputTool"] = renderCtx.SubAgent.AgentOutputTool
		data["ListAgentsTool"] = renderCtx.SubAgent.ListAgentsTool
	}

	// Agent context
	if renderCtx.Agent != nil {
		data["AgentID"] = renderCtx.Agent.AgentID
		data["Task"] = renderCtx.Agent.Task
		// Message history for context awareness
		if len(renderCtx.Agent.MessageHistory) > 0 {
			data["MessageHistory"] = renderCtx.Agent.MessageHistory
		}
	}

	return data
}
