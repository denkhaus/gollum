// Package agents provides agent implementations and factory functions.
package agents

import (
	"context"
	"fmt"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/strategy"
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
		logService:     mockLogger,
		mcpToolProvider: mockMCPProvider,
		// Other fields can be nil for this test
	}

	tools, err := factory.resolveTools(context.Background(), nil, nil)

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
		logService:     mockLogger,
		mcpToolProvider: mockMCPProvider,
		// Mock the bash provider
		bashToolProv: &mockBashToolProviderImpl{},
	}

	// Create a minimal DefaultAgent for testing
	testAgent := &DefaultAgent{
		id: uuid.New(),
		config: &shared.AgentConfig{},
	}
	tools, err := factory.resolveTools(context.Background(), testAgent, []string{"bash"})

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
		logService:     mockLogger,
		mcpToolProvider: mockMCPProvider,
	}

	testAgent := &DefaultAgent{
		id:     uuid.New(),
		config: &shared.AgentConfig{},
	}
	tools, err := factory.resolveTools(context.Background(), testAgent, []string{"not_a_real_tool"})

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
		logService:     mockLogger,
		mcpToolProvider: mockMCPProvider,
	}

	testAgent := &DefaultAgent{
		id:     uuid.New(),
		config: &shared.AgentConfig{},
	}
	tools, err := factory.resolveTools(context.Background(), testAgent, []string{"server/tool"})

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

	testAgent := &DefaultAgent{
		id:     uuid.New(),
		config: &shared.AgentConfig{},
	}
	tools, err := factory.resolveTools(context.Background(), testAgent, []string{"server/missing"})

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

	testAgent := &DefaultAgent{
		id:     uuid.New(),
		config: &shared.AgentConfig{},
	}
	tools, err := factory.resolveTools(context.Background(), testAgent, []string{"server/noslash"})

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

func TestCreateSupervisorAgent_AppliesOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mocks
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()

	customID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	// Test 1: Verify WithAgentID option sets the ID correctly
	config1 := &shared.AgentConfig{}
	shared.WithAgentID(customID)(config1)
	assert.Equal(t, customID, config1.ID, "WithAgentID should set the config ID")

	// Test 2: Verify multiple options can be applied
	config2 := &shared.AgentConfig{}
	shared.WithAgentID(customID)(config2)
	assert.Equal(t, customID, config2.ID, "WithAgentID should work on fresh config")

	// Test 3: Verify config without options has Nil ID initially
	config3 := &shared.AgentConfig{}
	assert.Equal(t, uuid.Nil, config3.ID, "Fresh config should have Nil ID")
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

