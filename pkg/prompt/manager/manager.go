// Package manager provides prompt management with ID-based access and lazy bootstrapping.
package manager

import (
	"context"
	"embed"

	"github.com/denkhaus/gollum/pkg/prompt"
	promptstore "github.com/denkhaus/gollum/pkg/prompt/store"
	"github.com/denkhaus/gollum/pkg/skills"
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

	// Workspace context management
	SetWorkspaceContext(ctx *prompt.WorkspaceContext) error
	GetWorkspaceContext() *prompt.WorkspaceContext
	UpdateWorkspaceFromSkillService(ctx context.Context, currentPath string) error

	// Existing methods for backward compatibility
	GetCompacterPrompt(data any) (string, error)
	GetSystemPrompt() (string, error)
	GetSupervisorPrompt() (string, error)
	GetSubagentPrompt(role, description string) (string, error)
}

type promptManager struct {
	store     promptstore.PromptStore
	skillSvc  skills.SkillService
	workspace *prompt.WorkspaceContext
}

// NewPromptManager creates a new Manager instance with PromptStore and optional SkillService.
func NewPromptManager(store promptstore.PromptStore, skillSvc skills.SkillService) PromptManager {
	return &promptManager{
		store:     store,
		skillSvc:  skillSvc,
		workspace: &prompt.WorkspaceContext{},
	}
}

// SetWorkspaceContext updates the workspace context for prompt rendering.
func (p *promptManager) SetWorkspaceContext(ctx *prompt.WorkspaceContext) error {
	if ctx == nil {
		p.workspace = &prompt.WorkspaceContext{}
	} else {
		p.workspace = ctx
	}
	return nil
}

// GetWorkspaceContext returns the current workspace context.
func (p *promptManager) GetWorkspaceContext() *prompt.WorkspaceContext {
	return p.workspace
}

// UpdateWorkspaceFromSkillService discovers skills and updates the workspace context.
// If skillSvc is nil, this is a no-op.
func (p *promptManager) UpdateWorkspaceFromSkillService(ctx context.Context, currentPath string) error {
	if p.skillSvc == nil {
		return nil
	}

	// Discover skills in the workspace
	if err := p.skillSvc.Discover(ctx); err != nil {
		return err
	}

	// Get discovered skills
	discoveredSkills := p.skillSvc.List()

	// Build skill info list
	skillInfos := make([]prompt.SkillInfo, 0, len(discoveredSkills))
	for _, skill := range discoveredSkills {
		skillInfos = append(skillInfos, prompt.SkillInfo{
			Name:        skill.Name,
			Description: skill.Description,
			Location:    skill.FilePath,
		})
	}

	// Get skills as XML
	skillsXML := discoveredSkills.ToPromptXML()

	// Update workspace context
	p.workspace = &prompt.WorkspaceContext{
		CurrentPath: currentPath,
		Skills:      skillInfos,
		SkillsXML:   skillsXML,
	}

	return nil
}
