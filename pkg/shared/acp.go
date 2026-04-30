package shared

import (
	"context"
	"net/http"
	"time"

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
	SetSessionStore(store acp.SessionStore[*Session])

	// HTTP transport support
	GetHandler() http.Handler
}

// NewSession creates a new Session with the given context and working directory.
// This is a convenience function for ACP session creation.
func NewSession(ctx context.Context, cancel context.CancelFunc, cwd string) *Session {
	return &Session{
		SessionContext: SessionContext{
			SessionID: uuid.New(),
			Cwd:       cwd,
		},
		Context:    ctx,
		CancelFunc: cancel,
		CreatedAt:  time.Now(),
	}
}
