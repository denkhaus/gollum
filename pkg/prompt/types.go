// Package prompt provides core types for versioned prompt management.
package prompt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/m-mizutani/gollem"
)

// =============================================================================
// PROMPT ID TYPES
// =============================================================================

// PromptID is a strongly-typed base identifier for prompts.
// A base ID is the unversioned identifier (e.g., "subagent_system", "supervisor_system").
//
// Base IDs are used for:
//   - Creating new versions via store.SaveNewVersion
//   - Listing all versions via store.ListVersions
//   - Bootstrap registration of built-in prompts
//
// To create a versioned ID, use PromptID.WithVersion() or VersionedPromptID.FromBaseID().
type PromptID string

// String returns the string representation of the base ID.
func (id PromptID) String() string {
	return string(id)
}

// WithVersion creates a VersionedPromptID from this base ID and a semantic version.
// Example: PromptIDSubagentSystem.WithVersion("1.0.0") -> "subagent_system@1.0.0"
func (id PromptID) WithVersion(version string) VersionedPromptID {
	return VersionedPromptID(fmt.Sprintf("%s@%s", id, version))
}

// WithSemVer creates a VersionedPromptID from this base ID and a semver.Version.
func (id PromptID) WithSemVer(version *semver.Version) VersionedPromptID {
	return VersionedPromptID(fmt.Sprintf("%s@%s", id, version.String()))
}

// IsBuiltin checks if this base ID is one of the built-in prompt constants.
func (id PromptID) IsBuiltin() bool {
	switch id {
	case PromptIDSubagentSystem,
		PromptIDSupervisorSystem,
		PromptIDCompacter,
		PromptIDSubagentTask,
		PromptIDOptimizerGradient,
		PromptIDOptimizerGradientMeta,
		PromptIDOptimizerMeta,
		PromptIDOptimizerMemory:
		return true
	default:
		return false
	}
}

// Built-in prompt ID constants.
// These are the base IDs for prompts that are embedded in the binary.
const (
	// PromptIDSubagentSystem is the subagent system prompt.
	PromptIDSubagentSystem PromptID = "subagent_system"
	// PromptIDSupervisorSystem is the supervisor system prompt.
	PromptIDSupervisorSystem PromptID = "supervisor_system"
	// PromptIDCompacter is the compacter prompt.
	PromptIDCompacter PromptID = "compacter"
	// PromptIDSubagentTask is the subagent task prompt.
	PromptIDSubagentTask PromptID = "subagent_task"

	// Optimizer prompt IDs
	// PromptIDOptimizerGradient is the gradient strategy reflection prompt.
	PromptIDOptimizerGradient PromptID = "optimizer_gradient"
	// PromptIDOptimizerGradientMeta is the gradient strategy metaprompt (phase 2).
	PromptIDOptimizerGradientMeta PromptID = "optimizer_gradient_meta"
	// PromptIDOptimizerMeta is the meta-prompt strategy.
	PromptIDOptimizerMeta PromptID = "optimizer_meta"
	// PromptIDOptimizerMemory is the prompt memory strategy.
	PromptIDOptimizerMemory PromptID = "optimizer_memory"
)

// =============================================================================
// VERSIONED PROMPT ID TYPE
// =============================================================================

const (
	// versionLatest is the default version string when no version is specified.
	versionLatest = "latest"
)

// VersionedPromptID is a strongly-typed versioned identifier for prompts.
// A versioned ID combines a base ID with a semantic version (e.g., "subagent_system@1.0.0").
//
// Versioned IDs are used for:
//   - Loading a specific prompt version via store.Load
//   - Deleting a specific prompt version via store.Delete
//   - The Prompt.ID field
//
// Use ParseVersionedPromptID() to parse from string, or PromptID.WithVersion() to create.
type VersionedPromptID string

// Errors for VersionedPromptID parsing.
var (
	ErrInvalidVersionedPromptID = errors.New("invalid versioned prompt ID format")
	ErrInvalidPromptVersion     = errors.New("invalid semantic version in prompt ID")
)

// ParseVersionedPromptID parses a string into a VersionedPromptID and validates it.
// Accepts formats: "base_id@version" or "base_id@latest".
// Returns an error if the format is invalid.
func ParseVersionedPromptID(s string) (VersionedPromptID, error) {
	vid := VersionedPromptID(s)
	if err := vid.Validate(); err != nil {
		return "", err
	}
	return vid, nil
}

// MustParseVersionedPromptID parses a string into a VersionedPromptID.
// Panics if the format is invalid. Use for compile-time known constants.
func MustParseVersionedPromptID(s string) VersionedPromptID {
	vid, err := ParseVersionedPromptID(s)
	if err != nil {
		panic(fmt.Sprintf("prompt: invalid versioned prompt ID %q: %v", s, err))
	}
	return vid
}

// FromBaseID creates a VersionedPromptID from a base ID and version string.
// If version is empty, it defaults to "latest".
func FromBaseID(baseID PromptID, version string) VersionedPromptID {
	if version == "" {
		version = versionLatest
	}
	return VersionedPromptID(fmt.Sprintf("%s@%s", baseID, version))
}

