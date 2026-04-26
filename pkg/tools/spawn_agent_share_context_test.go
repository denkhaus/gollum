package tools

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestSpawnAgentTool_WithShareContext tests that share_context parameter properly passes parent message history to subagent
func TestSpawnAgentTool_WithShareContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	senderAgent := shared.NewMockAgent(ctrl)
	senderID := uuid.New()
	senderAgent.EXPECT().GetID().Return(senderID).AnyTimes()
	taskID := uuid.New()

	// Setup mocks
	mockFactory := shared.NewMockAgentFactory(ctrl)
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockPromptMgr := manager.NewMockPromptManager(ctrl)
	mockExecHelper := NewMockAgentExecutionHelper(ctrl)
	mockConfigService := setupMockConfigService(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Create mock parent agent with message history
	mockParentAgent := shared.NewMockAgent(ctrl)
	expectedHistory := &gollem.History{
		Messages: []gollem.Message{
			{Role: gollem.RoleUser},
			{Role: gollem.RoleAssistant},
			{Role: gollem.RoleUser},
		},
	}

	mockParentAgent.EXPECT().GetConfig().Return(&shared.AgentConfig{
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	})
	mockParentAgent.EXPECT().GetID().Return(senderID).Times(4) // GetID is called at lines 211, 215, 251, 258
	mockParentAgent.EXPECT().GetMessageHistory(gomock.Any()).Return(expectedHistory, nil)

	tool := &spawnAgentToolImpl{
		logService:      logService,
		agentFactory:    mockFactory,
		registry:        mockRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: mockExecHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		agent:           senderAgent,
	}

	ctx := context.Background()

	mockPromptMgr.EXPECT().GetSubagentTaskPrompt(gomock.Any(), gomock.Any()).Return("System prompt", nil)
	mockRegistry.EXPECT().GetAgent(senderAgent).Return(mockParentAgent, true)
	mockRegistry.EXPECT().StoreAgentResult(gomock.Any()).Return(nil).Times(1)
	mockFactory.EXPECT().CreateAgent(ctx, gomock.Any()).Do(func(_ context.Context, cfg *shared.AgentConfig) {
		// Verify message history was passed to subagent config
		assert.Equal(t, expectedHistory, cfg.History)
		assert.NotNil(t, cfg.History)
		assert.Equal(t, 3, len(cfg.History.Messages))
	}).Return(mockParentAgent, nil)

	mockRegistry.EXPECT().Register(mockParentAgent, gomock.Any()).Return(nil)

	// Mock execution helper
	expectedResponse := map[string]any{
		"success":  true,
		"agent_id": taskID.String(),
		"response": "Task completed",
		"status":   "completed",
		"message":  "Agent completed",
	}
	mockExecHelper.EXPECT().ExecuteSynchronously(ctx, mockParentAgent, gomock.Any()).Return(expectedResponse, nil)

	result, err := tool.Run(ctx, map[string]any{
		"role":          "Tester",
		"description":   "Test task",
		"prompt":        "Do something",
		"share_context": true,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "completed", result["status"].(string))
}
