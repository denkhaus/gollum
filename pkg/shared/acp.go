package shared

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/ironpark/go-acp"
)

// Service defines the public interface for ACP agent operations
type ACPService interface {
	// acp.Agent interface methods
	Initialize(ctx context.Context, params *acp.InitializeRequest) (*acp.InitializeResponse, error)
	Authenticate(ctx context.Context, params *acp.AuthenticateRequest) (*acp.AuthenticateResponse, error)
	SetSessionMode(ctx context.Context, params *acp.SetSessionModeRequest) (*acp.SetSessionModeResponse, error)
	SetSessionConfigOption(ctx context.Context, params *acp.SetSessionConfigOptionRequest) (*acp.SetSessionConfigOptionResponse, error)
	Prompt(ctx context.Context, params *acp.PromptRequest) (*acp.PromptResponse, error)
	Cancel(ctx context.Context, params *acp.CancelNotification) error

	// Dependency injection setters (called by connection factory)
	SetClient(client acp.Client)
	SetSessionStore(store acp.SessionStore[*ACPSession])

	// HTTP transport support
	GetHandler() http.Handler
}

// ACPSession holds session state for ACP connections
type ACPSession struct {
	// Context for the current prompt/turn - can be cancelled
	Context context.Context
	// CancelFunc cancels the current turn
	CancelFunc context.CancelFunc
	// SessionID from ACP protocol (UUID)
	SessionID uuid.UUID
	// Cwd is the current working directory for this session
	Cwd string
}

// NewAcpSession creates a new session with cancellable context and working directory
func NewAcpSession(ctx context.Context, cancel context.CancelFunc, cwd string) *ACPSession {
	return &ACPSession{
		Context:    ctx,
		CancelFunc: cancel,
		Cwd:        cwd,
	}
}
