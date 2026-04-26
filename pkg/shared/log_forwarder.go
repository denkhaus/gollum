// Package shared provides shared types and interfaces for the Gollum application.
package shared

import (
	"time"
)

// LogEntry represents a log line for display polling
type LogEntry struct {
	// Level is the log level (e.g., "info", "error", "debug")
	Level string
	// Message is the log message content
	Message string
	// Timestamp is when the log entry was created
	Timestamp time.Time
	// Fields contains additional structured data for the log entry
	Fields map[string]any
	// SessionContext holds routing information (session, channel, agent)
	SessionContext
}

// LogForwarder defines the interface for forwarding log entries to the channel system.
// This allows the logger to route logs to specific channels without depending on
// the concrete ChannelFacade implementation.
type LogForwarder interface {
	// ForwardLog sends a log entry to the channel system for routing.
	// The entry's SessionID and ChannelID control which channel receives the log.
	ForwardLog(entry LogEntry)
}
