package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/mocks"
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
	mockLogger := mocks.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	// Register HookManager using the real implementation
	do.Provide(injector, hooks.NewHookManager)

	// Register other required dependencies using test helpers
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, flowregistry.FlowRegistry(&testFlowRegistry{}))
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
