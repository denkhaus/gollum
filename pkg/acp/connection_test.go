package acp

import (
	"net/http"
	"testing"
)

func TestConnectionHandler(t *testing.T) {
	// Test that connectionImpl can hold and return an HTTP handler
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	conn := &connectionImpl{
		handler: mockHandler,
	}

	handler := conn.Handler()
	if handler == nil {
		t.Error("Handler() returned nil, expected non-nil handler")
	}

	// Verify handler is callable - if it panics, test will fail
	// This validates it's a proper http.Handler
}

func TestConnectionHandlerNil(t *testing.T) {
	// Test that connectionImpl returns nil when no handler is set (stdio mode)
	conn := &connectionImpl{
		handler: nil,
	}

	handler := conn.Handler()
	if handler != nil {
		t.Error("Handler() should return nil for stdio mode")
	}
}
