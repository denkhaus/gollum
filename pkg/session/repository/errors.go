package repository

import (
	"fmt"

	"github.com/google/uuid"
)

var (
	// ErrSessionNotFound indicates a session was not found.
	ErrSessionNotFound = fmt.Errorf("session not found")

	// ErrSessionClosed indicates a session is already closed.
	ErrSessionClosed = fmt.Errorf("session already closed")

	// ErrSessionCannotFork indicates a session cannot be forked.
	ErrSessionCannotFork = fmt.Errorf("session cannot be forked")

	// ErrDatabaseConnection indicates a database connection failure.
	ErrDatabaseConnection = fmt.Errorf("database connection failed")

	// ErrDatabaseMigration indicates a migration failure.
	ErrDatabaseMigration = fmt.Errorf("database migration failed")
)

// SessionError wraps errors with session context.
type SessionError struct {
	SessionID uuid.UUID
	Op        string
	Err       error
}

// Error returns the error message.
func (e *SessionError) Error() string {
	return fmt.Sprintf("session %s: %s failed: %v", e.SessionID, e.Op, e.Err)
}

// Unwrap returns the underlying error.
func (e *SessionError) Unwrap() error {
	return e.Err
}
