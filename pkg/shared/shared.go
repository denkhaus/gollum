package shared

import (
	"errors"

	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

type (
	// LLMProvider represents the LLM provider to use for an agent.
	LLMProvider string

	// OutputMode controls console output behavior for agents.
	OutputMode string

	// ToolResultKeys represents the type for tool result key constants.
	ToolResultKeys string
)

const (
	// LLMProviderGemini uses Google's Gemini API.
	LLMProviderGemini LLMProvider = "gemini"
	// LLMProviderAnthropic uses Anthropic's Claude API.
	LLMProviderAnthropic LLMProvider = "anthropic"
	// LLMProviderOpenAI uses OpenAI's GPT API.
	LLMProviderOpenAI LLMProvider = "openai"
)

const (
	// OutputModeFull shows all execution steps (main agent).
	OutputModeFull OutputMode = "full"
	// OutputModeSummary shows only final result (sub-agent).
	OutputModeSummary OutputMode = "summary"
	// OutputModeSilent suppresses all output.
	OutputModeSilent OutputMode = "silent"
)

// ToolResult key constants.
const (
	// KeySuccess indicates whether the tool operation succeeded (bool).
	KeySuccess ToolResultKeys = "success"
	// KeyError contains error message if operation failed (string).
	KeyError ToolResultKeys = "error"
	// KeyFilePath is the absolute path to the file (string).
	KeyFilePath ToolResultKeys = "file_path"
	// KeyChecksum is the file checksum for verification (string).
	KeyChecksum ToolResultKeys = "checksum"
	// KeySize is the file size in bytes (int64).
	KeySize ToolResultKeys = "size"
	// KeyModified is the file modification timestamp (int64).
	KeyModified ToolResultKeys = "modified"
	// KeyLockedBy is the agent ID that holds the file lock (uuid.UUID).
	KeyLockedBy ToolResultKeys = "locked_by"
	// KeyReplacements is the count of string replacements made (int).
	KeyReplacements ToolResultKeys = "replacements"
	// KeyOldLen is the length of the old string being replaced (int).
	KeyOldLen ToolResultKeys = "old_len"
	// KeyNewLen is the length of the new string (int).
	KeyNewLen ToolResultKeys = "new_len"
	// KeyDiff contains the unified diff of changes (string).
	KeyDiff ToolResultKeys = "diff"
	// KeyDiffAdditions is the count of lines added (int).
	KeyDiffAdditions ToolResultKeys = "diff_additions"
	// KeyDiffRemovals is the count of lines removed (int).
	KeyDiffRemovals ToolResultKeys = "diff_removals"
	// KeyDiffCompact is a compact representation of the diff (string).
	KeyDiffCompact ToolResultKeys = "diff_compact"
	// KeyIsNewFile indicates whether the file was newly created (bool).
	KeyIsNewFile ToolResultKeys = "is_new_file"
	// KeyStdout contains the standard output from command execution (string).
	KeyStdout ToolResultKeys = "stdout"
	// KeyStderr contains the standard error output from command execution (string).
	KeyStderr ToolResultKeys = "stderr"
	// KeyExitCode is the process exit code (int).
	KeyExitCode ToolResultKeys = "exit_code"
	// KeyDuration is the execution duration (string).
	KeyDuration ToolResultKeys = "duration"
	// KeyWarning contains warning messages (string).
	KeyWarning ToolResultKeys = "warning"
	// KeyFileChanges contains detected file changes ([]FileChange).
	KeyFileChanges ToolResultKeys = "file_changes"
	// KeyFileDiffs contains file diff information (map[string]interface{}).
	KeyFileDiffs ToolResultKeys = "file_diffs"
)

// ToolResult is a type alias for tool execution result maps.
type ToolResult map[string]any

// GetString returns a string value from the result.
// Returns empty string if the key doesn't exist or is not a string.
func (tr ToolResult) GetString(key ToolResultKeys) string {
	if val, ok := tr[string(key)]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt returns an integer value from the result.
// Returns 0 if the key doesn't exist or is not an int.
func (tr ToolResult) GetInt(key ToolResultKeys) int {
	if val, ok := tr[string(key)]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}

// GetBool returns a boolean value from the result.
// Returns false if the key doesn't exist or is not a bool.
func (tr ToolResult) GetBool(key ToolResultKeys) bool {
	if val, ok := tr[string(key)]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// HasDiff returns true if the result contains diff information.
func (tr ToolResult) HasDiff() bool {
	_, hasFullDiff := tr[string(KeyDiff)]
	_, hasAdditions := tr[string(KeyDiffAdditions)]
	_, hasRemovals := tr[string(KeyDiffRemovals)]
	_, hasCompact := tr[string(KeyDiffCompact)]

	return hasFullDiff || hasAdditions || hasRemovals || hasCompact
}

// Common errors
var (
	ErrTargetAgentNotFound     = errors.New("target agent not found")
	ErrLLMProviderNotSupported = errors.New("LLM provider not supported")
	ErrInvalidModelFormat      = errors.New("model must be in 'provider/model' format")
	ErrAgentNotFound           = errors.New("agent not found")
	ErrPermissionDenied        = errors.New("permission denied")
)

// AgentConfig holds configuration for an agent, usable by both parent and sub-agents.
type AgentConfig struct {
	ID              uuid.UUID        `json:"id"`
	ParentID        *uuid.UUID       `json:"parent_id,omitempty"`
	SystemPrompt    string           `json:"system_prompt"`
	Role            string           `json:"role"`
	Description     string           `json:"description"`
	Strategy        gollem.Strategy  `json:"-"`
	Tools           []gollem.Tool    `json:"-"`
	ToolSets        []gollem.ToolSet `json:"-"`
	LLMClientConfig *LLMClientConfig `json:"llm_client_config"`
	OutputMode      OutputMode       `json:"output_mode"`
	AllowCompaction bool             `json:"allow_compaction"`
	History         *gollem.History  `json:"history,omitempty"` // Optional parent message history for context awareness
}

// AgentStatus represents the execution status of a background agent.
type AgentStatus string

const (
	// AgentStatusRunning indicates the agent is currently executing.
	AgentStatusRunning AgentStatus = "running"
	// AgentStatusCompleted indicates the agent completed successfully.
	AgentStatusCompleted AgentStatus = "completed"
	// AgentStatusFailed indicates the agent execution failed.
	AgentStatusFailed AgentStatus = "failed"
)

// AgentResult stores the execution result of a background agent
type AgentResult struct {
	AgentID     uuid.UUID              `json:"agent_id"`
	Status      AgentStatus            `json:"status"`
	Output      map[string]interface{} `json:"output"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   int64                  `json:"started_at"`             // Unix timestamp
	CompletedAt *int64                 `json:"completed_at,omitempty"` // Unix timestamp
}
