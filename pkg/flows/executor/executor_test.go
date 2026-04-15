package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/llm"
	"github.com/denkhaus/gollum/pkg/logger"
	mcpregistry "github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/strategy"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// setupTestDI creates a DI injector with mock services for testing
func setupTestDI(t *testing.T) do.Injector {
	injector := do.New()

	// Create mock logger
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any Debug calls with variadic arguments
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	// Allow any Warn calls (used by registry)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
	// Allow InfoWithFlowStep calls (state transitions)
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().ErrorWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager
	do.Provide(injector, hooks.NewHookManager)

	// Create and register generated mocks
	mockMCPRegistry := mcpregistry.NewMockMCPRegistry(ctrl)
	mockMCPRegistry.EXPECT().GetToolSets().Return([]gollem.ToolSet{}).AnyTimes()
	mockMCPRegistry.EXPECT().GetToolNames().Return([]string{}).AnyTimes()
	mockMCPRegistry.EXPECT().Close().Return(nil).AnyTimes()
	do.ProvideValue(injector, mcpregistry.MCPRegistry(mockMCPRegistry))

	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFlowRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
	mockFlowRegistry.EXPECT().GetFlow(gomock.Any()).Return(nil, flowregistry.ErrFlowNotFound).AnyTimes()
	do.ProvideValue(injector, flowregistry.FlowRegistry(mockFlowRegistry))

	// Register remaining custom mocks (to be migrated)
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock AgentFactory for LLM step testing
	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

	// Create mock strategy builder
	mockStrategyBuilder := &mockStrategyBuilderImpl{}
	do.ProvideValue[strategy.Builder](injector, mockStrategyBuilder)

	// Create mock LLM client provider
	mockLLMClientProvider := &mockClientProviderImpl{}
	do.ProvideValue[llm.ClientProvider](injector, mockLLMClientProvider)

	// Register the flow executor service
	do.Provide(injector, NewFlowExecutor)

	return injector
}

// setupTestDIWithRegistry creates a DI injector with mock services and a custom registry
func setupTestDIWithRegistry(t *testing.T, registry flowregistry.FlowRegistry) do.Injector {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Create mock logger
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any Debug calls with variadic arguments
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	// Allow any Warn calls (used by registry)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
	// Allow new InfoWithFlowStep calls
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().ErrorWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager
	do.Provide(injector, hooks.NewHookManager)

	// Create and register generated mocks
	mockMCPRegistry := mcpregistry.NewMockMCPRegistry(ctrl)
	mockMCPRegistry.EXPECT().GetToolSets().Return([]gollem.ToolSet{}).AnyTimes()
	mockMCPRegistry.EXPECT().GetToolNames().Return([]string{}).AnyTimes()
	mockMCPRegistry.EXPECT().Close().Return(nil).AnyTimes()
	do.ProvideValue(injector, mcpregistry.MCPRegistry(mockMCPRegistry))

	// Register dependencies
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, registry)
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock AgentFactory for LLM step testing
	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

	// Create mock strategy builder
	mockStrategyBuilder := &mockStrategyBuilderImpl{}
	do.ProvideValue[strategy.Builder](injector, mockStrategyBuilder)

	// Create mock LLM client provider
	mockLLMClientProvider := &mockClientProviderImpl{}
	do.ProvideValue[llm.ClientProvider](injector, mockLLMClientProvider)

	do.Provide(injector, NewFlowExecutor)

	return injector
}

// setupTestDIWithBashProvider creates a DI injector with mock services and a custom BashToolProvider
func setupTestDIWithBashProvider(t *testing.T, provider tools.BashToolProvider) do.Injector {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Create mock logger
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any Debug calls with variadic arguments
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	// Allow any InfoWithFlowStep calls (used for state transitions)
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().ErrorWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager
	do.Provide(injector, hooks.NewHookManager)

	// Create and register generated mocks
	mockMCPRegistry := mcpregistry.NewMockMCPRegistry(ctrl)
	mockMCPRegistry.EXPECT().GetToolSets().Return([]gollem.ToolSet{}).AnyTimes()
	mockMCPRegistry.EXPECT().GetToolNames().Return([]string{}).AnyTimes()
	mockMCPRegistry.EXPECT().Close().Return(nil).AnyTimes()
	do.ProvideValue(injector, mcpregistry.MCPRegistry(mockMCPRegistry))

	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFlowRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
	mockFlowRegistry.EXPECT().GetFlow(gomock.Any()).Return(nil, flowregistry.ErrFlowNotFound).AnyTimes()
	do.ProvideValue(injector, flowregistry.FlowRegistry(mockFlowRegistry))

	// Register dependencies
	do.ProvideValue(injector, provider)
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock AgentFactory for LLM step testing
	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

	// Create mock strategy builder
	mockStrategyBuilder := &mockStrategyBuilderImpl{}
	do.ProvideValue[strategy.Builder](injector, mockStrategyBuilder)

	// Create mock LLM client provider
	mockLLMClientProvider := &mockClientProviderImpl{}
	do.ProvideValue[llm.ClientProvider](injector, mockLLMClientProvider)

	do.Provide(injector, NewFlowExecutor)

	return injector
}

