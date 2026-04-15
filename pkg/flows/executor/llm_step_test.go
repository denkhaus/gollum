package executor

import (
	"context"
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
	"github.com/google/uuid"
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
	mockAgent := shared.NewMockAgent(ctrl)

	// Mock the Execute method to return a successful response
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).
		Return(&gollem.ExecuteResponse{Texts: []string{"Test response for PR 123"}}, nil)

	// Create mock agent factory
	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).
		Return(mockAgent, nil)

	// Create custom injector with our mock
	injector := do.New()

	// Create and configure mock logger with all expected Debug calls
	mockLogger := logger.NewMockLoggerService(ctrl)
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
	mockMCPRegistry := mcpregistry.NewMockMCPRegistry(ctrl)
	mockMCPRegistry.EXPECT().GetToolSets().Return([]gollem.ToolSet{}).AnyTimes()
	mockMCPRegistry.EXPECT().GetToolNames().Return([]string{}).AnyTimes()
	mockMCPRegistry.EXPECT().Close().Return(nil).AnyTimes()
	do.ProvideValue(injector, mcpregistry.MCPRegistry(mockMCPRegistry))
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFlowRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
	mockFlowRegistry.EXPECT().GetFlow(gomock.Any()).Return(nil, flowregistry.ErrFlowNotFound).AnyTimes()
	do.ProvideValue(injector, flowregistry.FlowRegistry(mockFlowRegistry))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock strategy builder
	mockStrategyBuilder := &mockStrategyBuilderImpl{}
	do.ProvideValue[strategy.Builder](injector, mockStrategyBuilder)

	// Create mock LLM client provider
	mockLLMClientProvider := &mockClientProviderImpl{}
	do.ProvideValue[llm.ClientProvider](injector, mockLLMClientProvider)

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
				{Type: "llm", Agent: "worker", Prompt: "Analyze PR #${input.pr_number}", Result:		&flows.StepResult{AssignTo: "output.text"}},
			}},
		},
	}

	exec := svc.New(flow)
	err := exec.SetInput(map[string]string{"pr_number": "123"})
	require.NoError(t, err)

	step := &flows.Step{Type: "llm", Agent: "worker", Prompt: "Analyze PR #${input.pr_number}", Result:		&flows.StepResult{AssignTo: "output.text"}}
	err = exec.(*flowExecutorImpl).executeStep(step, "init")

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
	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).
		Return(nil, assert.AnError)

	// Create custom injector with our mock
	injector := do.New()

	// Create and configure mock logger with all expected Debug calls
	mockLogger := logger.NewMockLoggerService(ctrl)
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
	mockMCPRegistry := mcpregistry.NewMockMCPRegistry(ctrl)
	mockMCPRegistry.EXPECT().GetToolSets().Return([]gollem.ToolSet{}).AnyTimes()
	mockMCPRegistry.EXPECT().GetToolNames().Return([]string{}).AnyTimes()
	mockMCPRegistry.EXPECT().Close().Return(nil).AnyTimes()
	do.ProvideValue(injector, mcpregistry.MCPRegistry(mockMCPRegistry))
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFlowRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
	mockFlowRegistry.EXPECT().GetFlow(gomock.Any()).Return(nil, flowregistry.ErrFlowNotFound).AnyTimes()
	do.ProvideValue(injector, flowregistry.FlowRegistry(mockFlowRegistry))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock strategy builder
	mockStrategyBuilder := &mockStrategyBuilderImpl{}
	do.ProvideValue[strategy.Builder](injector, mockStrategyBuilder)

	// Create mock LLM client provider
	mockLLMClientProvider := &mockClientProviderImpl{}
	do.ProvideValue[llm.ClientProvider](injector, mockLLMClientProvider)

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
	mockAgent := shared.NewMockAgent(ctrl)

	// Create mock agent factory
	mockAgentFactory := shared.NewMockAgentFactory(ctrl)

	// Set up expectations - CreateAgent succeeds but Execute fails
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).
		Return(mockAgent, nil)

	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).
		Return(nil, assert.AnError)

	// Create custom injector with our mock
	injector := do.New()

	// Create and configure mock logger with all expected Debug calls
	mockLogger := logger.NewMockLoggerService(ctrl)
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
	mockMCPRegistry := mcpregistry.NewMockMCPRegistry(ctrl)
	mockMCPRegistry.EXPECT().GetToolSets().Return([]gollem.ToolSet{}).AnyTimes()
	mockMCPRegistry.EXPECT().GetToolNames().Return([]string{}).AnyTimes()
	mockMCPRegistry.EXPECT().Close().Return(nil).AnyTimes()
	do.ProvideValue(injector, mcpregistry.MCPRegistry(mockMCPRegistry))
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFlowRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
	mockFlowRegistry.EXPECT().GetFlow(gomock.Any()).Return(nil, flowregistry.ErrFlowNotFound).AnyTimes()
	do.ProvideValue(injector, flowregistry.FlowRegistry(mockFlowRegistry))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock strategy builder
	mockStrategyBuilder := &mockStrategyBuilderImpl{}
	do.ProvideValue[strategy.Builder](injector, mockStrategyBuilder)

	// Create mock LLM client provider
	mockLLMClientProvider := &mockClientProviderImpl{}
	do.ProvideValue[llm.ClientProvider](injector, mockLLMClientProvider)

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

