// Package hooks provides typed payload structs for hook context data.
// Each payload type contains only fields relevant to its hook category,
// replacing the bloated HookContext fields and untyped Data map.
package hooks

import (
	"time"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
)

// ToolPayload contains data for tool execution hooks.
// Used with BeforeToolExecution, AfterToolExecution, and OnToolError hook points.
//
// Hooks can modify Args before execution (BeforeToolExecution) and
// modify Result after execution (AfterToolExecution).
type ToolPayload struct {
	shared.LoggingContext

	// Name is the tool name being executed.
	Name shared.ToolName

	// Args contains the tool arguments passed to the tool.
	// Hooks can modify this map before tool execution.
	Args map[string]any

	// Result contains the tool execution result.
	// Hooks can modify this map after tool execution.
	Result map[string]any

	// Error contains the tool execution error if one occurred.
	// Only populated for OnToolError hook point.
	Error error
}

// LLMPayload contains data for LLM hooks.
// Used with BeforeLLMRequest, AfterLLMResponse, and OnLLMError hook points.
//
// Hooks can modify Input before the request and Response after receiving it.
type LLMPayload struct {
	shared.LoggingContext

	// Input is the prompt sent to the LLM.
	// Hooks can modify this before the request is sent.
	Input string

	// Response is the text response from the LLM.
	// Hooks can modify this before it's returned to the caller.
	Response string

	// Model is the LLM model identifier (e.g., "claude-3-5-sonnet").
	Model string

	// Options contains additional LLM parameters (temperature, max_tokens, etc.).
	Options map[string]any

	// Error contains the LLM error if one occurred.
	// Only populated for OnLLMError hook point.
	Error error
}

// FileOperation represents the type of file operation being performed.
type FileOperation string

const (
	// FileOperationRead indicates a file read operation.
	FileOperationRead FileOperation = "read"
	// FileOperationWrite indicates a file write operation.
	FileOperationWrite FileOperation = "write"
	// FileOperationDelete indicates a file delete operation.
	FileOperationDelete FileOperation = "delete"
	// FileOperationModify indicates a file modify/edit operation.
	FileOperationModify FileOperation = "modify"
)

// FilePayload contains data for file operation hooks.
// Used with BeforeFileRead/AfterFileRead, BeforeFileWrite/AfterFileWrite,
// BeforeFileDelete/AfterFileDelete, and BeforeFileModify/AfterFileModify hook points.
//
// Hooks can modify Content before write operations and access both
// OldContent and NewContent for modify operations.
type FilePayload struct {
	shared.LoggingContext

	// Path is the file path for the operation.
	Path string

	// Content is the file content for read/write operations.
	// For reads: populated after the file is read.
	// For writes: hooks can modify this before the file is written.
	Content string

	// OldContent is the original content before modification.
	// Only populated for modify operations.
	OldContent string

	// NewContent is the new content for modification.
	// Hooks can modify this before the file is modified.
	NewContent string

	// Operation indicates the type of file operation being performed.
	Operation FileOperation
}

// SessionPayload contains data for session lifecycle hooks.
// Used with BeforeSessionStart and AfterSessionEnd hook points.
type SessionPayload struct {
	shared.LoggingContext

	// Metadata contains additional session-specific data.
	// Hooks can use this to pass information between before/after hooks.
	Metadata map[string]any
}

// AgentEvent represents the type of agent lifecycle event.
type AgentEvent string

const (
	// AgentEventSpawn indicates an agent is being spawned.
	AgentEventSpawn AgentEvent = "spawn"
	// AgentEventRemove indicates an agent is being removed.
	AgentEventRemove AgentEvent = "remove"
)

// AgentPayload contains data for agent lifecycle hooks.
// Used with BeforeAgentSpawn/AfterAgentSpawn and BeforeAgentRemove/AfterAgentRemove hook points.
type AgentPayload struct {
	shared.LoggingContext

	// Event indicates whether this is a spawn or remove event.
	Event AgentEvent

	// NewAgentID is the ID of the agent being spawned or removed.
	NewAgentID uuid.UUID

	// ParentID is the ID of the parent agent (for spawn events).
	// Empty uuid.Nil if this is a root agent.
	ParentID uuid.UUID
}

// SkillType represents the type of skill being invoked.
type SkillType string

const (
	// SkillTypePrompt indicates a prompt-based skill.
	SkillTypePrompt SkillType = "prompt"
	// SkillTypeAgent indicates an agent-based skill.
	SkillTypeAgent SkillType = "agent"
)

// SkillContextMode represents how context is inherited for skill invocation.
type SkillContextMode string

const (
	// ContextModeNone indicates no context inheritance.
	ContextModeNone SkillContextMode = "none"
	// ContextModeInherited indicates context is inherited from parent agent.
	ContextModeInherited SkillContextMode = "inherited"
	// ContextModeIsolated indicates the skill starts with a fresh isolated context.
	// Alias for ContextModeNone for compatibility with tools package.
	ContextModeIsolated SkillContextMode = "isolated"
)

// SkillPayload contains data for skill invocation hooks.
// Used with BeforeSkillInvoked, AfterSkillInvoked, and OnSkillError hook points.
type SkillPayload struct {
	shared.LoggingContext

	// Name is the skill name being invoked.
	Name string

	// Type indicates whether this is a prompt or agent-based skill.
	Type SkillType

	// ContextMode indicates how context is inherited for this invocation.
	ContextMode SkillContextMode

	// Model is the LLM provider/model used for skill execution.
	Model string

	// FilePath is the path to the skill definition file.
	FilePath string

	// Version is the skill version identifier.
	Version string

	// Result contains the skill execution result.
	// Populated after successful skill execution.
	Result map[string]any

	// Error contains the skill execution error if one occurred.
	// Only populated for OnSkillError hook point.
	Error error
}

// TracingPayload contains data for observability and tracing hooks.
// Used by tracing implementations like Langfuse for distributed tracing.
//
// This payload is available for all hook points to enable consistent
// observability across the entire hook lifecycle.
type TracingPayload struct {
	shared.LoggingContext

	// SpanID is the unique identifier for this span within the trace.
	SpanID string

	// TraceID is the unique identifier for the overall trace.
	TraceID string

	// StartTime is when the operation began.
	StartTime time.Time

	// EndTime is when the operation completed.
	EndTime time.Time

	// Metadata contains additional tracing-specific data.
	// Can include operation names, parent span IDs, custom attributes, etc.
	Metadata map[string]any
}
