package tui

import (
	"log"
	"time"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/google/uuid"
)

// TUIChannel implements the channel.Channel interface for the TUI.
// It forwards messages to the TUI model via the message channel for display.
type TUIChannel struct {
	id          uuid.UUID
	messageChan chan<- channel.Message
	agentID     uuid.UUID
	agentRole   string
}

// NewTUIChannel creates a new TUIChannel instance.
// The messageChan is used to send messages to the TUI model for display.
// If messageChan is nil, messages will be logged instead.
func NewTUIChannel(messageChan chan<- channel.Message) *TUIChannel {
	return &TUIChannel{
		id:          uuid.New(),
		messageChan: messageChan,
		agentID:     uuid.Nil, // Will be set by SetAgentInfo
		agentRole:   "assistant",
	}
}

// SetAgentInfo sets the agent ID and role for messages sent via this channel.
func (c *TUIChannel) SetAgentInfo(id uuid.UUID, role string) {
	c.agentID = id
	c.agentRole = role
}

// ID returns the unique identifier for this channel.
func (c *TUIChannel) ID() uuid.UUID {
	return c.id
}

// OnMessage handles incoming messages from the channel.
// Messages are sent to the TUI model's message channel for display.
func (c *TUIChannel) OnMessage(msg channel.Message) {
	if c.messageChan == nil {
		log.Printf("[TUIChannel] No message channel configured, message not displayed: %+v", msg)
		return
	}

	// Send message to TUI model
	// Use non-blocking send to avoid blocking if channel is full
	select {
	case c.messageChan <- msg:
		// Message sent successfully
	default:
		log.Printf("[TUIChannel] Message channel full, message dropped: %+v", msg)
	}
}

// OnLog handles log entries from the channel.
// Log entries are converted to system messages and sent to the TUI.
func (c *TUIChannel) OnLog(entry channel.LogEntry) {
	if c.messageChan == nil {
		log.Printf("[TUIChannel] No message channel configured, log entry not displayed: %+v", entry)
		return
	}

	// Convert log entry to a system message
	msg := channel.Message{
		Type:      channel.MessageTypeSystemInfo,
		Content:   formatLogEntry(entry),
		Timestamp: entry.Timestamp,
		AgentID:   c.agentID,
		AgentRole: c.agentRole,
	}

	// Send to TUI model
	select {
	case c.messageChan <- msg:
	default:
		log.Printf("[TUIChannel] Message channel full, log entry dropped: %+v", entry)
	}
}

// OnAgentLifecycle handles agent lifecycle events.
// Lifecycle events are converted to system messages and sent to the TUI.
func (c *TUIChannel) OnAgentLifecycle(event channel.AgentLifecycleEvent) {
	if c.messageChan == nil {
		log.Printf("[TUIChannel] No message channel configured, lifecycle event not displayed: %+v", event)
		return
	}

	// Convert lifecycle event to a system message
	msg := channel.Message{
		Type:      channel.MessageTypeSystemInfo,
		Content:   formatLifecycleEvent(event),
		Timestamp: time.Now(),
		AgentID:   event.AgentID,
		AgentRole: event.Role,
	}

	// Send to TUI model
	select {
	case c.messageChan <- msg:
	default:
		log.Printf("[TUIChannel] Message channel full, lifecycle event dropped: %+v", event)
	}
}

// formatLogEntry formats a log entry for display as a message.
func formatLogEntry(entry channel.LogEntry) string {
	// Format: [LEVEL] message
	prefix := "[" + entry.Level + "] "

	// Extract component from fields if present
	component := ""
	if comp, ok := entry.Fields["component"].(string); ok && comp != "" {
		component = comp + ": "
	}

	return prefix + component + entry.Message
}

// formatLifecycleEvent formats an agent lifecycle event for display as a message.
func formatLifecycleEvent(event channel.AgentLifecycleEvent) string {
	// Format: Agent lifecycle: action - agent_id
	action := "added"
	if !event.Added {
		action = "removed"
	}

	return "Agent " + action + ": " + event.AgentID.String() + " (" + event.Role + ")"
}

// Compile-time check to ensure TUIChannel implements channel.Channel
var _ channel.Channel = (*TUIChannel)(nil)
