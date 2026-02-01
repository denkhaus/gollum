// Package manager provides prompt management with ID-based access and lazy bootstrapping.
package manager

import (
	"context"
	"embed"
	"sync"

	"github.com/denkhaus/gollum/pkg/prompt"
	promptstore "github.com/denkhaus/gollum/pkg/prompt/store"
)

//go:embed templates/*.md
var promptTemplates embed.FS

// PromptManager provides prompt management services.
type PromptManager interface {
	// ID-based prompt access with store integration
	GetPromptByID(ctx context.Context, id string) (*prompt.Prompt, error)
	GetPromptWithContext(ctx context.Context, id string, renderCtx *prompt.RenderContext) (string, error)
	SetPrompt(ctx context.Context, id string, content string, name string) (*prompt.Prompt, error)
	DeletePrompt(ctx context.Context, id string) error
	ListPrompts(ctx context.Context, filter *prompt.ListFilter) ([]*prompt.Prompt, error)
	RenderPrompt(ctx context.Context, p *prompt.Prompt, renderCtx *prompt.RenderContext) (string, error)
	GetStore() promptstore.PromptStore

	// Existing methods for backward compatibility
	GetCompacterPrompt(data any) (string, error)
	GetSystemPrompt() (string, error)
	GetSupervisorPrompt() (string, error)
	GetSubagentPrompt(role, description string) (string, error)
}

type promptManager struct {
	store          promptstore.PromptStore
	systemOnce     sync.Once
	supervisorOnce sync.Once
	compacterOnce  sync.Once
	subagentOnce   sync.Once
}

// NewPromptManager creates a new Manager instance with PromptStore.
func NewPromptManager(store promptstore.PromptStore) PromptManager {
	return &promptManager{
		store: store,
	}
}
