package hooks

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockLogger is a simple mock logger for testing
type mockLogger struct {
	debugCalls [][]zap.Field
	infoCalls  [][]zap.Field
	warnCalls  [][]zap.Field
	errorCalls [][]zap.Field
}

func (m *mockLogger) Debug(msg string, fields ...zap.Field) {
	m.debugCalls = append(m.debugCalls, fields)
}

func (m *mockLogger) Debugf(format string, args ...any) {}

func (m *mockLogger) Info(msg string, fields ...zap.Field) {
	m.infoCalls = append(m.infoCalls, fields)
}

func (m *mockLogger) Infof(format string, args ...any) {}

func (m *mockLogger) Warn(msg string, fields ...zap.Field) {
	m.warnCalls = append(m.warnCalls, fields)
}

func (m *mockLogger) Warnf(format string, args ...any) {}

func (m *mockLogger) Error(msg string, fields ...zap.Field) {
	m.errorCalls = append(m.errorCalls, fields)
}

func (m *mockLogger) Errorf(format string, args ...any) {}

func (m *mockLogger) GetLogger() *zap.Logger { return nil }
func (m *mockLogger) SetTUIWriter(writer io.Writer) {}
func (m *mockLogger) ResetToStdout() {}
func (m *mockLogger) GetLogs(filter logger.LogFilter) []logger.LogEntry { return nil }
func (m *mockLogger) GetLogStats() map[string]interface{} { return nil }

var _ logger.LoggerService = (*mockLogger)(nil)

// newTestHookManager creates a HookManager for testing
func newTestHookManager() HookManager {
	log := &mockLogger{}
	return &hookManagerImpl{
		log:        log,
		registries: make(map[HookPoint]*hookRegistry),
	}
}

// TestHookManager_RegisterHook tests hook registration
func TestHookManager_RegisterHook(t *testing.T) {
	t.Run("successfully registers a hook", func(t *testing.T) {
		hm := newTestHookManager()

		// Initialize registries (normally done in NewHookManager)
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		fn := func(ctx context.Context, hc *HookContext, next func() error) error {
			return next()
		}
		meta := HookMetadata{
			Name:       "test-hook",
			Point:      BeforeSessionStart,
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(fn, meta)
		require.NoError(t, err)
	})

	t.Run("rejects nil hook function", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		meta := HookMetadata{
			Name:       "test-hook",
			Point:      BeforeSessionStart,
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(nil, meta)
		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects empty hook name", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		fn := func(ctx context.Context, hc *HookContext, next func() error) error {
			return next()
		}
		meta := HookMetadata{
			Name:       "",
			Point:      BeforeSessionStart,
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(fn, meta)
		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects unknown hook point", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		fn := func(ctx context.Context, hc *HookContext, next func() error) error {
			return next()
		}
		meta := HookMetadata{
			Name:       "test-hook",
			Point:      HookPoint("UnknownPoint"),
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(fn, meta)
		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects duplicate hook name", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		fn := func(ctx context.Context, hc *HookContext, next func() error) error {
			return next()
		}
		meta := HookMetadata{
			Name:       "test-hook",
			Point:      BeforeSessionStart,
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(fn, meta)
		require.NoError(t, err)

		err = hm.RegisterHook(fn, meta)
		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeConflict))
	})
}

// TestHookManager_UnregisterHook tests hook unregistration
func TestHookManager_UnregisterHook(t *testing.T) {
	t.Run("successfully unregisters a hook", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		fn := func(ctx context.Context, hc *HookContext, next func() error) error {
			return next()
		}
		meta := HookMetadata{
			Name:       "test-hook",
			Point:      BeforeSessionStart,
			Priority:   0,
			FatalError: false,
		}

		err := hm.RegisterHook(fn, meta)
		require.NoError(t, err)

		unregistered := hm.UnregisterHook("test-hook")
		assert.True(t, unregistered)

		// Should not be able to unregister again
		unregistered = hm.UnregisterHook("test-hook")
		assert.False(t, unregistered)
	})

	t.Run("returns false for non-existent hook", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		unregistered := hm.UnregisterHook("non-existent")
		assert.False(t, unregistered)
	})
}

