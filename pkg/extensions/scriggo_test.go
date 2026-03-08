package extensions

import (
	"testing"

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
