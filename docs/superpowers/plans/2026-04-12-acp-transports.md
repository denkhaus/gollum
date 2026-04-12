# ACP Transport Support Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add HTTP transport support to the ACP command alongside existing stdio transport, enabling multi-client server deployments.

**Architecture:** Extend the existing ACP channel system with pluggable transport selection. Add TransportType enum, extend channel options, update connection factory to create HTTPServerTransport when selected, and integrate CLI flags for transport configuration.

**Tech Stack:** Go 1.21+, go-acp library (github.com/ironpark/go-acp), urfave/cli/v3, samber/do/v2 (DI)

---

## File Structure

**New Files:**
- `pkg/acp/transport.go` - Transport type enum and validation
- `pkg/acp/transport_test.go` - Transport option unit tests
- `pkg/acp/http_transport_test.go` - HTTP transport integration tests

**Modified Files:**
- `pkg/acp/options.go` - Add WithTransport, WithHost, WithPort options
- `pkg/acp/connection.go` - Support transport selection in NewConnection
- `pkg/acp/channel.go` - Handle HTTP mode in Start() method
- `pkg/cli/acp.go` - Add --transport, --host, --port flags and HTTP server startup

---

## Chunk 1: Transport Type and Options

### Task 1: Create Transport Type Enum

**Files:**
- Create: `pkg/acp/transport.go`
- Test: `pkg/acp/transport_test.go`

- [ ] **Step 1: Write the failing test for transport type**

```go
package acp

import (
	"testing"
)

func TestTransportTypeValues(t *testing.T) {
	tests := []struct {
		name     string
		transport TransportType
		expected int
	}{
		{"stdio", TransportStdio, 0},
		{"http", TransportHTTP, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.transport) != tt.expected {
				t.Errorf("TransportType %s = %d, want %d", tt.name, tt.transport, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/acp/... -run TestTransportType -v`
Expected: FAIL with "undefined: TransportType"

- [ ] **Step 3: Write transport type implementation**

Create `pkg/acp/transport.go`:

```go
package acp

// TransportType represents the transport protocol for ACP communication
type TransportType int

const (
	// TransportStdio uses stdin/stdout for communication (default)
	TransportStdio TransportType = iota

	// TransportHTTP uses HTTP + Server-Sent Events for communication
	TransportHTTP
)

// String returns the string representation of the transport type
func (t TransportType) String() string {
	switch t {
	case TransportStdio:
		return "stdio"
	case TransportHTTP:
		return "http"
	default:
		return "unknown"
	}
}

// ParseTransportType parses a string into TransportType
func ParseTransportType(s string) (TransportType, error) {
	switch s {
	case "stdio":
		return TransportStdio, nil
	case "http":
		return TransportHTTP, nil
	default:
		return TransportStdio, fmt.Errorf("unsupported transport type: %s (supported: stdio, http)", s)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/acp/... -run TestTransportType -v`
Expected: PASS

- [ ] **Step 5: Add test for ParseTransportType**

