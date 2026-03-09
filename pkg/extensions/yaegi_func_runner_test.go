package extensions

import (
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/require"
)

func TestYaegiFuncRunner_ExecuteArbitraryFunction_DynamicExecution(t *testing.T) {
	// This test proves DYNAMIC function execution
	// Using a NEW function (not Double) to avoid hardcoded wrappers
	injector := do.New()

	// Register GatewayService (dependency of YaegiFuncRunner)
	do.Provide(injector, NewGatewayService)

	// Create YaegiFuncRunner
	runner, err := NewYaegiFuncRunner(injector)
	require.NoError(t, err)

	// Load a NEW function that was never hardcoded
	source := `package math

func Add(a int, b int) int {
	return a + b
}
`

	err = runner.LoadFunc("add", source)
	require.NoError(t, err)

	// Execute with parameters
	result, err := runner.ExecuteFunc("math.Add", map[string]any{
		"a": 10,
		"b": 32,
	})
	require.NoError(t, err)
	require.Equal(t, 42, result, "10 + 32 should equal 42")
}
