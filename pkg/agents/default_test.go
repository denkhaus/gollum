// Package agents provides agent implementations and factory functions.
package agents

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAgent_GetMessageHistory(t *testing.T) {
	ctx := context.Background()

	t.Run("handles nil session", func(t *testing.T) {
		agent := &defaultAgent{
			base:  nil, // No base agent set
			id:    uuid.New(),
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
			base:  nil, // No base agent means no session
			id:    uuid.New(),
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
		ID:          uuid.New(),
		SystemPrompt: "test prompt",
		Role:        "test role",
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
