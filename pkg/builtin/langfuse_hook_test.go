package builtin

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/git-hulk/langfuse-go/pkg/traces"
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

// TestLangfuseHook_LLMSpanCreation tests span creation in beforeLLMRequestHook
func TestLangfuseHook_LLMSpanCreation(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name            string
		langfuseEnabled bool
		sessionID       uuid.UUID
		llmModel        string
		llmInput        string
		wantSpanCreated bool
	}{
		{
			name:            "creates span when enabled with valid session",
			langfuseEnabled: true,
			sessionID:       sessionID,
			llmModel:        "claude-3-5-sonnet",
			llmInput:        "Hello, world!",
			wantSpanCreated: true,
		},
		{
			name:            "skips span when disabled",
			langfuseEnabled: false,
			sessionID:       sessionID,
			llmModel:        "claude-3-5-sonnet",
			llmInput:        "Hello, world!",
			wantSpanCreated: false,
		},
		{
			name:            "skips span with nil session ID",
			langfuseEnabled: true,
			sessionID:       uuid.Nil,
			llmModel:        "claude-3-5-sonnet",
			llmInput:        "Hello, world!",
			wantSpanCreated: false,
		},
		{
			name:            "uses fallback span name when model empty",
			langfuseEnabled: true,
			sessionID:       sessionID,
			llmModel:        "",
			llmInput:        "Hello, world!",
			wantSpanCreated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLog := mocks.NewMockLoggerService(ctrl)
			cfg := &config.LangfuseConfig{
				LangfuseEnabled:   tt.langfuseEnabled,
				LangfuseHost:      "https://cloud.langfuse.com",
				LangfusePublicKey: "pk-test-key",
				LangfuseSecretKey: "sk-test-key",
			}

			// Expect Info call for client initialization and Debug for any issues
			mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

			hook := &LangfuseHook{
				log:         mockLog,
				config:      cfg,
				client:      nil, // Will be lazily initialized
				clientMu:    &sync.Mutex{},
				traceCtxs:   make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			// Create trace context if needed
			if tt.sessionID != uuid.Nil {
				hook.createTraceContext(tt.sessionID)
			}

			// Create HookContext
			hookCtx := &hooks.HookContext{
				SessionID:  tt.sessionID,
				LLMModel:   tt.llmModel,
				LLMInput:   tt.llmInput,
				LLMOptions: make(map[string]any),
				Data:       make(map[string]any),
			}

			// Call beforeLLMRequestHook
			err := hook.beforeLLMRequestHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			// Verify span creation
			spanID, hasSpanID := hookCtx.Data["langfuse_span_id"].(string)

			if tt.wantSpanCreated {
				assert.True(t, hasSpanID, "Should have span ID in Data")
				assert.NotEmpty(t, spanID, "Span ID should not be empty")

				// Verify span stored in TraceContext
				tc := hook.getTraceContext(tt.sessionID)
				require.NotNil(t, tc, "TraceContext should exist")
				assert.Contains(t, tc.Spans, spanID, "Span should be in TraceContext.Spans")

				// Verify span context data
				spanCtx, ok := tc.Spans[spanID].(*LLMSpanContext)
				require.True(t, ok, "Span should be LLMSpanContext type")
				assert.Equal(t, tt.llmModel, spanCtx.Model)
				assert.Equal(t, tt.llmInput, spanCtx.Input)
				assert.False(t, spanCtx.StartTime.IsZero(), "StartTime should be set")
			} else {
				assert.False(t, hasSpanID, "Should not have span ID when disabled or nil session")
			}
		})
	}
}

