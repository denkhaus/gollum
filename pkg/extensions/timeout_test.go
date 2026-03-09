package extensions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriggoRunner_ExecuteFunc_WithTimeout(t *testing.T) {
	injector := do.New()
	gateway, err := NewGatewayService(injector)
	require.NoError(t, err)
	do.ProvideValue(injector, gateway)

	runner, err := NewScriggoRunner(injector)
	require.NoError(t, err)

	runnerImpl, ok := runner.(*scriggoRunnerImpl)
	require.True(t, ok, "Runner should be *scriggoRunnerImpl")

	// Load a function that runs in an infinite loop
	infiniteLoopSource := `
package main

func main() {
	i := 0
	for {
		i = i + 1
	}
}
`
	err = runner.LoadFunc("infinite_loop", infiniteLoopSource)
	require.NoError(t, err)

	// Execute with timeout - should be cancelled after 50ms
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = runnerImpl.ExecuteFuncWithContext(ctx, "infinite_loop", nil)
	elapsed := time.Since(start)

	// Verify cancellation
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
		"Error should be context error, got: %v", err)

	// Should return quickly (< 150ms)
	assert.Less(t, elapsed.Milliseconds(), int64(150),
		"Execution should have been cancelled quickly, took %v", elapsed)
}
