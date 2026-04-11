package tools

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/registry"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAgentOutputTool_Run_TimeoutClamping_Int(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()

	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(senderID).AnyTimes()

	agentID := uuid.New()

	tests := []struct {
		name            string
		inputTimeout    int
		expectedTimeout time.Duration
	}{
		{"Too low", 0, 1 * time.Millisecond},
		{"Negative", -100, 1 * time.Millisecond},
		{"Too high", 700000, 600000 * time.Millisecond},
		{"Valid", 30000, 30000 * time.Millisecond},
		{"Min valid", 1, 1 * time.Millisecond},
		{"Max valid", 600000, 600000 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskResult := &shared.AgentResult{
				AgentID:   agentID,
				Status:    shared.AgentStatusRunning,
				StartedAt: time.Now().Unix(),
			}

			// Permission check: sender is direct parent
			mockRegistry.EXPECT().
				IsDirectParent(senderID, agentID).
				Return(true)

			mockRegistry.EXPECT().
				GetAgentResult(agentID).
				Return(taskResult, true)

			mockRegistry.EXPECT().
				WaitForAgent(gomock.Any(), agentID, tt.expectedTimeout).
				Return(taskResult, nil)

			tool := &agentOutputToolImpl{
				hookManager: mockHookManager,
				registry:    mockRegistry,
				agent:       mockAgent,
			}

			result, err := tool.Run(context.Background(), map[string]any{
				"agent_id": agentID.String(),
				"block":    true,
				"timeout":  tt.inputTimeout,
			})

			require.NoError(t, err)
			assert.True(t, result["success"].(bool))
			ctrl.Finish()
		})
	}
}

func TestAgentOutputTool_Run_TimeoutClamping_Float(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()

	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(senderID).AnyTimes()

	agentID := uuid.New()

	tests := []struct {
		name            string
		inputTimeout    float64
		expectedTimeout time.Duration
	}{
		{"Too low", 0.5, 1 * time.Millisecond},
		{"Valid float", 30000.0, 30000 * time.Millisecond},
		{"Too high", 700000.5, 600000 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskResult := &shared.AgentResult{
				AgentID:   agentID,
				Status:    shared.AgentStatusRunning,
				StartedAt: time.Now().Unix(),
			}

			// Permission check: sender is direct parent
			mockRegistry.EXPECT().
				IsDirectParent(senderID, agentID).
				Return(true)

			mockRegistry.EXPECT().
				GetAgentResult(agentID).
				Return(taskResult, true)

			mockRegistry.EXPECT().
				WaitForAgent(gomock.Any(), agentID, tt.expectedTimeout).
				Return(taskResult, nil)

			tool := &agentOutputToolImpl{
				hookManager: mockHookManager,
				registry:    mockRegistry,
				agent:       mockAgent,
			}

			result, err := tool.Run(context.Background(), map[string]any{
				"agent_id": agentID.String(),
				"block":    true,
				"timeout":  tt.inputTimeout,
			})

			require.NoError(t, err)
			assert.True(t, result["success"].(bool))
			ctrl.Finish()
		})
	}
}

func TestAgentOutputTool_Run_DefaultBlockValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()

	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(senderID).AnyTimes()

	agentID := uuid.New()
	startedAt := time.Now().Unix()

	taskResult := &shared.AgentResult{
		AgentID:   agentID,
		Status:    shared.AgentStatusRunning,
		StartedAt: startedAt,
	}

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(true)

	// When block is not specified, it should default to true (blocking mode)
	mockRegistry.EXPECT().
		GetAgentResult(agentID).
		Return(taskResult, true)

	mockRegistry.EXPECT().
		WaitForAgent(gomock.Any(), agentID, 30000*time.Millisecond).
		Return(taskResult, nil)

	tool := &agentOutputToolImpl{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		agent:       mockAgent,
	}

	// No "block" parameter provided - should default to blocking mode
	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
}

func TestAgentOutputToolProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	senderID := uuid.New()

	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(senderID).AnyTimes()

	provider := &agentOutputToolProvider{
		hookManager: mockHookManager,
		registry:    mockRegistry,
	}

	tool := provider.CreateTool(mockAgent)
	toolImpl := tool.(*agentOutputToolImpl)

	assert.NotNil(t, tool)
	assert.Equal(t, mockRegistry, toolImpl.registry)
	assert.Equal(t, mockHookManager, toolImpl.hookManager)
	assert.Equal(t, senderID, toolImpl.agent.GetID())
}

// TestAgentOutputTool_Run_PermissionDenied tests permission check when caller is not direct parent
func TestAgentOutputTool_Run_PermissionDenied(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()

	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(senderID).AnyTimes()

	agentID := uuid.New()

	// Permission check: sender is NOT direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(false)

	tool := &agentOutputToolImpl{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		agent:       mockAgent,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": agentID.String(),
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "permission denied")
	assert.Contains(t, result["error"].(string), "direct subagents")
	assert.Contains(t, result["error"].(string), "not your direct child")
}
