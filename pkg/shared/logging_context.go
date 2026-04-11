// Package shared provides shared types and interfaces for the Gollum application.
package shared

import (
	"github.com/google/uuid"
)

// LoggingContext holds routing information for session/channel-aware logging.
// All fields must be valid for the context to be considered complete.
type LoggingContext struct {
	SessionID string    // Session identifier for routing
	ChannelID uuid.UUID // Channel identifier for routing
	AgentID   uuid.UUID // Agent identifier for the log source
}

// IsValid returns true if all required fields are populated with valid values.
func (c LoggingContext) IsValid() bool {
	return c.SessionID != "" &&
		c.ChannelID != uuid.Nil &&
		c.AgentID != uuid.Nil
}