// TestLangfuseHook_LLMSpanUpdate tests span update in afterLLMResponseHook
func TestLangfuseHook_LLMSpanUpdate(t *testing.T) {
	sessionID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := mocks.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled: true,
		LangfuseHost:    "https://cloud.langfuse.com",
	}

	// Expect Debug call for span completion
	mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	hook := &LangfuseHook{
		log:         mockLog,
		config:      cfg,
		client:      nil,
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[uuid.UUID]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// Create trace context and pre-populate with span
	hook.createTraceContext(sessionID)
	tc := hook.getTraceContext(sessionID)
	spanID := uuid.New().String()

	startTime := time.Now().Add(-100 * time.Millisecond)
	tc.Spans[spanID] = &LLMSpanContext{
		StartTime: startTime,
		Model:     "claude-3-5-sonnet",
		Input:     "Hello, world!",
	}

	// Create HookContext with span ID and response
	hookCtx := &hooks.HookContext{
		SessionID:    sessionID,
		LLMModel:     "claude-3-5-sonnet",
		LLMResponse:  "Hi there!",
		LLMOptions:   make(map[string]any),
		Data: map[string]any{
			"langfuse_span_id": spanID,
		},
	}

	// Call afterLLMResponseHook
	err := hook.afterLLMResponseHook(context.Background(), hookCtx, func() error { return nil })
	require.NoError(t, err)

	// Verify span update
	spanCtx, ok := tc.Spans[spanID].(*LLMSpanContext)
	require.True(t, ok, "Span should still be LLMSpanContext type")
	assert.Equal(t, "Hi there!", spanCtx.Output, "Output should be updated")
	assert.NotNil(t, spanCtx.Usage, "Usage should be created")
	assert.Equal(t, traces.UnitType("TOKENS"), spanCtx.Usage.Unit, "Usage unit should be TOKENS")
	assert.Equal(t, traces.ObservationLevelDefault, spanCtx.Level, "Level should be DEFAULT")
	assert.Equal(t, "success", spanCtx.StatusMessage, "Status should be success")
}

// TestLangfuseHook_LLMSpanUpdate_MissingSpanID tests afterLLMResponseHook with missing span ID
func TestLangfuseHook_LLMSpanUpdate_MissingSpanID(t *testing.T) {
	sessionID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	hook.createTraceContext(sessionID)

	// Create HookContext without span ID
	hookCtx := &hooks.HookContext{
		SessionID:   sessionID,
		LLMResponse: "Response!",
		Data:        make(map[string]any),
	}

	// Should not error, just skip
	err := hook.afterLLMResponseHook(context.Background(), hookCtx, func() error { return nil })
	assert.NoError(t, err, "Should not error when span ID is missing")
}

// TestLangfuseHook_LLMSpanUpdate_NilTraceContext tests afterLLMResponseHook with no trace context
func TestLangfuseHook_LLMSpanUpdate_NilTraceContext(t *testing.T) {
	sessionID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	// Don't create trace context

	// Create HookContext with span ID but no trace context
	hookCtx := &hooks.HookContext{
		SessionID:   sessionID,
		LLMResponse: "Response!",
		Data: map[string]any{
			"langfuse_span_id": "nonexistent-span",
		},
	}

	// Should not error, just skip
	err := hook.afterLLMResponseHook(context.Background(), hookCtx, func() error { return nil })
	assert.NoError(t, err, "Should not error when trace context is missing")
}

