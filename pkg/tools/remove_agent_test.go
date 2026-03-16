package tools

import (

	"context"
	"errors"
	"github.com/denkhaus/gollum/pkg/registry"
	"testing"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

)

// NewMockAgentConfig creates a test agent config
func NewMockAgentConfig(id uuid.UUID, parentID *uuid.UUID) *shared.AgentConfig {
	return &shared.AgentConfig{
		ID:       id,
		ParentID: parentID,
	}
}

func TestRemoveAgentTool_Spec(t *testing.T) {
	tool := &removeAgentToolImpl{}
	spec := tool.Spec()

	assert.Equal(t, "remove_agent", spec.Name)
	assert.Contains(t, spec.Description, "Removes an agent")
	assert.Equal(t, gollem.TypeString, spec.Parameters["agent_id"].Type)
	assert.Equal(t, gollem.TypeBoolean, spec.Parameters["force"].Type)
}

func TestRemoveAgentTool_Run_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	targetID := uuid.New()
	childID := uuid.New()

	// Setup mocks
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	targetAgent := shared.NewMockAgent(ctrl)
	childAgent := shared.NewMockAgent(ctrl)

	// Setup mock expectations
	mockRegistry.EXPECT().GetAgent(targetID).Return(targetAgent, true)
	mockRegistry.EXPECT().GetChildren(senderID).Return([]shared.Agent{targetAgent})
	mockRegistry.EXPECT().GetChildren(targetID).Return([]shared.Agent{childAgent})
	mockRegistry.EXPECT().Cleanup(targetID).Return(nil)

	// Mock agent configs
	targetAgentConfig := NewMockAgentConfig(targetID, &senderID)
	targetAgent.EXPECT().GetConfig().Return(targetAgentConfig).AnyTimes()
	targetAgent.EXPECT().GetID().Return(targetID).AnyTimes()
	childAgent.EXPECT().GetID().Return(childID).AnyTimes()

	tool := &removeAgentToolImpl{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	args := map[string]any{
		"agent_id": targetID.String(),
		"force":    true,
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, args)

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, targetID.String(), result["removed_agent"].(string))
	assert.Equal(t, senderID.String(), result["removed_by"].(string))
	assert.Equal(t, 1, result["child_count"])
}

func TestRemoveAgentTool_Run_InvalidAgentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	mockRegistry := registry.NewMockAgentRegistry(ctrl)

	tool := &removeAgentToolImpl{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	args := map[string]any{
		"agent_id": "invalid-uuid",
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, args)

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "must be a valid UUID")
}

func TestRemoveAgentTool_Run_AgentNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	targetID := uuid.New()

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().GetAgent(targetID).Return(nil, false)

	tool := &removeAgentToolImpl{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	args := map[string]any{
		"agent_id": targetID.String(),
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, args)

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "not found")
}

func TestRemoveAgentTool_Run_SelfRemoval(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	mockRegistry := registry.NewMockAgentRegistry(ctrl)

	tool := &removeAgentToolImpl{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	args := map[string]any{
		"agent_id": senderID.String(),
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, args)

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "cannot remove yourself")
}

func TestRemoveAgentTool_Run_PermissionDenied(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	targetID := uuid.New()
	thirdPartyID := uuid.New()

	// Setup mocks - target agent is child of third party, not sender
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	targetAgent := shared.NewMockAgent(ctrl)

	mockRegistry.EXPECT().GetAgent(targetID).Return(targetAgent, true)
	mockRegistry.EXPECT().GetChildren(senderID).Return([]shared.Agent{})

	// Mock target agent config
	targetAgentConfig := NewMockAgentConfig(targetID, &thirdPartyID)
	targetAgent.EXPECT().GetConfig().Return(targetAgentConfig)
	targetAgent.EXPECT().GetID().Return(targetID).AnyTimes() // May be called multiple times

	tool := &removeAgentToolImpl{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	args := map[string]any{
		"agent_id": targetID.String(),
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, args)

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "permission denied")
}

func TestRemoveAgentTool_Run_HasChildrenNoForce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	targetID := uuid.New()

	// Setup mocks
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	targetAgent := shared.NewMockAgent(ctrl)
	childAgent := shared.NewMockAgent(ctrl)

	mockRegistry.EXPECT().GetAgent(targetID).Return(targetAgent, true)
	mockRegistry.EXPECT().GetChildren(senderID).Return([]shared.Agent{targetAgent})
	mockRegistry.EXPECT().GetChildren(targetID).Return([]shared.Agent{childAgent})

	// Mock agent configs and IDs
	targetAgentConfig := NewMockAgentConfig(targetID, &senderID)
	targetAgent.EXPECT().GetConfig().Return(targetAgentConfig)
	targetAgent.EXPECT().GetID().Return(targetID).AnyTimes()
	// childAgent.GetID() is not called since we only count children when force=false

	tool := &removeAgentToolImpl{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	args := map[string]any{
		"agent_id": targetID.String(),
		// force is not set, defaults to false
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, args)

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "has 1 active children")
	assert.True(t, result["has_children"].(bool))
	assert.Equal(t, 1, result["child_count"])
}

func TestRemoveAgentTool_Run_CleanupError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	senderID := uuid.New()
	targetID := uuid.New()

	// Setup mocks
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	targetAgent := shared.NewMockAgent(ctrl)
	cleanupError := errors.New("cleanup failed")

	mockRegistry.EXPECT().GetAgent(targetID).Return(targetAgent, true)
	mockRegistry.EXPECT().GetChildren(senderID).Return([]shared.Agent{targetAgent})
	mockRegistry.EXPECT().GetChildren(targetID).Return([]shared.Agent{})
	mockRegistry.EXPECT().Cleanup(targetID).Return(cleanupError)

	// Mock agent config and ID
	targetAgentConfig := NewMockAgentConfig(targetID, &senderID)
	targetAgent.EXPECT().GetConfig().Return(targetAgentConfig)
	targetAgent.EXPECT().GetID().Return(targetID).AnyTimes() // May be called multiple times

	tool := &removeAgentToolImpl{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    mockRegistry,
		senderID:    senderID,
	}

	args := map[string]any{
		"agent_id": targetID.String(),
	}

	ctx := context.Background()
	result, err := tool.Run(ctx, args)

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "failed to remove agent")
}

func TestRemoveAgentToolProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)

	// Test provider creation with mock registry and hook manager
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)

	provider := &removeAgentToolProvider{
		logService:  logService,
		hookManager: mockHookManager,
		registry:    mockRegistry,
	}

	// Test tool creation
	senderID := uuid.New()
	tool := provider.CreateTool(senderID)
	toolImpl := tool.(*removeAgentToolImpl)

	assert.NotNil(t, tool)
	assert.Equal(t, senderID, toolImpl.senderID)
	assert.Equal(t, mockRegistry, toolImpl.registry)
	assert.Equal(t, logService, toolImpl.logService)
	assert.Equal(t, mockHookManager, toolImpl.hookManager)
}
