// Package ui provides unit tests for agent messenger components.
package ui

import (
	"testing"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
)

// TestNewAgentMessenger tests creation
func TestNewAgentMessenger(t *testing.T) {
	injector := do.New()

	messenger, err := NewAgentMessenger(injector)
	assert.NoError(t, err)
	assert.NotNil(t, messenger)
}

// TestAgentMessenger_DisplayAgentMessage tests message display
func TestAgentMessenger_DisplayAgentMessage(t *testing.T) {
	injector := do.New()

	messenger, err := NewAgentMessenger(injector)
	assert.NoError(t, err)

	impl := messenger.(*agentMessengerImpl)
	assert.NotNil(t, impl)

	// This should not panic - it outputs to stdout
	impl.DisplayAgentMessage(uuid.New(), "TestAgent", "Test message", false)
	impl.DisplayAgentMessage(uuid.New(), "TestAgent", "Tool execution", true)
}

// TestAgentMessenger_DisplayUserMessage tests user message display
func TestAgentMessenger_DisplayUserMessage(t *testing.T) {
	injector := do.New()

	messenger, err := NewAgentMessenger(injector)
	assert.NoError(t, err)

	impl := messenger.(*agentMessengerImpl)

	// This should not panic - it outputs to stdout
	impl.DisplayUserMessage("Hello from user")
}

// TestAgentMessenger_DisplaySystemInfo tests system info display
func TestAgentMessenger_DisplaySystemInfo(t *testing.T) {
	injector := do.New()

	messenger, err := NewAgentMessenger(injector)
	assert.NoError(t, err)

	impl := messenger.(*agentMessengerImpl)

	// This should not panic - it outputs to stdout
	impl.DisplaySystemInfo("System info message")
}

// TestAgentMessenger_ShortenAgentName tests agent name shortening
func TestAgentMessenger_ShortenAgentName(t *testing.T) {
	injector := do.New()

	messenger, err := NewAgentMessenger(injector)
	assert.NoError(t, err)

	impl := messenger.(*agentMessengerImpl)

	tests := []struct {
		name     string
		agentID  uuid.UUID
		role     string
		expected string
	}{
		{
			name:     "short role",
			agentID:  uuid.New(),
			role:     "Supervisor",
			expected: "Supervisor",
		},
		{
			name:     "long role uses ID prefix",
			agentID:  uuid.MustParse("12345678-1234-1234-1234-123456789abc"),
			role:     "VeryLongAgentRoleName",
			expected: "1234",
		},
		{
			name:     "empty role uses ID prefix",
			agentID:  uuid.MustParse("12345678-1234-1234-1234-123456789abc"),
			role:     "",
			expected: "1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := impl.shortenAgentName(tt.agentID, tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}
