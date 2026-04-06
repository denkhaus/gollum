// Package shared provides common types and interfaces used across the Gollum agent system.
package shared

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SessionManager manages active sessions
type SessionManager interface {
	// CreateSession creates a new session with the given ID (from ACP request).
	// The sessionID is provided by the caller, not generated internally.
	CreateSession(sessionID string, channelID uuid.UUID) (*Session, error)
	// GetOrCreateSession retrieves an existing session or creates a new one.
	GetOrCreateSession(sessionID string, channelID uuid.UUID) (*Session, error)
	// GetSession retrieves a session by its ID.
	GetSession(sessionID string) (*Session, bool)
	// CloseSession closes a session and cancels its context.
	CloseSession(sessionID string) error
	// GetSessionsByChannel returns all sessions for a given channel ID.
	GetSessionsByChannel(channelID uuid.UUID) []*Session
}

// Session represents an active session with a supervisor agent.
type Session struct {
	ID           string
	ChannelID    uuid.UUID
	SupervisorID uuid.UUID
	Context      context.Context
	CancelFunc   context.CancelFunc
	CreatedAt    time.Time
}
