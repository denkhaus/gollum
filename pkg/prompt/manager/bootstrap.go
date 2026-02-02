// Package manager provides lazy bootstrapping for built-in prompts.
package manager

import (
	"context"
	"fmt"
	"sync"

	"github.com/denkhaus/gollum/pkg/prompt"
)

// templateInfo maps built-in prompt IDs to their template files
var templateInfoMap = map[string]struct {
	file string
	name string
}{
	prompt.PromptIDSystem:              {"subagent_system_prompt.md", "System Prompt"},
	prompt.PromptIDSupervisor:          {"supervisor_system_prompt.md", "Supervisor System Prompt"},
	prompt.PromptIDCompacter:           {"compacter_prompt.md", "Compacter Prompt"},
	prompt.PromptIDSubagent:            {"subagent_task_prompt.md", "Subagent Task Prompt"},
	prompt.PromptIDOptimizerGradient:   {"optimizer_gradient_prompt.md", "Optimizer Gradient Reflection Prompt"},
	prompt.PromptIDOptimizerGradientMeta: {"optimizer_gradient_metaprompt.md", "Optimizer Gradient Metaprompt"},
	prompt.PromptIDOptimizerMeta:        {"optimizer_metaprompt.md", "Optimizer Metaprompt"},
	prompt.PromptIDOptimizerMemory:     {"optimizer_prompt_memory.md", "Optimizer Prompt Memory"},
}

// isBuiltinID checks if an ID is a built-in prompt constant
func (p *promptManager) isBuiltinID(id string) bool {
	switch id {
	case prompt.PromptIDSystem, prompt.PromptIDSupervisor, prompt.PromptIDCompacter, prompt.PromptIDSubagent,
		prompt.PromptIDOptimizerGradient, prompt.PromptIDOptimizerGradientMeta, prompt.PromptIDOptimizerMeta, prompt.PromptIDOptimizerMemory:
		return true
	default:
		return false
	}
}

// getBootstrapOnce returns the sync.Once for a given built-in ID
func (p *promptManager) getBootstrapOnce(id string) (*sync.Once, error) {
	switch id {
	case prompt.PromptIDSystem:
		return &p.systemOnce, nil
	case prompt.PromptIDSupervisor:
		return &p.supervisorOnce, nil
	case prompt.PromptIDCompacter:
		return &p.compacterOnce, nil
	case prompt.PromptIDSubagent:
		return &p.subagentOnce, nil
	case prompt.PromptIDOptimizerGradient:
		return &p.optimizerGradientOnce, nil
	case prompt.PromptIDOptimizerGradientMeta:
		return &p.optimizerGradientMetaOnce, nil
	case prompt.PromptIDOptimizerMeta:
		return &p.optimizerMetaOnce, nil
	case prompt.PromptIDOptimizerMemory:
		return &p.optimizerMemoryOnce, nil
	default:
		return nil, fmt.Errorf("unknown built-in prompt ID: %s", id)
	}
}

// getTemplateInfo returns the template file and name for a built-in ID
func (p *promptManager) getTemplateInfo(id string) (string, string, error) {
	info, ok := templateInfoMap[id]
	if !ok {
		return "", "", fmt.Errorf("no template info for built-in prompt ID: %s", id)
	}
	return info.file, info.name, nil
}

// bootstrapBuiltinPrompt loads a built-in prompt from embedded FS and saves to store
func (p *promptManager) bootstrapBuiltinPrompt(ctx context.Context, baseID string, once *sync.Once, templateFile string, promptName string) (*prompt.Prompt, error) {
	var bootstrapped *prompt.Prompt
	var bootstrapErr error

	once.Do(func() {
		// Read template from embedded FS
		content, err := promptTemplates.ReadFile("templates/"+templateFile)
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