// TestHookManager_TriggerHooks_PriorityOrdering tests priority-based execution
func TestHookManager_TriggerHooks_PriorityOrdering(t *testing.T) {
	t.Run("executes hooks in priority order (lowest first)", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		executed := []string{}
		fn1 := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "priority-0")
			return next()
		}
		fn2 := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "priority-10")
			return next()
		}
		fn3 := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "priority-5")
			return next()
		}

		// Register in reverse priority order
		hm.RegisterHook(fn1, HookMetadata{Name: "h1", Point: BeforeSessionStart, Priority: 10, FatalError: false})
		hm.RegisterHook(fn2, HookMetadata{Name: "h2", Point: BeforeSessionStart, Priority: 0, FatalError: false})
		hm.RegisterHook(fn3, HookMetadata{Name: "h3", Point: BeforeSessionStart, Priority: 5, FatalError: false})

		result := hm.TriggerHooks(context.Background(), BeforeSessionStart, &HookContext{})

		assert.False(t, result.Stopped)
		assert.NoError(t, result.Error)
		assert.Equal(t, []string{"priority-10", "priority-5", "priority-0"}, executed)
	})
}

// TestHookManager_TriggerHooks_ErrorHandling tests error handling
func TestHookManager_TriggerHooks_ErrorHandling(t *testing.T) {
	t.Run("fatal error stops execution", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		executed := []string{}
		fn1 := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "hook1")
			return next()
		}
		fn2 := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "hook2")
			return errors.New("fatal error")
		}
		fn3 := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "hook3")
			return next()
		}

		hm.RegisterHook(fn1, HookMetadata{Name: "h1", Point: BeforeSessionStart, Priority: 0, FatalError: false})
		hm.RegisterHook(fn2, HookMetadata{Name: "h2", Point: BeforeSessionStart, Priority: 1, FatalError: true})
		hm.RegisterHook(fn3, HookMetadata{Name: "h3", Point: BeforeSessionStart, Priority: 2, FatalError: false})

		result := hm.TriggerHooks(context.Background(), BeforeSessionStart, &HookContext{})

		assert.True(t, result.Stopped)
		assert.Error(t, result.Error)
		assert.Equal(t, []string{"hook1", "hook2"}, executed)
	})

	t.Run("non-fatal error continues execution", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		executed := []string{}
		fn1 := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "hook1")
			return next()
		}
		fn2 := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "hook2")
			return errors.New("non-fatal error")
		}
		fn3 := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "hook3")
			return next()
		}

		hm.RegisterHook(fn1, HookMetadata{Name: "h1", Point: BeforeSessionStart, Priority: 0, FatalError: false})
		hm.RegisterHook(fn2, HookMetadata{Name: "h2", Point: BeforeSessionStart, Priority: 1, FatalError: false})
		hm.RegisterHook(fn3, HookMetadata{Name: "h3", Point: BeforeSessionStart, Priority: 2, FatalError: false})

		result := hm.TriggerHooks(context.Background(), BeforeSessionStart, &HookContext{})

		assert.False(t, result.Stopped)
		assert.NoError(t, result.Error)
		assert.Equal(t, []string{"hook1", "hook2", "hook3"}, executed)
	})
}

// TestHookManager_WithSessionHooks tests session hook wrapping
func TestHookManager_WithSessionHooks(t *testing.T) {
	t.Run("successfully wraps work with session hooks", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		executed := []string{}
		beforeHook := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "before")
			return next()
		}
		afterHook := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = append(executed, "after")
			return next()
		}

		hm.RegisterHook(beforeHook, HookMetadata{Name: "before", Point: BeforeSessionStart, Priority: 0, FatalError: false})
		hm.RegisterHook(afterHook, HookMetadata{Name: "after", Point: AfterSessionEnd, Priority: 0, FatalError: false})

		sessionID := uuid.New()
		workExecuted := false
		err := hm.(*hookManagerImpl).WithSessionHooks(context.Background(), sessionID, func() error {
			executed = append(executed, "work")
			workExecuted = true
			return nil
		})

		require.NoError(t, err)
		assert.True(t, workExecuted)
		assert.Equal(t, []string{"before", "work", "after"}, executed)
	})

	t.Run("rejects nil session ID", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		err := hm.(*hookManagerImpl).WithSessionHooks(context.Background(), uuid.Nil, func() error {
			return nil
		})

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})
}

