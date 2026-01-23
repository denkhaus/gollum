package middleware_test

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/middleware"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/ui"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

const (
	testAgentRole = "TestAgent"
)

// TestNewSummaryMiddlewareProvider tests the provider creation via DI
func TestNewSummaryMiddlewareProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mocks
	mockMessenger := mocks.NewMockAgentMessenger(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)

	// Create injector with mocks
	injector := do.New()
	do.ProvideValue[ui.AgentMessenger](injector, mockMessenger)
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	provider, err := middleware.NewSummaryMiddlewareProvider(injector)
	if err != nil {
		t.Fatalf("NewSummaryMiddlewareProvider failed: %v", err)
	}

	if provider == nil {
		t.Fatal("expected provider, got nil")
	}

	// Test creating middleware
	agentID := uuid.New()
	agentRole := testAgentRole

	middlewareInstance := provider.CreateSummaryMiddleware(agentID, agentRole)

	if middlewareInstance == nil {
		t.Fatal("expected middleware, got nil")
	}
}

// TestSummaryMiddleware_ToolMiddleware tests that tool middleware suppresses output
func TestSummaryMiddleware_ToolMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mocks
	mockMessenger := mocks.NewMockAgentMessenger(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)

	// Create injector with mocks
	injector := do.New()
	do.ProvideValue[ui.AgentMessenger](injector, mockMessenger)
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	provider, err := middleware.NewSummaryMiddlewareProvider(injector)
	if err != nil {
		t.Fatalf("NewSummaryMiddlewareProvider failed: %v", err)
	}

	sm := provider.CreateSummaryMiddleware(uuid.New(), testAgentRole)

	// Create a mock next handler
	nextCalled := false
	nextHandler := func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		nextCalled = true
		return &gollem.ToolExecResponse{}, nil
	}

	// Wrap with tool middleware
	wrapped := sm.ToolMiddleware(nextHandler)

	// Execute
	_, err = wrapped(context.Background(), &gollem.ToolExecRequest{})
	if err != nil {
		t.Fatalf("ToolMiddleware failed: %v", err)
	}

	// Verify next handler was called
	if !nextCalled {
		t.Error("expected next handler to be called")
	}
}

// TestSummaryMiddleware_ContentBlockMiddleware tests that content buffering works
func TestSummaryMiddleware_ContentBlockMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mocks
	mockMessenger := mocks.NewMockAgentMessenger(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)

	// Create injector with mocks
	injector := do.New()
	do.ProvideValue[ui.AgentMessenger](injector, mockMessenger)
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	provider, err := middleware.NewSummaryMiddlewareProvider(injector)
	if err != nil {
		t.Fatalf("NewSummaryMiddlewareProvider failed: %v", err)
	}

	sm := provider.CreateSummaryMiddleware(uuid.New(), testAgentRole)

	// Create content block handler
	nextHandler := func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return &gollem.ContentResponse{
			Texts: []string{"test content"},
		}, nil
	}

	wrapped := sm.ContentBlockMiddleware(nextHandler)

	// Execute
	_, err = wrapped(context.Background(), &gollem.ContentRequest{})
	if err != nil {
		t.Fatalf("ContentBlockMiddleware failed: %v", err)
	}
}

// TestSummaryMiddleware_FlushAndReset tests buffer management
func TestSummaryMiddleware_FlushAndReset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mocks
	mockMessenger := mocks.NewMockAgentMessenger(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)

	// Create injector with mocks
	injector := do.New()
	do.ProvideValue[ui.AgentMessenger](injector, mockMessenger)
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	provider, err := middleware.NewSummaryMiddlewareProvider(injector)
	if err != nil {
		t.Fatalf("NewSummaryMiddlewareProvider failed: %v", err)
	}

	sm := provider.CreateSummaryMiddleware(uuid.New(), testAgentRole)

	// Note: We can't directly test Flush/Reset since the buffer is private
	// and there's no way to write to it without using the ContentBlockMiddleware
	// This is a limitation of the current design - consider exposing buffer methods
	// or adding a way to inject content for testing

	// For now, just verify the methods exist and don't panic
	sm.Flush()
	sm.Reset()
}

// TestSummaryMiddleware_DisplayWelcome tests welcome message display (suppressed in summary mode)
func TestSummaryMiddleware_DisplayWelcome(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mocks
	mockMessenger := mocks.NewMockAgentMessenger(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)

	// Create injector with mocks
	injector := do.New()
	do.ProvideValue[ui.AgentMessenger](injector, mockMessenger)
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	provider, err := middleware.NewSummaryMiddlewareProvider(injector)
	if err != nil {
		t.Fatalf("NewSummaryMiddlewareProvider failed: %v", err)
	}

	sm := provider.CreateSummaryMiddleware(uuid.New(), testAgentRole)

	// NO mock expectations - summary mode suppresses welcome

	// Execute - should not panic
	sm.DisplayWelcome()
}

// TestSummaryMiddleware_DisplaySystemInfo tests system info display (suppressed in summary mode)
func TestSummaryMiddleware_DisplaySystemInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mocks
	mockMessenger := mocks.NewMockAgentMessenger(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)

	// Create injector with mocks
	injector := do.New()
	do.ProvideValue[ui.AgentMessenger](injector, mockMessenger)
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	provider, err := middleware.NewSummaryMiddlewareProvider(injector)
	if err != nil {
		t.Fatalf("NewSummaryMiddlewareProvider failed: %v", err)
	}

	sm := provider.CreateSummaryMiddleware(uuid.New(), testAgentRole)

	// NO mock expectations - summary mode suppresses system info

	// Execute - should not panic
	testMessage := "System info test message"
	sm.DisplaySystemInfo(testMessage)
}
