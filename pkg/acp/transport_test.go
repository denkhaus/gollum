package acp

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/samber/do/v2"
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

func TestParseTransportType(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  TransportType
		wantError bool
	}{
		{"valid stdio", "stdio", TransportStdio, false},
		{"valid http", "http", TransportHTTP, false},
		{"invalid transport", "tcp", TransportStdio, true},
		{"empty string", "", TransportStdio, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTransportType(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseTransportType(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
				return
			}
			if !tt.wantError && result != tt.expected {
				t.Errorf("ParseTransportType(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTransportTypeString(t *testing.T) {
	tests := []struct {
		name     string
		transport TransportType
		expected string
	}{
		{"stdio", TransportStdio, "stdio"},
		{"http", TransportHTTP, "http"},
		{"unknown", TransportType(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.transport.String(); result != tt.expected {
				t.Errorf("TransportType.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

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

func TestWithHost(t *testing.T) {
	mockService := &acpServiceImpl{}

	option := WithHost("127.0.0.1")
	err := option.Apply(mockService)

	if err != nil {
		t.Errorf("WithHost() error = %v", err)
	}
	if mockService.host != "127.0.0.1" {
		t.Errorf("WithHost() set host = %v, want %v", mockService.host, "127.0.0.1")
	}
}

func TestWithPort(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		wantError bool
	}{
		{"valid port", 3000, false},
		{"min port", 1, false},
		{"max port", 65535, false},
		{"zero port", 0, true},
		{"negative port", -1, true},
		{"too large port", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &acpServiceImpl{}

			option := WithPort(tt.port)
			err := option.Apply(mockService)

			if (err != nil) != tt.wantError {
				t.Errorf("WithPort(%d) error = %v, wantError %v", tt.port, err, tt.wantError)
				return
			}

			if !tt.wantError && mockService.port != tt.port {
				t.Errorf("WithPort(%d) set port = %v, want %v", tt.port, mockService.port, tt.port)
			}
		})
	}
}

func TestHTTPModeStart(t *testing.T) {
	// Create injector
	injector := do.New()

	// Create service with HTTP transport
	mockService := &acpServiceImpl{
		injector:      injector,
		transportType: TransportHTTP,
	}

	// Start the service
	err := mockService.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify connection was created
	if mockService.conn == nil {
		t.Error("Start() did not create connection")
	}

	// Verify handler is available
	handler := mockService.GetHandler()
	if handler == nil {
		t.Error("GetHandler() returned nil, want non-nil handler")
	}

	// Verify handler is functional
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Handler should respond (even if with 404 or method not allowed)
	// The important part is that it doesn't panic
}

func TestHTTPModeStartWithContextCancellation(t *testing.T) {
	// Create injector
	injector := do.New()

	// Create service with HTTP transport
	mockService := &acpServiceImpl{
		injector:      injector,
		transportType: TransportHTTP,
	}

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Start the service
	err := mockService.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify connection was created
	if mockService.conn == nil {
		t.Error("Start() did not create connection")
	}

	// Verify handler is available
	handler := mockService.GetHandler()
	if handler == nil {
		t.Error("GetHandler() returned nil, want non-nil handler")
	}

	// Cancel the context
	cancel()

	// Verify context is cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled as expected
	default:
		t.Error("Context was not cancelled")
	}

	// Handler should still be accessible after context cancellation
	// (HTTP handler lifecycle is managed by the server, not the context)
	handler = mockService.GetHandler()
	if handler == nil {
		t.Error("GetHandler() returned nil after context cancellation")
	}
}