// TestHookManager_WithAgentHooks tests agent hook wrapping
func TestHookManager_WithAgentHooks(t *testing.T) {
	t.Run("successfully triggers agent hooks", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		executed := false
		agentHook := func(ctx context.Context, hc *HookContext, next func() error) error {
			executed = true
			return next()
		}

		hm.RegisterHook(agentHook, HookMetadata{Name: "agent-hook", Point: BeforeAgentSpawn, Priority: 0, FatalError: false})

		sessionID := uuid.New()
		agentID := uuid.New()
		err := hm.(*hookManagerImpl).WithAgentHooks(context.Background(), sessionID, agentID, BeforeAgentSpawn, func() error {
			return nil
		})

		require.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("rejects nil agent ID", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		err := hm.(*hookManagerImpl).WithAgentHooks(context.Background(), uuid.New(), uuid.Nil, BeforeAgentSpawn, nil)

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})

	t.Run("rejects invalid agent hook point", func(t *testing.T) {
		hm := newTestHookManager()
		for _, point := range []HookPoint{
			BeforeSessionStart, AfterSessionEnd,
			BeforeAgentSpawn, AfterAgentSpawn,
			BeforeAgentRemove, AfterAgentRemove,
		} {
			hm.(*hookManagerImpl).registries[point] = &hookRegistry{}
		}

		sessionID := uuid.New()
		agentID := uuid.New()
		err := hm.(*hookManagerImpl).WithAgentHooks(context.Background(), sessionID, agentID, BeforeSessionStart, nil)

		require.Error(t, err)
		assert.True(t, errs.IsType(err, errs.TypeValidation))
	})
}

// TestHookContext_Clone tests HookContext cloning
func TestHookContext_Clone(t *testing.T) {
	t.Run("creates a shallow copy of HookContext", func(t *testing.T) {
		original := &HookContext{
			SessionID: uuid.New(),
			AgentID:   uuid.New(),
			ToolName:  "test-tool",
			Data:      map[string]any{"key1": "value1", "key2": 42},
		}

		cloned := original.Clone()

		assert.Equal(t, original.SessionID, cloned.SessionID)
		assert.Equal(t, original.AgentID, cloned.AgentID)
		assert.Equal(t, original.ToolName, cloned.ToolName)
		assert.Equal(t, original.Data, cloned.Data)

		// Modify cloned data
		cloned.Data["key1"] = "modified"
		assert.Equal(t, "value1", original.Data["key1"])
		assert.Equal(t, "modified", cloned.Data["key1"])
	})

	t.Run("handles nil HookContext", func(t *testing.T) {
		cloned := (*HookContext)(nil).Clone()
		assert.NotNil(t, cloned)
		assert.NotNil(t, cloned.Data)
	})
}

// TestHookPoint_String tests HookPoint string representation
func TestHookPoint_String(t *testing.T) {
	tests := []struct {
		name     string
		point    HookPoint
		expected string
	}{
		{"BeforeSessionStart", BeforeSessionStart, "BeforeSessionStart"},
		{"AfterSessionEnd", AfterSessionEnd, "AfterSessionEnd"},
		{"BeforeAgentSpawn", BeforeAgentSpawn, "BeforeAgentSpawn"},
		{"AfterAgentSpawn", AfterAgentSpawn, "AfterAgentSpawn"},
		{"BeforeAgentRemove", BeforeAgentRemove, "BeforeAgentRemove"},
		{"AfterAgentRemove", AfterAgentRemove, "AfterAgentRemove"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.point.String())
		})
	}
}
