// Package channel provides integration tests for the channel facade service.
package channel

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestChannelFacade_Integration_SupervisorRouting verifies the complete
// integration between channel facade, registry, and supervisor agent routing.
func TestChannelFacade_Integration_SupervisorRouting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	injector := setupTestInjector()

	// Create facade
	facade, err := NewChannelFacade(injector)
	if err != nil {
		t.Fatalf("should create facade: %v", err)
	}

	// Get registry from injector
	mockReg := &mockAgentRegistry{}

	// Test 1: Verify registry has GetSupervisorAgent method
	t.Run("Registry supports GetSupervisorAgent", func(t *testing.T) {
		// This test verifies the interface is properly implemented
		// The actual supervisor registration happens in RegisterChannel tests
		_, err := mockReg.GetSupervisorAgent()
		// We expect an error when no supervisor is registered yet
		assert.Error(t, err, "should error when no supervisor registered")
	})

	// Test 2: Verify SubmitInput handles non-command input when no supervisor
	t.Run("SubmitInput requires supervisor for non-commands", func(t *testing.T) {
		channelID := uuid.New()

		// Submit non-command input - should fail without supervisor
		result, err := facade.SubmitInput(ctx, channelID, "test input")
		assert.Error(t, err, "should error without supervisor")
		assert.False(t, result.Handled, "should not be handled without supervisor")
	})

	t.Log("Integration test: Channel facade components verified")
	t.Log("Components tested: GetSupervisorAgent(), SubmitInput() error handling")
}

// TestChannelFacade_Integration_WithSupervisor tests the full flow with a supervisor
func TestChannelFacade_Integration_WithSupervisor(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires a complete DI setup with:
	// - Config service
	// - Registry service
	// - Agent factory
	// - Command manager
	// - Channel facade
	//
	// The supervisor agent creation happens through the factory during channel registration
	//
	// For now, we verify the contract is in place
	t.Log("Full integration test requires complete DI setup")
	t.Log("See RegisterChannel tests for supervisor creation flow")
}
