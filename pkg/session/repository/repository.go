package repository

import (
	"context"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
)

// SessionRepository defines the interface for session persistence operations.
type SessionRepository interface {
	// Create persists a new session.
	Create(ctx context.Context, session *shared.Session) error

	// Get retrieves a session by ID.
	Get(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error)

	// List retrieves sessions with optional filtering.
	List(ctx context.Context, filter *SessionFilter) ([]*shared.Session, error)

	// Update updates an existing session.
	Update(ctx context.Context, session *shared.Session) error

	// Delete removes a session.
	Delete(ctx context.Context, sessionID uuid.UUID) error

	// Fork creates a copy of a session.
	Fork(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error)

	// Close marks a session as closed.
	Close(ctx context.Context, sessionID uuid.UUID) error

	// Archive removes old sessions.
	Archive(ctx context.Context, olderThan time.Duration) (int64, error)

	// AddMessage adds a message to a session.
	AddMessage(ctx context.Context, sessionID uuid.UUID, msg shared.Message) error

	// GetMessages retrieves messages from a session.
	GetMessages(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]shared.Message, error)

	// Exists checks if a session exists.
	Exists(ctx context.Context, sessionID uuid.UUID) (bool, error)
}

// SessionFilter defines filtering options for List operations.
type SessionFilter struct {
	ChannelID     uuid.UUID
	AgentID       uuid.UUID
	State         string
	CreatedAfter  time.Time
	CreatedBefore time.Time
}
