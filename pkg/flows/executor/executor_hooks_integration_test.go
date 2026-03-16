package executor

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	mcpregistry "github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestFlowExecutor_WithHookManagerIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Create mock logger
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().InfoWithFlowStep(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager using the real implementation
	do.Provide(injector, hooks.NewHookManager)

	// Register other required dependencies using test helpers
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, flowregistry.FlowRegistry(&testFlowRegistry{}))
	do.ProvideValue(injector, mcpregistry.MCPRegistry(&testMCPRegistry{}))
	do.ProvideValue(injector, tools.FlowToolsProvider(&testFlowToolsProvider{}))
	// Create mock AgentFactory for tests that don't need LLM functionality
	mockAgentFactory := shared.NewMockAgentFactory(ctrl)
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)
	do.Provide(injector, NewFlowExecutor)

	// Create executor
	execService := do.MustInvoke[FlowExecutorService](injector)
	assert.NotNil(t, execService)

	// Verify executor can create instances
	flow := &flows.Flow{
		Name: "test-flow",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "name", Required: true}},
		},
		States: []flows.State{
			{
				Name:    "initial",
				Initial: true,
				Steps: []flows.Step{
					{Type: "func", Function: "test"},
				},
				Transitions: []flows.Transition{
					{To: "final"},
				},
			},
			{Name: "final"},
		},
	}

	instance := execService.New(flow)
	assert.NotNil(t, instance)

	// Note: Hooks won't be called yet - that's next task
	// This test just verifies the executor can be created with HookManager
}

func TestFlowExecutor_ExecuteStepWithHooks(t *testing.T) {
	injector := setupTestDI(t)
	hm := do.MustInvoke[hooks.HookManager](injector)

	var capturedBefore, capturedAfter *hooks.ExecutorPayload

	// Register before hook
	_ = hm.RegisterExecutorHook(func(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.ExecutorPayload], next func() error) error {
		capturedBefore = &hookCtx.Payload
		return next()
	}, hooks.TypedHookMetadata{Name: "before", Point: hooks.BeforeFlowStep})

	// Register after hook
	_ = hm.RegisterExecutorHook(func(_ context.Context, hookCtx *hooks.TypedHookContext[hooks.ExecutorPayload], next func() error) error {
		capturedAfter = &hookCtx.Payload
		return next()
	}, hooks.TypedHookMetadata{Name: "after", Point: hooks.AfterFlowStep})

	execService, _ := NewFlowExecutor(injector)

	flow := &flows.Flow{
		Name: "hook-test-flow",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "name", Required: true}},
		},
		States: []flows.State{
			{
				Name:    "initial",
				Initial: true,
				Steps: []flows.Step{
					{Type: "func", Function: "noop"}, // Will be mocked
				},
				Transitions: []flows.Transition{
					{To: "final", When: "true"},
				},
			},
			{Name: "final"},
		},
	}

	instance := execService.New(flow)
	instance.SetInput(map[string]string{"name": "test"})

	// Mock the func step to return successfully
	// (For now, we'll test that the wrapper works even if step fails)

	err := instance.Run()

	// We expect this to fail (func not found), but hooks should still be called
	assert.Error(t, err)

	// Verify hooks were called
	assert.NotNil(t, capturedBefore, "Before hook should be called")
	assert.NotNil(t, capturedAfter, "After hook should be called")
	assert.Equal(t, "hook-test-flow", capturedBefore.FlowName)
	assert.Equal(t, "func", capturedBefore.StepType)
	assert.Equal(t, "initial", capturedBefore.CurrentState)
	assert.Greater(t, capturedAfter.Duration, int64(0))
}
