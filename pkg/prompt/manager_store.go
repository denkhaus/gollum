// Package prompt provides PromptStore integration for prompt manager.
package prompt

import (
	"context"
	"errors"
	"fmt"
)

// Local error definitions to avoid import cycle with store package
var (
	ErrPromptNotFound  = errors.New("prompt not found")
	ErrPromptIsBuiltin = errors.New("cannot delete built-in prompt")
)

// GetStore returns the underlying PromptStore for advanced operations
func (p *promptManager) GetStore() PromptStore {
	return p.store
}

// GetPromptByID retrieves a prompt by ID from store or bootstraps built-in
func (p *promptManager) GetPromptByID(ctx context.Context, id string) (*Prompt, error) {
	if p.store == nil {
		return nil, fmt.Errorf("prompt store not initialized")
	}

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
	prompt, err := p.store.Load(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load prompt %s: %w", id, err)
	}

	// Return nil, not error, if not found
	if prompt == nil {
		return nil, nil
	}

	return prompt, nil
}

// SetPrompt saves a new prompt version
func (p *promptManager) SetPrompt(ctx context.Context, id string, content string, name string) (*Prompt, error) {
	if p.store == nil {
		return nil, fmt.Errorf("prompt store not initialized")
	}

	prompt, err := p.store.SaveNewVersion(ctx, id, content, name)
	if err != nil {
		return nil, fmt.Errorf("failed to save prompt %s: %w", id, err)
	}

	return prompt, nil
}

// DeletePrompt deletes a prompt (respects IsBuiltin flag)
func (p *promptManager) DeletePrompt(ctx context.Context, id string) error {
	if p.store == nil {
		return fmt.Errorf("prompt store not initialized")
	}

	// Load prompt to check IsBuiltin flag
	prompt, err := p.store.Load(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load prompt %s for deletion check: %w", id, err)
	}

	// If prompt exists and IsBuiltin is true, return error
	if prompt != nil && prompt.IsBuiltin {
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
func (p *promptManager) ListPrompts(ctx context.Context, filter *ListFilter) ([]*Prompt, error) {
	if p.store == nil {
		return nil, fmt.Errorf("prompt store not initialized")
	}

	prompts, err := p.store.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	return prompts, nil
}