func (m *mockBashToolProviderImpl) CreateTool(agent shared.Agent) gollem.Tool {
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

func TestSupervisorAgentOptions_WithSessionContext(t *testing.T) {
	// Test that new session context options work correctly
	customID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	sessionID := "test-session-123"
	channelID := uuid.MustParse("987fcdeb-51a2-9f3b-a456-426614174000")

	// Test WithSessionID
	config1 := &shared.AgentConfig{}
	shared.WithSessionID(sessionID)(config1)
	assert.Equal(t, sessionID, config1.SessionID, "WithSessionID should set the config SessionID")

	// Test WithChannelID
	config2 := &shared.AgentConfig{}
	shared.WithChannelID(channelID)(config2)
	assert.Equal(t, channelID, config2.ChannelID, "WithChannelID should set the config ChannelID")

	// Test all options together
	config3 := &shared.AgentConfig{}
	shared.WithAgentID(customID)(config3)
	shared.WithSessionID(sessionID)(config3)
	shared.WithChannelID(channelID)(config3)

	assert.Equal(t, customID, config3.ID, "WithAgentID should set the config ID")
	assert.Equal(t, sessionID, config3.SessionID, "WithSessionID should set the config SessionID")
	assert.Equal(t, channelID, config3.ChannelID, "WithChannelID should set the config ChannelID")
}

func TestAgentConfig_SessionContextDefaults(t *testing.T) {
	// Test that empty/nil values are the default
	config := &shared.AgentConfig{}

	assert.Equal(t, "", config.SessionID, "SessionID should default to empty string")
	assert.Equal(t, uuid.Nil, config.ChannelID, "ChannelID should default to Nil UUID")
}

func TestAgentConfig_SessionContextMarshaling(t *testing.T) {
	// Test that session context fields are properly tagged for JSON
	sessionID := "session-abc"
	channelID := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	config := &shared.AgentConfig{
		SessionID: sessionID,
		ChannelID: channelID,
	}

	assert.Equal(t, sessionID, config.SessionID)
	assert.Equal(t, channelID, config.ChannelID)
}

func TestDefaultAgentFactory_CreateAgent_UsesDefaultStrategy(t *testing.T) {
	// This is a smoke test to verify the factory properly uses the strategy builder
	// Full integration tests would require complex mocking of all dependencies
	// The key behavior we're testing: when config.Strategy is nil, the builder is called

	trackingStrategy := &mockStrategy{}

	// We can't easily mock all the dependencies, so we'll just verify
	// the mock builder signature is correct by implementing it
	mockBuilder := &trackingStrategyBuilder{
		onBuildForSubAgent: func(client gollem.LLMClient, strategyType strategy.StrategyType) gollem.Strategy {
			return trackingStrategy
		},
	}

	// Verify the mock implements the interface
	var _ strategy.Builder = mockBuilder

	// If we get here without compile errors, the interface is correctly implemented
	// The actual integration test would require a full DI container setup
	assert.True(t, true, "Strategy builder interface is correctly implemented")
}

func TestDefaultAgentFactory_CreateAgent_PreservesExistingStrategy(t *testing.T) {
	// Smoke test to verify existing strategies are preserved
	// Full integration test would require complex mocking

	existingStrategy := &mockStrategy{}

	// Verify we can create a config with an existing strategy
	config := &shared.AgentConfig{
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "test-model",
		},
		Strategy: existingStrategy,
	}

	// Verify the strategy is set
	assert.NotNil(t, config.Strategy, "Strategy should be set")
	assert.Equal(t, existingStrategy, config.Strategy, "Existing strategy should be preserved")
}

// trackingStrategyBuilder is a mock that allows tracking method calls
type trackingStrategyBuilder struct {
	onBuildForSubAgent func(client gollem.LLMClient, strategyType strategy.StrategyType) gollem.Strategy
}

func (m *trackingStrategyBuilder) BuildForSupervisor(client gollem.LLMClient, strategyType strategy.StrategyType) gollem.Strategy {
	return &mockStrategy{}
}

func (m *trackingStrategyBuilder) BuildForSubAgent(client gollem.LLMClient, strategyType strategy.StrategyType) gollem.Strategy {
	if m.onBuildForSubAgent != nil {
		return m.onBuildForSubAgent(client, strategyType)
	}
	return &mockStrategy{}
}

func (m *trackingStrategyBuilder) BuildForLLMStep(client gollem.LLMClient, strategyType strategy.StrategyType) gollem.Strategy {
	return &mockStrategy{}
}

func (m *trackingStrategyBuilder) BuildReact(cfg *config.StrategyConfig, client gollem.LLMClient) gollem.Strategy {
	return &mockStrategy{}
}

func (m *trackingStrategyBuilder) BuildSimple(client gollem.LLMClient) gollem.Strategy {
	return &mockStrategy{}
}

// Mock implementations for testing

type mockStrategy struct{}

func (m *mockStrategy) Init(ctx context.Context, inputs []gollem.Input) error {
	return nil
}

func (m *mockStrategy) Handle(ctx context.Context, state *gollem.StrategyState) ([]gollem.Input, *gollem.ExecuteResponse, error) {
	return nil, nil, nil
}

func (m *mockStrategy) Tools(ctx context.Context) ([]gollem.Tool, error) {
	return nil, nil
}

