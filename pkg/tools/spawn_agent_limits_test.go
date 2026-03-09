package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSpawnAgentTool_Run_AgentFactoryError_LimitExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	mockAgentFactory := mocks.NewMockAgentFactory(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockExecHelper := mocks.NewMockAgentExecutionHelper(ctrl)
	mockConfigService := setupMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Set up default behavior for response helper methods
	setupMockExecutionHelperWithDefaults(mockExecHelper)

	senderID := uuid.New()

	// Mock agent factory that returns a limit error
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).Return(
		nil, fmt.Errorf("maximum agent limit reached (50)"),
	)

	mockPromptMgr.EXPECT().GetSubagentTaskPrompt(gomock.Any(), gomock.Any()).Return("System prompt", nil)
	mockRegistry.EXPECT().GetAgent(senderID).Return(nil, false)
	// No StoreTaskResult expected when CreateAgent fails

	tool := &spawnAgentToolImpl{
		logService:      logService,
		agentFactory:    mockAgentFactory,
		registry:        mockRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: mockExecHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        senderID,
	}

	args := map[string]any{
		"role":        "Test Role",
		"description": "Test",
		"prompt":      "Do something",
	}

	result, err := tool.Run(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "failed to create subagent")
	assert.Contains(t, result["error"].(string), "maximum agent limit reached")
}

func TestSpawnAgentTool_Run_AgentFactoryError_SubAgentLimitExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	mockAgentFactory := mocks.NewMockAgentFactory(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockExecHelper := mocks.NewMockAgentExecutionHelper(ctrl)
	mockConfigService := setupMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Set up default behavior for response helper methods
	setupMockExecutionHelperWithDefaults(mockExecHelper)

	senderID := uuid.New()

	// Mock agent factory that returns a sub-agent limit error
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).Return(
		nil, fmt.Errorf("maximum sub-agent limit reached for parent (3)"),
	)

	mockPromptMgr.EXPECT().GetSubagentTaskPrompt(gomock.Any(), gomock.Any()).Return("System prompt", nil)
	mockRegistry.EXPECT().GetAgent(senderID).Return(nil, false)
	// No StoreTaskResult expected when CreateAgent fails

	tool := &spawnAgentToolImpl{
		logService:      logService,
		agentFactory:    mockAgentFactory,
		registry:        mockRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: mockExecHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        senderID,
	}

	args := map[string]any{
		"role":        "Test Role",
		"description": "Test",
		"prompt":      "Do something",
	}

	result, err := tool.Run(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "failed to create subagent")
	assert.Contains(t, result["error"].(string), "maximum sub-agent limit reached")
}

func TestSpawnAgentTool_Run_AgentFactoryError_GenericError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	mockAgentFactory := mocks.NewMockAgentFactory(ctrl)
	mockRegistry := mocks.NewMockAgentRegistry(ctrl)
	mockPromptMgr := mocks.NewMockPromptManager(ctrl)
	mockExecHelper := mocks.NewMockAgentExecutionHelper(ctrl)
	mockConfigService := setupMockConfigService(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Set up default behavior for response helper methods
	setupMockExecutionHelperWithDefaults(mockExecHelper)

	senderID := uuid.New()

	// Mock agent factory that returns a generic error
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).Return(
		nil, fmt.Errorf("LLM provider not supported"),
	)

	mockPromptMgr.EXPECT().GetSubagentTaskPrompt(gomock.Any(), gomock.Any()).Return("System prompt", nil)
	mockRegistry.EXPECT().GetAgent(senderID).Return(nil, false)
	// No StoreTaskResult expected when CreateAgent fails

	tool := &spawnAgentToolImpl{
		logService:      logService,
		agentFactory:    mockAgentFactory,
		registry:        mockRegistry,
		promptManager:   mockPromptMgr,
		executionHelper: mockExecHelper,
		configService:   mockConfigService,
		hookManager:     mockHookManager,
		senderID:        senderID,
	}

	args := map[string]any{
		"role":        "Test Role",
		"description": "Test",
		"prompt":      "Do something",
	}

	result, err := tool.Run(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "failed to create subagent")
}
