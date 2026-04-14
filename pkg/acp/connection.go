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
	"net/http"

	"github.com/denkhaus/gollum/pkg/shared"
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
		return s.newHTTPConnection(handler)
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

// newStdioConnection creates a stdio-based ACP connection
func (s *acpServiceImpl) newStdioConnection() (Connection, error) {
	// Create session store
	store := acppkg.NewMemoryStore[*shared.ACPSession]()

	// Set ACP-specific fields
	s.SetClient(nil) // Will be set after connection creation
	s.SetSessionStore(store)

	// Create connection with session store and middleware
	conn := acppkg.NewAgentSideConnection(s, s.stdin, s.stdout,
		acppkg.WithSessionStore(store, func(ctx context.Context, params *acppkg.NewSessionRequest) (acppkg.SessionID, *shared.ACPSession, error) {
			ctx, cancel := context.WithCancel(ctx)
			return acppkg.GenerateSessionID(), shared.NewAcpSession(ctx, cancel), nil
		}),
		acppkg.WithMiddleware(acppkg.RecoveryMiddleware()),
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
func (s *acpServiceImpl) newHTTPConnection(handler http.Handler) (Connection, error) {
	// Create HTTP transport from handler
	httpTransport := acppkg.NewHTTPServerTransport()
	store := acppkg.NewMemoryStore[*shared.ACPSession]()

	// Set ACP-specific fields
	s.SetClient(nil)
	s.SetSessionStore(store)

	// Create connection with HTTP transport
	conn := acppkg.NewAgentSideConnection(s, nil, nil,
		acppkg.WithTransport(httpTransport),
		acppkg.WithSessionStore(store, func(parentCtx context.Context, params *acppkg.NewSessionRequest) (acppkg.SessionID, *shared.ACPSession, error) {
			// Use passed context as parent for cancellation propagation
			ctx, cancel := context.WithCancel(parentCtx)
			return acppkg.GenerateSessionID(), shared.NewAcpSession(ctx, cancel), nil
		}),
		acppkg.WithMiddleware(acppkg.RecoveryMiddleware()),
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
