// Package prompt provides lazy initialization for built-in prompts from embedded FS.
package prompt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
)

// Built-in prompt template filenames
const (
	templateSystem     = "templates/subagent_system_prompt.md"
	templateSupervisor = "templates/supervisor_system_prompt.md"
	templateCompacter  = "templates/compacter_prompt.md"
	templateSubagent   = "templates/subagent_task_prompt.md"
)

// Built-in prompt names
const (
	nameSystem     = "System Prompt"
	nameSupervisor = "Supervisor Prompt"
	nameCompacter  = "Compacter Prompt"
	nameSubagent   = "Subagent Task Prompt"
)

// bootstrapBuiltinPrompt lazy-loads a built-in prompt from embedded FS.
// Uses sync.Once to ensure single initialization per built-in prompt.
func (p *promptManager) bootstrapBuiltinPrompt(ctx context.Context, baseID string, once *sync.Once, templateFile string, promptName string) (*Prompt, error) {
	if p.store == nil {
		return nil, fmt.Errorf("prompt store not initialized")
	}

	var bootstrapErr error
	var bootstrappedPrompt *Prompt

	once.Do(func() {
		// Read template from embedded FS
		content, err := promptTemplates.ReadFile(templateFile)
		if err != nil {
			bootstrapErr = fmt.Errorf("failed to read built-in prompt template %s: %w", baseID, err)
			return
		}

		// Create prompt with IsBuiltin=true, Version=1.0.0
		now := time.Now()
		prompt := &Prompt{
			ID:        baseID + "@1.0.0",
			Name:      promptName,
			Content:   string(content),
			Context:   make(map[string]interface{}),
			Tags:      []string{baseID, baseID + "@latest"},
			CreatedAt: now,
			UpdatedAt: now,
			Version:   semver.New(1, 0, 0, "", ""),
			IsBuiltin: true,
		}

		// Save to store via SaveNewVersion
		_, err = p.store.SaveNewVersion(ctx, baseID, prompt.Content, prompt.Name)
		if err != nil {
			bootstrapErr = fmt.Errorf("failed to save built-in prompt %s to store: %w", baseID, err)
			return
		}

		// Load the saved prompt to get the stored version
		bootstrappedPrompt, err = p.store.Load(ctx, baseID)
		if err != nil {
			bootstrapErr = fmt.Errorf("failed to load bootstrapped prompt %s: %w", baseID, err)
			return
		}
	})

	if bootstrapErr != nil {
		return nil, bootstrapErr
	}

	// If bootstrappedPrompt is nil, try loading from store
	// (might have been bootstrapped by another goroutine)
	if bootstrappedPrompt == nil {
		bootstrappedPrompt, bootstrapErr = p.store.Load(ctx, baseID)
		if bootstrapErr != nil {
			return nil, bootstrapErr
		}
	}

	return bootstrappedPrompt, nil
}

// isBuiltinID checks if an ID corresponds to a built-in prompt
func (p *promptManager) isBuiltinID(id string) bool {
	switch id {
	case PromptIDSystem, PromptIDSupervisor, PromptIDCompacter, PromptIDSubagent:
		return true
	default:
		return false
	}
}

// getTemplateInfo returns the template file and name for a built-in prompt ID
func (p *promptManager) getTemplateInfo(baseID string) (templateFile string, promptName string, err error) {
	switch baseID {
	case PromptIDSystem:
		return templateSystem, nameSystem, nil
	case PromptIDSupervisor:
		return templateSupervisor, nameSupervisor, nil
	case PromptIDCompacter:
		return templateCompacter, nameCompacter, nil
	case PromptIDSubagent:
		return templateSubagent, nameSubagent, nil
	default:
		return "", "", fmt.Errorf("unknown built-in prompt ID: %s", baseID)
	}
}

// getBootstrapOnce returns the sync.Once for a given built-in prompt ID
func (p *promptManager) getBootstrapOnce(baseID string) (*sync.Once, error) {
	switch baseID {
	case PromptIDSystem:
		return &p.systemOnce, nil
	case PromptIDSupervisor:
		return &p.supervisorOnce, nil
	case PromptIDCompacter:
		return &p.compacterOnce, nil
	case PromptIDSubagent:
		return &p.subagentOnce, nil
	default:
		return nil, fmt.Errorf("unknown built-in prompt ID: %s", baseID)
	}
}
