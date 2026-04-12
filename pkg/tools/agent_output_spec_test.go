package tools

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/registry"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAgentOutputTool_Spec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockAgent := shared.NewMockAgent(ctrl)
	agentID := uuid.New()
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &agentOutputToolImpl{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		agent:       mockAgent,
	}

	spec := tool.Spec()

	// Verify spec structure
	assert.Equal(t, shared.ToolNameAgentOutput.String(), spec.Name)
	assert.NotEmpty(t, spec.Description)

	// Verify required parameters

	// Verify agent_id parameter
	assert.NotNil(t, spec.Parameters["agent_id"])
	assert.Equal(t, gollem.TypeString, spec.Parameters["agent_id"].Type)

	// Verify block parameter
	assert.NotNil(t, spec.Parameters["block"])
	assert.Equal(t, gollem.TypeBoolean, spec.Parameters["block"].Type)

	// Verify timeout parameter
	assert.NotNil(t, spec.Parameters["timeout"])
	assert.Equal(t, gollem.TypeInteger, spec.Parameters["timeout"].Type)
}

func TestAgentOutputTool_Run_MissingAgentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockAgent := shared.NewMockAgent(ctrl)
	agentID := uuid.New()
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &agentOutputToolImpl{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		agent:       mockAgent,
	}

	result, err := tool.Run(context.Background(), map[string]any{})

	// Tool returns nil error with error response
	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "agent_id is required")
}

func TestAgentOutputTool_Run_EmptyAgentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockAgent := shared.NewMockAgent(ctrl)
	agentID := uuid.New()
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &agentOutputToolImpl{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		agent:       mockAgent,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": "",
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "agent_id is required")
}

func TestAgentOutputTool_Run_InvalidUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockAgent := shared.NewMockAgent(ctrl)
	agentID := uuid.New()
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &agentOutputToolImpl{
		hookManager: mockHookManager,
		registry:    mockRegistry,
		agent:       mockAgent,
	}

	result, err := tool.Run(context.Background(), map[string]any{
		"agent_id": "not-a-uuid",
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "agent_id must be a valid UUID")
}

func TestAgentOutputTool_Run_AgentNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	agentID := uuid.New()

	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(senderID).AnyTimes()
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", senderID, uuid.Nil)).AnyTimes()

	// Permission check: sender is direct parent
	mockRegistry.EXPECT().
		IsDirectParent(senderID, agentID).
		Return(true)

	mockRegistry.EXPECT().
		GetAgentResult(agentID).
		Return(nil, false)

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
	assert.Contains(t, result["error"].(string), "not found")
}
