// Package acp provides Agent Communication Protocol (ACP) support for Gollum.
//
// The Connection interface supports two transport modes:
//   - HTTP mode: Uses HTTP handler for web-based communication (Handler() returns non-nil)
//   - Stdio mode: Uses standard input/output for CLI-based communication (Handler() returns nil)
//
// The connection manages ACP sessions, middleware, and service lifecycle through
// dependency injection using the samber/do/v2 framework.
package acp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	acppkg "github.com/ironpark/go-acp"
)

// Connection defines the ACP connection interface
type Connection interface {
	Start(ctx context.Context) error
	Close() error
	Done() <-chan struct{}
	Handler() http.Handler // Returns handler for HTTP transport, nil for stdio
}

// connectionImpl implements Connection (PRIVATE)
type connectionImpl struct {
	conn    *acppkg.AgentSideConnection
	service shared.ACPService
	handler http.Handler // HTTP handler for HTTP transport
}

// newConnection creates a new ACP connection
//
// Modes:
//   - Stdio mode (handler == nil): uses s.stdin and s.stdout
//   - HTTP mode (handler != nil): uses the provided handler
func (s *acpServiceImpl) newConnection(handler http.Handler) (Connection, error) {
	// Detect HTTP mode by transport type
	if s.transportType == TransportHTTP {
		// HTTP mode: create HTTP connection
		return s.newHTTPConnection()
	}

	// Stdio mode: stdin and stdout are required
	if s.stdin == nil {
		return nil, errors.New("stdin cannot be nil for stdio mode")
	}
	if s.stdout == nil {
		return nil, errors.New("stdout cannot be nil for stdio mode")
	}

	return s.newStdioConnection()
}

func (s *acpServiceImpl) newSessionFactory(
	parentCtx context.Context,
	params *acppkg.NewSessionRequest,
) (acppkg.SessionID, *shared.Session, error) {

	// Use passed context as parent for cancellation propagation
	ctx, cancel := context.WithCancel(parentCtx)

	// s.sessionManager is initialized in constructor
	if s.sessionManager == nil {
		cancel()
		return "", nil, fmt.Errorf("sessionManager not initialized in newSessionFactory")
	}

	cwd := params.Cwd
	if cwd == "" {
		cwd = "."
	}
	// Generate session ID and parse it as UUID
	sessionID := uuid.New()
	acpSessionID := acppkg.SessionID(sessionID.String())

	// session := shared.NewAcpSession(ctx, cancel, cwd)
	// session.SessionID = sessionID

	// Register session in Gollum's SessionManager for system integration
	session, err := s.sessionManager.CreateSession(ctx, &shared.SessionContext{
		SessionID:        sessionID,
		ChannelID:        s.id,
		AgentID:          uuid.Nil, // Will be set when supervisor is created
		StartupDirectory: cwd,
	})
	if err != nil {
		cancel()
		return "", nil, fmt.Errorf("failed to register session in SessionManager: %w", err)
	}

	// TODO: Send session metadata to client after session creation
	// This should be done asynchronously to not block the session/new response
	//
	// Example implementation:
	// go func() {
	//     // Send available commands/skills to the client
	//     if err := s.sendAvailableCommands(ctx, acpSessionID); err != nil {
	//         s.logger.Warn("failed to send available commands", zap.Error(err))
	//     }
	//
	//     // Optionally send model info and config options
	//     if err := s.sendModelInfoUpdate(ctx, acpSessionID); err != nil {
	//         s.logger.Warn("failed to send model info", zap.Error(err))
	//     }
	// }()

	return acpSessionID, session, nil
}

// newStdioConnection creates a stdio-based ACP connection
func (s *acpServiceImpl) newStdioConnection() (Connection, error) {
	// Create session store
	store := acppkg.NewMemoryStore[*shared.Session]()

	// Set ACP-specific fields
	s.SetClient(nil) // Will be set after connection creation
	s.SetSessionStore(store)

	// Create connection with session store and middleware
	// LoggingMiddleware logs all ACP method calls for debugging
	conn := acppkg.NewAgentSideConnection(s, s.stdin, s.stdout,
		acppkg.WithSessionStore(store, s.newSessionFactory),
		acppkg.WithMiddleware(
			acppkg.RecoveryMiddleware(),
			acppkg.LoggingMiddleware(nil), // Use default logger (log.Printf)
		),
	)

	// Set client on service
	s.SetClient(conn.Client())

	return &connectionImpl{
		conn:    conn,
		service: s,
		handler: nil, // Stdio mode has no handler
	}, nil
}

// newHTTPConnection creates an HTTP-based ACP connection
func (s *acpServiceImpl) newHTTPConnection() (Connection, error) {
	// Create HTTP transport from handler
	httpTransport := acppkg.NewHTTPServerTransport()
	store := acppkg.NewMemoryStore[*shared.Session]()

	// Set ACP-specific fields
	s.SetClient(nil)
	s.SetSessionStore(store)

	// Create connection with HTTP transport and middleware
	// LoggingMiddleware logs all ACP method calls for debugging
	conn := acppkg.NewAgentSideConnection(s, nil, nil,
		acppkg.WithTransport(httpTransport),
		acppkg.WithSessionStore(store, s.newSessionFactory),
		acppkg.WithMiddleware(
			acppkg.RecoveryMiddleware(),
			acppkg.LoggingMiddleware(nil), // Use default logger (log.Printf)
		),
	)

	// Set client on service
	s.SetClient(conn.Client())

	return &connectionImpl{
		conn:    conn,
		service: s,
		handler: httpTransport.Handler(),
	}, nil
}

func (p *connectionImpl) Start(ctx context.Context) error {
	return p.conn.Start(ctx)
}

func (p *connectionImpl) Close() error {
	// Note: Channel unregistration is handled by the service lifecycle
	return p.conn.Close()
}

func (p *connectionImpl) Done() <-chan struct{} {
	return p.conn.Done()
}

func (p *connectionImpl) Handler() http.Handler {
	return p.handler
}
