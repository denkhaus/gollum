package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	mcpregistry "github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestExecutor_FuncStep_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create a temporary workspace directory
	tempDir := t.TempDir()
	funcsDir := filepath.Join(tempDir, ".gollum", "functions")
	require.NoError(t, os.MkdirAll(funcsDir, 0755))

	// Write a double.go function file
	doubleGoPath := filepath.Join(funcsDir, "double.go")
	doubleSource := `package double

func Double(x int) int {
	return x * 2
}
`
	require.NoError(t, os.WriteFile(doubleGoPath, []byte(doubleSource), 0644))

	// Set up DI injector with real extension service that loads from temp dir
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Create mock logger
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any logging calls
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()
	// Allow InfoWithFlowStep calls (state transitions)
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().ErrorWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Create mock workspace service that returns our temp dir
	mockWorkspace := workspace.NewMockService(ctrl)
	mockWorkspace.EXPECT().GetCurrentWorkspace().Return(tempDir).AnyTimes()
	mockWorkspace.EXPECT().GetWorkspaceHistory().Return([]string{}).AnyTimes()
	do.ProvideValue[workspace.Service](injector, mockWorkspace)

	// Register real extension service (will load from temp dir)
	do.Provide(injector, extensions.NewGatewayService)
	do.Provide(injector, extensions.NewYaegiFuncRunner)
	do.Provide(injector, extensions.NewYaegiLoader)
	do.Provide(injector, extensions.NewExtensionServiceWithWorkspace)

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

	// Register mock dependencies for other services
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))
	// Create mock AgentFactory for tests that don't need LLM functionality
	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)
	do.Provide(injector, NewFlowExecutor)

	// Load extensions (this will load double.go)
	extService := do.MustInvoke[extensions.ExtensionService](injector)
	ctx := context.Background()
	require.NoError(t, extService.LoadAll(ctx), "Failed to load extensions")

	// Verify double function was loaded
	funcRunner := extService.GetFuncRunner()
	funcs := funcRunner.ListFuncs()
	t.Logf("Loaded functions: %v", funcs)
	require.Contains(t, funcs, "double.Double", "double.Double function should be loaded")

	// Create a flow that uses the double function
	flow := &flows.Flow{
		Name:    "test-double-flow",
		Version: "1.0",
		Input: &flows.InputBlock{
			Ints: []flows.FieldDef{{Name: "value", Required: true}},
		},
		Output: &flows.OutputBlock{
			Ints: []flows.FieldDef{{Name: "result"}},
		},
		States: []flows.State{
			{
				Name:    "init",
				Initial: true,
				Steps: []flows.Step{
					{
						Type:     "func",
						Function: "double.Double",
						Params:   []flows.StepParam{{Name: "x", AssignFrom: "input.value"}},
						Result:   &flows.StepResult{AssignTo: "output.result"},
					},
				},
				Transitions: []flows.Transition{{To: "done"}},
			},
			{Name: "done"},
		},
	}

	// Create executor and run the flow
	execSvc := do.MustInvoke[FlowExecutorService](injector)
	exec := execSvc.New(flow)
	exec.SetInput(map[string]string{"value": "21"})

	_, err := exec.Run()
	require.NoError(t, err, "Flow execution should succeed")

	// Verify the output
	result, err := exec.GetContext().GetOutputField("result")
	require.NoError(t, err, "Result field should exist")
	assert.Equal(t, 42, result, "21 doubled should be 42")
}
