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

	// Expect logging calls
	mockLogger.EXPECT().Info("ExecuteFlow operation started", gomock.Any()).Times(1)
	mockLogger.EXPECT().Info("ExecuteFlow operation completed successfully", gomock.Any()).Times(1)

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

	// Run without flowName - now returns error response map, not Go error
	result, err := tool.Run(context.Background(), map[string]any{
		"inputs": map[string]any{},
	})

	require.NoError(t, err) // No Go error, error is in response map
	assert.NotNil(t, result)
	assert.Equal(t, false, result[string(shared.KeySuccess)])
	assert.Contains(t, result[string(shared.KeyError)], "flowName")
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

	// Expect logging calls
	mockLogger.EXPECT().Info("ExecuteFlow operation started", gomock.Any()).Times(1)
	mockLogger.EXPECT().Error("Flow not found", gomock.Any()).Times(1)

	// Run the tool - now returns error response map, not Go error
	result, err := tool.Run(context.Background(), map[string]any{
		"flowName": "non-existent",
		"inputs":   map[string]any{},
	})

	require.NoError(t, err) // No Go error, error is in response map
	assert.NotNil(t, result)
	assert.Equal(t, false, result[string(shared.KeySuccess)])
	assert.Contains(t, result[string(shared.KeyError)], "flow not found")
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

	// Expect logging calls
	mockLogger.EXPECT().Info("ExecuteFlow operation started", gomock.Any()).Times(1)
	mockLogger.EXPECT().Info("ExecuteFlow operation completed successfully", gomock.Any()).Times(1)

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

	// Expect logging calls
	mockLogger.EXPECT().Info("ExecuteFlow operation started", gomock.Any()).Times(1)
	mockLogger.EXPECT().Error("Flow execution failed", gomock.Any()).Times(1)

	// Setup executor to return error
	mockExecutor.executeFunc = func(ctx context.Context, flow *flows.Flow, inputs map[string]any) (*flows.FlowExecutionResult, error) {
		return nil, assert.AnError
	}

	// Run the tool - now returns error response map, not Go error
	result, err := tool.Run(context.Background(), map[string]any{
		"flowName": "test-flow",
		"inputs":   map[string]any{},
	})

	require.NoError(t, err) // No Go error, error is in response map
	assert.NotNil(t, result)
	assert.Equal(t, false, result[string(shared.KeySuccess)])
	assert.Contains(t, result[string(shared.KeyError)], "flow execution failed")
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

func TestListFlowsTool_Run_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	tool := NewListFlowsTool(mockRegistry, mockLogger)

	// Create test flow info
	testFlows := []*flowregistry.FlowInfo{
		{
			Name:         "test-flow-1",
			Description:  "First test flow",
			Version:      "1.0.0",
			InputFields:  []flowregistry.FieldInfo{{Name: "url", Type: "string", Required: true}},
			OutputFields: []flowregistry.FieldInfo{{Name: "result", Type: "string"}},
			States:       []string{"start", "end"},
		},
		{
			Name:         "test-flow-2",
			Description:  "Second test flow",
			Version:      "2.0.0",
			InputFields:  []flowregistry.FieldInfo{{Name: "data", Type: "int", Required: false}},
			OutputFields: []flowregistry.FieldInfo{{Name: "output", Type: "int"}},
			States:       []string{"process"},
		},
	}

	// Expect ListFlows call
	mockRegistry.EXPECT().ListFlows().Return(testFlows, nil)

	// Expect logging calls
	mockLogger.EXPECT().Info("ListFlows operation started", gomock.Any()).Times(1)
	mockLogger.EXPECT().Info("ListFlows operation completed successfully", gomock.Any()).Times(1)

	// Run the tool
	result, err := tool.Run(context.Background(), map[string]any{})

	require.NoError(t, err)
	assert.NotNil(t, result["flows"])
	flows := result["flows"].([]*flowregistry.FlowInfo)
	assert.Len(t, flows, 2)
	assert.Equal(t, "test-flow-1", flows[0].Name)
	assert.Equal(t, "test-flow-2", flows[1].Name)
}

func TestListFlowsTool_Run_ListFlowsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	tool := NewListFlowsTool(mockRegistry, mockLogger)

	// Expect ListFlows to fail
	mockRegistry.EXPECT().ListFlows().Return(nil, assert.AnError)

	// Expect logging calls
	mockLogger.EXPECT().Info("ListFlows operation started", gomock.Any()).Times(1)
	mockLogger.EXPECT().Error("Failed to list flows", gomock.Any()).Times(1)

	// Run the tool - should return error response map
	result, err := tool.Run(context.Background(), map[string]any{})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, false, result[string(shared.KeySuccess)])
	assert.Contains(t, result[string(shared.KeyError)], "failed to list flows")
}

func TestListFlowsTool_Spec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	tool := NewListFlowsTool(mockRegistry, mockLogger)

	spec := tool.Spec()

	assert.Equal(t, shared.ToolNameListFlows.String(), spec.Name)
	assert.Contains(t, spec.Description, "Lists all available flows")
	assert.NotNil(t, spec.Parameters)
	// ListFlows should have no required parameters
	assert.Empty(t, spec.Parameters)
}
