package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	mcpregistry "github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// testMCPRegistry is a simple mock for testing
type testMCPRegistry struct{}

func (m *testMCPRegistry) GetToolSets() []gollem.ToolSet {
	return []gollem.ToolSet{}
}

func (m *testMCPRegistry) Close() error {
	return nil
}

// setupTestDI creates a DI injector with mock services for testing
func setupTestDI(t *testing.T) do.Injector {
	injector := do.New()

	// Create mock logger
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any Debug calls with variadic arguments
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager
	do.Provide(injector, hooks.NewHookManager)

	// Register mock dependencies
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, flowregistry.FlowRegistry(&testFlowRegistry{}))
	do.ProvideValue(injector, mcpregistry.MCPRegistry(&testMCPRegistry{}))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock AgentFactory for LLM step testing
	mockAgentFactory := mocks.NewMockAgentFactory(ctrl)
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

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
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any Debug calls with variadic arguments
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager
	do.Provide(injector, hooks.NewHookManager)

	// Register dependencies
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, registry)
	do.ProvideValue(injector, mcpregistry.MCPRegistry(&testMCPRegistry{}))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock AgentFactory for LLM step testing
	mockAgentFactory := mocks.NewMockAgentFactory(ctrl)
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

	do.Provide(injector, NewFlowExecutor)

	return injector
}

// setupTestDIWithBashProvider creates a DI injector with mock services and a custom BashToolProvider
func setupTestDIWithBashProvider(t *testing.T, provider tools.BashToolProvider) do.Injector {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Create mock logger
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any Debug calls with variadic arguments
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager
	do.Provide(injector, hooks.NewHookManager)

	// Register dependencies
	do.ProvideValue(injector, provider)
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, flowregistry.FlowRegistry(&testFlowRegistry{}))
	do.ProvideValue(injector, mcpregistry.MCPRegistry(&testMCPRegistry{}))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock AgentFactory for LLM step testing
	mockAgentFactory := mocks.NewMockAgentFactory(ctrl)
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

	do.Provide(injector, NewFlowExecutor)

	return injector
}

// setupTestDIWithBashProviderAndMCPRegistry creates a DI injector with mock services, custom BashToolProvider, and MCPRegistry
func setupTestDIWithBashProviderAndMCPRegistry(t *testing.T, provider tools.BashToolProvider, mcpReg mcpregistry.MCPRegistry) do.Injector {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Create mock logger
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any Debug calls with variadic arguments
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager
	do.Provide(injector, hooks.NewHookManager)

	// Register dependencies
	do.ProvideValue(injector, provider)
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, flowregistry.FlowRegistry(&testFlowRegistry{}))
	do.ProvideValue(injector, mcpregistry.MCPRegistry(mcpReg))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock AgentFactory for LLM step testing
	mockAgentFactory := mocks.NewMockAgentFactory(ctrl)
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

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
