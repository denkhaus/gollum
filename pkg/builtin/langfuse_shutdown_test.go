package builtin

import (

	"github.com/denkhaus/gollum/pkg/logger"
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

)

func TestLangfuseHook_Shutdown_UnitTests(t *testing.T) {
	tests := []struct {
		name      string
		hasClient bool
		hasTraces bool
	}{
		{
			name:      "shutdown with client and traces",
			hasClient: true,
			hasTraces: true,
		},
		{
			name:      "shutdown with client no traces",
			hasClient: true,
			hasTraces: false,
		},
		{
			name:      "shutdown without client",
			hasClient: false,
			hasTraces: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLog := logger.NewMockLoggerService(ctrl)
			mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
			mockLog.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

			cfg := &config.LangfuseConfig{
				LangfuseEnabled:   tt.hasClient,
				LangfuseHost:      "https://cloud.langfuse.com",
				LangfusePublicKey: "pk-test-key",
				LangfuseSecretKey: "sk-test-key",
			}

			hook := &LangfuseHook{
				log:         mockLog,
				config:      cfg,
				client:      nil, // No real client for unit tests
				clientMu:    &sync.Mutex{},
				traceCtxs:   make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			// Add trace contexts if needed
			if tt.hasTraces {
				for i := 0; i < 2; i++ {
					hook.createTraceContext(uuid.New())
				}
			}

			// Shutdown should not error
			err := hook.Shutdown()
			require.NoError(t, err, "Shutdown should not return error")

			// Verify all contexts cleaned
			hook.traceCtxsMu.RLock()
			assert.Equal(t, 0, len(hook.traceCtxs), "All trace contexts should be cleaned up")
			hook.traceCtxsMu.RUnlock()
		})
	}
}

// TestLangfuseHook_CleanupAllTraceContexts tests trace context cleanup
func TestLangfuseHook_CleanupAllTraceContexts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := logger.NewMockLoggerService(ctrl)
	mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	cfg := &config.LangfuseConfig{
		LangfuseEnabled: true,
		LangfuseHost:    "https://cloud.langfuse.com",
	}

	hook := &LangfuseHook{
		log:         mockLog,
		config:      cfg,
		client:      nil,
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[uuid.UUID]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// Create multiple trace contexts
	sessionIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for _, sid := range sessionIDs {
		hook.createTraceContext(sid)
	}

	// Verify contexts exist
	hook.traceCtxsMu.RLock()
	assert.Equal(t, 3, len(hook.traceCtxs))
	hook.traceCtxsMu.RUnlock()

	// Cleanup all
	hook.cleanupAllTraceContexts()

	// Verify all removed
	hook.traceCtxsMu.RLock()
	assert.Equal(t, 0, len(hook.traceCtxs), "All contexts should be removed")
	for _, sid := range sessionIDs {
		_, exists := hook.traceCtxs[sid]
		assert.False(t, exists, "Context should not exist for session %s", sid)
	}
	hook.traceCtxsMu.RUnlock()
}

// TestLangfuseHook_FlushTraces tests trace flushing
func TestLangfuseHook_FlushTraces(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := logger.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled: true,
		LangfuseHost:    "https://cloud.langfuse.com",
	}

	hook := &LangfuseHook{
		log:         mockLog,
		config:      cfg,
		client:      nil, // No client - tests no-op behavior
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[uuid.UUID]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// flushTraces should not error even without client
	err := hook.flushTraces()
	require.NoError(t, err, "flushTraces should not error without client")
}

// TestLangfuseHook_ShutdownWithOrphanedTraces tests Shutdown cleanup of orphaned traces
func TestLangfuseHook_ShutdownWithOrphanedTraces(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := logger.NewMockLoggerService(ctrl)
	mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	cfg := &config.LangfuseConfig{
		LangfuseEnabled: true,
		LangfuseHost:    "https://cloud.langfuse.com",
	}

	hook := &LangfuseHook{
		log:         mockLog,
		config:      cfg,
		client:      nil,
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[uuid.UUID]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// Create some orphaned trace contexts (session started but not ended)
	for i := 0; i < 3; i++ {
		sessionID := uuid.New()
		hook.createTraceContext(sessionID)
	}

	// Verify contexts exist
	hook.traceCtxsMu.RLock()
	assert.Equal(t, 3, len(hook.traceCtxs), "Should have 3 trace contexts")
	hook.traceCtxsMu.RUnlock()

	// Shutdown should clean them up
	err := hook.Shutdown()
	require.NoError(t, err)

	// Verify all contexts cleaned up
	hook.traceCtxsMu.RLock()
	assert.Equal(t, 0, len(hook.traceCtxs), "Should have 0 trace contexts after shutdown")
	hook.traceCtxsMu.RUnlock()
}
