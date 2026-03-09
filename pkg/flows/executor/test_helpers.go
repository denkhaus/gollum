package executor

import (
	"context"

	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// NewExecutor creates a test executor with mocked dependencies
func NewExecutor(flow *flows.Flow) *flowExecutorImpl {
	// Find initial state for backwards compatibility with tests
	currentState := ""
	for _, state := range flow.States {
		if state.Initial {
			currentState = state.Name
			break
		}
	}

	return &flowExecutorImpl{
		flow:             flow,
		ctx:              NewContext(flow.Input, nil),
		history:          NewExecutionHistory(),
		currentState:     currentState,
		bashToolProvider: &testBashToolProvider{},
		extService:       &testExtensionService{},
		flowRegistry:     &testFlowRegistry{},
		hookManager:      &hooks.NoOpHookManager{},
	}
}

// NewExecutorWithRegistry creates a test executor with a specific flow registry
func NewExecutorWithRegistry(flow *flows.Flow, registry flowregistry.FlowRegistry) *flowExecutorImpl {
	return &flowExecutorImpl{
		flow:             flow,
		ctx:              NewContext(flow.Input, nil),
		history:          NewExecutionHistory(),
		bashToolProvider: &testBashToolProvider{},
		extService:       &testExtensionService{},
		flowRegistry:     registry,
		hookManager:      &hooks.NoOpHookManager{},
	}
}

// NewExecutorWithProvider creates a test executor with a specific bash tool provider
func NewExecutorWithProvider(flow *flows.Flow, provider tools.BashToolProvider) *flowExecutorImpl {
	return &flowExecutorImpl{
		flow:             flow,
		ctx:              NewContext(flow.Input, nil),
		history:          NewExecutionHistory(),
		bashToolProvider: provider,
		extService:       &testExtensionService{},
		flowRegistry:     &testFlowRegistry{},
		hookManager:      &hooks.NoOpHookManager{},
	}
}

// Mock implementations for testing

type testBashToolProvider struct{}

func (m *testBashToolProvider) CreateTool(agentID uuid.UUID) gollem.Tool {
	return &testBashTool{}
}

type testBashTool struct{}

func (m *testBashTool) Name() string {
	return "bash"
}

func (m *testBashTool) Description() string {
	return "Test bash tool"
}

func (m *testBashTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "bash",
		Description: "Test bash tool",
	}
}

func (m *testBashTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Return mock success response
	return map[string]any{
		"stdout":    "test output",
		"stderr":    "",
		"exit_code": 0,
	}, nil
}

type testExtensionService struct{}

func (m *testExtensionService) LoadAll(ctx context.Context) error {
	return nil
}

func (m *testExtensionService) GetFuncRunner() extensions.ScriggoRunner {
	return &testFuncRunner{}
}

func (m *testExtensionService) GetExtension(name string) (*extensions.Extension, error) {
	return nil, extensions.ErrExtensionNotFound
}

func (m *testExtensionService) ListExtensions() []string {
	return []string{}
}

type testFuncRunner struct{}

func (m *testFuncRunner) ExecuteFunc(name string, args map[string]any) (any, error) {
	// Use the built-in registry for actual testing
	return flowregistry.GetBuiltinRegistry().Execute(name, args)
}

func (m *testFuncRunner) LoadFunc(name string, source string) error {
	return nil
}

func (m *testFuncRunner) ListFuncs() []string {
	return []string{}
}

type testFlowRegistry struct{}

func (m *testFlowRegistry) Register(name string, flow *flows.Flow) {
	// No-op for test registry
}

func (m *testFlowRegistry) GetFlow(name string) (*flows.Flow, error) {
	return nil, flowregistry.ErrFlowNotFound
}
