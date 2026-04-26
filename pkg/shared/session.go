// Package shared provides common types and interfaces used across the Gollum agent system.
package shared

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents an active session with a supervisor agent.
type Session struct {
	ID           uuid.UUID
	ChannelID    uuid.UUID
	SupervisorID uuid.UUID
	Context      context.Context
	CancelFunc   context.CancelFunc
	CreatedAt    time.Time
	Cwd          string // Current working directory for this session

	// Session-owned supervisor (lazy initialized)
	supervisor   Agent
	supervisorMu sync.RWMutex
}

// GetOrCreateSupervisor lazily creates and returns the session's supervisor agent.
// This method is thread-safe and uses double-checked locking.
func (s *Session) GetOrCreateSupervisor(factory AgentFactory) (Agent, error) {
	// Fast path: read lock to check if supervisor already exists
	s.supervisorMu.RLock()
	if s.supervisor != nil {
		s.supervisorMu.RUnlock()
		return s.supervisor, nil
	}
	s.supervisorMu.RUnlock()

	// Slow path: acquire write lock for creation
	s.supervisorMu.Lock()
	defer s.supervisorMu.Unlock()

	// Double-check: another goroutine might have created it while we waited
	if s.supervisor != nil {
		return s.supervisor, nil
	}

	// Create supervisor for this session
	supervisor, _, err := factory.CreateSupervisorAgent(
		s.Context,
		WithSessionID(s.ID),
		WithChannelID(s.ChannelID),
	)
	if err != nil {
		return nil, err
	}

	s.supervisor = supervisor
	s.SupervisorID = supervisor.GetID()
	return supervisor, nil
}

// Close cleans up session resources including the supervisor agent.
// This method is thread-safe and prevents memory leaks by releasing the supervisor reference.
func (s *Session) Close() error {
	s.supervisorMu.Lock()
	defer s.supervisorMu.Unlock()

	// Clear supervisor reference to allow garbage collection
	s.supervisor = nil

	return nil
}
