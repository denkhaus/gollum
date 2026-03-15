package registry

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetPromptIDForAgent(t *testing.T) {
	// Create a fresh registry for each test to avoid shared state
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	registry := &agentRegistry{
		agents: make(map[uuid.UUID]*agentHandle),
	}

	t.Run("supervisor agent returns supervisor prompt ID", func(t *testing.T) {
		handle := &agentHandle{
			config: &shared.AgentConfig{
				ID:       uuid.New(),
				ParentID: nil, // No parent = supervisor
			},
		}

		promptID := registry.getPromptIDForAgent(handle)
		assert.Equal(t, prompt.PromptIDSupervisorSystem, promptID)
	})

	t.Run("subagent returns subagent prompt ID", func(t *testing.T) {
		parentID := uuid.New()
		handle := &agentHandle{
			config: &shared.AgentConfig{
				ID:       uuid.New(),
				ParentID: &parentID, // Has parent = subagent
			},
		}

		promptID := registry.getPromptIDForAgent(handle)
		assert.Equal(t, prompt.PromptIDSubagentSystem, promptID)
	})
}

func TestIsAgentIdle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	registry := &agentRegistry{
		agents:       make(map[uuid.UUID]*agentHandle),
		agentResults: make(map[uuid.UUID]*shared.AgentResult),
	}

	agentID := uuid.New()

	t.Run("agent with no cancel and no result is idle", func(t *testing.T) {
		handle := &agentHandle{
			config: &shared.AgentConfig{ID: agentID},
			cancel: nil,
		}

		assert.True(t, registry.isAgentIdle(handle))
	})

	t.Run("agent with no cancel and running result is not idle", func(t *testing.T) {
		handle := &agentHandle{
			config: &shared.AgentConfig{ID: agentID},
			cancel: nil,
		}
		registry.agentResults[agentID] = &shared.AgentResult{
			AgentID: agentID,
			Status:  shared.AgentStatusRunning,
		}

		assert.False(t, registry.isAgentIdle(handle))
	})

	t.Run("agent with no cancel and completed result is idle", func(t *testing.T) {
		handle := &agentHandle{
			config: &shared.AgentConfig{ID: agentID},
			cancel: nil,
		}
		registry.agentResults[agentID] = &shared.AgentResult{
			AgentID: agentID,
			Status:  shared.AgentStatusCompleted,
		}

		assert.True(t, registry.isAgentIdle(handle))
	})

	t.Run("agent with no cancel and failed result is idle", func(t *testing.T) {
		handle := &agentHandle{
			config: &shared.AgentConfig{ID: agentID},
			cancel: nil,
		}
		registry.agentResults[agentID] = &shared.AgentResult{
			AgentID: agentID,
			Status:  shared.AgentStatusFailed,
		}

		assert.True(t, registry.isAgentIdle(handle))
	})

	t.Run("agent with cancel and running result is not idle", func(t *testing.T) {
		handle := &agentHandle{
			config: &shared.AgentConfig{ID: agentID},
			cancel: func() {}, // Has cancel function
		}
		registry.agentResults[agentID] = &shared.AgentResult{
			AgentID: agentID,
			Status:  shared.AgentStatusRunning,
		}

		assert.False(t, registry.isAgentIdle(handle))
	})

	t.Run("agent with cancel but completed result is idle", func(t *testing.T) {
		handle := &agentHandle{
			config: &shared.AgentConfig{ID: agentID},
			cancel: func() {}, // Has cancel function
		}
		registry.agentResults[agentID] = &shared.AgentResult{
			AgentID: agentID,
			Status:  shared.AgentStatusCompleted,
		}

		assert.True(t, registry.isAgentIdle(handle))
	})
}

func TestCopyWorkspaceContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	registry := &agentRegistry{
		agents: make(map[uuid.UUID]*agentHandle),
	}

	t.Run("nil source returns nil", func(t *testing.T) {
		result := registry.copyWorkspaceContext(nil)
		assert.Nil(t, result)
	})

	t.Run("copies all fields", func(t *testing.T) {
		src := &shared.WorkspaceContext{
			CurrentPath: "/test/path",
			SkillsXML:   "<skills>test</skills>",
			Skills: []shared.SkillInfo{
				{Name: "skill1", Description: "Test skill", Location: "/path/skill1.md"},
			},
		}

		result := registry.copyWorkspaceContext(src)

		require.NotNil(t, result)
		assert.Equal(t, src.CurrentPath, result.CurrentPath)
		assert.Equal(t, src.SkillsXML, result.SkillsXML)
		require.Len(t, result.Skills, 1)
		assert.Equal(t, src.Skills[0], result.Skills[0])
	})

	t.Run("deep copy of skills slice", func(t *testing.T) {
		src := &shared.WorkspaceContext{
			CurrentPath: "/test/path",
			Skills: []shared.SkillInfo{
				{Name: "skill1", Description: "Test", Location: "/path/1.md"},
				{Name: "skill2", Description: "Test", Location: "/path/2.md"},
			},
		}

		result := registry.copyWorkspaceContext(src)

		// Modify original to ensure deep copy
		src.Skills[0].Name = "modified"
		src.Skills = append(src.Skills, shared.SkillInfo{Name: "skill3"})

		// Result should not be affected
		require.Len(t, result.Skills, 2)
		assert.Equal(t, "skill1", result.Skills[0].Name)
		assert.Equal(t, "skill2", result.Skills[1].Name)
	})

	t.Run("empty skills slice", func(t *testing.T) {
		src := &shared.WorkspaceContext{
			CurrentPath: "/test/path",
			Skills:      []shared.SkillInfo{},
		}

		result := registry.copyWorkspaceContext(src)
		require.NotNil(t, result)
		require.Len(t, result.Skills, 00)
	})
}
