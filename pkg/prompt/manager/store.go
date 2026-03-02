// Package manager provides PromptStore integration for prompt manager.
package manager

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/prompt"
	promptstore "github.com/denkhaus/gollum/pkg/prompt/store"
)

var (
	// ErrPromptNotFound is returned when a prompt is not found
	ErrPromptNotFound = promptstore.ErrPromptNotFound
	// ErrPromptIsBuiltin is returned when trying to delete a built-in prompt
	ErrPromptIsBuiltin = promptstore.ErrPromptIsBuiltin
)

// GetStore returns the underlying PromptStore for advanced operations
func (p *promptManager) GetStore() promptstore.PromptStore {
	return p.store
}

// GetPromptByID retrieves a prompt by ID from store or bootstraps built-in.
// Parameter baseID: the unversioned base identifier (e.g., prompt.PromptIDSubagentSystem).
func (p *promptManager) GetPromptByID(ctx context.Context, baseID prompt.PromptID) (*prompt.Prompt, error) {
	// Check if ID is a built-in constant
	if p.isBuiltinID(baseID) {
		// Get bootstrap sync.Once for this built-in
		once, err := p.getBootstrapOnce(baseID)
		if err != nil {
			return nil, err
		}

		// Get template info
		templateFile, promptName, err := p.getTemplateInfo(baseID)
		if err != nil {
			return nil, err
		}

		// Bootstrap the built-in prompt
		_, err = p.bootstrapBuiltinPrompt(ctx, baseID, once, templateFile, promptName)
		if err != nil {
			return nil, fmt.Errorf("failed to bootstrap built-in prompt %s: %w", baseID, err)
		}
	}

	// Resolve alias to get the actual versioned prompt
	// This works for both memory store (which stores aliases as keys) and file store (which scans for @latest tag)
	loadedPrompt, err := p.store.ResolveAlias(ctx, prompt.FromBaseID(baseID, "latest"))
	if err != nil {
		return nil, fmt.Errorf("failed to load prompt %s: %w", baseID, err)
	}

	// Return nil, not error, if not found
	if loadedPrompt == nil {
		return nil, nil
	}

	return loadedPrompt, nil
}

// SetPrompt saves a new prompt version.
// Parameter baseID: the unversioned base identifier (e.g., prompt.PromptIDSubagentSystem).
func (p *promptManager) SetPrompt(ctx context.Context, baseID prompt.PromptID, content string, name string) (*prompt.Prompt, error) {
	savedPrompt, err := p.store.SaveNewVersion(ctx, baseID, content, name)
	if err != nil {
		return nil, fmt.Errorf("failed to save prompt %s: %w", baseID, err)
	}

	return savedPrompt, nil
}

// DeletePrompt deletes a prompt (respects IsBuiltin flag).
// Parameter baseID: the unversioned base identifier (e.g., prompt.PromptIDSubagentSystem).
func (p *promptManager) DeletePrompt(ctx context.Context, baseID prompt.PromptID) error {
	// Load prompt to check IsBuiltin flag
	loadedPrompt, err := p.store.Load(ctx, prompt.FromBaseID(baseID, "latest"))
	if err != nil {
		return fmt.Errorf("failed to load prompt %s for deletion check: %w", baseID, err)
	}

	// If prompt exists and IsBuiltin is true, return error
	if loadedPrompt != nil && loadedPrompt.IsBuiltin {
		return ErrPromptIsBuiltin
	}

	// Call store Delete (returns nil if not found)
	err = p.store.Delete(ctx, prompt.FromBaseID(baseID, "latest"))
	if err != nil {
		return fmt.Errorf("failed to delete prompt %s: %w", baseID, err)
	}

	return nil
}

// ListPrompts lists prompts with filter
func (p *promptManager) ListPrompts(ctx context.Context, filter *prompt.ListFilter) ([]*prompt.Prompt, error) {
	// Convert prompt.ListFilter to store.ListFilter
	storeFilter := &promptstore.ListFilter{
		Tags: filter.Tags,
		IDs:  filter.IDs,
	}

	prompts, err := p.store.List(ctx, storeFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	return prompts, nil
}