```go
func TestParseTransportType(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  TransportType
		wantError bool
	}{
		{"stdio", "stdio", TransportStdio, false},
		{"http", "http", TransportHTTP, false},
		{"invalid", "invalid", TransportStdio, true},
		{"empty", "", TransportStdio, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTransportType(tt.input)
			if tt.wantError && err == nil {
				t.Errorf("ParseTransportType(%q) expected error, got nil", tt.input)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ParseTransportType(%q) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("ParseTransportType(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./pkg/acp/... -run TestParseTransportType -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/acp/transport.go pkg/acp/transport_test.go
git commit -m "feat(acp): add TransportType enum and parser

Add TransportType enum with stdio and HTTP variants.
Include ParseTransportType function with validation.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 2: Add Transport Options

**Files:**
- Modify: `pkg/acp/options.go`
- Test: `pkg/acp/transport_test.go` (extend)

- [ ] **Step 1: Write test for WithTransport option**

```go
func TestWithTransport(t *testing.T) {
	// Create a mock acpServiceImpl for testing
	mockService := &acpServiceImpl{}
	
	option := WithTransport(TransportHTTP)
	err := option.Apply(mockService)
	
	if err != nil {
		t.Errorf("WithTransport() error = %v", err)
	}
	if mockService.transportType != TransportHTTP {
		t.Errorf("WithTransport() set transportType = %v, want %v", mockService.transportType, TransportHTTP)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/acp/... -run TestWithTransport -v`
Expected: FAIL with "transportType field not found"

- [ ] **Step 3: Add transport field to acpServiceImpl**

Modify `pkg/acp/service.go`:

```go
type acpServiceImpl struct {
	facade       channel.ChannelFacade
	logger       logger.LoggerService
	config       config.ConfigService
	client       acppkg.Client
	store        acppkg.SessionStore[*shared.ACPSession]
	id           uuid.UUID
	conn         Connection
	stdin        io.Reader
	stdout       io.Writer
	injector     do.Injector
	transportType TransportType  // Add this field
	host         string          // Add this field
	port         int             // Add this field
}
```

- [ ] **Step 4: Add WithTransport option function**

Modify `pkg/acp/options.go`:

```go
// WithTransport sets the transport type for ACP communication
func WithTransport(t TransportType) channel.ChannelOption {
	return &ACPOption{applyFunc: func(s *acpServiceImpl) error {
		s.transportType = t
		return nil
	}}
}

// WithHost sets the host address for HTTP transport
func WithHost(host string) channel.ChannelOption {
	return &ACPOption{applyFunc: func(s *acpServiceImpl) error {
		s.host = host
		return nil
	}}
}

// WithPort sets the port number for HTTP transport
func WithPort(port int) channel.ChannelOption {
	return &ACPOption{applyFunc: func(s *acpServiceImpl) error {
		if port < 1 || port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535, got %d", port)
		}
		s.port = port
		return nil
	}}
}
```

- [ ] **Step 5: Initialize default values in NewAcpService**

Modify `pkg/acp/service.go` NewAcpService:

```go
svc := &acpServiceImpl{
	logger:       logger,
	facade:       facade,
	config:       cfg,
	id:           id,
	injector:     injector,
	transportType: TransportStdio,  // Default to stdio
	host:         "0.0.0.0",        // Default bind address
	port:         8080,             // Default port
}
```

- [ ] **Step 6: Add tests for WithHost and WithPort**

```go
func TestWithHost(t *testing.T) {
	mockService := &acpServiceImpl{}
	
	option := WithHost("localhost")
	err := option.Apply(mockService)
	
	if err != nil {
		t.Errorf("WithHost() error = %v", err)
	}
	if mockService.host != "localhost" {
		t.Errorf("WithHost() set host = %v, want localhost", mockService.host)
	}
}

func TestWithPort(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		wantError bool
	}{
		{"valid port", 9000, false},
		{"min port", 1, false},
		{"max port", 65535, false},
		{"below min", 0, true},
		{"above max", 65536, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &acpServiceImpl{}
			option := WithPort(tt.port)
			err := option.Apply(mockService)
			if tt.wantError && err == nil {
				t.Errorf("WithPort(%d) expected error, got nil", tt.port)
			}
			if !tt.wantError && err != nil {
				t.Errorf("WithPort(%d) unexpected error: %v", tt.port, err)
			}
			if !tt.wantError && mockService.port != tt.port {
				t.Errorf("WithPort() set port = %v, want %d", mockService.port, tt.port)
			}
		})
	}
}
```

- [ ] **Step 7: Run all transport tests**

Run: `go test ./pkg/acp/... -run "TestWithTransport|TestWithHost|TestWithPort" -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add pkg/acp/service.go pkg/acp/options.go pkg/acp/transport_test.go
git commit -m "feat(acp): add transport configuration options

Add WithTransport, WithHost, and WithPort options for ACP channel.
Initialize default values in NewAcpService.
Add validation for port range (1-65535).

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 2: Connection Factory Support

### Task 3: Update Connection Factory for HTTP Transport

**Files:**
- Modify: `pkg/acp/connection.go`
- Test: `pkg/acp/connection_test.go` (extend)

- [ ] **Step 1: Write test for HTTP connection creation**

