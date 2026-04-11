// Package channel provides the channel abstraction layer for Gollum.
package channel

import (
	"context"

	"github.com/google/uuid"
)

// Channel is the interface that all channel implementations must satisfy
type Channel interface {
	// ID returns a unique identifier for this channel
	ID() uuid.UUID

	// OnMessage is called when a new message should be displayed
	OnMessage(msg Message)

	// OnLog is called for log entries (channel can ignore if not applicable)
	OnLog(entry LogEntry)

	// OnAgentLifecycle is called when agent registration/removal events occur
	OnAgentLifecycle(event AgentLifecycleEvent)
}

// ChannelFacade is the central coordinator for all channels
type ChannelFacade interface {
	// DisplayMessage sends a message to all registered channels
	DisplayMessage(msg Message)

	// DisplayLog sends a log entry to channels (entry.SessionID and entry.ChannelID control routing)
	DisplayLog(entry LogEntry)

	// SubmitInput handles user input from any channel
	SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (InputResult, error)

	// CancelInput cancels an in-flight input for the given session
	CancelInput(sessionID string) error

	// RegisterChannel adds a channel to receive events
	RegisterChannel(channel Channel) error

	// UnregisterChannel removes a channel
	UnregisterChannel(channelID uuid.UUID) error

	// NotifyAgentLifecycle broadcasts agent lifecycle event
	NotifyAgentLifecycle(agentID uuid.UUID, role string, added bool)
}
