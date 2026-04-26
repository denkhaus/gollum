package builtin

import (
	"github.com/denkhaus/gollum/pkg/logger"
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
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
		mockHM := hooks.NewMockHookManager(ctrl)
		mockLog := logger.NewMockLoggerService(ctrl)
		cfg := &config.LangfuseConfig{
			LangfuseEnabled: true,
		}
		hook := &LangfuseHook{
			log:         mockLog,
			config:      cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[string]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		// Expect all typed hook registrations (20 total)
		mockHM.EXPECT().RegisterSessionHook(gomock.Any(), gomock.Any()).Times(2)
		mockHM.EXPECT().RegisterAgentHook(gomock.Any(), gomock.Any()).Times(4)
		mockHM.EXPECT().RegisterToolHook(gomock.Any(), gomock.Any()).Times(3)
		mockHM.EXPECT().RegisterFileHook(gomock.Any(), gomock.Any()).Times(8)
		mockHM.EXPECT().RegisterLLMHook(gomock.Any(), gomock.Any()).Times(3)

		err := RegisterLangfuseHooks(mockHM, hook)

		assert.NoError(t, err)
	})

	t.Run("skips registration when Langfuse disabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockHM := hooks.NewMockHookManager(ctrl)
		mockLog := logger.NewMockLoggerService(ctrl)
		cfg := &config.LangfuseConfig{
			LangfuseEnabled: false,
		}
		hook := &LangfuseHook{
			log:         mockLog,
			config:      cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[string]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		// No expectations = no calls should be made

		err := RegisterLangfuseHooks(mockHM, hook)

		assert.NoError(t, err)
	})

	t.Run("verifies hook metadata", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Track captured metadata
		var capturedMetadata []hooks.TypedHookMetadata

		mockHM := hooks.NewMockHookManager(ctrl)
		mockLog := logger.NewMockLoggerService(ctrl)
		cfg := &config.LangfuseConfig{
			LangfuseEnabled: true,
		}
		hook := &LangfuseHook{
			log:         mockLog,
			config:      cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[string]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		// Capture metadata from all registrations
		captureMeta := func(_ any, meta any) {
			capturedMetadata = append(capturedMetadata, meta.(hooks.TypedHookMetadata))
		}

		mockHM.EXPECT().RegisterSessionHook(gomock.Any(), gomock.Any()).Do(captureMeta).Times(2)
		mockHM.EXPECT().RegisterAgentHook(gomock.Any(), gomock.Any()).Do(captureMeta).Times(4)
		mockHM.EXPECT().RegisterToolHook(gomock.Any(), gomock.Any()).Do(captureMeta).Times(3)
		mockHM.EXPECT().RegisterFileHook(gomock.Any(), gomock.Any()).Do(captureMeta).Times(8)
		mockHM.EXPECT().RegisterLLMHook(gomock.Any(), gomock.Any()).Do(captureMeta).Times(3)

		err := RegisterLangfuseHooks(mockHM, hook)

		assert.NoError(t, err)
		assert.Equal(t, 20, len(capturedMetadata), "Should have 20 registered hooks")

		// Verify some expected hook names
		hookNames := make([]string, 0, len(capturedMetadata))
		for _, meta := range capturedMetadata {
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
		for _, meta := range capturedMetadata {
			assert.Equal(t, LangfuseHookPriority, meta.Priority, "All hooks should have priority 500")
			assert.False(t, meta.FatalError, "Langfuse hooks should not be fatal")
		}
	})
}
