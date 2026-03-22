package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
)

// TestLLMStep_VerboseIntegration verifies the verbose flag flows through the system
func TestLLMStep_VerboseIntegration(t *testing.T) {
	// Create a simple flow for testing
	flow := &flows.Flow{
		Name: "test-flow",
		Agents: []flows.Agent{
			{Name: "test", Model: "test-model", Prompt: "You are a test agent"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow).(*flowExecutorImpl)

	// Test 1: Verify verbose=false (default) uses OutputModeSilent
	step1 := &flows.Step{Type: "llm", Agent: "test", Verbose: false}
	outputMode := exec.getOutputModeForStep(step1)
	assert.Equal(t, shared.OutputModeSilent, outputMode, "verbose=false should use OutputModeSilent")

	// Test 2: Verify verbose=true uses OutputModeFull
	step2 := &flows.Step{Type: "llm", Agent: "test", Verbose: true}
	outputMode = exec.getOutputModeForStep(step2)
	assert.Equal(t, shared.OutputModeFull, outputMode, "verbose=true should use OutputModeFull")

	// Test 3: Verify step without verbose attribute defaults to false
	step3 := &flows.Step{Type: "llm", Agent: "test"}
	outputMode = exec.getOutputModeForStep(step3)
	assert.Equal(t, shared.OutputModeSilent, outputMode, "missing verbose should default to OutputModeSilent")
}
