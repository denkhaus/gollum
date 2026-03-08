package registry

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
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
