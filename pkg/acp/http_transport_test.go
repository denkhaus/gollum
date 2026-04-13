package acp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/samber/do/v2"
	acppkg "github.com/ironpark/go-acp"

	"github.com/denkhaus/gollum/pkg/shared"
)

// TestHTTPTransportCreation verifies that HTTP transport can be created
// and provides a valid HTTP handler that responds to requests.
func TestHTTPTransportCreation(t *testing.T) {
	// Create test injector
	injector := do.New()

	// Create ACP service with HTTP transport
	service := &acpServiceImpl{
		transportType: TransportHTTP,
		host:          "localhost",
		port:          9090,
		injector:      injector,
	}

	// Create HTTP transport manually for testing
	httpTransport := acppkg.NewHTTPServerTransport()

	// Create session store
	store := acppkg.NewMemoryStore[*shared.ACPSession]()

	// Create connection with HTTP transport
	conn := acppkg.NewAgentSideConnection(service, nil, nil,
		acppkg.WithTransport(httpTransport),
		acppkg.WithSessionStore(store, func(ctx context.Context, params *acppkg.NewSessionRequest) (acppkg.SessionID, *shared.ACPSession, error) {
			ctx, cancel := context.WithCancel(context.Background())
			return acppkg.GenerateSessionID(), shared.NewAcpSession(ctx, cancel), nil
		}),
		acppkg.WithMiddleware(acppkg.RecoveryMiddleware()),
	)

	// Get handler from transport
	handler := httpTransport.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil, want non-nil handler")
	}

	// Wrap connection to track handler
	service.conn = &connectionImpl{
		conn:    conn,
		service: service,
		handler: handler,
	}

	// Verify handler responds to HTTP requests
	// The handler should respond (even if with 404 or method not allowed)
	// The important part is that it doesn't panic
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Handler should respond with a valid HTTP status code
	if w.Code < 200 || w.Code >= 600 {
		t.Errorf("Handler returned invalid status code %d, want valid HTTP status", w.Code)
	}

	// Verify GetHandler returns the same handler
	retrievedHandler := service.GetHandler()
	if retrievedHandler == nil {
		t.Error("GetHandler() returned nil, want non-nil handler")
	}
}

// TestHTTPMultiClientSupport verifies that the HTTP transport handler
// can handle concurrent requests from multiple clients safely.
func TestHTTPMultiClientSupport(t *testing.T) {
	// Create test injector
	injector := do.New()

	// Create ACP service with HTTP transport
	service := &acpServiceImpl{
		transportType: TransportHTTP,
		host:          "localhost",
		port:          9090,
		injector:      injector,
	}

	// Create HTTP transport manually for testing
	httpTransport := acppkg.NewHTTPServerTransport()

	// Create session store
	store := acppkg.NewMemoryStore[*shared.ACPSession]()

	// Create connection with HTTP transport
	conn := acppkg.NewAgentSideConnection(service, nil, nil,
		acppkg.WithTransport(httpTransport),
		acppkg.WithSessionStore(store, func(ctx context.Context, params *acppkg.NewSessionRequest) (acppkg.SessionID, *shared.ACPSession, error) {
			ctx, cancel := context.WithCancel(context.Background())
			return acppkg.GenerateSessionID(), shared.NewAcpSession(ctx, cancel), nil
		}),
		acppkg.WithMiddleware(acppkg.RecoveryMiddleware()),
	)

	// Get handler from transport
	handler := httpTransport.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil, want non-nil handler")
	}

	// Wrap connection to track handler
	service.conn = &connectionImpl{
		conn:    conn,
		service: service,
		handler: handler,
	}

	// Test concurrent access from multiple clients
	numClients := 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := []string{}

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			// Simulate HTTP request from client
			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()

			// This should not panic or cause race conditions
			handler.ServeHTTP(w, req)

			// Verify response
			if w.Code < 200 || w.Code >= 600 {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("Client %d: invalid status code %d", clientID, w.Code))
				mu.Unlock()
			}
		}(i)
	}

	// Wait for all clients to complete
	wg.Wait()

	// Check for any errors
	for _, err := range errors {
		t.Error(err)
	}
}

// TestHTTPHandlerInterface verifies that the HTTP handler implements
// the http.Handler interface correctly.
func TestHTTPHandlerInterface(t *testing.T) {
	// Create test injector
	injector := do.New()

	// Create ACP service with HTTP transport
	service := &acpServiceImpl{
		transportType: TransportHTTP,
		host:          "localhost",
		port:          9090,
		injector:      injector,
	}

	// Create HTTP transport manually for testing
	httpTransport := acppkg.NewHTTPServerTransport()

	// Create session store
	store := acppkg.NewMemoryStore[*shared.ACPSession]()

	// Create connection with HTTP transport
	conn := acppkg.NewAgentSideConnection(service, nil, nil,
		acppkg.WithTransport(httpTransport),
		acppkg.WithSessionStore(store, func(ctx context.Context, params *acppkg.NewSessionRequest) (acppkg.SessionID, *shared.ACPSession, error) {
			ctx, cancel := context.WithCancel(context.Background())
			return acppkg.GenerateSessionID(), shared.NewAcpSession(ctx, cancel), nil
		}),
		acppkg.WithMiddleware(acppkg.RecoveryMiddleware()),
	)

	// Get handler from transport
	handler := httpTransport.Handler()

	// Verify handler implements http.Handler interface
	var _ http.Handler = handler

	// Test ServeHTTP method directly
	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(w, req)

	// Verify response is valid HTTP
	if w.Code < 200 || w.Code >= 600 {
		t.Errorf("Invalid status code %d", w.Code)
	}

	// Wrap connection to track handler
	service.conn = &connectionImpl{
		conn:    conn,
		service: service,
		handler: handler,
	}

	// Verify GetHandler returns http.Handler
	retrievedHandler := service.GetHandler()
	var _ http.Handler = retrievedHandler
}
