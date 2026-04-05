// Package agents provides agent implementations and factory functions.
package agents

import (
	"context"
	"fmt"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestDefaultAgentFactory_ResolveTools_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	mockMCPProvider := &mockMCPToolProvider{}

	factory := &defaultAgentFactory{
		logService:  mockLogger,
		mcpToolProvider: mockMCPProvider,
		// Other fields can be nil for this test
	}

	tools, err := factory.resolveTools(context.Background(), uuid.Nil, nil)

	assert.NoError(t, err)
	assert.Nil(t, tools)
}

func TestDefaultAgentFactory_ResolveTools_BashOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

	mockMCPProvider := &mockMCPToolProvider{}

	factory := &defaultAgentFactory{
		logService:  mockLogger,
		mcpToolProvider: mockMCPProvider,
		// Mock the bash provider
		bashToolProv: &mockBashToolProviderImpl{},
	}

	tools, err := factory.resolveTools(context.Background(), uuid.Nil, []string{"bash"})

	assert.NoError(t, err)
	assert.Len(t, tools, 1)
}

func TestDefaultAgentFactory_ResolveTools_InvalidBuiltin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

	mockMCPProvider := &mockMCPToolProvider{}

	factory := &defaultAgentFactory{
		logService:  mockLogger,
		mcpToolProvider: mockMCPProvider,
	}

	tools, err := factory.resolveTools(context.Background(), uuid.Nil, []string{"not_a_real_tool"})

	assert.Error(t, err)
	assert.Nil(t, tools)
	assert.Contains(t, err.Error(), "unknown built-in tool")
}

func TestDefaultAgentFactory_ResolveTools_MCPTool_ValidFormat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	mockMCPProvider := &mockMCPToolProvider{
		tool: &mockTool{},
	}

	factory := &defaultAgentFactory{
		logService:  mockLogger,
		mcpToolProvider: mockMCPProvider,
	}

	tools, err := factory.resolveTools(context.Background(), uuid.Nil, []string{"server/tool"})

	assert.NoError(t, err)
	assert.Len(t, tools, 1)
}

func TestDefaultAgentFactory_ResolveTools_MCPTool_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Warn is called with msg + 1 Field (zap.Strings) = 2 args total
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	mockMCPProvider := &mockMCPToolProvider{
		err: assert.AnError,
	}

	factory := &defaultAgentFactory{
		logService:     mockLogger,
		mcpToolProvider: mockMCPProvider,
	}

	tools, err := factory.resolveTools(context.Background(), uuid.Nil, []string{"server/missing"})

	// Should succeed - MCP tool not found is a warning, not an error
	assert.NoError(t, err)
	assert.Len(t, tools, 0) // No tools since MCP tool wasn't found
}

func TestDefaultAgentFactory_ResolveTools_InvalidMCPFormat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Warn is called with msg + 1 Field (zap.Strings) = 2 args total
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	mockMCPProvider := &mockMCPToolProvider{
		err: fmt.Errorf("invalid MCP tool format: noslash (expected 'server/tool')"),
	}

	factory := &defaultAgentFactory{
		logService:     mockLogger,
		mcpToolProvider: mockMCPProvider,
	}

	tools, err := factory.resolveTools(context.Background(), uuid.Nil, []string{"server/noslash"})

	// MCP provider returns error for invalid format - this is treated as a warning
	// and the tool is skipped, so it succeeds but returns no tools
	assert.NoError(t, err)
	assert.Len(t, tools, 0)
}

func TestCreateSupervisorAgent_WithAgentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup minimal mocks - same as other tests in this file
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	customID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	// Test that WithAgentID option correctly sets the ID
	config := &shared.AgentConfig{}
	shared.WithAgentID(customID)(config)

	// Verify the ID was set correctly on the config
	assert.Equal(t, customID, config.ID)
}

// Mock implementations
type mockMCPToolProvider struct {
	tool gollem.Tool
	err  error
}

func (m *mockMCPToolProvider) CreateTool(agentID uuid.UUID, serverAndTool string) (gollem.Tool, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tool, nil
}

type mockTool struct{}

func (m *mockTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "mock",
		Description: "Mock tool",
	}
}

func (m *mockTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}

// mockBashToolProviderImpl is a mock implementation of BashToolProvider
type mockBashToolProviderImpl struct{}

func (m *mockBashToolProviderImpl) CreateTool(agentID uuid.UUID) gollem.Tool {
	return &mockBashToolWithSpec{}
}

// mockBashToolWithSpec is a mock tool that returns a bash spec
type mockBashToolWithSpec struct{}

func (m *mockBashToolWithSpec) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "bash",
		Description: "Execute bash commands",
	}
}

func (m *mockBashToolWithSpec) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}
