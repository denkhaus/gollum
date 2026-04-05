package executor

import (
	"context"

	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/shared"
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
		flow:              flow,
		ctx:               newContext(flow.Input, flow.Output, flow.Context, nil),
		history:           NewExecutionHistory(),
		currentState:      currentState,
		bashToolProvider:  &testBashToolProvider{},
		extService:        &testExtensionService{},
		flowRegistry:      &testFlowRegistry{},
		hookManager:       &hooks.NoOpHookManager{},
		flowToolsProvider: &testFlowToolsProvider{},
	}
}

// NewExecutorWithRegistry creates a test executor with a specific flow registry
func NewExecutorWithRegistry(flow *flows.Flow, registry flowregistry.FlowRegistry) *flowExecutorImpl {
	return &flowExecutorImpl{
		flow:              flow,
		ctx:               newContext(flow.Input, flow.Output, flow.Context, nil),
		history:           NewExecutionHistory(),
		bashToolProvider:  &testBashToolProvider{},
		extService:        &testExtensionService{},
		flowRegistry:      registry,
		hookManager:       &hooks.NoOpHookManager{},
		flowToolsProvider: &testFlowToolsProvider{},
	}
}

// NewExecutorWithProvider creates a test executor with a specific bash tool provider
func NewExecutorWithProvider(flow *flows.Flow, provider tools.BashToolProvider) *flowExecutorImpl {
	return &flowExecutorImpl{
		flow:              flow,
		ctx:               newContext(flow.Input, flow.Output, flow.Context, nil),
		history:           NewExecutionHistory(),
		bashToolProvider:  provider,
		extService:        &testExtensionService{},
		flowRegistry:      &testFlowRegistry{},
		hookManager:       &hooks.NoOpHookManager{},
		flowToolsProvider: &testFlowToolsProvider{},
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

func (m *testExtensionService) GetFuncRunner() extensions.YaegiFuncRunner {
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

type testFlowRegistry struct {
	flows map[string]*flows.Flow
}

func (m *testFlowRegistry) Register(name string, flow *flows.Flow) {
	if m.flows == nil {
		m.flows = make(map[string]*flows.Flow)
	}
	m.flows[name] = flow
}

func (m *testFlowRegistry) GetFlow(name string) (*flows.Flow, error) {
	flow, ok := m.flows[name]
	if !ok {
		return nil, flowregistry.ErrFlowNotFound
	}
	return flow, nil
}

func (m *testFlowRegistry) GetFlowInfo(name string) (*flowregistry.FlowInfo, error) {
	flow, err := m.GetFlow(name)
	if err != nil {
		return nil, err
	}
	return &flowregistry.FlowInfo{
		Name:        name,
		Description: flow.Description,
		States:      []string{},
	}, nil
}

func (m *testFlowRegistry) ListFlows() ([]*flowregistry.FlowInfo, error) {
	var infos []*flowregistry.FlowInfo
	for name := range m.flows {
		info, err := m.GetFlowInfo(name)
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}
	return infos, nil
}

type testFlowToolsProvider struct{}

func (m *testFlowToolsProvider) CreateTool(agentID uuid.UUID, flowCtx flows.FlowContext, toolName shared.ToolName) (gollem.Tool, error) {
	// Return a mock tool that does nothing
	return &testFlowTool{}, nil
}

type testFlowTool struct{}

func (m *testFlowTool) Name() string {
	return "test_flow_tool"
}

func (m *testFlowTool) Description() string {
	return "Test flow tool"
}

func (m *testFlowTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        "test_flow_tool",
		Description: "Test flow tool for testing",
	}
}

func (m *testFlowTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return map[string]any{"success": true}, nil
}
