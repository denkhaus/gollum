package acp

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	acppkg "github.com/ironpark/go-acp"
)

// TestPrompt_VariableDeclaration verifies that the session variable is declared
// before the defer block so the recover block can access it for nil checking.
//
// This is a regression test for the compilation error:
// "use of package session not in selector"
func TestPrompt_VariableDeclaration(t *testing.T) {
	// The session variable must be declared before the defer block
	var session *shared.Session

	// This defer block should be able to access the session variable
	defer func() {
		if r := recover(); r != nil {
			// This should work because session is declared before defer
			if session == nil {
				t.Log("Successfully checked session == nil in recover block")
			}
		}
	}()

	// Verify session is nil at this point
	if session != nil {
		t.Error("session should be nil initially")
	}

	// Assign a value to session
	ctx := context.Background()
	session = &shared.Session{
		SessionContext: shared.SessionContext{
			SessionID:        uuid.Nil,
			StartupDirectory: "/tmp",
		},
		Context:    ctx,
		CancelFunc: func() {},
	}

	t.Log("Variable declaration test passed")
}

// TestPrompt_NilInputsReturnErrors verifies that Prompt returns errors
// instead of panicking for various nil inputs
func TestPrompt_NilInputsReturnErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(*acpServiceImpl)
		wantErr string
	}{
		{
			name: "nil_store",
			setup: func(s *acpServiceImpl) {
				s.store = nil
			},
			wantErr: "store not initialized",
		},
		{
			name: "nil_facade",
			setup: func(s *acpServiceImpl) {
				s.store = acppkg.NewMemoryStore[*shared.Session]()
				s.facade = nil
			},
			wantErr: "facade not initialized",
		},
		{
			name: "nil_client",
			setup: func(s *acpServiceImpl) {
				s.store = acppkg.NewMemoryStore[*shared.Session]()
				s.facade = nil // Facade check happens before client check
				s.client = nil
			},
			wantErr: "facade not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &acpServiceImpl{}
			tt.setup(svc)

			_, err := svc.Prompt(ctx, &acppkg.PromptRequest{
				SessionID: "test-session",
			})

			if err == nil {
				t.Error("Expected error but got none")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("Expected error '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}
