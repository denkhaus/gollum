package acp

import (
	"context"

	acppkg "github.com/ironpark/go-acp"
)

// AcpSession holds session state for ACP connections
type AcpSession struct {
	// Context for the current prompt/turn - can be cancelled
	Context context.Context
	// CancelFunc cancels the current turn
	CancelFunc context.CancelFunc
	// SessionID from ACP protocol
	SessionID acppkg.SessionID
}

// NewAcpSession creates a new session with cancellable context
func NewAcpSession(ctx context.Context, cancel context.CancelFunc) *AcpSession {
	return &AcpSession{
		Context:    ctx,
		CancelFunc: cancel,
	}
}
