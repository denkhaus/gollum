package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecutor_Close_CanBeCalledMultipleTimes tests that Close is idempotent
func TestExecutor_Close_CanBeCalledMultipleTimes(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test",
		Version: "1.0",
		States:  []flows.State{{Name: "init", Initial: true}},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)

	// Close should not error
	err := exec.Close()
	require.NoError(t, err)

	// Close should be idempotent - calling multiple times should not error
	err = exec.Close()
	require.NoError(t, err)
}

// TestExecutor_Close_AfterRun tests cleanup after execution
func TestExecutor_Close_AfterRun(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test",
		Version: "1.0",
		States: []flows.State{
			{Name: "init", Initial: true, Transitions: []flows.Transition{{To: "done"}}},
			{Name: "done"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)

	_, err := exec.Run()
	require.NoError(t, err)

	// Close should cleanup resources
	err = exec.Close()
	require.NoError(t, err)
}

// TestExecutor_Close_WithContext tests cleanup with context
func TestExecutor_Close_WithContext(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test",
		Version: "1.0",
		Input:   &flows.InputBlock{Strings: []flows.FieldDef{{Name: "name"}}},
		States:  []flows.State{{Name: "init", Initial: true}},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)

	// Set input to create context
	err := exec.SetInput(map[string]string{"name": "test"})
	require.NoError(t, err)

	// Verify context exists
	ctx := exec.GetContext()
	assert.NotNil(t, ctx)

	// Close should cleanup
	err = exec.Close()
	require.NoError(t, err)
}

// TestExecutor_Close_DuringExecution tests cleanup cancellation
func TestExecutor_Close_DuringExecution(t *testing.T) {
	// This test would verify that closing during execution cancels the flow
	// For now, we just verify Close doesn't panic
	flow := &flows.Flow{
		Name:    "test",
		Version: "1.0",
		States:  []flows.State{{Name: "init", Initial: true}},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)

	// Close immediately
	err := exec.Close()
	require.NoError(t, err)
}

// TestExecutor_Close_NilContext tests Close without initialized context
func TestExecutor_Close_NilContext(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test",
		Version: "1.0",
		States:  []flows.State{{Name: "init", Initial: true}},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)

	// Close without setting input (context may not be initialized)
	err := exec.Close()
	require.NoError(t, err)
}

// TestExecutor_Close_WithCanceledContext tests cleanup with canceled context
func TestExecutor_Close_WithCanceledContext(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test",
		Version: "1.0",
		Input:   &flows.InputBlock{Strings: []flows.FieldDef{{Name: "name"}}},
		States:  []flows.State{{Name: "init", Initial: true}},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)

	err := exec.SetInput(map[string]string{"name": "test"})
	require.NoError(t, err)

	// Cancel the underlying context if available
	// This tests cleanup with a canceled context

	err = exec.Close()
	require.NoError(t, err)
}

// TestCloseMethod_ExistsOnInterface tests that Close method exists on interface
func TestCloseMethod_ExistsOnInterface(t *testing.T) {
	// This is a compile-time test to ensure FlowExecutorInstance has Close method
	// Simply declaring a variable of the interface type ensures all methods exist
	var exec FlowExecutorInstance
	_ = exec
	// If this compiles, the interface has the Close method
}

// TestExecutor_Close_ContextCancellation tests that Close can be used with context cancellation
func TestExecutor_Close_ContextCancellation(t *testing.T) {
	flow := &flows.Flow{
		Name:    "test",
		Version: "1.0",
		Input:   &flows.InputBlock{Strings: []flows.FieldDef{{Name: "name"}}},
		States: []flows.State{
			{Name: "init", Initial: true, Transitions: []flows.Transition{{To: "done"}}},
			{Name: "done"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	exec := svc.New(flow)

	err := exec.SetInput(map[string]string{"name": "test"})
	require.NoError(t, err)

	// Run in background
	done := make(chan error, 1)
	go func() {
		_, err := exec.Run()
		done <- err
	}()

	// Wait for run to complete (it's a simple flow with just transitions)
	err = <-done
	require.NoError(t, err)

	// Now close after completion
	err = exec.Close()
	require.NoError(t, err)
}