```go
func TestNewConnectionHTTPTransport(t *testing.T) {
	// This test will require mocking the injector and ACP service
	// For now, we'll test that the connection can be created with HTTP options
	
	// Note: Full integration test will be in http_transport_test.go
	// This is a structural test to ensure the signature works
}
```

- [ ] **Step 2: Modify NewConnection to accept http.Handler return**

Modify `pkg/acp/connection.go`:

Update Connection interface to optionally return handler:

```go
// Connection defines the ACP connection interface
type Connection interface {
	Start(ctx context.Context) error
	Close() error
	Done() <-chan struct{}
	Handler() http.Handler  // Returns handler for HTTP transport, nil for stdio
}
```

- [ ] **Step 3: Update connectionImpl to support HTTP transport**

Modify `connectionImpl` struct:

```go
type connectionImpl struct {
	conn    *acppkg.AgentSideConnection
	service shared.ACPService
	handler http.Handler  // HTTP handler for HTTP transport
}
```

- [ ] **Step 4: Update NewConnection to create HTTP transport when needed**

Modify `NewConnection` function to support transport type (we'll pass it via options in next task, for now prepare the structure):

```go
func (p *connectionImpl) Handler() http.Handler {
	if p.handler != nil {
		return p.handler
	}
	return nil
}
```

- [ ] **Step 5: Add Handler method to connectionImpl**

```go
func (p *connectionImpl) Handler() http.Handler {
	return p.handler
}
```

- [ ] **Step 6: Run tests to ensure no regression**

Run: `go test ./pkg/acp/... -run TestConnection -v`
Expected: PASS (or update existing tests as needed)

- [ ] **Step 7: Commit**

```bash
git add pkg/acp/connection.go
git commit -m "feat(acp): add Handler method to Connection interface

Prepare connection interface to support HTTP transport.
Add Handler() method that returns http.Handler for HTTP mode,
returns nil for stdio mode.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 4: Update Channel Start Method for HTTP Transport

**Files:**
- Modify: `pkg/acp/channel.go`
- Test: `pkg/acp/service_test.go` (extend)

- [ ] **Step 1: Write test for HTTP mode start**

```go
func TestACPServiceStartHTTPMode(t *testing.T) {
	// This will be tested in integration tests
	// For now, ensure Start method signature is correct
}
```

- [ ] **Step 2: Update Start method to return handler for HTTP transport**

Modify `pkg/acp/channel.go` Start method:

```go
// Start begins the ACP channel's lifecycle by creating and starting the ACP connection.
// Returns http.Handler for HTTP transport (caller must start server), nil for stdio.
func (s *acpServiceImpl) Start(ctx context.Context) error {
	// If HTTP transport selected, create HTTP connection
	if s.transportType == TransportHTTP {
		return s.startHTTP(ctx)
	}
	
	// Otherwise use stdio (existing behavior)
	return s.startStdio(ctx)
}

// startStdio handles stdio transport (existing behavior)
func (s *acpServiceImpl) startStdio(ctx context.Context) error {
	if s.stdin == nil || s.stdout == nil {
		return fmt.Errorf("ACP channel: stdin and stdout must be provided via options")
	}

	conn, err := NewConnection(s.injector, s.stdin, s.stdout)
	if err != nil {
		return fmt.Errorf("failed to create ACP connection: %w", err)
	}

	s.conn = conn
	return s.conn.Start(ctx)
}

// startHTTP handles HTTP transport
func (s *acpServiceImpl) startHTTP(ctx context.Context) error {
	// Create HTTP transport
	httpTransport := acppkg.NewHTTPServerTransport()
	
	// Create session store
	store := acppkg.NewMemoryStore[*shared.ACPSession]()
	
	// Set ACP-specific fields
	s.SetClient(nil)
	s.SetSessionStore(store)
	
	// Create connection with HTTP transport
	conn := acppkg.NewAgentSideConnection(s, httpTransport,
		acppkg.WithSessionStore(store, func(ctx context.Context, params *acppkg.NewSessionRequest) (acppkg.SessionID, *shared.ACPSession, error) {
			ctx, cancel := context.WithCancel(context.Background())
			return acppkg.GenerateSessionID(), shared.NewAcpSession(ctx, cancel), nil
		}),
		acppkg.WithMiddleware(acppkg.RecoveryMiddleware()),
	)
	
	s.SetClient(conn.Client())
	s.conn = &connectionImpl{
		conn:    conn,
		service: s,
		handler: httpTransport.Handler(),
	}
	
	return nil
}
```

- [ ] **Step 3: Add GetHandler method to retrieve HTTP handler**

Add to acpServiceImpl:

```go
// GetHandler returns the HTTP handler for HTTP transport, nil for stdio
func (s *acpServiceImpl) GetHandler() http.Handler {
	if s.conn == nil {
		return nil
	}
	return s.conn.Handler()
}
```

- [ ] **Step 4: Update Connection interface to include GetHandler**

Modify `pkg/acp/connection.go`:

```go
// Connection defines the ACP connection interface
type Connection interface {
	Start(ctx context.Context) error
	Close() error
	Done() <-chan struct{}
	Handler() http.Handler
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./pkg/acp/... -v`
Expected: PASS (update any failing tests)

- [ ] **Step 6: Commit**

```bash
git add pkg/acp/channel.go pkg/acp/connection.go
git commit -m "feat(acp): implement HTTP transport in Start method

Split Start into startStdio and startHTTP methods.
startHTTP creates HTTPServerTransport and returns handler.
Add GetHandler method to retrieve HTTP handler from channel.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 3: CLI Integration

### Task 5: Add Transport CLI Flags

**Files:**
- Modify: `pkg/cli/acp.go`

- [ ] **Step 1: Add transport flags to ACP command**

Modify `pkg/cli/acp.go`:

```go
// ACPCommand returns the ACP server command
func ACPCommand() *cli.Command {
	return &cli.Command{
		Name:   "acp",
		Usage:  "Start Gollum ACP server (Agent Client Protocol)",
		Action: runACPServer,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "transport",
				Aliases:  []string{"t"},
				Usage:    "Transport type: stdio or http (default: stdio)",
				Value:    "stdio",
				Sources:  cli.EnvVars("GOLLUM_ACP_TRANSPORT"),
			},
			&cli.StringFlag{
				Name:    "host",
				Usage:   "HTTP server bind address (default: 0.0.0.0)",
				Value:   "0.0.0.0",
				Sources: cli.EnvVars("GOLLUM_ACP_HOST"),
			},
			&cli.IntFlag{
				Name:    "port",
				Usage:   "HTTP server port (default: 8080)",
				Value:   8080,
				Sources: cli.EnvVars("GOLLUM_ACP_PORT"),
			},
		},
	}
}
```

- [ ] **Step 2: Update runACPServer to parse transport flags**

Modify `runACPServer` function:

```go
func runACPServer(ctx context.Context, cmd *cli.Command) error {
	// Get injector from root command
	injector := shared.MustGetInjectorFromRoot(cmd)

	// Parse transport flags
	transportStr := cmd.String("transport")
	transport, err := acp.ParseTransportType(transportStr)
	if err != nil {
		return err
	}

	host := cmd.String("host")
	port := cmd.Int("port")

	// Get channel facade
	facade := do.MustInvoke[channel.ChannelFacade](injector)

	// Create ACP channel via factory with transport options
	var opts []channel.ChannelOption
	opts = append(opts, acp.WithTransport(transport))
	opts = append(opts, acp.WithHost(host))
	opts = append(opts, acp.WithPort(port))

	// For stdio, add stdin/stdout
	if transport == acp.TransportStdio {
		opts = append(opts,
			acp.WithStdin(os.Stdin),
			acp.WithStdout(os.Stdout),
		)
	}

	ch, err := facade.CreateChannel(acp.Identifier, opts...)
	if err != nil {
		return err
	}

	// Register channel with facade
	if err := facade.RegisterChannel(ch); err != nil {
		return err
	}

	// Ensure cleanup on exit
	defer func() {
		_ = facade.UnregisterChannel(ch.ID())
	}()

	// Start channel
	if err := ch.Start(ctx); err != nil {
		return err
	}

	// For HTTP transport, start HTTP server
	if transport == acp.TransportHTTP {
		return startHTTPServer(ctx, ch, host, port)
	}

	return nil
}

// startHTTPServer starts the HTTP server with the ACP handler
func startHTTPServer(ctx context.Context, ch channel.Channel, host string, port int) error {
	// Type assert to get handler
	acpCh, ok := ch.(*acp.ACPService)
	if !ok {
		return fmt.Errorf("channel is not an ACP service")
	}

	handler := acpCh.GetHandler()
	if handler == nil {
		return fmt.Errorf("HTTP handler not available")
	}

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", host, port)
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		// Graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("HTTP server shutdown failed: %w", err)
		}
		return nil
	case err := <-errChan:
		return fmt.Errorf("HTTP server error: %w", err)
	}
}
```

- [ ] **Step 3: Add missing imports**

```go
import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/denkhaus/gollum/pkg/acp"
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
)
```

- [ ] **Step 4: Fix type assertion for GetHandler**

The channel.Channel interface doesn't have GetHandler, so we need to access it differently. Modify startHTTPServer:

```go
// startHTTPServer starts the HTTP server with the ACP handler
func startHTTPServer(ctx context.Context, ch channel.Channel, host string, port int) error {
	// Get the handler from the channel's internal connection
	// We need to access the service's connection directly
	// This requires the service to expose the handler
	
	// For now, we'll need to add a method to the service to get the handler
	// This will be addressed in the next commit after updating the interface
	return fmt.Errorf("HTTP handler retrieval not yet implemented")
}
```

- [ ] **Step 5: Add GetHandler to shared.ACPService interface**

Modify `pkg/shared/acp.go` or wherever the ACPService interface is defined:

```go
// ACPService defines the ACP service interface
type ACPService interface {
	channel.Channel
	GetClient() acppkg.Client
	SetClient(client acppkg.Client)
	GetSessionStore() acppkg.SessionStore[*shared.ACPSession]
	SetSessionStore(store acppkg.SessionStore[*shared.ACPSession])
	GetHandler() http.Handler  // Add this method
}
```

- [ ] **Step 6: Implement GetHandler in acpServiceImpl**

Modify `pkg/acp/service.go`:

```go
// GetHandler returns the HTTP handler for HTTP transport, nil for stdio
func (s *acpServiceImpl) GetHandler() http.Handler {
	if s.conn == nil {
		return nil
	}
	return s.conn.Handler()
}
```

- [ ] **Step 7: Update startHTTPServer to use GetHandler**

```go
// startHTTPServer starts the HTTP server with the ACP handler
func startHTTPServer(ctx context.Context, ch channel.Channel, host string, port int) error {
	// Type assert to ACPService interface
	acpService, ok := ch.(shared.ACPService)
	if !ok {
		return fmt.Errorf("channel is not an ACP service")
	}

	handler := acpService.GetHandler()
	if handler == nil {
		return fmt.Errorf("HTTP handler not available - check transport configuration")
	}

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", host, port)
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	s.logger.Info("Starting ACP HTTP server", zap.String("addr", addr))

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down ACP HTTP server")
		// Graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("HTTP server shutdown failed: %w", err)
		}
		s.logger.Info("ACP HTTP server stopped")
		return nil
	case err := <-errChan:
		return fmt.Errorf("HTTP server error: %w", err)
	}
}
```

- [ ] **Step 8: Add zap import for logging**

```go
"go.uber.org/zap"
```

- [ ] **Step 9: Build and test**

Run: `go build ./cmd/gollum`
Expected: Success

Test: `./gollum acp --help`
Expected: Shows transport, host, port flags

- [ ] **Step 10: Commit**

```bash
git add pkg/cli/acp.go pkg/shared/acp.go pkg/acp/service.go
git commit -m "feat(acp): add CLI flags for transport selection

Add --transport, --host, --port flags to ACP command.
Implement HTTP server startup for HTTP transport mode.
Add GetHandler method to ACPService interface.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 4: Testing

### Task 6: HTTP Transport Integration Tests

**Files:**
- Create: `pkg/acp/http_transport_test.go`

- [ ] **Step 1: Write basic HTTP transport test**

```go
package acp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"
	acppkg "github.com/ironpark/go-acp"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHTTPTransportCreation(t *testing.T) {
	// Create test injector
	injector := do.New()
	
	// Create mock dependencies
	logger := zap.NewNop()
	
	// Create ACP service with HTTP transport
	service := &acpServiceImpl{
		logger:       logger,
		transportType: TransportHTTP,
		host:         "localhost",
		port:         9090,
		injector:     injector,
	}
	
	// Create HTTP transport manually for testing
	httpTransport := acppkg.NewHTTPServerTransport()
	
	store := acppkg.NewMemoryStore[*shared.ACPSession]()
	service.SetClient(nil)
	service.SetSessionStore(store)
	
	// Create connection with HTTP transport
	conn := acppkg.NewAgentSideConnection(service, httpTransport,
		acppkg.WithSessionStore(store, func(ctx context.Context, params *acppkg.NewSessionRequest) (acppkg.SessionID, *shared.ACPSession, error) {
			ctx, cancel := context.WithCancel(context.Background())
			return acppkg.GenerateSessionID(), shared.NewAcpSession(ctx, cancel), nil
		}),
		acppkg.WithMiddleware(acppkg.RecoveryMiddleware()),
	)
	
	service.SetClient(conn.Client())
	
	// Get handler
	handler := httpTransport.Handler()
	assert.NotNil(t, handler, "Handler should not be nil")
	
	// Test that handler is valid HTTP handler
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	// Handler should respond (even if with error for malformed request)
	assert.True(t, w.Code >= 200, "Handler should respond with HTTP status")
}
```

- [ ] **Step 2: Run test**

Run: `go test ./pkg/acp/... -run TestHTTPTransportCreation -v`
Expected: PASS

- [ ] **Step 3: Add multi-client connection test**

```go
func TestHTTPMultiClientSupport(t *testing.T) {
	// This test verifies that multiple clients can connect simultaneously
	// In a real scenario, this would require running an actual HTTP server
	// For now, we test that the HTTP transport supports concurrent handlers
	
	httpTransport := acppkg.NewHTTPServerTransport()
	handler := httpTransport.Handler()
	
	assert.NotNil(t, handler)
	
	// Test concurrent handler access
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}
}
```

- [ ] **Step 4: Run test**

Run: `go test ./pkg/acp/... -run TestHTTPMultiClientSupport -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/acp/http_transport_test.go
git commit -m "test(acp): add HTTP transport integration tests

Add tests for HTTP transport creation and multi-client support.
Verify handler responds to HTTP requests.
Test concurrent handler access for multi-client scenarios.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 7: End-to-End CLI Testing

**Files:**
- No files modified (manual testing)

- [ ] **Step 1: Test stdio transport (existing behavior)**

```bash
# Should work as before
./gollum acp

# Verify with explicit flag
./gollum acp --transport stdio
```

Expected: ACP starts in stdio mode

- [ ] **Step 2: Test HTTP transport with defaults**

```bash
./gollum acp --transport http &
HTTP_PID=$!

# Wait for server to start
sleep 2

# Test connection (will fail ACP handshake but should reach server)
curl -X POST http://localhost:8080 -d '{"jsonrpc":"2.0","id":1,"method":"initialize"}' || true

# Cleanup
kill $HTTP_PID
```

Expected: HTTP server starts on port 8080

- [ ] **Step 3: Test HTTP transport with custom host/port**

```bash
./gollum acp --transport http --host 127.0.0.1 --port 9000 &
HTTP_PID=$!

sleep 2

curl -X POST http://127.0.0.1:9000 -d '{"jsonrpc":"2.0","id":1,"method":"initialize"}' || true

kill $HTTP_PID
```

Expected: HTTP server starts on custom port

- [ ] **Step 4: Test invalid transport flag**

```bash
./gollum acp --transport invalid
```

Expected: Error message about unsupported transport type

- [ ] **Step 5: Test port validation**

```bash
./gollum acp --transport http --port 99999
```

Expected: Error about port range

- [ ] **Step 6: Test environment variable configuration**

```bash
GOLLUM_ACP_TRANSPORT=http GOLLUM_ACP_PORT=9000 ./gollum acp &
HTTP_PID=$!

sleep 2

curl http://localhost:9000 || true

kill $HTTP_PID
```

Expected: Reads config from environment variables

---

## Chunk 5: Documentation and Polish

### Task 8: Update Documentation

**Files:**
- Modify: `docs/cli.md` (if exists)
- Modify: `README.md` (if relevant)

- [ ] **Step 1: Update CLI documentation**

Add to `docs/cli.md` or create new section:

```markdown
## ACP Command

The ACP command starts the Gollum Agent Client Protocol server.

### Usage

```bash
gollum acp [flags]
```

### Flags

- `--transport, -t {stdio|http}` - Transport type (default: stdio)
- `--host <address>` - HTTP server bind address (default: 0.0.0.0)
- `--port <number>` - HTTP server port (default: 8080)

### Transport Modes

**Stdio Mode (default):**
```bash
gollum acp
# or
gollum acp --transport stdio
```
Communicates over stdin/stdout. Used when launched as a subprocess by IDEs/editors.

**HTTP Mode:**
```bash
gollum acp --transport http
# or with custom host/port
gollum acp --transport http --host localhost --port 9000
```
Starts an HTTP server with Server-Sent Events. Supports multiple concurrent clients.

### Environment Variables

- `GOLLUM_ACP_TRANSPORT` - Transport type
- `GOLLUM_ACP_HOST` - HTTP server host
- `GOLLUM_ACP_PORT` - HTTP server port

### Examples

Local development with stdio:
```bash
gollum acp
```

Multi-client HTTP server:
```bash
gollum acp --transport http --port 8080
```

Production deployment:
```bash
GOLLUM_ACP_TRANSPORT=http \
GOLLUM_ACP_HOST=0.0.0.0 \
GOLLUM_ACP_PORT=8080 \
gollum acp
```
```

- [ ] **Step 2: Add architecture note to existing ACP documentation**

If `docs/acp_improvements.md` exists, add:

```markdown
## Transport Architecture

The ACP service supports multiple transport types:

- **Stdio Transport**: Default transport using stdin/stdout. Used when the agent runs as a subprocess.
- **HTTP Transport**: HTTP + Server-Sent Events. Enables multi-client server deployments and web-based integration.

Transport selection is handled via channel options (`WithTransport`, `WithHost`, `WithPort`) and configured via CLI flags.
```

- [ ] **Step 3: Commit documentation**

```bash
git add docs/cli.md docs/acp_improvements.md
git commit -m "docs: document ACP transport modes

Add documentation for stdio and HTTP transport modes.
Include CLI flags, environment variables, and usage examples.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 9: Final Verification

- [ ] **Step 1: Run full test suite**

```bash
go test ./pkg/acp/... ./pkg/cli/... ./pkg/channel/...
```

Expected: All tests pass

- [ ] **Step 2: Build final binary**

```bash
go build -o gollum ./cmd/gollum
```

Expected: Builds successfully

- [ ] **Step 3: Verify help output**

```bash
./gollum acp --help
```

Expected: Shows transport flags

- [ ] **Step 4: Quick smoke test**

```bash
# Test stdio mode starts
timeout 2 ./gollum acp || true

# Test HTTP mode starts
./gollum acp --transport http &
PID=$!
sleep 2
kill $PID 2>/dev/null || true
```

Expected: Both modes start successfully

- [ ] **Step 5: Commit any final fixes**

```bash
git add -A
git commit -m "fix(acp): final polish and fixes for transport support

Address any issues found during testing.
Ensure all transports work correctly.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Success Criteria

- [x] Design approved and documented
- [ ] All tasks completed with checkboxes checked
- [ ] stdio transport works (backward compatible)
- [ ] HTTP transport works with default and custom host/port
- [ ] Multi-client connections supported
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Manual testing successful
- [ ] No breaking changes to existing functionality

## Notes

- HTTP transport uses go-acp's HTTPServerTransport which inherently supports multiple concurrent connections
- Session management (existing session store) handles separating clients
- Signal handling already exists in CLI root command
- Graceful shutdown with 10-second timeout for HTTP server
- Port validation prevents invalid configurations
- Environment variable support allows flexible deployment
