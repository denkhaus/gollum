package tools

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// mockFlowExecutor is a mock implementation of flows.Executor
type mockFlowExecutor struct {
	executeFunc func(ctx context.Context, flow *flows.Flow, inputs map[string]any) (*flows.FlowExecutionResult, error)
}

func (m *mockFlowExecutor) Execute(ctx context.Context, flow *flows.Flow, inputs map[string]any) (*flows.FlowExecutionResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, flow, inputs)
	}
	return &flows.FlowExecutionResult{
		Outputs: map[string]any{},
	}, nil
}

func TestExecuteFlowTool_Run_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockExecutor := &mockFlowExecutor{}
	mockLogger := logger.NewMockLoggerService(ctrl)

	tool := NewExecuteFlowTool(mockRegistry, mockExecutor, mockLogger)

	testFlow := &flows.Flow{
		Name: "test-flow",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{
				{Name: "url", Type: flows.TypeString, Required: true},
			},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{{Name: "result", Type: flows.TypeString}},
		},
	}

	// Expect flow lookup
	mockRegistry.EXPECT().GetFlow("test-flow").Return(testFlow, nil)

	// Setup executor to return success
	mockExecutor.executeFunc = func(ctx context.Context, flow *flows.Flow, inputs map[string]any) (*flows.FlowExecutionResult, error) {
		return &flows.FlowExecutionResult{
			Outputs: map[string]any{
				"result": "success",
			},
		}, nil
	}

	// Run the tool
	result, err := tool.Run(context.Background(), map[string]any{
		"flowName": "test-flow",
		"inputs": map[string]any{
			"url": "https://example.com",
		},
	})

	require.NoError(t, err)
	assert.NotNil(t, result["outputs"])
	outputs := result["outputs"].(map[string]any)
	assert.Equal(t, "success", outputs["result"])
}

func TestExecuteFlowTool_Run_MissingFlowName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockExecutor := &mockFlowExecutor{}
	mockLogger := logger.NewMockLoggerService(ctrl)

	tool := NewExecuteFlowTool(mockRegistry, mockExecutor, mockLogger)

	// Run without flowName
	result, err := tool.Run(context.Background(), map[string]any{
		"inputs": map[string]any{},
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "flowName is required")
}

func TestExecuteFlowTool_Run_FlowNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockExecutor := &mockFlowExecutor{}
	mockLogger := logger.NewMockLoggerService(ctrl)

	tool := NewExecuteFlowTool(mockRegistry, mockExecutor, mockLogger)

	// Expect flow lookup to fail
	mockRegistry.EXPECT().GetFlow("non-existent").Return(nil, assert.AnError)

	// Run the tool
	result, err := tool.Run(context.Background(), map[string]any{
		"flowName": "non-existent",
		"inputs":   map[string]any{},
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "flow not found")
}

func TestExecuteFlowTool_Run_NilInputs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockExecutor := &mockFlowExecutor{}
	mockLogger := logger.NewMockLoggerService(ctrl)

	tool := NewExecuteFlowTool(mockRegistry, mockExecutor, mockLogger)

	testFlow := &flows.Flow{
		Name: "test-flow",
	}

	// Expect flow lookup
	mockRegistry.EXPECT().GetFlow("test-flow").Return(testFlow, nil)

	// Run the tool with nil inputs
	result, err := tool.Run(context.Background(), map[string]any{
		"flowName": "test-flow",
	})

	require.NoError(t, err)
	assert.NotNil(t, result["outputs"])
}

func TestExecuteFlowTool_Run_ExecutionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockExecutor := &mockFlowExecutor{}
	mockLogger := logger.NewMockLoggerService(ctrl)

	tool := NewExecuteFlowTool(mockRegistry, mockExecutor, mockLogger)

	testFlow := &flows.Flow{
		Name: "test-flow",
	}

	// Expect flow lookup
	mockRegistry.EXPECT().GetFlow("test-flow").Return(testFlow, nil)

	// Setup executor to return error
	mockExecutor.executeFunc = func(ctx context.Context, flow *flows.Flow, inputs map[string]any) (*flows.FlowExecutionResult, error) {
		return nil, assert.AnError
	}

	// Run the tool
	result, err := tool.Run(context.Background(), map[string]any{
		"flowName": "test-flow",
		"inputs":   map[string]any{},
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "flow execution failed")
}

func TestExecuteFlowTool_Spec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockExecutor := &mockFlowExecutor{}
	mockLogger := logger.NewMockLoggerService(ctrl)

	tool := NewExecuteFlowTool(mockRegistry, mockExecutor, mockLogger)

	spec := tool.Spec()

	assert.Equal(t, shared.ToolNameExecuteFlow.String(), spec.Name)
	assert.Contains(t, spec.Description, "Executes a Gollum flow")
	assert.NotNil(t, spec.Parameters)
	assert.Contains(t, spec.Parameters, "flowName")
	assert.Contains(t, spec.Parameters, "inputs")
}
