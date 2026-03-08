package extensions

import (
	"errors"
	"testing"
	"time"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHookRegistry_RegisterAndExecute(t *testing.T) {
	injector := do.New()
	registry, err := NewHookRegistry(injector)
	require.NoError(t, err)

	executed := false
	hookFn := func(_ *HookContext) error {
		executed = true
		return nil
	}

	err = registry.Register(HookAgentPreExecute, "test_hook", hookFn)
	require.NoError(t, err)

	hookCtx := &HookContext{
		Type:      HookAgentPreExecute,
		AgentID:   "test-agent",
		Timestamp: mockTime(),
		Metadata:  make(map[string]any),
	}

	err = registry.Execute(HookAgentPreExecute, hookCtx)
	require.NoError(t, err)
	assert.True(t, executed, "hook should have been executed")
}

func TestHookRegistry_Unregister(t *testing.T) {
	injector := do.New()
	registry, err := NewHookRegistry(injector)
	require.NoError(t, err)

	hookFn := func(_ *HookContext) error {
		return nil
	}

	// Register
	err = registry.Register(HookToolPreExecute, "hook1", hookFn)
	require.NoError(t, err)

	// Unregister
	err = registry.Unregister(HookToolPreExecute, "hook1")
	require.NoError(t, err)

	// Verify hook is no longer executed
	hookCtx := &HookContext{
		Type: HookToolPreExecute,
	}
	err = registry.Execute(HookToolPreExecute, hookCtx)
	require.NoError(t, err, "no hooks should remain")
}

func TestHookRegistry_HookErrorPropagation(t *testing.T) {
	injector := do.New()
	registry, err := NewHookRegistry(injector)
	require.NoError(t, err)

	expectedErr := errors.New("hook failed")
	hookFn := func(_ *HookContext) error {
		return expectedErr
	}

	err = registry.Register(HookAgentPostExecute, "failing_hook", hookFn)
	require.NoError(t, err)

	hookCtx := &HookContext{
		Type: HookAgentPostExecute,
	}
	err = registry.Execute(HookAgentPostExecute, hookCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failing_hook failed")
}

func TestHookRegistry_RegisterNilFunction(t *testing.T) {
	injector := do.New()
	registry, err := NewHookRegistry(injector)
	require.NoError(t, err)

	err = registry.Register(HookAgentPreExecute, "nil_hook", nil)
	assert.ErrorIs(t, err, ErrNilHookFunction)
}

func mockTime() time.Time {
	return time.Date(2025, 3, 8, 12, 0, 0, 0, time.UTC)
}