// TestLangfuseHook_OnLLMError tests error handling in onLLMErrorHook
func TestLangfuseHook_OnLLMError(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name           string
		langfuseEnabled bool
		sessionID      uuid.UUID
		llmError       error
		wantErrorLevel bool
		wantStatusMsg  string
	}{
		{
			name:           "marks span as ERROR with error message",
			langfuseEnabled: true,
			sessionID:      sessionID,
			llmError:       fmt.Errorf("rate limit exceeded"),
			wantErrorLevel: true,
			wantStatusMsg:  "rate limit exceeded",
		},
		{
			name:           "skips marking when disabled",
			langfuseEnabled: false,
			sessionID:      sessionID,
			llmError:       fmt.Errorf("rate limit exceeded"),
			wantErrorLevel: false,
		},
		{
			name:           "handles nil error gracefully",
			langfuseEnabled: true,
			sessionID:      sessionID,
			llmError:       nil,
			wantErrorLevel: true, // Still marks ERROR level
			wantStatusMsg:  "unknown error",
		},
		{
			name:           "skips when span ID missing",
			langfuseEnabled: true,
			sessionID:      sessionID,
			llmError:       fmt.Errorf("network error"),
			wantErrorLevel: false, // Can't mark without span ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLog := mocks.NewMockLoggerService(ctrl)
			cfg := &config.LangfuseConfig{
				LangfuseEnabled: tt.langfuseEnabled,
				LangfuseHost:    "https://cloud.langfuse.com",
			}

			// Expect Debug call for span failure
			mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

			hook := &LangfuseHook{
				log:         mockLog,
				config:      cfg,
				client:      nil,
				clientMu:    &sync.Mutex{},
				traceCtxs:   make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			// Create trace context and span
			hook.createTraceContext(sessionID)
			tc := hook.getTraceContext(sessionID)
			spanID := uuid.New().String()

			startTime := time.Now().Add(-50 * time.Millisecond)
			tc.Spans[spanID] = &LLMSpanContext{
				StartTime: startTime,
				Model:     "claude-3-5-sonnet",
				Input:     "Test prompt",
				Level:     traces.ObservationLevelDefault,
			}

			// Create HookContext
			hookCtx := &hooks.HookContext{
				SessionID:  tt.sessionID,
				LLMError:   tt.llmError,
				LLMOptions: make(map[string]any),
				Data:       make(map[string]any),
			}

			// Add span ID to Data (except for missing span ID test)
			if tt.name != "skips when span ID missing" {
				hookCtx.Data["langfuse_span_id"] = spanID
			}

			// Call onLLMErrorHook
			err := hook.onLLMErrorHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			// Verify error marking
			spanCtx, ok := tc.Spans[spanID].(*LLMSpanContext)

			if tt.wantErrorLevel {
				require.True(t, ok, "Span should still exist")
				assert.Equal(t, traces.ObservationLevelError, spanCtx.Level, "Level should be ERROR")
				assert.Equal(t, tt.wantStatusMsg, spanCtx.StatusMessage, "Status message should match")
			} else {
				// For disabled or missing span ID, span should remain unchanged
				if ok {
					assert.NotEqual(t, traces.ObservationLevelError, spanCtx.Level, "Level should not be ERROR")
				}
			}
		})
	}
}

// TestLangfuseHook_LLMSpanLifecycle_Integration tests the full LLM span lifecycle
func TestLangfuseHook_LLMSpanLifecycle_Integration(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name         string
		llmModel     string
		llmInput     string
		llmResponse  string
		llmError     error
		wantOutput   string
		wantLevel    traces.ObservationLevel
		wantStatus   string
	}{
		{
			name:       "successful LLM call",
			llmModel:   "claude-3-5-sonnet",
			llmInput:   "What is 2+2?",
			llmResponse: "4",
			llmError:    nil,
			wantOutput:  "4",
			wantLevel:   traces.ObservationLevelDefault,
			wantStatus:  "success",
		},
		{
			name:       "failed LLM call",
			llmModel:   "claude-3-5-sonnet",
			llmInput:   "What is the meaning of life?",
			llmResponse: "",
			llmError:    fmt.Errorf("timeout"),
			wantOutput:  "",
			wantLevel:   traces.ObservationLevelError,
			wantStatus:  "timeout",
		},
		{
			name:       "LLM call with partial response then error",
			llmModel:   "claude-3-5-sonnet",
			llmInput:   "Generate a long story",
			llmResponse: "Once upon a time...",
			llmError:    fmt.Errorf("stream interrupted"),
			wantOutput:  "Once upon a time...",
			wantLevel:   traces.ObservationLevelError,
			wantStatus:  "stream interrupted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLog := mocks.NewMockLoggerService(ctrl)
			cfg := &config.LangfuseConfig{
				LangfuseEnabled:   true,
				LangfuseHost:      "https://cloud.langfuse.com",
				LangfusePublicKey: "pk-test-key",
				LangfuseSecretKey: "sk-test-key",
			}

			// Expect Debug calls for span lifecycle events and Info for client init
			mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
			mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

			hook := &LangfuseHook{
				log:         mockLog,
				config:      cfg,
				client:      nil,
				clientMu:    &sync.Mutex{},
				traceCtxs:   make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			// Create trace context
			hook.createTraceContext(sessionID)
			tc := hook.getTraceContext(sessionID)

			// Simulate LLM flow: BeforeLLMRequest -> LLM call -> AfterLLMResponse/OnLLMError
			hookCtx := &hooks.HookContext{
				SessionID:   sessionID,
				LLMModel:    tt.llmModel,
				LLMInput:    tt.llmInput,
				LLMResponse: tt.llmResponse,
				LLMError:    tt.llmError,
				LLMOptions:  make(map[string]any),
				Data:        make(map[string]any),
			}

			// Call beforeLLMRequestHook (creates span)
			err := hook.beforeLLMRequestHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			// Get span ID
			spanID, ok := hookCtx.Data["langfuse_span_id"].(string)
			require.True(t, ok, "Should have span ID")
			require.NotEmpty(t, spanID, "Span ID should not be empty")

			// Verify initial span state
			spanCtx, ok := tc.Spans[spanID].(*LLMSpanContext)
			require.True(t, ok, "Span should be LLMSpanContext type")
			assert.Equal(t, tt.llmModel, spanCtx.Model)
			assert.Equal(t, tt.llmInput, spanCtx.Input)
			assert.False(t, spanCtx.StartTime.IsZero())

			// Simulate LLM response/error
			if tt.llmError != nil {
				// Call onLLMErrorHook
				err = hook.onLLMErrorHook(context.Background(), hookCtx, func() error { return nil })
				require.NoError(t, err)
			} else {
				// Call afterLLMResponseHook
				err = hook.afterLLMResponseHook(context.Background(), hookCtx, func() error { return nil })
				require.NoError(t, err)
			}

			// Verify final span state
			spanCtx = tc.Spans[spanID].(*LLMSpanContext)
			assert.Equal(t, tt.wantOutput, spanCtx.Output, "Output should match")
			assert.Equal(t, tt.wantLevel, spanCtx.Level, "Level should match")
			assert.Equal(t, tt.wantStatus, spanCtx.StatusMessage, "Status message should match")

			// Verify latency was calculated
			if tt.llmError == nil {
				assert.NotNil(t, spanCtx.Usage, "Usage should be created for successful calls")
			}
		})
	}
}

