// Package manager provides template rendering for prompts.
package manager

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/denkhaus/gollum/pkg/prompt"
	promptstore "github.com/denkhaus/gollum/pkg/prompt/store"
)

// partialsCache caches parsed partials templates
var partialsCache struct {
	once      sync.Once
	templates *template.Template
	err       error
}

// templateNameMap maps prompt IDs to their template names
var templateNameMap = map[prompt.PromptID]string{
	prompt.PromptIDSubagentSystem:   "systemprompt",
	prompt.PromptIDSupervisorSystem: "supervisorprompt",
	prompt.PromptIDCompacter:        "compacter",
	prompt.PromptIDSubagentTask:     "subagenttaskprompt",
	// Optimizer templates (not loaded from store, embedded only)
	prompt.PromptIDOptimizerGradient:     "optimizergradientprompt",
	prompt.PromptIDOptimizerGradientMeta: "optimizergradientmetaprompt",
	prompt.PromptIDOptimizerMeta:         "optimizermetapromptprompt",
	prompt.PromptIDOptimizerMemory:       "optimizerpromptmemory",
}

// extractBaseID extracts the base prompt ID from a versioned ID
// e.g., "system@1.0.0" -> "system", "system" -> "system"
func extractBaseID(id string) prompt.PromptID {
	if idx := strings.Index(id, "@"); idx != -1 {
		return prompt.PromptID(id[:idx])
	}
	return prompt.PromptID(id)
}

// loadPartials loads partial templates from embedded FS
func loadPartials() (*template.Template, error) {
	partialsCache.once.Do(func() {
		// Parse all partials from the embedded FS
		partialsCache.templates, partialsCache.err = template.ParseFS(promptTemplates, "templates/partials/*.md")
	})
	return partialsCache.templates, partialsCache.err
}

// RenderPrompt renders a prompt template with the given context
func (p *promptManager) RenderPrompt(ctx context.Context, prompt *prompt.Prompt, renderCtx *prompt.RenderContext) (string, error) {
	_ = ctx // Reserved for future use (logging, tracing, cancellation)
	// Check for nil prompt
	if prompt == nil {
		return "", fmt.Errorf("prompt cannot be nil")
	}

	// Load partials
	partials, err := loadPartials()
	if err != nil {
		return "", fmt.Errorf("failed to load partials: %w", err)
	}

	// Parse template with partials
	tmpl, err := partials.Clone()
	if err != nil {
		return "", fmt.Errorf("failed to clone partials: %w", err)
	}

	tmpl, err = tmpl.New(prompt.ID).Parse(prompt.Content)
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
func (p *promptManager) GetPromptWithContext(ctx context.Context, id prompt.PromptID, renderCtx *prompt.RenderContext) (string, error) {
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

	// Workspace context - comes from renderCtx.Workspace (managed by WorkspaceService)
	if renderCtx.Workspace != nil {
		data["Workspace"] = renderCtx.Workspace
	}

	return data
}
