// Package prompt provides prompt management and template rendering for agent system prompts.
package prompt

import (
	"context"
	"embed"
	"sync"

	"github.com/samber/do/v2"
)

//go:embed templates/*.md
var promptTemplates embed.FS

// PromptStore defines the contract for prompt persistence operations.
// This is a local copy to avoid import cycle with store package.
type PromptStore interface {
	SaveNewVersion(ctx context.Context, baseID string, content string, name string) (*Prompt, error)
	Load(ctx context.Context, id string) (*Prompt, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *ListFilter) ([]*Prompt, error)
	Exists(ctx context.Context, id string) (bool, error)
}

type (
	promptManager struct {
		store          PromptStore
		systemOnce     sync.Once
		supervisorOnce sync.Once
		compacterOnce  sync.Once
		subagentOnce   sync.Once
	}

	// PromptManager provides prompt rendering services for agents.
	//revive:disable-next-line:exported
	PromptManager interface {
		// New methods for ID-based prompt access with store integration
		GetPromptByID(ctx context.Context, id string) (*Prompt, error)
		GetPromptWithContext(ctx context.Context, id string, renderCtx *RenderContext) (string, error)
		SetPrompt(ctx context.Context, id string, content string, name string) (*Prompt, error)
		DeletePrompt(ctx context.Context, id string) error
		ListPrompts(ctx context.Context, filter *ListFilter) ([]*Prompt, error)
		RenderPrompt(ctx context.Context, p *Prompt, renderCtx *RenderContext) (string, error)
		GetStore() PromptStore

		// Existing methods for backward compatibility
		GetCompacterPrompt(data any) (string, error)
		GetSystemPrompt() (string, error)
		GetSupervisorPrompt() (string, error)
		GetSubagentPrompt(role, description string) (string, error)
	}
)

// NewPromptManager creates a new Manager instance for prompt rendering.
func NewPromptManager(injector do.Injector) (PromptManager, error) {
	// Note: store will be injected via separate provider to avoid import cycles
	// For now, initialize without store
	pm := &promptManager{
		store: nil,
	}
	return pm, nil
}

// SetStore sets the prompt store (called by DI provider after initialization)
func (p *promptManager) SetStore(store PromptStore) {
	p.store = store
}

// Backward-compatible wrapper methods are implemented in manager_backward_compat.go
// These delegate to the new store-based API while maintaining existing signatures.
