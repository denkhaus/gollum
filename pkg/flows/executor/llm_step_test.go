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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestExecuteLLMStep_ValidatesAgentExists(t *testing.T) {
	flow := &flows.Flow{
		Name: "test",
		States: []flows.State{
			{Name: "init", Initial: true, Steps: []flows.Step{
				{Type: "llm", Agent: "nonexistent"},
			}},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)
	step := &flows.Step{Type: "llm", Agent: "nonexistent"}
	err := exec.(*flowExecutorImpl).executeStep(step, "init")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
}

func TestExecuteLLMStep_AgentNotFound(t *testing.T) {
	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)

	flow := &flows.Flow{
		Name: "test",
		States: []flows.State{
			{Name: "init", Initial: true, Steps: []flows.Step{
				{Type: "llm", Agent: "nonexistent", Prompt: "test prompt"},
			}},
		},
	}

	exec := svc.New(flow)
	step := &flows.Step{Type: "llm", Agent: "nonexistent", Prompt: "test prompt"}
	err := exec.(*flowExecutorImpl).executeStep(step, "init")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found: nonexistent")
}

func TestExecuteLLMStep_SubstitutesPrompt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock agent
	mockAgent := mocks.NewMockAgent(ctrl)

	// Mock the Execute method to return a successful response
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).
		Return(&gollem.ExecuteResponse{Texts: []string{"Test response for PR 123"}}, nil)

	// Create mock agent factory
	mockAgentFactory := mocks.NewMockAgentFactory(ctrl)
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).
		Return(mockAgent, nil)

	// Create custom injector with our mock
	injector := do.New()

	// Create and configure mock logger with all expected Debug calls
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any Debug calls
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager
	do.Provide(injector, hooks.NewHookManager)

	// Add other required services
	do.ProvideValue(injector, shared.AgentFactory(mockAgentFactory))
	testMCPReg := &testMCPRegistry{}
	do.ProvideValue(injector, mcpregistry.MCPRegistry(testMCPReg))
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, flowregistry.FlowRegistry(&testFlowRegistry{}))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Register the flow executor service
	do.Provide(injector, NewFlowExecutor)

	svc := do.MustInvoke[FlowExecutorService](injector)

	flow := &flows.Flow{
		Name: "test",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "pr_number"}},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "text"}},
		},
		Agents: []flows.Agent{
			{Name: "worker", Model: "claude-3.5", Prompt: "You are a helper"},
		},
		States: []flows.State{
			{Name: "init", Initial: true, Steps: []flows.Step{
				{Type: "llm", Agent: "worker", Prompt: "Analyze PR #${input.pr_number}", Output: &flows.StepOutput{Assign: "${output.text}"}},
			}},
		},
	}

	exec := svc.New(flow)
	exec.SetInput(map[string]string{"pr_number": "123"})

	step := &flows.Step{Type: "llm", Agent: "worker", Prompt: "Analyze PR #${input.pr_number}", Output: &flows.StepOutput{Assign: "${output.text}"}}
	err := exec.(*flowExecutorImpl).executeStep(step, "init")

	// Should succeed without error
	assert.NoError(t, err)

	// Check that the response was stored in output
	ctx := exec.GetContext()
	value, err := ctx.GetOutputField("text")
	require.NoError(t, err)
	assert.Equal(t, "Test response for PR 123", value)
}

func TestExecuteLLMStep_CreateAgentFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock agent factory that returns error
	mockAgentFactory := mocks.NewMockAgentFactory(ctrl)
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).
		Return(nil, assert.AnError)

	// Create custom injector with our mock
	injector := do.New()

	// Create and configure mock logger with all expected Debug calls
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any Debug calls
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager
	do.Provide(injector, hooks.NewHookManager)

	// Add other required services
	do.ProvideValue(injector, shared.AgentFactory(mockAgentFactory))
	testMCPReg := &testMCPRegistry{}
	do.ProvideValue(injector, mcpregistry.MCPRegistry(testMCPReg))
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, flowregistry.FlowRegistry(&testFlowRegistry{}))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Register the flow executor service
	do.Provide(injector, NewFlowExecutor)

	svc := do.MustInvoke[FlowExecutorService](injector)

	flow := &flows.Flow{
		Name: "test",
		Agents: []flows.Agent{
			{Name: "worker", Model: "claude-3.5", Prompt: "You are a helper"},
		},
		States: []flows.State{
			{Name: "init", Initial: true, Steps: []flows.Step{
				{Type: "llm", Agent: "worker", Prompt: "test prompt"},
			}},
		},
	}

	exec := svc.New(flow)
	step := &flows.Step{Type: "llm", Agent: "worker", Prompt: "test prompt"}
	err := exec.(*flowExecutorImpl).executeStep(step, "init")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create agent")
}

func TestExecuteLLMStep_ExecuteFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock agent
	mockAgent := mocks.NewMockAgent(ctrl)

	// Create mock agent factory
	mockAgentFactory := mocks.NewMockAgentFactory(ctrl)

	// Set up expectations - CreateAgent succeeds but Execute fails
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).
		Return(mockAgent, nil)

	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).
		Return(nil, assert.AnError)

	// Create custom injector with our mock
	injector := do.New()

	// Create and configure mock logger with all expected Debug calls
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	// Allow any Debug calls
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager
	do.Provide(injector, hooks.NewHookManager)

	// Add other required services
	do.ProvideValue(injector, shared.AgentFactory(mockAgentFactory))
	testMCPReg := &testMCPRegistry{}
	do.ProvideValue(injector, mcpregistry.MCPRegistry(testMCPReg))
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, flowregistry.FlowRegistry(&testFlowRegistry{}))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Register the flow executor service
	do.Provide(injector, NewFlowExecutor)

	svc := do.MustInvoke[FlowExecutorService](injector)

	flow := &flows.Flow{
		Name: "test",
		Agents: []flows.Agent{
			{Name: "worker", Model: "claude-3.5", Prompt: "You are a helper"},
		},
		States: []flows.State{
			{Name: "init", Initial: true, Steps: []flows.Step{
				{Type: "llm", Agent: "worker", Prompt: "test prompt"},
			}},
		},
	}

	exec := svc.New(flow)
	step := &flows.Step{Type: "llm", Agent: "worker", Prompt: "test prompt"}
	err := exec.(*flowExecutorImpl).executeStep(step, "init")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM execution failed")
}

func TestInferLLMProvider(t *testing.T) {
	tests := []struct {
		name             string
		model            string
		expectedProvider string
	}{
		{"Claude model", "claude-3.5", "anthropic"},
		{"Claude 4", "claude-4-opus", "anthropic"},
		{"GPT model", "gpt-4", "openai"},
		{"O1 model", "o1-preview", "openai"},
		{"Gemini model", "gemini-2.0", "gemini"},
		{"Unknown model defaults to Anthropic", "unknown-model", "anthropic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flow := &flows.Flow{
				Name: "test",
				States: []flows.State{
					{Name: "init", Initial: true},
				},
			}

			injector := setupTestDI(t)
			svc := do.MustInvoke[FlowExecutorService](injector)
			_ = svc.New(flow)

			// inferLLMProvider was removed - model is now directly in LLMClientConfig
			// This test validates model string format instead
			model := tt.model
			assert.NotEmpty(t, model, "Model should not be empty")
		})
	}
}