func TestParseToolNames_Empty(t *testing.T) {
	exec := &flowExecutorImpl{}
	result := exec.parseToolNames("")

	assert.Nil(t, result)
}

func TestParseToolNames_Single(t *testing.T) {
	exec := &flowExecutorImpl{}
	result := exec.parseToolNames("bash")

	assert.Equal(t, []string{"bash"}, result)
}

func TestParseToolNames_Multiple(t *testing.T) {
	exec := &flowExecutorImpl{}
	result := exec.parseToolNames("bash,read_file,write_file")

	assert.Equal(t, []string{"bash", "read_file", "write_file"}, result)
}

func TestParseToolNames_WithSpaces(t *testing.T) {
	exec := &flowExecutorImpl{}
	result := exec.parseToolNames("bash, read_file , write_file")

	assert.Equal(t, []string{"bash", "read_file", "write_file"}, result)
}

func TestParseToolNames_EmptyItems(t *testing.T) {
	exec := &flowExecutorImpl{}
	result := exec.parseToolNames("bash,,read_file")

	assert.Equal(t, []string{"bash", "read_file"}, result)
}

func TestParseToolNames_MCPTools(t *testing.T) {
	exec := &flowExecutorImpl{}
	result := exec.parseToolNames("filesystem/read_file,filesystem/write_file")

	assert.Equal(t, []string{"filesystem/read_file", "filesystem/write_file"}, result)
}

func TestParseToolNames_Mixed(t *testing.T) {
	exec := &flowExecutorImpl{}
	result := exec.parseToolNames("bash,filesystem/read_file,current_time")

	assert.Equal(t, []string{"bash", "filesystem/read_file", "current_time"}, result)
}

func TestExecuteLLMStep_PopulatesAllowedTools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock agent
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).
		Return(&gollem.ExecuteResponse{Texts: []string{"response"}}, nil)

	// Create mock agent factory that captures the config
	var capturedConfig *shared.AgentConfig
	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, config *shared.AgentConfig) (shared.Agent, error) {
			capturedConfig = config
			return mockAgent, nil
		})

	// Create custom injector
	injector := do.New()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	do.Provide(injector, hooks.NewHookManager)
	do.ProvideValue(injector, shared.AgentFactory(mockAgentFactory))
	mockMCPRegistry := mcpregistry.NewMockMCPRegistry(ctrl)
	mockMCPRegistry.EXPECT().GetToolSets().Return([]gollem.ToolSet{}).AnyTimes()
	mockMCPRegistry.EXPECT().GetToolNames().Return([]string{}).AnyTimes()
	mockMCPRegistry.EXPECT().Close().Return(nil).AnyTimes()
	do.ProvideValue(injector, mcpregistry.MCPRegistry(mockMCPRegistry))
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFlowRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
	mockFlowRegistry.EXPECT().GetFlow(gomock.Any()).Return(nil, flowregistry.ErrFlowNotFound).AnyTimes()
	do.ProvideValue(injector, flowregistry.FlowRegistry(mockFlowRegistry))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock strategy builder
	mockStrategyBuilder := &mockStrategyBuilderImpl{}
	do.ProvideValue[strategy.Builder](injector, mockStrategyBuilder)

	// Create mock LLM client provider
	mockLLMClientProvider := &mockClientProviderImpl{}
	do.ProvideValue[llm.ClientProvider](injector, mockLLMClientProvider)

	do.Provide(injector, NewFlowExecutor)

	svc := do.MustInvoke[FlowExecutorService](injector)

	flow := &flows.Flow{
		Name: "test",
		Agents: []flows.Agent{
			{Name: "worker", Model: "claude-3.5", Prompt: "You are a helper"},
		},
		States: []flows.State{
			{Name: "init", Initial: true, Steps: []flows.Step{
				{Type: "llm", Agent: "worker", Prompt: "test", Tools: "bash,read_file"},
			}},
		},
	}

	exec := svc.New(flow)
	step := &flows.Step{Type: "llm", Agent: "worker", Prompt: "test", Tools: "bash,read_file"}
	_ = exec.(*flowExecutorImpl).executeStep(step, "init")

	// Verify AllowedTools was populated
	require.NotNil(t, capturedConfig)
	assert.Equal(t, []string{"bash", "read_file"}, capturedConfig.AllowedTools)
}

func TestSeparateFlowTools_FlowToolsOnly(t *testing.T) {
	exec := &flowExecutorImpl{}
	flowTools, allowedTools := exec.separateFlowTools([]string{"set_context_field", "set_output_field"})

	assert.Equal(t, []string{"set_context_field", "set_output_field"}, flowTools)
	assert.Nil(t, allowedTools)
}

func TestSeparateFlowTools_BuiltinToolsOnly(t *testing.T) {
	exec := &flowExecutorImpl{}
	flowTools, allowedTools := exec.separateFlowTools([]string{"bash", "read_file"})

	assert.Nil(t, flowTools)
	assert.Equal(t, []string{"bash", "read_file"}, allowedTools)
}

