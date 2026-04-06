// Package session provides session management for Gollum.
//
// The SessionManager is responsible for creating and tracking active sessions,
// each associated with a channel and a supervisor agent. Sessions maintain
// their own context for cancellation and lifecycle management.
package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
)

// SessionManager manages active sessions
type SessionManager interface {
	// CreateSession creates a new session with the given ID (from ACP request).
	// The sessionID is provided by the caller, not generated internally.
	CreateSession(sessionID string, channelID uuid.UUID) (*shared.Session, error)
	// GetOrCreateSession retrieves an existing session or creates a new one.
	GetOrCreateSession(sessionID string, channelID uuid.UUID) (*shared.Session, error)
	// GetSession retrieves a session by its ID.
	GetSession(sessionID string) (*shared.Session, bool)
	// CloseSession closes a session and cancels its context.
	CloseSession(sessionID string) error
	// GetSessionsByChannel returns all sessions for a given channel ID.
	GetSessionsByChannel(channelID uuid.UUID) []*shared.Session
}

// sessionManagerImpl implements SessionManager with in-memory storage.
type sessionManagerImpl struct {
	sessions sync.Map
}

// Ensure sessionManagerImpl implements SessionManager at compile time
var _ SessionManager = (*sessionManagerImpl)(nil)

// NewSessionManager creates a new SessionManager (DI constructor).
func NewSessionManager(injector do.Injector) (SessionManager, error) {
	return &sessionManagerImpl{}, nil
}

// CreateSession creates a new session with the given ID from ACP request.
func (p *sessionManagerImpl) CreateSession(sessionID string, channelID uuid.UUID) (*shared.Session, error) {
	// ID is provided from ACP request, not generated here
	sessionCtx, cancel := context.WithCancel(context.Background())
	session := &shared.Session{
		ID:         sessionID,
		ChannelID:  channelID,
		Context:    sessionCtx,
		CancelFunc: cancel,
		CreatedAt:  time.Now(),
	}
	p.sessions.Store(sessionID, session)
	return session, nil
}

// GetOrCreateSession retrieves an existing session or creates a new one.
func (p *sessionManagerImpl) GetOrCreateSession(sessionID string, channelID uuid.UUID) (*shared.Session, error) {
	if session, ok := p.GetSession(sessionID); ok {
		return session, nil
	}
	return p.CreateSession(sessionID, channelID)
}

// GetSession retrieves a session by its ID.
func (p *sessionManagerImpl) GetSession(sessionID string) (*shared.Session, bool) {
	if val, ok := p.sessions.Load(sessionID); ok {
		return val.(*shared.Session), true
	}
	return nil, false
}

// CloseSession closes a session and cancels its context.
// It also cleans up the supervisor agent to prevent memory leaks.
func (p *sessionManagerImpl) CloseSession(sessionID string) error {
	if val, ok := p.sessions.Load(sessionID); ok {
		session := val.(*shared.Session)

		// Cleanup supervisor to prevent memory leak
		if err := session.Close(); err != nil {
			// Log but don't fail - context cancellation is more important
			// The supervisor reference will still be cleared
		}

		// Cancel the session context
		session.CancelFunc()

		// Remove from session map
		p.sessions.Delete(sessionID)
		return nil
	}
	return fmt.Errorf("session not found")
}

// GetSessionsByChannel returns all sessions for a given channel ID.
func (p *sessionManagerImpl) GetSessionsByChannel(channelID uuid.UUID) []*shared.Session {
	var result []*shared.Session
	p.sessions.Range(func(key, value any) bool {
		session := value.(*shared.Session)
		if session.ChannelID == channelID {
			result = append(result, session)
		}
		return true
	})
	return result
}
