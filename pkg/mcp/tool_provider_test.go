package mcp

import (

	"context"
	"github.com/denkhaus/gollum/pkg/logger"
	"testing"

	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

)

func TestMCPToolProvider_CreateTool_ValidFormat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

	// Create a mock tool set with a tool spec
	mockToolSet := &mockToolSet{
		specs: []gollem.ToolSpec{
			{
				Name:        "test_tool",
				Description: "A test tool",
			},
		},
	}

	provider := &mcpToolProviderImpl{
		registry: &mockMCPRegistry{tools: []gollem.ToolSet{mockToolSet}},
		logger:   mockLogger,
	}

	tool, err := provider.CreateTool(uuid.Nil, "testserver/test_tool")

	require.NoError(t, err)
	assert.NotNil(t, tool)
	assert.Equal(t, "test_tool", tool.Spec().Name)
}

func TestMCPToolProvider_CreateTool_InvalidFormat_NoSlash(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

	provider := &mcpToolProviderImpl{
		registry: &mockMCPRegistry{},
		logger:   mockLogger,
	}

	tool, err := provider.CreateTool(uuid.Nil, "invalidformat")

	assert.Error(t, err)
	assert.Nil(t, tool)
	assert.Contains(t, err.Error(), "invalid MCP tool format")
}

func TestMCPToolProvider_CreateTool_ToolNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

	provider := &mcpToolProviderImpl{
		registry: &mockMCPRegistry{}, // Empty registry
		logger:   mockLogger,
	}

	tool, err := provider.CreateTool(uuid.Nil, "testserver/missing_tool")

	assert.Error(t, err)
	assert.Nil(t, tool)
	assert.Contains(t, err.Error(), "not found")
}

func TestMCPToolProvider_CreateTool_MultipleToolSets(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

	// Create multiple tool sets
	toolSet1 := &mockToolSet{
		specs: []gollem.ToolSpec{{Name: "tool1", Description: "Tool 1"}},
	}
	toolSet2 := &mockToolSet{
		specs: []gollem.ToolSpec{{Name: "tool2", Description: "Tool 2"}},
	}

	provider := &mcpToolProviderImpl{
		registry: &mockMCPRegistry{
			tools: []gollem.ToolSet{toolSet1, toolSet2},
		},
		logger: mockLogger,
	}

	// Should find tool2 even though it's in the second tool set
	tool, err := provider.CreateTool(uuid.Nil, "server2/tool2")

	assert.NoError(t, err)
	assert.NotNil(t, tool)
}

// Mock implementations
type mockMCPRegistry struct {
	tools []gollem.ToolSet
}

func (m *mockMCPRegistry) GetToolSets() []gollem.ToolSet {
	return m.tools
}

func (m *mockMCPRegistry) GetToolNames() []string {
	// Return empty list for mock
	return []string{}
}

func (m *mockMCPRegistry) Close() error {
	return nil
}

type mockToolSet struct {
	specs []gollem.ToolSpec
}

func (m *mockToolSet) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return m.specs, nil
}

func (m *mockToolSet) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return map[string]any{"result": "mock result"}, nil
}