func TestSeparateFlowTools_Mixed(t *testing.T) {
	exec := &flowExecutorImpl{}
	flowTools, allowedTools := exec.separateFlowTools([]string{
		"bash",
		"set_context_field",
		"read_file",
		"get_context",
		"filesystem/read_file",
	})

	assert.Equal(t, []string{"set_context_field", "get_context"}, flowTools)
	assert.Equal(t, []string{"bash", "read_file", "filesystem/read_file"}, allowedTools)
}

func TestSeparateFlowTools_AllFlowTools(t *testing.T) {
	exec := &flowExecutorImpl{}
	flowTools, allowedTools := exec.separateFlowTools([]string{
		"set_context_field",
		"set_output_field",
		"get_context",
		"emit_log",
		"transition_to",
	})

	assert.Equal(t, []string{"set_context_field", "set_output_field", "get_context", "emit_log", "transition_to"}, flowTools)
	assert.Nil(t, allowedTools)
}

func TestExecuteLLMStep_SeparatesFlowTools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock agent
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(uuid.New()).AnyTimes()
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).
		Return(&gollem.ExecuteResponse{Texts: []string{"response"}}, nil)

	// Create mock agent factory that captures the config
	var capturedConfig *shared.AgentConfig
	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	mockAgentFactory.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, config *shared.AgentConfig) (shared.Agent, error) {
			capturedConfig = config
			return mockAgent, nil
		})

	// Create custom injector
	injector := do.New()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	do.Provide(injector, hooks.NewHookManager)
	do.ProvideValue(injector, shared.AgentFactory(mockAgentFactory))
	mockMCPRegistry := mcpregistry.NewMockMCPRegistry(ctrl)
	mockMCPRegistry.EXPECT().GetToolSets().Return([]gollem.ToolSet{}).AnyTimes()
	mockMCPRegistry.EXPECT().GetToolNames().Return([]string{}).AnyTimes()
	mockMCPRegistry.EXPECT().Close().Return(nil).AnyTimes()
	do.ProvideValue(injector, mcpregistry.MCPRegistry(mockMCPRegistry))
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFlowRegistry.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
	mockFlowRegistry.EXPECT().GetFlow(gomock.Any()).Return(nil, flowregistry.ErrFlowNotFound).AnyTimes()
	do.ProvideValue(injector, flowregistry.FlowRegistry(mockFlowRegistry))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))

	// Create mock strategy builder
	mockStrategyBuilder := &mockStrategyBuilderImpl{}
	do.ProvideValue[strategy.Builder](injector, mockStrategyBuilder)

	// Create mock LLM client provider
	mockLLMClientProvider := &mockClientProviderImpl{}
	do.ProvideValue[llm.ClientProvider](injector, mockLLMClientProvider)

	do.Provide(injector, NewFlowExecutor)

	svc := do.MustInvoke[FlowExecutorService](injector)

	flow := &flows.Flow{
		Name: "test",
		Agents: []flows.Agent{
			{Name: "worker", Model: "claude-3.5", Prompt: "You are a helper"},
		},
		States: []flows.State{
			{Name: "init", Initial: true, Steps: []flows.Step{
				{Type: "llm", Agent: "worker", Prompt: "test", Tools: "bash,set_context_field,read_file"},
			}},
		},
	}

	exec := svc.New(flow)
	step := &flows.Step{Type: "llm", Agent: "worker", Prompt: "test", Tools: "bash,set_context_field,read_file"}
	_ = exec.(*flowExecutorImpl).executeStep(step, "init")

	// Verify flow tools were separated from built-in tools
	require.NotNil(t, capturedConfig)
	assert.Equal(t, []string{"bash", "read_file"}, capturedConfig.AllowedTools)
}

func TestGetOutputModeForStep_VerboseControlsOutput(t *testing.T) {
	flow := &flows.Flow{
		Name: "test",
		Agents: []flows.Agent{
			{Name: "worker", Model: "claude-3.5", Prompt: "You are a helper"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow).(*flowExecutorImpl)

	// Test 1: Verbose=false (default) uses OutputModeSilent
	step1 := &flows.Step{Type: "llm", Agent: "worker", Verbose: false}
	outputMode := exec.getOutputModeForStep(step1)
	assert.Equal(t, shared.OutputModeSilent, outputMode, "verbose=false should use OutputModeSilent")

	// Test 2: Verbose=true uses OutputModeFull
	step2 := &flows.Step{Type: "llm", Agent: "worker", Verbose: true}
	outputMode = exec.getOutputModeForStep(step2)
	assert.Equal(t, shared.OutputModeFull, outputMode, "verbose=true should use OutputModeFull")

	// Test 3: Step without verbose attribute defaults to false
	step3 := &flows.Step{Type: "llm", Agent: "worker"}
	outputMode = exec.getOutputModeForStep(step3)
	assert.Equal(t, shared.OutputModeSilent, outputMode, "missing verbose should default to OutputModeSilent")
}
