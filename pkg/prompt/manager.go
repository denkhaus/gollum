// Package prompt provides prompt management and template rendering for agent system prompts.
package prompt

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"sync"

	"github.com/denkhaus/gollum/pkg/shared"
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

// Pre-parsed templates for better performance
var (
	compacterTemplate        = template.Must(template.New("compacter").ParseFS(promptTemplates, "templates/compacter_prompt.md"))
	systemPromptTemplate     = template.Must(template.New("systemprompt").ParseFS(promptTemplates, "templates/subagent_system_prompt.md"))
	supervisorPromptTemplate = template.Must(template.New("supervisorprompt").ParseFS(promptTemplates, "templates/supervisor_system_prompt.md"))
	subagentTaskTemplate     = template.Must(template.New("subagenttaskprompt").ParseFS(promptTemplates, "templates/subagent_task_prompt.md"))
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

func (p *promptManager) GetCompacterPrompt(data any) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := compacterTemplate.Execute(buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to render compacter prompt: %v", err)
	}

	return buf.String(), nil
}

func (p *promptManager) GetSystemPrompt() (string, error) {
	buf := bytes.NewBuffer(nil)
	err := systemPromptTemplate.Execute(buf, nil)
	if err != nil {
		return "", fmt.Errorf("failed to render system prompt: %v", err)
	}

	return buf.String(), nil
}

func (p *promptManager) GetSupervisorPrompt() (string, error) {
	buf := bytes.NewBuffer(nil)
	err := supervisorPromptTemplate.Execute(buf, nil)
	if err != nil {
		return "", fmt.Errorf("failed to render supervisor prompt: %v", err)
	}

	return buf.String(), nil
}

// GetSubagentPrompt generates a specialized prompt for subagents with their role and description
func (p *promptManager) GetSubagentPrompt(role, description string) (string, error) {
	// Prepare context with role, description, and tool names
	ctx := SubAgentContext{
		Role:            role,
		Description:     description,
		SpawnAgentTool:  shared.ToolNameSpawnAgent,
		RemoveAgentTool: shared.ToolNameRemoveAgent,
		ResumeAgentTool: shared.ToolNameResumeAgent,
		AgentOutputTool: shared.ToolNameAgentOutput,
		ListAgentsTool:  shared.ToolNameListAgents,
	}

	buf := bytes.NewBuffer(nil)
	err := subagentTaskTemplate.Execute(buf, ctx)
	if err != nil {
		return "", fmt.Errorf("failed to render subagent task prompt: %v", err)
	}

	return buf.String(), nil
}

// New interface methods are implemented in manager_bootstrap.go and manager_store.go
