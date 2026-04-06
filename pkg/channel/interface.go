// Package channel provides the channel abstraction layer for Gollum.
package channel

import (
	"context"
	"time"

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

// CommandHandler is a function that executes a slash command
type CommandHandler func(ctx context.Context, args string) (string, error)

// Command represents a registered slash command
type Command struct {
	Name        string
	Description string
	Handler     CommandHandler
}

// CommandManager handles slash command registration and execution
type CommandManager interface {
	// Register adds a new slash command
	Register(cmd Command) error

	// Unregister removes a command
	Unregister(name string) error

	// Execute parses input and executes command if it starts with "/"
	Execute(ctx context.Context, input string) (handled bool, response string, err error)

	// List returns all available commands
	List() []Command

	// IsCommand checks if input starts with "/"
	IsCommand(input string) bool
}

// ChannelFacade is the central coordinator for all channels
type ChannelFacade interface {
	// DisplayMessage sends a message to all registered channels
	DisplayMessage(msg Message)

	// DisplayLog sends a log entry to all registered channels
	DisplayLog(entry LogEntry)

	// SubmitInput handles user input from any channel
	SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (InputResult, error)

	// CancelInput cancels an in-flight input for the given channel
	CancelInput(channelID uuid.UUID) error

	// GetLogs returns recent log entries for channels to poll
	GetLogs(since time.Time, limit int) []LogEntry

	// RegisterChannel adds a channel to receive events
	RegisterChannel(channel Channel) error

	// UnregisterChannel removes a channel
	UnregisterChannel(channelID uuid.UUID) error

	// NotifyAgentLifecycle broadcasts agent lifecycle event
	NotifyAgentLifecycle(agentID uuid.UUID, role string, added bool)
}
