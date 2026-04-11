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

	// ToolParamKeys represents the type for tool parameter key constants.
	ToolParamKeys string
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

// ToolParam key constants for tool input parameters.
const (
	// ParamCommand is the command to execute (string).
	ParamCommand ToolParamKeys = "command"
	// ParamFilePath is the absolute path to the file (string).
	ParamFilePath ToolParamKeys = "file_path"
	// ParamPath is a directory or file path (string).
	ParamPath ToolParamKeys = "path"
	// ParamContent is the content to read/write (string).
	ParamContent ToolParamKeys = "content"
	// ParamPattern is the search pattern (string).
	ParamPattern ToolParamKeys = "pattern"
	// ParamTimeout is the timeout in seconds (float64).
	ParamTimeout ToolParamKeys = "timeout"
	// ParamOutputMode is the output mode (string).
	ParamOutputMode ToolParamKeys = "output_mode"
	// ParamLimit is the maximum number of results (int).
	ParamLimit ToolParamKeys = "limit"
	// ParamOffset is the starting line number (int).
	ParamOffset ToolParamKeys = "offset"
	// ParamGlob is the glob pattern for file filtering (string).
	ParamGlob ToolParamKeys = "glob"
	// ParamOldString is the string to replace (string).
	ParamOldString ToolParamKeys = "old_string"
	// ParamNewString is the replacement string (string).
	ParamNewString ToolParamKeys = "new_string"
	// ParamReplaceAll = " replace all occurrences (bool).
	ParamReplaceAll ToolParamKeys = "replace_all"
	// ParamCreateDirs = " create parent directories (bool).
	ParamCreateDirs ToolParamKeys = "create_dirs"
	// ParamContextBefore = " context lines before match (int).
	ParamContextBefore ToolParamKeys = "-B"
	// ParamContextAfter = " context lines after match (int).
	ParamContextAfter ToolParamKeys = "-A"
	// ParamContextBoth = " context lines before and after (int).
	ParamContextBoth ToolParamKeys = "-C"
	// ParamCaseIgnore = " case-insensitive search (bool).
	ParamCaseIgnore ToolParamKeys = "-i"
	// ParamShowNumbers = " show line numbers (bool).
	ParamShowNumbers ToolParamKeys = "-n"
	// ParamMultiline = " multiline regex mode (bool).
	ParamMultiline ToolParamKeys = "multiline"
	// ParamHeadLimit is the maximum results to return (int).
	ParamHeadLimit ToolParamKeys = "head_limit"
	// ParamAgentID is the agent ID (string).
	ParamAgentID ToolParamKeys = "agent_id"
	// ParamAgentName is the agent name (string).
	ParamAgentName ToolParamKeys = "name"
	// ParamModel is the model identifier (string).
	ParamModel ToolParamKeys = "model"
	// ParamPrompt is " prompt text (string).
	ParamPrompt ToolParamKeys = "prompt"
	// ParamQuery is the search query (string).
	ParamQuery ToolParamKeys = "query"
	// ParamURL is the URL (string).
	ParamURL ToolParamKeys = "url"
	// ParamMode is the operation mode (string).
	ParamMode ToolParamKeys = "mode"
	// ParamCount is the count of items (int).
	ParamCount ToolParamKeys = "count"
	// ParamLevel is the log level (string).
	ParamLevel ToolParamKeys = "level"
	// ParamSinceSeq is the sequence number to start from (int64).
	ParamSinceSeq ToolParamKeys = "since_seq"
	// ParamBlock is whether to block for completion (bool).
	ParamBlock ToolParamKeys = "block"
	// ParamRecursive is whether to search recursively (bool).
	ParamRecursive ToolParamKeys = "recursive"
	// ParamTree is whether to format as tree (bool).
	ParamTree ToolParamKeys = "tree"
	// ParamForce is whether to force an operation (bool).
	ParamForce ToolParamKeys = "force"
	// ParamTimezone is the timezone (string).
	ParamTimezone ToolParamKeys = "timezone"
	// ParamRunInBackground is whether to run asynchronously (bool).
	ParamRunInBackground ToolParamKeys = "run_in_background"
	// ParamInput is the input text (string).
	ParamInput ToolParamKeys = "input"
	// ParamContextMode is the context mode (string).
	ParamContextMode ToolParamKeys = "context_mode"
	// ParamRole is the agent role (string).
	ParamRole ToolParamKeys = "role"
	// ParamDescription is a short description (string).
	ParamDescription ToolParamKeys = "description"
	// ParamShareContext is whether to share context (bool).
	ParamShareContext ToolParamKeys = "share_context"
	// ParamAllowedTools is the list of allowed tools ([]string).
	ParamAllowedTools ToolParamKeys = "allowed_tools"
	// ParamValue is a generic value parameter (string).
	ParamValue ToolParamKeys = "value"
	// ParamTo is the target destination (string).
	ParamTo ToolParamKeys = "to"
	// ParamFields is a list of fields ([]string).
	ParamFields ToolParamKeys = "fields"
	// ParamMessage is a message (string).
	ParamMessage ToolParamKeys = "message"
	// ParamFlowName is the name of the flow to execute (string).
	ParamFlowName ToolParamKeys = "flowName"
	// ParamInputs is the input parameters for the flow (map[string]any).
	ParamInputs ToolParamKeys = "inputs"
) // ToolResult is a type alias for tool execution result maps.
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
	LLMClientConfig *LLMClientConfig `json:"llm_client_config"`
	OutputMode      OutputMode       `json:"output_mode"`
	AllowCompaction bool             `json:"allow_compaction"`
	History         *gollem.History  `json:"history,omitempty"` // Optional parent message history for context awareness
	IsSupervisor    bool             `json:"is_supervisor"`     // Indicates this is the singleton supervisor agent
	SessionID       string           `json:"session_id"`        // Session identifier for multi-session support
	ChannelID       uuid.UUID        `json:"channel_id"`        // Channel identifier for multi-session support

	// AllowedTools specifies which tools the agent can access.
	// Built-in tools use ToolName constants (e.g., "bash", "current_time").
	// MCP tools use "server_name/tool_name" format (e.g., "filesystem/read_file").
	// Empty or nil means no tools available.
	AllowedTools []string `json:"allowed_tools"`
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