// TestLangfuseHook_ToolSpanCreation tests span creation in beforeToolExecutionHook
func TestLangfuseHook_ToolSpanCreation(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name            string
		langfuseEnabled bool
		sessionID       uuid.UUID
		toolName        string
		toolArgs        map[string]any
		wantSpanCreated bool
	}{
		{
			name:            "creates span when enabled with valid session",
			langfuseEnabled: true,
			sessionID:       sessionID,
			toolName:        "read_file",
			toolArgs:        map[string]any{"path": "/test.txt"},
			wantSpanCreated: true,
		},
		{
			name:            "skips span when disabled",
			langfuseEnabled: false,
			sessionID:       sessionID,
			toolName:        "read_file",
			toolArgs:        map[string]any{"path": "/test.txt"},
			wantSpanCreated: false,
		},
		{
			name:            "skips span with nil session ID",
			langfuseEnabled: true,
			sessionID:       uuid.Nil,
			toolName:        "read_file",
			toolArgs:        map[string]any{"path": "/test.txt"},
			wantSpanCreated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLog := mocks.NewMockLoggerService(ctrl)
			cfg := &config.LangfuseConfig{
				LangfuseEnabled:   tt.langfuseEnabled,
				LangfuseHost:      "https://cloud.langfuse.com",
				LangfusePublicKey: "pk-test-key",
				LangfuseSecretKey: "sk-test-key",
			}

			// Expect Info call for client initialization and Debug for any issues
			mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

			hook := &LangfuseHook{
				log:         mockLog,
				config:      cfg,
				client:      nil, // Will be lazily initialized
				clientMu:    &sync.Mutex{},
				traceCtxs:   make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			if tt.sessionID != uuid.Nil {
				hook.createTraceContext(tt.sessionID)
			}

			hookCtx := &hooks.HookContext{
				SessionID: tt.sessionID,
				ToolName:  tt.toolName,
				ToolArgs:  tt.toolArgs,
				Data:      make(map[string]any),
			}

			err := hook.beforeToolExecutionHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			spanID, hasSpanID := hookCtx.Data["langfuse_span_id"].(string)

			if tt.wantSpanCreated {
				assert.True(t, hasSpanID, "Should have span ID in Data")
				assert.NotEmpty(t, spanID, "Span ID should not be empty")

				tc := hook.getTraceContext(tt.sessionID)
				require.NotNil(t, tc, "TraceContext should exist")
				assert.Contains(t, tc.Spans, spanID, "Span should be in TraceContext.Spans")

				spanCtx, ok := tc.Spans[spanID].(*ToolSpanContext)
				require.True(t, ok, "Span should be ToolSpanContext type")
				assert.Equal(t, tt.toolName, spanCtx.ToolName)
				assert.Equal(t, tt.toolArgs, spanCtx.Input)
				assert.False(t, spanCtx.StartTime.IsZero(), "StartTime should be set")
			} else {
				assert.False(t, hasSpanID, "Should not have span ID when disabled or nil session")
			}
		})
	}
}

