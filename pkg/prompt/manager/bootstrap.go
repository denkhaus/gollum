// Package manager provides lazy bootstrapping for built-in prompts.
package manager

import (
	"context"
	"fmt"
	"sync"

	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/shared"
)

// Bootstrap entries for each built-in prompt.
var (
	systemOnce                sync.Once
	supervisorOnce            sync.Once
	compacterOnce             sync.Once
	subagentOnce              sync.Once
	optimizerGradientOnce     sync.Once
	optimizerGradientMetaOnce sync.Once
	optimizerMetaOnce         sync.Once
	optimizerMemoryOnce       sync.Once
)

// templateInfo maps built-in prompt IDs to their template metadata.
type templateInfo struct {
	file string
	name string
	once *sync.Once
}

// templateRegistry maps built-in prompt IDs to their template entries.
var templateRegistry = map[prompt.PromptID]templateInfo{
	prompt.PromptIDSubagentSystem:        {"subagent_system_prompt.md", "System Prompt", &systemOnce},
	prompt.PromptIDSupervisorSystem:      {"supervisor_system_prompt.md", "Supervisor System Prompt", &supervisorOnce},
	prompt.PromptIDCompacter:             {"compacter_prompt.md", "Compacter Prompt", &compacterOnce},
	prompt.PromptIDSubagentTask:          {"subagent_task_prompt.md", "Subagent Task Prompt", &subagentOnce},
	prompt.PromptIDOptimizerGradient:     {"optimizer_gradient_prompt.md", "Optimizer Gradient Reflection Prompt", &optimizerGradientOnce},
	prompt.PromptIDOptimizerGradientMeta: {"optimizer_gradient_metaprompt.md", "Optimizer Gradient Metaprompt", &optimizerGradientMetaOnce},
	prompt.PromptIDOptimizerMeta:         {"optimizer_metaprompt.md", "Optimizer Metaprompt", &optimizerMetaOnce},
	prompt.PromptIDOptimizerMemory:       {"optimizer_prompt_memory.md", "Optimizer Prompt Memory", &optimizerMemoryOnce},
}

// isBuiltinID checks if an ID is a built-in prompt.
func (p *promptManager) isBuiltinID(id prompt.PromptID) bool {
	_, ok := templateRegistry[id]
	return ok
}

// getBootstrapOnce returns the sync.Once for a given built-in ID.
func (p *promptManager) getBootstrapOnce(id prompt.PromptID) (*sync.Once, error) {
	info, ok := templateRegistry[id]
	if !ok {
		return nil, fmt.Errorf("unknown built-in prompt ID: %s", id)
	}
	return info.once, nil
}

// getTemplateInfo returns the template file and name for a built-in ID.
func (p *promptManager) getTemplateInfo(id prompt.PromptID) (string, string, error) {
	info, ok := templateRegistry[id]
	if !ok {
		return "", "", fmt.Errorf("no template info for built-in prompt ID: %s", id)
	}
	return info.file, info.name, nil
}

// bootstrapBuiltinPrompt loads a built-in prompt from embedded FS and saves to store.
func (p *promptManager) bootstrapBuiltinPrompt(ctx context.Context, baseID prompt.PromptID, once *sync.Once, templateFile string, promptName string) (*prompt.Prompt, error) {
	var bootstrapped *prompt.Prompt
	var bootstrapErr error

	once.Do(func() {
		// Read template from embedded FS
		content, err := promptTemplates.ReadFile("templates/" + templateFile)
		if err != nil {
			bootstrapErr = fmt.Errorf("failed to read template %s: %w", templateFile, err)
			return
		}

		// Save to store with IsBuiltin=true (built-in prompts cannot be deleted)
		saved, err := p.store.SaveBuiltinVersion(ctx, baseID, string(content), promptName)
		if err != nil {
			bootstrapErr = fmt.Errorf("failed to save bootstrapped prompt to store: %w", err)
			return
		}

		bootstrapped = saved
	})

	if bootstrapErr != nil {
		return nil, bootstrapErr
	}

	return bootstrapped, nil
}

// GetSubagentTaskPrompt returns the subagent task prompt with role and description
func (p *promptManager) GetSubagentTaskPrompt(role, description string) (string, error) {
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
