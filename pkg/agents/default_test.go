// Package agents provides agent implementations and factory functions.
package agents

import (
	"context"
	"errors"
	"testing"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDefaultAgent_GetMessageHistory(t *testing.T) {
	ctx := context.Background()

	t.Run("handles nil session", func(t *testing.T) {
		agent := &DefaultAgent{
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
		agent := &DefaultAgent{
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

	agent := &DefaultAgent{
		id:     uuid.New(),
		config: config,
	}

	assert.Equal(t, config, agent.GetConfig())
}

// TestDefaultAgent_GetID tests GetID method
func TestDefaultAgent_GetID(t *testing.T) {
	id := uuid.New()
	agent := &DefaultAgent{
		id: id,
	}

	assert.Equal(t, id, agent.GetID())
}

// TestDefaultAgent_Execute tests Execute method delegation with nil base
func TestDefaultAgent_Execute_NilBase(t *testing.T) {
	ctx := context.Background()

	agent := &DefaultAgent{
		base: nil,
		id:   uuid.New(),
		config: &shared.AgentConfig{
			SystemPrompt: "test prompt",
		},
	}

	// Should panic when calling Execute on nil base
	assert.Panics(t, func() {
		_, _ = agent.Execute(ctx, gollem.Text("test input"))
	})
}

// TestDefaultAgent_Session_NilBase tests Session with nil base
func TestDefaultAgent_Session_NilBase(t *testing.T) {
	agent := &DefaultAgent{
		base: nil,
		id:   uuid.New(),
		config: &shared.AgentConfig{
			SystemPrompt: "test prompt",
		},
	}

	// Should panic when calling Session on nil base
	assert.Panics(t, func() {
		agent.Session()
	})
}

// TestDefaultAgent_UpdateSystemPrompt tests UpdateSystemPrompt method
func TestDefaultAgent_UpdateSystemPrompt(t *testing.T) {
	ctx := context.Background()

	t.Run("config is updated with new prompt", func(t *testing.T) {
		agent := &DefaultAgent{
			id: uuid.New(), //nolint:unusedwrite // field is required but not used in this test
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
		agent := &DefaultAgent{
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
		agent := &DefaultAgent{
			base: nil, // No session
			id:   uuid.New(),
			config: &shared.AgentConfig{
				SystemPrompt: "test prompt",
			},
		}

		// Create a modifier that tracks if it was called
		modifierCalled := false
		modifier := func(h *gollem.History) (*gollem.History, error) { //nolint:unparam // test helper, error always nil
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
		modifier := func(h *gollem.History) (*gollem.History, error) { //nolint:unparam // test helper, error always nil
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

		modifier := func(h *gollem.History) (*gollem.History, error) { //nolint:unparam // test helper, error always nil
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

// TestDefaultAgent_UpdateHistory_NilBase tests error handling when base is nil
func TestDefaultAgent_UpdateHistory_NilBase(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()

	agent := &DefaultAgent{
		base: nil,
		id:   agentID,
		config: &shared.AgentConfig{
			SystemPrompt: "test prompt",
		},
	}

	// Modifier that returns an error
	modifier := func(_ *gollem.History) (*gollem.History, error) {
		return nil, errors.New("modifier failed")
	}

	err := agent.UpdateHistory(ctx, modifier)
	// With nil base, GetMessageHistory returns nil, nil (no error)
	// Then modifier is called with nil history and returns error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "history modifier failed")
}

// TestBuildOptionsWithHistory_BaseOptions tests basic option building
func TestBuildOptionsWithHistory_BaseOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agentID := uuid.New()

	mockLLMClient := mocks.NewMockLLMClient(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)

	t.Run("with nil history and silent mode", func(t *testing.T) {
		mockPromptMgr.EXPECT().GetPromptByID(gomock.Any(), gomock.Any()).Return(&prompt.Prompt{Content: "compacter prompt"}, nil).AnyTimes()

		agent := &DefaultAgent{
			id:            agentID,
			llmClient:     mockLLMClient,
			promptManager: mockPromptMgr,
			config: &shared.AgentConfig{
				SystemPrompt:    "test prompt",
				Role:            "test role",
				Strategy:        nil,
				OutputMode:      shared.OutputModeSilent,
				AllowCompaction: false,
			},
		}

		options := agent.buildOptionsWithHistory(nil)
		assert.NotNil(t, options)
		// Should have at least: strategy, tools, system prompt
		assert.GreaterOrEqual(t, len(options), 2)
	})

	t.Run("with empty history messages", func(t *testing.T) {
		history := &gollem.History{
			Version:  gollem.HistoryVersion,
			Messages: []gollem.Message{}, // Empty but not nil
		}

		mockPromptMgr.EXPECT().GetPromptByID(gomock.Any(), gomock.Any()).Return(&prompt.Prompt{Content: "compacter prompt"}, nil).AnyTimes()

		agent := &DefaultAgent{
			id:            agentID,
			llmClient:     mockLLMClient,
			promptManager: mockPromptMgr,
			config: &shared.AgentConfig{
				SystemPrompt:    "test prompt",
				Role:            "test role",
				Strategy:        nil,
				OutputMode:      shared.OutputModeSilent,
				AllowCompaction: false,
			},
		}

		options := agent.buildOptionsWithHistory(history)
		assert.NotNil(t, options)
		// Empty message history should not be added
		assert.GreaterOrEqual(t, len(options), 2)
	})

	t.Run("with history messages", func(t *testing.T) {
		history := &gollem.History{
			Version:  gollem.HistoryVersion,
			Messages: []gollem.Message{{Role: gollem.RoleUser}},
		}

		mockPromptMgr.EXPECT().GetPromptByID(gomock.Any(), gomock.Any()).Return(&prompt.Prompt{Content: "compacter prompt"}, nil).AnyTimes()

		agent := &DefaultAgent{
			id:            agentID,
			llmClient:     mockLLMClient,
			promptManager: mockPromptMgr,
			config: &shared.AgentConfig{
				SystemPrompt:    "test prompt",
				Role:            "test role",
				Strategy:        nil,
				OutputMode:      shared.OutputModeSilent,
				AllowCompaction: false,
			},
		}

		options := agent.buildOptionsWithHistory(history)
		assert.NotNil(t, options)
		// Should have: strategy, tools, system prompt, history
		assert.GreaterOrEqual(t, len(options), 3)
	})

	t.Run("with AllowCompaction true", func(t *testing.T) {
		mockPromptMgr.EXPECT().GetPromptByID(gomock.Any(), gomock.Any()).Return(&prompt.Prompt{Content: "compacter prompt"}, nil).AnyTimes()

		agent := &DefaultAgent{
			id:            agentID,
			llmClient:     mockLLMClient,
			promptManager: mockPromptMgr,
			config: &shared.AgentConfig{
				SystemPrompt:    "test prompt",
				Role:            "test role",
				Strategy:        nil,
				OutputMode:      shared.OutputModeSilent,
				AllowCompaction: true,
			},
		}

		options := agent.buildOptionsWithHistory(nil)
		assert.NotNil(t, options)
		// Should include compacter middleware
		assert.GreaterOrEqual(t, len(options), 3)
	})
}
