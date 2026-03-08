package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
)

// setupTestDI creates a DI injector with mock services for testing
func setupTestDI(t *testing.T) do.Injector {
	injector := do.New()

	// Register mock dependencies
	do.ProvideValue(injector, tools.BashToolProvider(&testBashToolProvider{}))
	do.ProvideValue(injector, extensions.ExtensionService(&testExtensionService{}))
	do.ProvideValue(injector, flowregistry.FlowRegistry(&testFlowRegistry{}))
	// Register the flow executor service
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
