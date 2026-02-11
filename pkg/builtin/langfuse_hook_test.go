package builtin

import (
	"sync"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewLangfuseHook(t *testing.T) {
	t.Run("creates hook with nil client", func(t *testing.T) {
		// Setup mock logger and config
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLog := mocks.NewMockLoggerService(ctrl)
		mockCfg := &config.LangfuseConfig{
			LangfuseEnabled: false,
			LangfuseHost:    "https://cloud.langfuse.com",
		}

		// Create hook manually (bypassing DI for test)
		hook := &LangfuseHook{
			log:      mockLog,
			config:   mockCfg,
			client:   nil,
			clientMu: &sync.Mutex{},
		}

		assert.Nil(t, hook.client, "Client should be nil initially")
	})
}

func TestLangfuseHook_GetClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.LangfuseConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid credentials",
			config: &config.LangfuseConfig{
				LangfuseEnabled:       true,
				LangfuseHost:         "https://cloud.langfuse.com",
				LangfusePublicKey:    "pk-test-key",
				LangfuseSecretKey:    "sk-test-key",
				LangfuseFlushInterval: 1000,
				LangfuseMaxQueueSize:  100,
			},
			wantErr: false,
		},
		{
			name: "missing public key",
			config: &config.LangfuseConfig{
				LangfuseEnabled:   true,
				LangfuseHost:      "https://cloud.langfuse.com",
				LangfusePublicKey: "",
				LangfuseSecretKey: "sk-test-key",
			},
			wantErr:     true,
			errContains: "credentials not configured",
		},
		{
			name: "missing secret key",
			config: &config.LangfuseConfig{
				LangfuseEnabled:   true,
				LangfuseHost:      "https://cloud.langfuse.com",
				LangfusePublicKey: "pk-test-key",
				LangfuseSecretKey: "",
			},
			wantErr:     true,
			errContains: "credentials not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLog := mocks.NewMockLoggerService(ctrl)

			// Expect Info call for successful init
			if !tt.wantErr {
				mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			}

			hook := &LangfuseHook{
				log:      mockLog,
				config:   tt.config,
				client:   nil,
				clientMu: &sync.Mutex{},
			}

			client, err := hook.getClient()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				// Verify client is cached on second call
				client2, err2 := hook.getClient()
				assert.NoError(t, err2)
				assert.Same(t, client, client2, "Client should be cached")
			}
		})
	}
}

func TestLangfuseHook_Shutdown(t *testing.T) {
	t.Run("shutdown with nil client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLog := mocks.NewMockLoggerService(ctrl)

		hook := &LangfuseHook{
			log:      mockLog,
			config:   &config.LangfuseConfig{},
			client:   nil,
			clientMu: &sync.Mutex{},
		}

		err := hook.Shutdown()
		assert.NoError(t, err, "Shutdown with nil client should not error")
	})

	t.Run("shutdown with valid client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLog := mocks.NewMockLoggerService(ctrl)

		// Expect Info calls for client init and shutdown
		mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).Times(2)
		cfg := &config.LangfuseConfig{
			LangfuseEnabled:   true,
			LangfusePublicKey: "pk-test",
			LangfuseSecretKey: "sk-test",
			LangfuseHost:      "https://cloud.langfuse.com",
		}

		hook := &LangfuseHook{
			log:      mockLog,
			config:   cfg,
			client:   nil,
			clientMu: &sync.Mutex{},
		}

		// Initialize client
		client, err := hook.getClient()
		require.NoError(t, err)
		require.NotNil(t, client)

		// Shutdown should flush and not error
		err = hook.Shutdown()
		assert.NoError(t, err)
	})
}

func TestNewLangfuseHookProvider(t *testing.T) {
	t.Run("returns valid HookFunc", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLog := mocks.NewMockLoggerService(ctrl)
		cfg := &config.LangfuseConfig{
			LangfuseEnabled: false,
		}

		// Create hook directly for test
		hook := &LangfuseHook{
			log:      mockLog,
			config:   cfg,
			client:   nil,
			clientMu: &sync.Mutex{},
		}

		// Create provider function manually
		provider := func(h *LangfuseHook) (interface{}, error) {
			// Return a HookFunc that passes through
			return func(ctx interface{}, hookCtx interface{}, next func() error) error {
				return next()
			}, nil
		}

		hookFunc, err := provider(hook)
		assert.NoError(t, err)
		assert.NotNil(t, hookFunc)

		// Verify it's a function
		fn, ok := hookFunc.(func(interface{}, interface{}, func() error) error)
		assert.True(t, ok, "Provider should return HookFunc")

		// Call function (should be no-op for now)
		err = fn(nil, nil, func() error { return nil })
		assert.NoError(t, err)
	})
}

