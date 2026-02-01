// Package prompt provides core types for versioned prompt management.
package prompt

import (
	"time"

	"github.com/Masterminds/semver/v3"
)

// Built-in prompt ID constants
const (
	PromptIDSystem     = "system"
	PromptIDSupervisor = "supervisor"
	PromptIDCompacter  = "compacter"
	PromptIDSubagent   = "subagent"
)

// Prompt represents a versioned prompt with metadata.
type Prompt struct {
	ID        string                 // Unique identifier (e.g., "system", "subagent@1.0.0")
	Name      string                 // Human-readable name
	Content   string                 // Prompt text, may contain template syntax
	Context   map[string]interface{} // Optional context values for template rendering
	Tags      []string               // Optional categorization tags
	CreatedAt time.Time              // When this prompt was created
	UpdatedAt time.Time              // When this prompt was last modified
	Version   *semver.Version        // SemVer version for change tracking
	IsBuiltin bool                   // Built-in prompts cannot be deleted
}

// RenderContext contains general context values for prompt rendering.
type RenderContext struct {
	Values   map[string]interface{} // General context values
	SubAgent *SubAgentContext       // SubAgent-specific context
	Agent    *AgentContext          // Agent-specific context
}

// SubAgentContext contains context specific to subagent operations.
type SubAgentContext struct {
	Role            string // The role of the subagent
	Description     string // Description of the subagent's purpose
	SpawnAgentTool  string // Name of the tool to spawn new agents
	RemoveAgentTool string // Name of the tool to remove agents
	ResumeAgentTool string // Name of the tool to resume agents
	AgentOutputTool string // Name of the tool for agent output
	ListAgentsTool  string // Name of the tool to list agents
}

// AgentContext contains context specific to agent operations.
type AgentContext struct {
	AgentID string // The agent's unique identifier
	Task    string // The task the agent is working on
}

// ListFilter provides filtering options for listing prompts.
type ListFilter struct {
	Tags []string // Filter by tags (OR logic)
	IDs  []string // Filter by IDs (OR logic)
}
