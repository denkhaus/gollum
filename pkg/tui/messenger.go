// Package tui provides Bubbletea-based terminal user interface components for Gollum.
//
// This file contains the messenger integration that allows AgentMessenger
// to send messages to the TUI via channels instead of printing to stdout.
package tui

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// MessageTypeAdapter adapts TUI MessageType for use in AgentMessenger.
// This allows the ui package to send messages without importing the tui package.
type MessageTypeAdapter int

const (
	// MessageTypeAdapterUser represents a message from the user.
	MessageTypeAdapterUser MessageTypeAdapter = iota
	// MessageTypeAdapterAgent represents a message from an agent (LLM output).
	MessageTypeAdapterAgent
	// MessageTypeAdapterTool represents a tool execution message.
	MessageTypeAdapterTool
	// MessageTypeAdapterSystem represents a system-level message.
	MessageTypeAdapterSystem
	// MessageTypeAdapterError represents an error message.
	MessageTypeAdapterError
)

// ToMessageType converts MessageTypeAdapter to TUI MessageType.
func (mt MessageTypeAdapter) ToMessageType() MessageType {
	return MessageType(mt)
}

// MessageAdapter is a simplified message structure for AgentMessenger
// to send messages to the TUI without importing the tui package.
type MessageAdapter struct {
	ID        uuid.UUID
	Type      MessageTypeAdapter
	Content   string
	AgentID   uuid.UUID
	AgentRole string
	IsTool    bool
}

// ToMessage converts MessageAdapter to TUI Message.
func (ma MessageAdapter) ToMessage() Message {
	return Message{
		ID:        ma.ID,
		Type:      ma.Type.ToMessageType(),
		Content:   ma.Content,
		Timestamp: time.Now(),
		AgentID:   ma.AgentID,
		AgentRole: ma.AgentRole,
		IsTool:    ma.IsTool,
		// Tool messages default to collapsed state
		Collapsed: ma.IsTool || ma.Type == MessageTypeAdapterTool,
	}
}

// MessengerChannel is the channel type for sending messages from AgentMessenger to TUI.
type MessengerChannel chan<- MessageAdapter

var (
	// globalMessengerChannel is the singleton channel for messenger communication.
	globalMessengerChannel MessengerChannel

	// globalMessengerMutex protects the global messenger channel.
	globalMessengerMutex sync.RWMutex
)

// SetMessengerChannel sets the global messenger channel for AgentMessenger integration.
// This allows AgentMessenger to send messages to the TUI instead of printing to stdout.
func SetMessengerChannel(ch MessengerChannel) {
	globalMessengerMutex.Lock()
	defer globalMessengerMutex.Unlock()
	globalMessengerChannel = ch
}

// GetMessengerChannel returns the global messenger channel.
func GetMessengerChannel() MessengerChannel {
	globalMessengerMutex.RLock()
	defer globalMessengerMutex.RUnlock()
	return globalMessengerChannel
}

// SendMessage sends a message to the TUI if the messenger channel is set.
// Returns true if the message was sent, false if no channel is configured.
func SendMessage(msg MessageAdapter) bool {
	if ch := GetMessengerChannel(); ch != nil {
		ch <- msg
		return true
	}
	return false
}