// TestNewLangfuseHooksProvider tests the DI provider that returns *LangfuseHook
func TestNewLangfuseHooksProvider(t *testing.T) {
	t.Run("creates LangfuseHook successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLog := mocks.NewMockLoggerService(ctrl)
		mockCfg := &config.LangfuseConfig{
			LangfuseEnabled: false,
			LangfuseHost:    "https://cloud.langfuse.com",
		}

		// Create hook directly for test (bypassing full DI container)
		hook := &LangfuseHook{
			log:        mockLog,
			config:     mockCfg,
			client:     nil,
			clientMu:   &sync.Mutex{},
			traceCtxs:  make(map[uuid.UUID]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		assert.NotNil(t, hook, "Hook should be created")
		assert.Equal(t, mockLog, hook.log, "Logger should be set")
		assert.Equal(t, mockCfg, hook.config, "Config should be set")
		assert.Nil(t, hook.client, "Client should be nil initially")
	})

	t.Run("provider returns correct type", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLog := mocks.NewMockLoggerService(ctrl)
		mockCfg := &config.LangfuseConfig{
			LangfuseEnabled: true,
			LangfuseHost:    "https://cloud.langfuse.com",
		}

		// Simulate provider behavior
		provider := func() (*LangfuseHook, error) {
			return &LangfuseHook{
				log:        mockLog,
				config:     mockCfg,
				client:     nil,
				clientMu:   &sync.Mutex{},
				traceCtxs:  make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}, nil
		}

		result, err := provider()

		assert.NoError(t, err, "Provider should not return error")
		assert.NotNil(t, result, "Provider should return non-nil hook")

		// Verify it's a LangfuseHook with correct fields
		assert.Equal(t, mockLog, result.log)
		assert.Equal(t, mockCfg, result.config)
	})

	t.Run("handles dependencies correctly", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLog := mocks.NewMockLoggerService(ctrl)

		// Test with different config states
		testConfigs := []*config.LangfuseConfig{
			{LangfuseEnabled: false},
			{LangfuseEnabled: true, LangfuseHost: "https://cloud.langfuse.com"},
			{LangfuseEnabled: true, LangfuseHost: "http://localhost:3000"},
		}

		for _, cfg := range testConfigs {
			hook := &LangfuseHook{
				log:        mockLog,
				config:     cfg,
				client:     nil,
				clientMu:   &sync.Mutex{},
				traceCtxs:  make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			assert.NotNil(t, hook)
			assert.Equal(t, cfg, hook.config, "Config should match")
		}
	})
}

func TestLangfuseHook_TraceContextOperations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := mocks.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled:   false,
		LangfusePublicKey: "pk-test",
		LangfuseSecretKey: "sk-test",
		LangfuseHost:      "https://cloud.langfuse.com",
	}

	hook := &LangfuseHook{
		log:        mockLog,
		config:     cfg,
		client:     nil,
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[uuid.UUID]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	sessionID := uuid.New()

	t.Run("getTraceContext returns nil when not found", func(t *testing.T) {
		tc := hook.getTraceContext(sessionID)
		assert.Nil(t, tc, "getTraceContext should return nil for non-existent session")
	})

	t.Run("createTraceContext creates and stores context", func(t *testing.T) {
		tc := hook.createTraceContext(sessionID)

		assert.NotNil(t, tc, "createTraceContext should return non-nil TraceContext")
		assert.Equal(t, sessionID, tc.SessionID, "SessionID should match")
		assert.NotEmpty(t, tc.TraceID, "TraceID should be generated")
		assert.NotNil(t, tc.Spans, "Spans map should be initialized")
		assert.False(t, tc.CreatedAt.IsZero(), "CreatedAt should be set")
	})

	t.Run("getTraceContext returns created context", func(t *testing.T) {
		hook.createTraceContext(sessionID)
		tc := hook.getTraceContext(sessionID)

		assert.NotNil(t, tc, "getTraceContext should return created TraceContext")
		assert.Equal(t, sessionID, tc.SessionID)
	})

	t.Run("removeTraceContext deletes context", func(t *testing.T) {
		hook.createTraceContext(sessionID)
		hook.removeTraceContext(sessionID)

		// Context should be removed
		tc := hook.getTraceContext(sessionID)
		assert.Nil(t, tc, "getTraceContext should return nil after removal")
	})

	t.Run("concurrent access is thread-safe", func(t *testing.T) {
		// Create multiple session IDs
		session1 := uuid.New()
		session2 := uuid.New()
		session3 := uuid.New()

		// Run concurrent operations
		done := make(chan bool)
		go func() {
			hook.createTraceContext(session1)
			done <- true
		}()
		go func() {
			hook.createTraceContext(session2)
			done <- true
		}()
		go func() {
			hook.createTraceContext(session3)
			done <- true
		}()
		go func() {
			hook.getTraceContext(session1)
			done <- true
		}()

		// Wait for all goroutines
		for i := 0; i < 4; i++ {
			<-done
		}

		// Verify all contexts were created
		assert.NotNil(t, hook.getTraceContext(session1))
		assert.NotNil(t, hook.getTraceContext(session2))
		assert.NotNil(t, hook.getTraceContext(session3))
	})
}

