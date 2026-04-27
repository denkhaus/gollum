package shared

import (
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// Message represents a structured message data for displays and communication.
// It is used across multiple subsystems (TUI, logging, agents, tools) to represent
// various types of communication including chat messages, tool executions, and system events.
type Message struct {
	ID        uuid.UUID
	Role      gollem.MessageRole // system, user, assistant, tool (message role in conversation)
	AgentRole string            // supervisor, sub-agent, etc. (agent type for UI)
	SessionContext
	Content   string
	Timestamp time.Time
	Metadata  map[string]any // tool_name, duration, collapsed, etc.
}

// IsTool returns true if this is a tool-related message (request or response).
func (m *Message) IsTool() bool {
	return m.Role == gollem.RoleTool
}

// IsUserMessage returns true if this is a user message.
func (m *Message) IsUserMessage() bool {
	return m.Role == gollem.RoleUser
}

// IsAssistantMessage returns true if this is an assistant/agent message.
func (m *Message) IsAssistantMessage() bool {
	return m.Role == gollem.RoleAssistant
}

// IsSystemMessage returns true if this is a system message.
func (m *Message) IsSystemMessage() bool {
	return m.Role == gollem.RoleSystem
}
