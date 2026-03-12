package tui

import (
	"log"

	"github.com/denkhaus/gollum/pkg/channel"
)

// TUIChannel is a stub implementation of the channel.Channel interface for the TUI.
// Actual TUI integration will be implemented later.
type TUIChannel struct {
	id string
}

// NewTUIChannel creates a new TUIChannel instance.
func NewTUIChannel() *TUIChannel {
	return &TUIChannel{
		id: "tui",
	}
}

// ID returns the unique identifier for this channel.
func (c *TUIChannel) ID() string {
	return c.id
}

// OnMessage handles incoming messages from the channel.
// Currently a stub implementation that logs the message.
func (c *TUIChannel) OnMessage(msg channel.Message) {
	log.Printf("[TUIChannel] Received message: %+v", msg)
}

// OnLog handles log entries from the channel.
// Currently a stub implementation that logs the entry.
func (c *TUIChannel) OnLog(entry channel.LogEntry) {
	log.Printf("[TUIChannel] Log entry: %+v", entry)
}

// OnAgentLifecycle handles agent lifecycle events.
// Currently a stub implementation that logs the event.
func (c *TUIChannel) OnAgentLifecycle(event channel.AgentLifecycleEvent) {
	log.Printf("[TUIChannel] Agent lifecycle event: %+v", event)
}

// Compile-time check to ensure TUIChannel implements channel.Channel
var _ channel.Channel = (*TUIChannel)(nil)