// Package shared provides shared types and interfaces for the Gollum application.
package shared

import (
	"time"

	"github.com/google/uuid"
)

// LogEntry represents a log line for display polling
type LogEntry struct {
	Level     string
	Message   string
	Timestamp time.Time
	Fields    map[string]any
	SessionID string    // Session identifier for routing
	ChannelID uuid.UUID // Channel identifier for routing
}

// LogForwarder defines the interface for forwarding log entries to the channel system.
// This allows the logger to route logs to specific channels without depending on
// the concrete ChannelFacade implementation.
type LogForwarder interface {
	// ForwardLog sends a log entry to the channel system for routing.
	// The entry's SessionID and ChannelID control which channel receives the log.
	ForwardLog(entry LogEntry)
}
