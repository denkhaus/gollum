package builtin

import (
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewBuiltinHooksProvider_WithLangfuse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("registers Langfuse hooks successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockHM := &mockHookManager{}
		mockLog := mocks.NewMockLoggerService(ctrl)
		cfg := &config.LangfuseConfig{
			LangfuseEnabled: true,
		}
		hook := &LangfuseHook{
			log:         mockLog,
			config:      cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[uuid.UUID]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		err := RegisterLangfuseHooks(mockHM, hook)

		assert.NoError(t, err)
		assert.Equal(t, 20, mockHM.registerCount, "All 20 Langfuse hooks should be registered")
	})

	t.Run("skips registration when Langfuse disabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockHM := &mockHookManager{}
		mockLog := mocks.NewMockLoggerService(ctrl)
		cfg := &config.LangfuseConfig{
			LangfuseEnabled: false,
		}
		hook := &LangfuseHook{
			log:         mockLog,
			config:      cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[uuid.UUID]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		err := RegisterLangfuseHooks(mockHM, hook)

		assert.NoError(t, err)
		assert.Equal(t, 0, mockHM.registerCount, "No hooks should be registered when disabled")
	})

	t.Run("verifies hook metadata", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create a more detailed mock to capture metadata
		metadataTracker := &metadataTrackingHookManager{}
		mockLog := mocks.NewMockLoggerService(ctrl)
		cfg := &config.LangfuseConfig{
			LangfuseEnabled: true,
		}
		hook := &LangfuseHook{
			log:         mockLog,
			config:      cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[uuid.UUID]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		err := RegisterLangfuseHooks(metadataTracker, hook)

		assert.NoError(t, err)
		assert.Equal(t, 20, len(metadataTracker.metadata), "Should have 20 registered hooks")

		// Verify some expected hook names
		hookNames := make([]string, 0, len(metadataTracker.metadata))
		for _, meta := range metadataTracker.metadata {
			hookNames = append(hookNames, meta.Name)
		}

		// Check for session hooks
		assert.Contains(t, hookNames, "langfuse-before-session-start")
		assert.Contains(t, hookNames, "langfuse-after-session-end")

		// Check for agent hooks
		assert.Contains(t, hookNames, "langfuse-before-agent-spawn")
		assert.Contains(t, hookNames, "langfuse-after-agent-spawn")

		// Check for LLM hooks
		assert.Contains(t, hookNames, "langfuse-before-llm-request")
		assert.Contains(t, hookNames, "langfuse-after-llm-response")
		assert.Contains(t, hookNames, "langfuse-on-llm-error")

		// Verify all hooks have correct priority
		for _, meta := range metadataTracker.metadata {
			assert.Equal(t, LangfuseHookPriority, meta.Priority, "All hooks should have priority 500")
			assert.False(t, meta.FatalError, "Langfuse hooks should not be fatal")
		}
	})
}

// metadataTrackingHookManager is a mock that tracks all registered hook metadata
type metadataTrackingHookManager struct {
	metadata []hooks.HookMetadata
}

func (m *metadataTrackingHookManager) RegisterHook(fn hooks.HookFunc, meta hooks.HookMetadata) error {
	m.metadata = append(m.metadata, meta)
	return nil
}

// mockHookManager is a minimal mock for testing hook registration
// It only implements RegisterHook since that's all RegisterLangfuseHooks needs
type mockHookManager struct {
	registerCount int
}

func (m *mockHookManager) RegisterHook(fn hooks.HookFunc, meta hooks.HookMetadata) error {
	m.registerCount++
	return nil
}
