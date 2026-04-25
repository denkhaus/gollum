// Package channel provides the channel abstraction layer for Gollum.
package channel

import (
	"context"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
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
	AgentID   uuid.UUID
	Role      string
	Added     bool
	SessionID string
	ChannelID uuid.UUID
}

// Channel is the interface that all channel implementations must satisfy
type Channel interface {
	// ID returns a unique identifier for this channel
	ID() uuid.UUID

	// OnMessage is called when a new message should be displayed
	OnMessage(msg Message)

	// OnLog is called for log entries (channel can ignore if not applicable)
	OnLog(entry shared.LogEntry)

	// OnAgentLifecycle is called when agent registration/removal events occur
	OnAgentLifecycle(event AgentLifecycleEvent)

	// Start begins the channel's lifecycle. For interactive channels like TUI,
	// this runs the main event loop. For passive channels, this may return immediately.
	// Blocks until the channel is complete or context is cancelled.
	Start(ctx context.Context) error
}

// ChannelFacade is the central coordinator for all channels
// It also implements shared.LogForwarder for routing logs to channels
//
// The facade delegates input handling business logic (command execution,
// session/supervisor management) to the InputHandler interface, allowing
// for easier testing and potential strategy swapping.
type ChannelFacade interface {
	shared.LogForwarder

	// DisplayMessage sends a message to all registered channels
	DisplayMessage(msg Message)

	// SubmitInput handles user input from any channel.
	// Delegates to InputHandler for business logic including:
	// - Slash command detection and execution
	// - Session creation/retrieval
	// - Supervisor agent creation and execution
	SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (*InputResult, error)

	// CancelInput cancels an in-flight input for the given session.
	// Delegates to InputHandler for session cancellation logic.
	CancelInput(sessionID string) error

	// RegisterChannel adds a channel to receive events
	RegisterChannel(channel Channel) error

	// UnregisterChannel removes a channel
	UnregisterChannel(channelID uuid.UUID) error

	// NotifyAgentLifecycle sends agent lifecycle event to specific channel
	NotifyAgentLifecycle(agentID uuid.UUID, channelID uuid.UUID, sessionID string, role string, added bool)

	// DiscoverProviders scans DI for channel providers
	DiscoverProviders(injector do.Injector) error

	// CreateChannel creates a channel instance by identifier with options
	CreateChannel(identifier ChannelIdentifier, opts ...ChannelOption) (Channel, error)
}
