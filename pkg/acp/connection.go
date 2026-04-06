package acp

import (
	"context"
	"errors"
	"io"

	"github.com/denkhaus/gollum/pkg/shared"
	acppkg "github.com/ironpark/go-acp"
	"github.com/samber/do/v2"
)

// Connection defines the ACP connection interface
type Connection interface {
	Start(ctx context.Context) error
	Close() error
	Done() <-chan struct{}
}

// connectionImpl implements Connection (PRIVATE)
type connectionImpl struct {
	conn    *acppkg.AgentSideConnection
	service shared.ACPService
}

// NewConnection creates a new ACP connection with DI
func NewConnection(injector do.Injector, reader io.Reader, writer io.Writer) (Connection, error) {
	// Validate parameters
	if reader == nil {
		return nil, errors.New("reader cannot be nil")
	}
	if writer == nil {
		return nil, errors.New("writer cannot be nil")
	}

	// Create session store
	store := acppkg.NewMemoryStore[*shared.ACPSession]()
	acpService := do.MustInvoke[shared.ACPService](injector)

	// Set ACP-specific fields
	acpService.SetClient(nil) // Will be set after connection creation
	acpService.SetSessionStore(store)

	// Create connection with session store and middleware
	conn := acppkg.NewAgentSideConnection(acpService, reader, writer,
		acppkg.WithSessionStore(store, func(ctx context.Context, params *acppkg.NewSessionRequest) (acppkg.SessionID, *shared.ACPSession, error) {
			ctx, cancel := context.WithCancel(context.Background())
			return acppkg.GenerateSessionID(), shared.NewAcpSession(ctx, cancel), nil
		}),
		acppkg.WithMiddleware(acppkg.RecoveryMiddleware()),
	)

	// Set client on service
	acpService.SetClient(conn.Client())

	return &connectionImpl{
		conn:    conn,
		service: acpService,
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
