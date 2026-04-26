// Package shared provides shared types and interfaces for the Gollum application.
package shared

import (
	"fmt"

	"github.com/google/uuid"
)

// SessionContext holds routing information for session-aware operations.
// It is used across logging, messaging, and agent lifecycle management.
// All routing fields (SessionID, ChannelID, AgentID) must be valid for the context to be complete.
type SessionContext struct {
	SessionID uuid.UUID // Session identifier for routing
	ChannelID uuid.UUID // Channel identifier for routing
	AgentID   uuid.UUID // Agent identifier for the log source
	Cwd       string    // Current working directory for this session
}

// IsValid returns true if all required routing fields are populated with valid values.
// Cwd is optional and may be empty.
func (c SessionContext) IsValid() bool {
	return c.SessionID != uuid.Nil &&
		c.ChannelID != uuid.Nil &&
		c.AgentID != uuid.Nil
}

// String returns a formatted representation of the session context for logging.
// Example: "session=abc-123 channel=def-456 agent=ghi-789 cwd=/home/user/project"
func (c SessionContext) String() string {
	cwd := c.Cwd
	if cwd == "" {
		cwd = "."
	}
	return fmt.Sprintf("session=%s channel=%s agent=%s cwd=%s",
		c.SessionID.String(),
		c.ChannelID.String(),
		c.AgentID.String(),
		cwd,
	)
}

// NewSessionContext creates a SessionContext with optional working directory.
func NewSessionContext(sessionID uuid.UUID, agentID uuid.UUID, channelID uuid.UUID, cwd string) *SessionContext {
	return &SessionContext{
		SessionID: sessionID,
		ChannelID: channelID,
		AgentID:   agentID,
		Cwd:       cwd,
	}
}
