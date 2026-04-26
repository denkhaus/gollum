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
	"github.com/denkhaus/gollum/pkg/session/repository"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
)

// SessionManager manages active sessions
type SessionManager interface {
	// CreateSession creates a new session with the given context.
	CreateSession(ctx *shared.SessionContext) (*shared.Session, error)
	// GetOrCreateSession retrieves an existing session or creates a new one.
	GetOrCreateSession(ctx *shared.SessionContext) (*shared.Session, error)
	// GetSession retrieves a session by its ID.
	GetSession(sessionID uuid.UUID) (*shared.Session, bool)
	// CloseSession closes a session and cancels its context.
	CloseSession(sessionID uuid.UUID) error
	// GetSessionsByChannel returns all sessions for a given channel ID.
	GetSessionsByChannel(channelID uuid.UUID) []*shared.Session

	// LoadSession loads a session from persistence.
	LoadSession(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error)

	// ListSessions lists all sessions.
	ListSessions(ctx context.Context) ([]*shared.Session, error)

	// ResumeSession resumes a closed session.
	ResumeSession(ctx context.Context, sessionID uuid.UUID) error

	// ForkSession creates a copy of a session.
	ForkSession(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error)
}

// sessionManagerImpl implements SessionManager with in-memory storage and persistence.
type sessionManagerImpl struct {
	sessions sync.Map
	repo     repository.SessionRepository
}

// Ensure sessionManagerImpl implements SessionManager at compile time
var _ SessionManager = (*sessionManagerImpl)(nil)

// NewSessionManager creates a new SessionManager (DI constructor).
func NewSessionManager(injector do.Injector) (SessionManager, error) {
	repo, err := do.Invoke[repository.SessionRepository](injector)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke SessionRepository: %w", err)
	}

	return &sessionManagerImpl{
		repo: repo,
	}, nil
}

// CreateSession creates a new session with the given context.
func (p *sessionManagerImpl) CreateSession(ctx *shared.SessionContext) (*shared.Session, error) {
	sessionCtx, cancel := context.WithCancel(context.Background())
	session := &shared.Session{
		ID:         ctx.SessionID,
		ChannelID:  ctx.ChannelID,
		Context:    sessionCtx,
		CancelFunc: cancel,
		CreatedAt:  time.Now(),
		Cwd:        ctx.Cwd,
	}

	// Persist to repository
	if err := p.repo.Create(context.Background(), session); err != nil {
		cancel() // Clean up context on failure
		return nil, fmt.Errorf("failed to persist session: %w", err)
	}

	p.sessions.Store(ctx.SessionID.String(), session)
	return session, nil
}

// GetOrCreateSession retrieves an existing session or creates a new one.
func (p *sessionManagerImpl) GetOrCreateSession(ctx *shared.SessionContext) (*shared.Session, error) {
	if session, ok := p.GetSession(ctx.SessionID); ok {
		return session, nil
	}
	return p.CreateSession(ctx)
}

// GetSession retrieves a session by its ID.
func (p *sessionManagerImpl) GetSession(sessionID uuid.UUID) (*shared.Session, bool) {
	if val, ok := p.sessions.Load(sessionID.String()); ok {
		return val.(*shared.Session), true
	}
	return nil, false
}

// CloseSession closes a session and cancels its context.
// It also cleans up the supervisor agent to prevent memory leaks.
func (p *sessionManagerImpl) CloseSession(sessionID uuid.UUID) error {
	if val, ok := p.sessions.Load(sessionID.String()); ok {
		session := val.(*shared.Session)

		// Cleanup supervisor to prevent memory leak
		if err := session.Close(); err != nil {
			// Log but don't fail - context cancellation is more important
			// The supervisor reference will still be cleared
		}

		// Cancel the session context
		session.CancelFunc()

		// Mark as closed in repository
		if err := p.repo.Close(context.Background(), sessionID); err != nil {
			// Log but don't fail - session is already removed from memory
		}

		// Remove from session map
		p.sessions.Delete(sessionID.String())
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

// LoadSession loads a session from persistence.
func (p *sessionManagerImpl) LoadSession(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error) {
	// Check if already in memory
	if val, ok := p.sessions.Load(sessionID.String()); ok {
		return val.(*shared.Session), nil
	}

	// Load from repository
	session, err := p.repo.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Add to in-memory cache
	p.sessions.Store(sessionID.String(), session)

	return session, nil
}

// ListSessions lists all sessions.
func (p *sessionManagerImpl) ListSessions(ctx context.Context) ([]*shared.Session, error) {
	return p.repo.List(ctx, nil)
}

// ResumeSession resumes a closed session.
func (p *sessionManagerImpl) ResumeSession(ctx context.Context, sessionID uuid.UUID) error {
	// Load session (from repo or cache)
	session, err := p.LoadSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Create new context
	sessionCtx, cancel := context.WithCancel(context.Background())
	session.Context = sessionCtx
	session.CancelFunc = cancel

	// Update in repository
	if err := p.repo.Update(ctx, session); err != nil {
		cancel() // Clean up context on failure
		return err
	}

	return nil
}

// ForkSession creates a copy of a session.
func (p *sessionManagerImpl) ForkSession(ctx context.Context, sessionID uuid.UUID) (*shared.Session, error) {
	// Fork in repository
	newSession, err := p.repo.Fork(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Add to in-memory cache
	p.sessions.Store(newSession.ID.String(), newSession)

	return newSession, nil
}