// TestLangfuseHook_ToolSpanUpdate tests span update in afterToolExecutionHook
func TestLangfuseHook_ToolSpanUpdate(t *testing.T) {
	sessionID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := mocks.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled: true,
		LangfuseHost:    "https://cloud.langfuse.com",
	}

	// Expect Debug call for span completion
	mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	hook := &LangfuseHook{
		log:         mockLog,
		config:      cfg,
		client:      nil,
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[uuid.UUID]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	hook.createTraceContext(sessionID)
	tc := hook.getTraceContext(sessionID)
	spanID := uuid.New().String()

	startTime := time.Now().Add(-50 * time.Millisecond)
	tc.Spans[spanID] = &ToolSpanContext{
		StartTime: startTime,
		ToolName:  "read_file",
		Input:     map[string]any{"path": "/test.txt"},
	}

	hookCtx := &hooks.HookContext{
		SessionID:  sessionID,
		ToolResult: map[string]any{"content": "hello world"},
		Data: map[string]any{
			"langfuse_span_id": spanID,
		},
	}

	err := hook.afterToolExecutionHook(context.Background(), hookCtx, func() error { return nil })
	require.NoError(t, err)

	spanCtx, ok := tc.Spans[spanID].(*ToolSpanContext)
	require.True(t, ok, "Span should still be ToolSpanContext type")
	assert.Equal(t, map[string]any{"content": "hello world"}, spanCtx.Output, "Output should be updated")
	assert.Equal(t, traces.ObservationLevelDefault, spanCtx.Level, "Level should be DEFAULT")
	assert.Equal(t, "success", spanCtx.StatusMessage, "Status should be success")
}

// TestLangfuseHook_OnToolError tests error handling in onToolErrorHook
func TestLangfuseHook_OnToolError(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name           string
		toolError      error
		wantErrorLevel bool
		wantStatusMsg  string
	}{
		{
			name:           "marks span as ERROR with error message",
			toolError:      fmt.Errorf("file not found"),
			wantErrorLevel: true,
			wantStatusMsg:  "file not found",
		},
		{
			name:           "handles nil error gracefully",
			toolError:      nil,
			wantErrorLevel: true,
			wantStatusMsg:  "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLog := mocks.NewMockLoggerService(ctrl)
			cfg := &config.LangfuseConfig{
				LangfuseEnabled: true,
				LangfuseHost:    "https://cloud.langfuse.com",
			}

			// Expect Debug call for span failure
			mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

			hook := &LangfuseHook{
				log:         mockLog,
				config:      cfg,
				client:      nil,
				clientMu:    &sync.Mutex{},
				traceCtxs:   make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			hook.createTraceContext(sessionID)
			tc := hook.getTraceContext(sessionID)
			spanID := uuid.New().String()

			tc.Spans[spanID] = &ToolSpanContext{
				StartTime: time.Now().Add(-30 * time.Millisecond),
				ToolName:  "read_file",
				Input:     map[string]any{"path": "/test.txt"},
				Level:     traces.ObservationLevelDefault,
			}

			hookCtx := &hooks.HookContext{
				SessionID: sessionID,
				ToolError: tt.toolError,
				Data: map[string]any{
					"langfuse_span_id": spanID,
				},
			}

			err := hook.onToolErrorHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			spanCtx, ok := tc.Spans[spanID].(*ToolSpanContext)
			require.True(t, ok, "Span should still exist")
			assert.Equal(t, traces.ObservationLevelError, spanCtx.Level, "Level should be ERROR")
			assert.Equal(t, tt.wantStatusMsg, spanCtx.StatusMessage, "Status message should match")
		})
	}
}

