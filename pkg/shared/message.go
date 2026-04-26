package shared

import (
	"time"

	"github.com/google/uuid"
)

// MessageType represents different types of messages
type MessageType int

const (
	MessageTypeUserChat     MessageType = iota
	MessageTypeAgentChat                // Chat response from agent
	MessageTypeToolRequest              // Tool execution request
	MessageTypeToolResponse             // Tool execution response
	MessageTypeThinking                 // Agent thinking blocks
	MessageTypeSystemInfo               // System information messages
	MessageTypeError                    // Error messages
)

// String returns a string representation of the MessageType
func (mt MessageType) String() string {
	switch mt {
	case MessageTypeUserChat:
		return "user_chat"
	case MessageTypeAgentChat:
		return "agent_chat"
	case MessageTypeToolRequest:
		return "tool_request"
	case MessageTypeToolResponse:
		return "tool_response"
	case MessageTypeThinking:
		return "thinking"
	case MessageTypeSystemInfo:
		return "system_info"
	case MessageTypeError:
		return "error"
	default:
		return "unknown"
	}
}

// Message represents a structured message data for displays and communication.
// It is used across multiple subsystems (TUI, logging, agents, tools) to represent
// various types of communication including chat messages, tool executions, and system events.
type Message struct {
	ID        uuid.UUID
	Type      MessageType
	AgentRole string
	SessionContext
	Content   string
	Timestamp time.Time
	Metadata  map[string]any // tool_name, duration, collapsed, etc.
}
