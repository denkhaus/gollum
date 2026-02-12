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

// Common errors
var (
	ErrTargetAgentNotFound     = errors.New("target agent not found")
	ErrLLMProviderNotSupported = errors.New("LLM provider not supported")
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
	LLMProvider     LLMProvider      `json:"llm_provider"`
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