// setupTestDIWithBashProviderAndMCPRegistry creates a DI injector with mock services, custom BashToolProvider, and MCPRegistry
func setupTestDIWithBashProviderAndMCPRegistry(t *testing.T, provider tools.BashToolProvider, mcpReg mcpregistry.MCPRegistry) do.Injector {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Create mock logger
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any Debug calls with variadic arguments
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	// Allow any InfoWithFlowStep calls (used for state transitions)
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().ErrorWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager
	do.Provide(injector, hooks.NewHookManager)

	// Create and register generated mocks
	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFlowRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
	mockFlowRegistry.EXPECT().GetFlow(gomock.Any()).Return(nil, flowregistry.ErrFlowNotFound).AnyTimes()
	do.ProvideValue(injector, flowregistry.FlowRegistry(mockFlowRegistry))

	// Register dependencies
	do.ProvideValue(injector, provider)
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, mcpReg)
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock AgentFactory for LLM step testing
	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

	// Create mock strategy builder
	mockStrategyBuilder := &mockStrategyBuilderImpl{}
	do.ProvideValue[strategy.Builder](injector, mockStrategyBuilder)

	// Create mock LLM client provider
	mockLLMClientProvider := &mockClientProviderImpl{}
	do.ProvideValue[llm.ClientProvider](injector, mockLLMClientProvider)

	do.Provide(injector, NewFlowExecutor)

	return injector
}

func TestFlowExecutorService_New_CreatesExecutorInstance(t *testing.T) {
	flow := &flows.Flow{
		Name: "test-flow",
		States: []flows.State{
			{Name: "init", Initial: true},
			{Name: "done"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	instance := svc.New(flow)

	assert.NotNil(t, instance)
	// Validate will set the initial state
	err := instance.Validate()
	assert.NoError(t, err)
}

func TestFlowExecutorService_Validate_ReturnsErrorForInvalidFlow(t *testing.T) {
	flow := &flows.Flow{
		Name: "invalid-flow",
		States: []flows.State{
			{Name: "init"}, // No initial state
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	instance := svc.New(flow)
	err := instance.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no initial state")
}

// TestNewExecutor_BackwardCompatibility ensures the test helper still works
func TestNewExecutor_BackwardCompatibility(t *testing.T) {
	flow := &flows.Flow{
		Name: "test-flow",
		States: []flows.State{
			{Name: "init", Initial: true},
			{Name: "done"},
		},
	}

	exec := NewExecutor(flow)

	assert.NotNil(t, exec)
	assert.Equal(t, "test-flow", exec.flow.Name)
	assert.Equal(t, "init", exec.currentState)
}

func TestExtractJSONPath_TopLevelField(t *testing.T) {
	exec := &flowExecutorImpl{}

	data := map[string]any{
		"name": "John",
		"age":  30,
	}

	result, err := exec.extractJSONPath(data, "$.name")
	assert.NoError(t, err)
	assert.Equal(t, "John", result)
}

func TestExtractJSONPath_NestedField(t *testing.T) {
	exec := &flowExecutorImpl{}

	data := map[string]any{
		"user": map[string]any{
			"name": "Alice",
			"age":  25,
		},
	}

	result, err := exec.extractJSONPath(data, "$.user.name")
	assert.NoError(t, err)
	assert.Equal(t, "Alice", result)
}

func TestExtractJSONPath_ArrayIndex(t *testing.T) {
	exec := &flowExecutorImpl{}

	data := map[string]any{
		"items": []any{
			"first",
			"second",
			"third",
		},
	}

	result, err := exec.extractJSONPath(data, "$.items[1]")
	assert.NoError(t, err)
	assert.Equal(t, "second", result)
}

func TestExtractJSONPath_ArrayElementField(t *testing.T) {
	exec := &flowExecutorImpl{}

	data := map[string]any{
		"users": []any{
			map[string]any{"name": "Bob"},
			map[string]any{"name": "Charlie"},
		},
	}

	result, err := exec.extractJSONPath(data, "$.users[0].name")
	assert.NoError(t, err)
	assert.Equal(t, "Bob", result)
}

func TestExtractJSONPath_RootOnly(t *testing.T) {
	exec := &flowExecutorImpl{}

	data := map[string]any{
		"name": "Test",
	}

	result, err := exec.extractJSONPath(data, "$")
	assert.NoError(t, err)
	assert.Equal(t, data, result)
}

func TestExtractJSONPath_FieldNotFound(t *testing.T) {
	exec := &flowExecutorImpl{}

	data := map[string]any{
		"name": "Test",
	}

	_, err := exec.extractJSONPath(data, "$.missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExtractJSONPath_InvalidArrayIndex(t *testing.T) {
	exec := &flowExecutorImpl{}

	data := map[string]any{
		"items": []any{"one", "two"},
	}

	_, err := exec.extractJSONPath(data, "$.items[5]")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of bounds")
}
