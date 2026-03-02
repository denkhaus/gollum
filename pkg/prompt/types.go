// Package prompt provides core types for versioned prompt management.
package prompt

import (
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/m-mizutani/gollem"
)

// Built-in prompt ID constants
const (
	PromptIDSubagentSystem = "subagent_system"
	// PromptIDSupervisorSystem is the supervisor system prompt.
	PromptIDSupervisorSystem = "supervisor_system"
	// PromptIDCompacter is the compacter prompt.
	PromptIDCompacter = "compacter"
	// PromptIDSubagentTask is the subagent task prompt.
	PromptIDSubagentTask = "subagent_task"

	// Optimizer prompt IDs
	// PromptIDOptimizerGradient is the gradient strategy reflection prompt.
	PromptIDOptimizerGradient = "optimizer_gradient"
	// PromptIDOptimizerGradientMeta is the gradient strategy metaprompt (phase 2).
	PromptIDOptimizerGradientMeta = "optimizer_gradient_meta"
	// PromptIDOptimizerMeta is the meta-prompt strategy.
	PromptIDOptimizerMeta = "optimizer_meta"
	// PromptIDOptimizerMemory is the prompt memory strategy.
	PromptIDOptimizerMemory = "optimizer_memory"
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
	Values    map[string]interface{} // General context values
	SubAgent  *SubAgentContext       // SubAgent-specific context
	Agent     *AgentContext          // Agent-specific context
	Workspace *WorkspaceContext      // Workspace-specific context (skills, path)
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
	AgentID        string           // The agent's unique identifier
	Task           string           // The task the agent is working on
	MessageHistory []gollem.Message // Optional message history for context awareness
}

// SkillInfo represents a discovered skill for template rendering.
type SkillInfo struct {
	Name        string `json:"name"`        // Skill name
	Description string `json:"description"` // Brief description
	Location    string `json:"location"`    // File path to the skill
}

// WorkspaceContext contains workspace-specific information for prompt rendering.
type WorkspaceContext struct {
	CurrentPath string      `json:"current_path"` // Current working directory
	SkillsXML   string      `json:"skills_xml"`   // Skills in XML format for LLM prompts
	Skills      []SkillInfo `json:"skills"`       // List of discovered skills
}

// ListFilter provides filtering options for listing prompts.
type ListFilter struct {
	Tags []string // Filter by tags (OR logic)
	IDs  []string // Filter by IDs (OR logic)
}
