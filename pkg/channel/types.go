// Package channel provides the channel abstraction layer for Gollum.
package channel

import (
	"context"
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

// Message represents a structured message data for displays
type Message struct {
	ID        uuid.UUID
	Type      MessageType
	AgentID   uuid.UUID
	AgentRole string
	SessionID string // Session identifier for multi-session support
	ChannelID uuid.UUID
	Content   string
	Timestamp time.Time
	Metadata  map[string]any // tool_name, duration, collapsed, etc.
}

// InputResult represents the result of user input submission
type InputResult struct {
	Handled   bool   // true if command was executed
	IsCommand bool   // true if input was slash command
	Response  string // optional response (e.g., command help)
	Error     error  // optional error if command failed
}

// AgentLifecycleEvent represents agent registration/removal events
type AgentLifecycleEvent struct {
	AgentID uuid.UUID
	Role    string
	Added   bool
}

// ChannelOption is the interface for channel configuration options.
// Each channel package implements its own option types that satisfy this interface.
type ChannelOption interface {
	Apply(Channel) error
}

// ChannelFactory creates a channel instance with options.
// Registered as named providers in DI (e.g., "channel_tui", "channel_acp").
type ChannelFactory func(opts ...ChannelOption) (Channel, error)

// ChannelIdentifier is the const type each channel exports.
// Provides type safety - no magic strings.
type ChannelIdentifier string

// ChannelStarter is the interface for channels that manage their own lifecycle.
// Channels like TUI implement this to run their main loop.
type ChannelStarter interface {
	Channel
	Start(ctx context.Context) error
}