// String returns the string representation of the versioned ID.
func (vid VersionedPromptID) String() string {
	return string(vid)
}

// Validate checks if the versioned ID has a valid format.
// Valid formats: "base_id@version" where version is semver or "latest".
func (vid VersionedPromptID) Validate() error {
	baseID, version := vid.Split()
	if baseID == "" {
		return fmt.Errorf("%w: empty base ID", ErrInvalidVersionedPromptID)
	}
	if version == "" {
		return fmt.Errorf("%w: missing version", ErrInvalidVersionedPromptID)
	}
	return nil
}

// Split returns the base ID and version parts.
// For "subagent_system@1.0.0", returns ("subagent_system", "1.0.0").
// If there's no @ separator, returns (vid, "").
func (vid VersionedPromptID) Split() (baseID PromptID, version string) {
	idx := strings.LastIndex(string(vid), "@")
	if idx == -1 {
		return PromptID(vid), ""
	}
	return PromptID(vid[:idx]), string(vid[idx+1:])
}

// BaseID returns the base ID part of the versioned ID.
// For "subagent_system@1.0.0", returns "subagent_system".
func (vid VersionedPromptID) BaseID() PromptID {
	baseID, _ := vid.Split()
	return baseID
}

// Version returns the version string part of the versioned ID.
// For "subagent_system@1.0.0", returns "1.0.0".
func (vid VersionedPromptID) Version() string {
	_, version := vid.Split()
	return version
}

// IsLatest returns true if the version is "latest".
func (vid VersionedPromptID) IsLatest() bool {
	return vid.Version() == "latest"
}

// SemVer parses the version part as a semantic version.
// Returns an error if the version is not valid semver (e.g., "latest").
func (vid VersionedPromptID) SemVer() (*semver.Version, error) {
	version := vid.Version()
	if version == "latest" {
		return nil, fmt.Errorf("%w: cannot parse 'latest' as semver", ErrInvalidPromptVersion)
	}
	return semver.NewVersion(version)
}

// IsBuiltin checks if the base ID of this versioned ID is a built-in prompt.
func (vid VersionedPromptID) IsBuiltin() bool {
	return vid.BaseID().IsBuiltin()
}

// =============================================================================
// PROMPT STRUCT
// =============================================================================

// Prompt represents a versioned prompt with metadata.
// The ID field contains the versioned ID (e.g., "subagent_system@1.0.0").
// Use BaseID() to get the unversioned base identifier as PromptID.
type Prompt struct {
	// ID is the fully qualified versioned identifier (e.g., "subagent_system@1.0.0").
	// This is the primary key for loading and deleting prompts.
	// JSON tag ensures backward compatibility with stored prompts.
	ID string `json:"id"`

	// Name is a human-readable name for the prompt.
	Name string `json:"name"`

	// Content is the prompt text, which may contain Go template syntax.
	Content string `json:"content"`

	// Context provides optional values for template rendering.
	Context map[string]any `json:"context,omitempty"`

	// Tags are optional categorization labels.
	Tags []string `json:"tags,omitempty"`

	// CreatedAt is when this prompt version was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this prompt version was last modified.
	UpdatedAt time.Time `json:"updated_at"`

	// Version is the semantic version of this prompt.
	Version *semver.Version `json:"version"`

	// IsBuiltin indicates whether this is a built-in prompt that cannot be deleted.
	IsBuiltin bool `json:"is_builtin"`
}

// VersionedPromptID returns the versioned ID as a strongly-typed VersionedPromptID.
func (p *Prompt) VersionedPromptID() VersionedPromptID {
	return VersionedPromptID(p.ID)
}

// BaseID returns the unversioned base ID from the prompt's versioned ID.
// For example, "subagent_system@1.0.0" -> "subagent_system".
func (p *Prompt) BaseID() PromptID {
	return p.VersionedPromptID().BaseID()
}

// =============================================================================
// CONTEXT TYPES
// =============================================================================

// RenderContext contains general context values for prompt rendering.
type RenderContext struct {
	Values    map[string]any           // General context values
	SubAgent  *SubAgentContext         // SubAgent-specific context
	Agent     *AgentContext            // Agent-specific context
	Workspace *shared.WorkspaceContext // Workspace-specific context (skills, path)
}

// SubAgentContext contains context specific to subagent operations.
type SubAgentContext struct {
	Role            string          // The role of the subagent
	Description     string          // Description of the subagent's purpose
	SpawnAgentTool  shared.ToolName // Name of the tool to spawn new agents
	RemoveAgentTool shared.ToolName // Name of the tool to remove agents
	ResumeAgentTool shared.ToolName // Name of the tool to resume agents
	AgentOutputTool shared.ToolName // Name of the tool for agent output
	ListAgentsTool  shared.ToolName // Name of the tool to list agents
}

// AgentContext contains context specific to agent operations.
type AgentContext struct {
	AgentID        string           // The agent's unique identifier
	Task           string           // The task the agent is working on
	MessageHistory []gollem.Message // Optional message history for context awareness
}

// ListFilter provides filtering options for listing prompts.
type ListFilter struct {
	Tags []string            // Filter by tags (OR logic)
	IDs  []VersionedPromptID // Filter by versioned IDs (OR logic)
}
