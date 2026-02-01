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

// GetPromptByID retrieves a prompt by ID from store or bootstraps built-in
func (p *promptManager) GetPromptByID(ctx context.Context, id string) (*prompt.Prompt, error) {
	// Check if ID is a built-in constant
	if p.isBuiltinID(id) {
		// Get bootstrap sync.Once for this built-in
		once, err := p.getBootstrapOnce(id)
		if err != nil {
			return nil, err
		}

		// Get template info
		templateFile, promptName, err := p.getTemplateInfo(id)
		if err != nil {
			return nil, err
		}

		// Bootstrap the built-in prompt
		_, err = p.bootstrapBuiltinPrompt(ctx, id, once, templateFile, promptName)
		if err != nil {
			return nil, fmt.Errorf("failed to bootstrap built-in prompt %s: %w", id, err)
		}
	}

	// Load from store
	loadedPrompt, err := p.store.Load(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load prompt %s: %w", id, err)
	}

	// Return nil, not error, if not found
	if loadedPrompt == nil {
		return nil, nil
	}

	return loadedPrompt, nil
}

// SetPrompt saves a new prompt version
func (p *promptManager) SetPrompt(ctx context.Context, id string, content string, name string) (*prompt.Prompt, error) {
	savedPrompt, err := p.store.SaveNewVersion(ctx, id, content, name)
	if err != nil {
		return nil, fmt.Errorf("failed to save prompt %s: %w", id, err)
	}

	return savedPrompt, nil
}

// DeletePrompt deletes a prompt (respects IsBuiltin flag)
func (p *promptManager) DeletePrompt(ctx context.Context, id string) error {
	// Load prompt to check IsBuiltin flag
	loadedPrompt, err := p.store.Load(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load prompt %s for deletion check: %w", id, err)
	}

	// If prompt exists and IsBuiltin is true, return error
	if loadedPrompt != nil && loadedPrompt.IsBuiltin {
		return ErrPromptIsBuiltin
	}

	// Call store Delete (returns nil if not found)
	err = p.store.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete prompt %s: %w", id, err)
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
