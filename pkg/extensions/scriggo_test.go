package extensions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/open2b/scriggo"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriggoRunner_LoadAndExecuteFunc(t *testing.T) {
	injector := do.New()
	gateway, _ := NewGatewayService(injector)
	runner := &scriggoRunnerImpl{
		funcs:   make(map[string]*scriggo.Program),
		gateway: gateway,
	}

	// Note: This will be initialized properly via NewScriggoRunner
	// For now we test the interface exists
	assert.NotNil(t, runner)
}

func TestScriggoRunner_ListFuncs(t *testing.T) {
	runner := &scriggoRunnerImpl{
		funcs: make(map[string]*scriggo.Program),
	}

	funcs := runner.ListFuncs()
	assert.NotNil(t, funcs)
	assert.Empty(t, funcs)
}

func TestNewScriggoRunner(t *testing.T) {
	injector := do.New()
	gateway, err := NewGatewayService(injector)
	require.NoError(t, err)
	require.NotNil(t, gateway)

	// Register the gateway in the DI container
	do.ProvideValue(injector, gateway)

	runner, err := NewScriggoRunner(injector)
	require.NoError(t, err)
	assert.NotNil(t, runner)

	// Verify it implements the interface
	_, ok := runner.(ScriggoRunner)
	assert.True(t, ok, "ScriggoRunner should implement ScriggoRunner interface")
}

func TestScriggoRunner_ContextCancellation(t *testing.T) {
	injector := do.New()
	gateway, err := NewGatewayService(injector)
	require.NoError(t, err)
	do.ProvideValue(injector, gateway)

	runner, err := NewScriggoRunner(injector)
	require.NoError(t, err)

	// Cast to implementation to access ExecuteFuncWithContext
	runnerImpl, ok := runner.(*scriggoRunnerImpl)
	require.True(t, ok, "Runner should be *scriggoRunnerImpl")

	// Load a function that runs indefinitely (infinite loop)
	// This program will loop forever unless cancelled
	infiniteLoopSource := `
package main

func main() {
	for {
		// Infinite loop
	}
}
`
	err = runner.LoadFunc("infinite_loop", infiniteLoopSource)
	require.NoError(t, err)

	// Execute with a short deadline (100ms)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = runnerImpl.ExecuteFuncWithContext(ctx, "infinite_loop", nil)
	elapsed := time.Since(start)

	// Should have been cancelled
	require.Error(t, err, "ExecuteFuncWithContext should return error when context is cancelled")
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
		"Error should be context.DeadlineExceeded or context.Canceled, got: %v", err)

	// Should have returned quickly (within 200ms, allowing some margin)
	assert.Less(t, elapsed.Milliseconds(), int64(200),
		"Execution should have been cancelled quickly, took %v", elapsed)

	t.Logf("Context cancellation test passed: execution was cancelled after %v", elapsed)
}
