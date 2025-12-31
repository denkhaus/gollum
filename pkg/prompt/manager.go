// Package prompt provides prompt management and template rendering for agent system prompts.
package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
)

//go:embed templates/*.md
var promptTemplates embed.FS

type (
	promptManager struct{}

	// PromptManager provides prompt rendering services for agents.
	PromptManager interface {
		GetCompacterPrompt(data any) (string, error)
		GetSystemPrompt() (string, error)
		GetSupervisorPrompt() (string, error)
		GetSubagentPrompt(role, description string) (string, error)
	}
)

// SubagentPromptContext contains data for rendering the subagent task prompt
type SubagentPromptContext struct {
	Role            string
	Description     string
	SpawnAgentTool  string
	RemoveAgentTool string
	ResumeAgentTool string
	AgentOutputTool string
	ListAgentsTool  string
}

// Pre-parsed templates for better performance
var (
	compacterTemplate        = template.Must(template.New("compacter").ParseFS(promptTemplates, "templates/compacter_prompt.md"))
	systemPromptTemplate     = template.Must(template.New("systemprompt").ParseFS(promptTemplates, "templates/subagent_system_prompt.md"))
	supervisorPromptTemplate = template.Must(template.New("supervisorprompt").ParseFS(promptTemplates, "templates/supervisor_system_prompt.md"))
	subagentTaskTemplate     = template.Must(template.New("subagenttaskprompt").ParseFS(promptTemplates, "templates/subagent_task_prompt.md"))
)

// NewPromptManager creates a new Manager instance for prompt rendering.
func NewPromptManager(_ do.Injector) (PromptManager, error) {
	pm := &promptManager{}
	return pm, nil
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
	ctx := SubagentPromptContext{
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
