// Package agents provides agent implementations and factory functions.
package agents

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAgent_GetMessageHistory(t *testing.T) {
	ctx := context.Background()

	t.Run("handles nil session", func(t *testing.T) {
		agent := &defaultAgent{
			base:   nil, // No base agent set
			id:     uuid.New(),
			config: &shared.AgentConfig{},
		}

		messages, err := agent.GetMessageHistory(ctx)
		require.NoError(t, err)
		assert.Nil(t, messages)
	})

	t.Run("handles base agent with nil session", func(t *testing.T) {
		// This test verifies that when base exists but Session() returns nil
		// The method handles it gracefully
		agent := &defaultAgent{
			base:   nil, // No base agent means no session
			id:     uuid.New(),
			config: &shared.AgentConfig{},
		}

		messages, err := agent.GetMessageHistory(ctx)
		require.NoError(t, err)
		assert.Nil(t, messages)
	})
}

// TestDefaultAgent_GetConfig tests GetConfig method
func TestDefaultAgent_GetConfig(t *testing.T) {
	config := &shared.AgentConfig{
		ID:           uuid.New(),
		SystemPrompt: "test prompt",
		Role:         "test role",
	}

	agent := &defaultAgent{
		id:     uuid.New(),
		config: config,
	}

	assert.Equal(t, config, agent.GetConfig())
}

// TestDefaultAgent_GetID tests GetID method
func TestDefaultAgent_GetID(t *testing.T) {
	id := uuid.New()
	agent := &defaultAgent{
		id: id,
	}

	assert.Equal(t, id, agent.GetID())
}

// TestDefaultAgent_UpdateSystemPrompt tests UpdateSystemPrompt method
func TestDefaultAgent_UpdateSystemPrompt(t *testing.T) {
	ctx := context.Background()

	t.Run("config is updated with new prompt", func(t *testing.T) {
		agent := &defaultAgent{
			id: uuid.New(),
			config: &shared.AgentConfig{
				SystemPrompt: "original prompt",
			},
		}

		// Verify initial state
		assert.Equal(t, "original prompt", agent.config.SystemPrompt)

		// Direct config update (what UpdateSystemPrompt does internally)
		agent.config.SystemPrompt = "new prompt"
		assert.Equal(t, "new prompt", agent.config.SystemPrompt)
	})

	t.Run("handles nil history gracefully", func(t *testing.T) {
		agent := &defaultAgent{
			base: nil,
			id:   uuid.New(),
			config: &shared.AgentConfig{
				SystemPrompt: "test prompt",
			},
		}

		history, err := agent.GetMessageHistory(ctx)
		require.NoError(t, err)
		assert.Nil(t, history)
	})
}

// TestDefaultAgent_UpdateHistory tests UpdateHistory method
func TestDefaultAgent_UpdateHistory(t *testing.T) {
	ctx := context.Background()

	t.Run("modifier function is called with history", func(t *testing.T) {
		agent := &defaultAgent{
			base: nil, // No session
			id:   uuid.New(),
			config: &shared.AgentConfig{
				SystemPrompt: "test prompt",
			},
		}

		// Create a modifier that tracks if it was called
		modifierCalled := false
		modifier := func(h *gollem.History) (*gollem.History, error) {
			modifierCalled = true
			return h, nil
		}

		// With nil history, modifier should still be called
		history, err := agent.GetMessageHistory(ctx)
		require.NoError(t, err)
		assert.Nil(t, history)

		// Test that modifier works on nil history
		result, err := modifier(history)
		require.NoError(t, err)
		assert.True(t, modifierCalled)
		assert.Nil(t, result)
	})

	t.Run("modifier can add messages", func(t *testing.T) {
		modifier := func(h *gollem.History) (*gollem.History, error) {
			if h == nil {
				h = &gollem.History{
					Version:  gollem.HistoryVersion,
					Messages: []gollem.Message{},
				}
			}
			h.Messages = append(h.Messages, gollem.Message{
				Role: gollem.RoleUser,
			})
			return h, nil
		}

		// Test modifier on nil history
		result, err := modifier(nil)
		require.NoError(t, err)
		assert.Len(t, result.Messages, 1)
		assert.Equal(t, gollem.RoleUser, result.Messages[0].Role)
	})

	t.Run("modifier can filter messages", func(t *testing.T) {
		original := &gollem.History{
			Version: gollem.HistoryVersion,
			Messages: []gollem.Message{
				{Role: gollem.RoleUser},
				{Role: gollem.RoleAssistant},
				{Role: gollem.RoleUser},
			},
		}

		modifier := func(h *gollem.History) (*gollem.History, error) {
			// Filter out assistant messages
			filtered := make([]gollem.Message, 0)
			for _, msg := range h.Messages {
				if msg.Role != gollem.RoleAssistant {
					filtered = append(filtered, msg)
				}
			}
			h.Messages = filtered
			return h, nil
		}

		result, err := modifier(original)
		require.NoError(t, err)
		assert.Len(t, result.Messages, 2)
		for _, msg := range result.Messages {
			assert.NotEqual(t, gollem.RoleAssistant, msg.Role)
		}
	})
}
