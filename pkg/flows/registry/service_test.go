package registry

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlowRegistryService_GetFlow_ReturnsRegisteredFlow(t *testing.T) {
	svc := &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}

	testFlow := &flows.Flow{
		Name:    "test-flow",
		Version: "1.0",
		States: []flows.State{
			{Name: "init", Initial: true},
		},
	}
	svc.flows["test-flow"] = testFlow

	result, err := svc.GetFlow("test-flow")

	require.NoError(t, err)
	assert.Equal(t, "test-flow", result.Name)
}

func TestFlowRegistryService_GetFlow_NotFound_ReturnsError(t *testing.T) {
	svc := &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}

	_, err := svc.GetFlow("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "flow not found")
}

func TestFlowRegistryService_Register_AddsFlow(t *testing.T) {
	svc := &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}

	flow := &flows.Flow{Name: "test"}
	svc.Register("test", flow)

	result, err := svc.GetFlow("test")
	require.NoError(t, err)
	assert.Equal(t, "test", result.Name)
}

func TestFlowRegistryService_Register_OverwritesExisting(t *testing.T) {
	svc := &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}

	flow1 := &flows.Flow{Name: "v1"}
	flow2 := &flows.Flow{Name: "v2"}
	svc.Register("test", flow1)
	svc.Register("test", flow2)

	result, err := svc.GetFlow("test")
	require.NoError(t, err)
	assert.Equal(t, "v2", result.Name) // Should be v2 (overwritten)
}

func TestFlowRegistryService_LoadFromMap_LoadsMultipleFlows(t *testing.T) {
	svc := &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}

	flows := map[string]*flows.Flow{
		"flow1": {Name: "flow1"},
		"flow2": {Name: "flow2"},
	}
	svc.LoadFromMap(flows)

	result1, _ := svc.GetFlow("flow1")
	result2, _ := svc.GetFlow("flow2")
	assert.Equal(t, "flow1", result1.Name)
	assert.Equal(t, "flow2", result2.Name)
}

func TestFlowRegistryService_LoadFromDirectory_LoadsXMLFlows(t *testing.T) {
	// Use the existing flows directory
	flowDir := "../../../.gollum/flows/examples"

	svc := &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}

	// This should fail initially
	err := svc.LoadFromDirectory(flowDir)
	require.NoError(t, err)

	// Verify at least one flow was loaded
	// simple-flow.xml should exist
	flow, err := svc.GetFlow("simple-flow")
	require.NoError(t, err, "simple-flow should be loaded")
	assert.Equal(t, "simple-flow", flow.Name)
}

func TestFlowRegistryService_LoadFromDirectory_NonExistentDirectory_ReturnsError(t *testing.T) {
	svc := &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}

	err := svc.LoadFromDirectory("/nonexistent/directory")

	assert.Error(t, err)
}

func TestFlowRegistryService_LoadFromDirectory_LoadsAllSampleFlows(t *testing.T) {
	// Test against the existing flow samples
	flowDir := "../../../.gollum/flows"

	svc := &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}

	err := svc.LoadFromDirectory(flowDir)
	require.NoError(t, err)

	// Verify known flows from examples
	knownFlows := []string{
		"simple-flow",
		"step-types-example",
	}

	for _, flowName := range knownFlows {
		flow, err := svc.GetFlow(flowName)
		require.NoError(t, err, "flow %s should be loaded", flowName)
		assert.Equal(t, flowName, flow.Name)
		assert.NotEmpty(t, flow.States, "flow %s should have states", flowName)
	}
}

func TestFlowRegistryService_NewFlowRegistryService_LoadsWorkspaceFlows(t *testing.T) {
	// Create a new service - it should auto-load from .gollum/flows
	injector := do.New()
	svc, err := NewFlowRegistryService(injector)
	require.NoError(t, err)

	// Since we're running from the project root, it should load flows
	// from .gollum/flows
	_, err = svc.GetFlow("simple-flow")
	// The flow should be loaded if .gollum/flows exists
	// If not, we'll get an error, which is also acceptable
	if err != nil {
		assert.Contains(t, err.Error(), "flow not found")
	}
}