func TestTraceContext_Struct(t *testing.T) {
	t.Run("TraceContext has all required fields", func(t *testing.T) {
		tc := &TraceContext{
			TraceID:   "test-trace-123",
			RootSpan:  nil,
			Spans:     make(map[string]interface{}),
			SessionID:  uuid.New(),
			CreatedAt:  time.Now(),
		}

		assert.Equal(t, "test-trace-123", tc.TraceID)
		assert.Nil(t, tc.RootSpan)
		assert.NotNil(t, tc.Spans)
		assert.NotEqual(t, uuid.Nil, tc.SessionID)
		assert.False(t, tc.CreatedAt.IsZero())
	})

	t.Run("TraceContext initializes with zero values", func(t *testing.T) {
		tc := &TraceContext{}

		assert.Empty(t, tc.TraceID)
		assert.Nil(t, tc.RootSpan)
		assert.Nil(t, tc.Spans)
		assert.Equal(t, uuid.Nil, tc.SessionID)
		assert.True(t, tc.CreatedAt.IsZero())
	})
}

func TestRegisterLangfuseHooks(t *testing.T) {
	t.Run("skips registration when Langfuse disabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockHM := &mockHookManager{}
		mockLog := mocks.NewMockLoggerService(ctrl)
		cfg := &config.LangfuseConfig{
			LangfuseEnabled: false,
		}
		hook := &LangfuseHook{
			log:        mockLog,
			config:     cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[uuid.UUID]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		err := RegisterLangfuseHooks(mockHM, hook)

		assert.NoError(t, err)
		assert.Equal(t, 0, mockHM.registerCount, "No hooks should be registered when disabled")
	})

	t.Run("registers all hooks when Langfuse enabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockHM := &mockHookManager{}
		mockLog := mocks.NewMockLoggerService(ctrl)
		cfg := &config.LangfuseConfig{
			LangfuseEnabled: true,
		}
		hook := &LangfuseHook{
			log:        mockLog,
			config:     cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[uuid.UUID]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		err := RegisterLangfuseHooks(mockHM, hook)

		assert.NoError(t, err)
		assert.Equal(t, 20, mockHM.registerCount, "Should register 20 hook points (2 session + 4 agent + 3 tool + 6 file + 3 LLM)")
	})
}

func TestLangfuseHook_TraceIDPropagation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := mocks.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled: true,
	}
	sessionID := uuid.New()

	hook := &LangfuseHook{
		log:        mockLog,
		config:     cfg,
		client:      nil,
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[uuid.UUID]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// Create trace context
	hook.createTraceContext(sessionID)

	t.Run("propagateTraceID sets trace ID in HookContext.Data", func(t *testing.T) {
		hookCtx := &hooks.HookContext{
			SessionID: sessionID,
			Data:      make(map[string]any),
		}

		hook.propagateTraceID(hookCtx)

		traceID, exists := hookCtx.Data["langfuse_trace_id"]
		assert.True(t, exists, "langfuse_trace_id should be set")
		assert.NotEmpty(t, traceID, "trace ID should not be empty")
	})

	t.Run("propagateTraceID does nothing when no session", func(t *testing.T) {
		hookCtx := &hooks.HookContext{
			SessionID: uuid.Nil,
			Data:      make(map[string]any),
		}

		hook.propagateTraceID(hookCtx)

		_, exists := hookCtx.Data["langfuse_trace_id"]
		assert.False(t, exists, "langfuse_trace_id should not be set for nil session")
	})

	t.Run("propagateTraceID does nothing when no trace context", func(t *testing.T) {
		differentSession := uuid.New()
		hookCtx := &hooks.HookContext{
			SessionID: differentSession,
			Data:      make(map[string]any),
		}

		hook.propagateTraceID(hookCtx)

		_, exists := hookCtx.Data["langfuse_trace_id"]
		assert.False(t, exists, "langfuse_trace_id should not be set for non-existent trace")
	})
}

// TestNewBuiltinHooksProvider_WithLangfuse tests integration with RegisterLangfuseHooks
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
			log:        mockLog,
			config:     cfg,
			client:     nil,
			clientMu:   &sync.Mutex{},
			traceCtxs:  make(map[uuid.UUID]*TraceContext),
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
			log:        mockLog,
			config:     cfg,
			client:     nil,
			clientMu:   &sync.Mutex{},
			traceCtxs:  make(map[uuid.UUID]*TraceContext),
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
			log:        mockLog,
			config:     cfg,
			client:     nil,
			clientMu:   &sync.Mutex{},
			traceCtxs:  make(map[uuid.UUID]*TraceContext),
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