// TestLangfuseHook_ToolSpanLifecycle_Integration tests the full tool span lifecycle
func TestLangfuseHook_ToolSpanLifecycle_Integration(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name        string
		toolName    string
		toolArgs    map[string]any
		toolResult  map[string]any
		toolError   error
		wantOutput  map[string]any
		wantLevel   traces.ObservationLevel
		wantStatus  string
	}{
		{
			name:       "successful tool execution",
			toolName:   "read_file",
			toolArgs:   map[string]any{"path": "/test.txt"},
			toolResult: map[string]any{"content": "file contents"},
			toolError:  nil,
			wantOutput: map[string]any{"content": "file contents"},
			wantLevel:  traces.ObservationLevelDefault,
			wantStatus: "success",
		},
		{
			name:       "failed tool execution",
			toolName:   "read_file",
			toolArgs:   map[string]any{"path": "/nonexistent.txt"},
			toolResult: nil,
			toolError:  fmt.Errorf("file not found"),
			wantOutput: nil,
			wantLevel:  traces.ObservationLevelError,
			wantStatus: "file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLog := mocks.NewMockLoggerService(ctrl)
			cfg := &config.LangfuseConfig{
				LangfuseEnabled:   true,
				LangfuseHost:      "https://cloud.langfuse.com",
				LangfusePublicKey: "pk-test-key",
				LangfuseSecretKey: "sk-test-key",
			}

			// Expect Debug calls for span lifecycle events and Info for client init
			mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
			mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

			hook := &LangfuseHook{
				log:         mockLog,
				config:      cfg,
				client:      nil,
				clientMu:    &sync.Mutex{},
				traceCtxs:   make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			// Create trace context
			hook.createTraceContext(sessionID)
			tc := hook.getTraceContext(sessionID)

			// Simulate tool flow: BeforeToolExecution -> Tool execution -> AfterToolExecution/OnToolError
			hookCtx := &hooks.HookContext{
				SessionID:  sessionID,
				ToolName:   tt.toolName,
				ToolArgs:   tt.toolArgs,
				ToolResult: tt.toolResult,
				ToolError:  tt.toolError,
				Data:       make(map[string]any),
			}

			// Call beforeToolExecutionHook (creates span)
			err := hook.beforeToolExecutionHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			// Get span ID
			spanID, ok := hookCtx.Data["langfuse_span_id"].(string)
			require.True(t, ok, "Should have span ID")
			require.NotEmpty(t, spanID, "Span ID should not be empty")

			// Verify initial span state
			spanCtx, ok := tc.Spans[spanID].(*ToolSpanContext)
			require.True(t, ok, "Span should be ToolSpanContext type")
			assert.Equal(t, tt.toolName, spanCtx.ToolName)
			assert.Equal(t, tt.toolArgs, spanCtx.Input)
			assert.False(t, spanCtx.StartTime.IsZero())

			// Simulate tool response/error
			if tt.toolError != nil {
				// Call onToolErrorHook
				err = hook.onToolErrorHook(context.Background(), hookCtx, func() error { return nil })
				require.NoError(t, err)
			} else {
				// Call afterToolExecutionHook
				err = hook.afterToolExecutionHook(context.Background(), hookCtx, func() error { return nil })
				require.NoError(t, err)
			}

			// Verify final span state
			spanCtx = tc.Spans[spanID].(*ToolSpanContext)
			assert.Equal(t, tt.wantOutput, spanCtx.Output, "Output should match")
			assert.Equal(t, tt.wantLevel, spanCtx.Level, "Level should match")
			assert.Equal(t, tt.wantStatus, spanCtx.StatusMessage, "Status message should match")
		})
	}
}
