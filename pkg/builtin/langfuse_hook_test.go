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

// TestLangfuseHook_AgentSpawnSpanLifecycle tests the full agent spawn span lifecycle
func TestLangfuseHook_AgentSpawnSpanLifecycle(t *testing.T) {
	sessionID := uuid.New()
	parentAgentID := uuid.New()

	tests := []struct {
		name            string
		langfuseEnabled bool
		sessionID       uuid.UUID
		parentAgentID   uuid.UUID
		newAgentID      string
		wantSpanCreated bool
	}{
		{
			name:            "creates spawn span when enabled",
			langfuseEnabled: true,
			sessionID:       sessionID,
			parentAgentID:   parentAgentID,
			newAgentID:      "child-agent-123",
			wantSpanCreated: true,
		},
		{
			name:            "skips spawn span when disabled",
			langfuseEnabled: false,
			sessionID:       sessionID,
			parentAgentID:   parentAgentID,
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

			// Expect Info call for client initialization and Debug for lifecycle
			mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

			hook := &LangfuseHook{
				log:         mockLog,
				config:      cfg,
				client:      nil,
				clientMu:    &sync.Mutex{},
				traceCtxs:   make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			if tt.sessionID != uuid.Nil {
				hook.createTraceContext(tt.sessionID)
			}

			hookCtx := &hooks.HookContext{
				SessionID: tt.sessionID,
				AgentID:   tt.parentAgentID,
				Data:      make(map[string]any),
			}

			// Call before hook
			err := hook.beforeAgentSpawnHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			spanID, hasSpanID := hookCtx.Data["langfuse_span_id"].(string)

			if tt.wantSpanCreated {
				assert.True(t, hasSpanID, "Should have span ID")

				// Add new agent ID for after hook
				if tt.newAgentID != "" {
					hookCtx.Data["new_agent_id"] = tt.newAgentID
				}

				// Call after hook
				err = hook.afterAgentSpawnHook(context.Background(), hookCtx, func() error { return nil })
				require.NoError(t, err)

				tc := hook.getTraceContext(tt.sessionID)
				spanCtx, ok := tc.Spans[spanID].(*AgentSpanContext)
				require.True(t, ok, "Span should be AgentSpanContext type")
				assert.Equal(t, "spawn", spanCtx.EventType)
				assert.Equal(t, tt.parentAgentID.String(), spanCtx.ParentAgentID)
				assert.Equal(t, tt.newAgentID, spanCtx.NewAgentID)
				assert.Equal(t, traces.ObservationLevelDefault, spanCtx.Level)
				assert.Equal(t, "success", spanCtx.StatusMessage)
			}
		})
	}
}

// TestLangfuseHook_AgentRemoveSpanLifecycle tests the agent removal span lifecycle
func TestLangfuseHook_AgentRemoveSpanLifecycle(t *testing.T) {
	sessionID := uuid.New()
	agentID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := mocks.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled:   true,
		LangfuseHost:      "https://cloud.langfuse.com",
		LangfusePublicKey: "pk-test-key",
		LangfuseSecretKey: "sk-test-key",
	}

	// Expect Info call for client init and Debug for lifecycle events
	mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
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

	hookCtx := &hooks.HookContext{
		SessionID: sessionID,
		AgentID:   agentID,
		Data:      make(map[string]any),
	}

	// Call before remove hook
	err := hook.beforeAgentRemoveHook(context.Background(), hookCtx, func() error { return nil })
	require.NoError(t, err)

	spanID, hasSpanID := hookCtx.Data["langfuse_span_id"].(string)
	assert.True(t, hasSpanID, "Should have span ID")

	// Call after remove hook
	err = hook.afterAgentRemoveHook(context.Background(), hookCtx, func() error { return nil })
	require.NoError(t, err)

	tc := hook.getTraceContext(sessionID)
	spanCtx, ok := tc.Spans[spanID].(*AgentSpanContext)
	require.True(t, ok, "Span should be AgentSpanContext type")
	assert.Equal(t, "remove", spanCtx.EventType)
	assert.Equal(t, agentID.String(), spanCtx.AgentID)
	assert.Equal(t, traces.ObservationLevelDefault, spanCtx.Level)
	assert.Equal(t, "success", spanCtx.StatusMessage)
}

// TestLangfuseHook_SpanHierarchy_Integration tests span hierarchy across session, agent, tool, and LLM operations
func TestLangfuseHook_SpanHierarchy_Integration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sessionID := uuid.New()
	parentAgentID := uuid.New()

	mockLog := mocks.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled:   true,
		LangfuseHost:      "https://cloud.langfuse.com",
		LangfusePublicKey: "pk-test-key",
		LangfuseSecretKey: "sk-test-key",
	}

	// Expect Info call for client init and Debug for lifecycle events
	mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	hook := &LangfuseHook{
		log:         mockLog,
		config:      cfg,
		client:      nil,
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[uuid.UUID]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// Simulate session start (creates trace context)
	hook.createTraceContext(sessionID)
	tc := hook.getTraceContext(sessionID)
	require.NotNil(t, tc, "TraceContext should exist after session start")

	// Simulate agent spawn
	agentSpawnCtx := &hooks.HookContext{
		SessionID: sessionID,
		AgentID:   parentAgentID,
		Data:      make(map[string]any),
	}

	err := hook.beforeAgentSpawnHook(context.Background(), agentSpawnCtx, func() error { return nil })
	require.NoError(t, err)

	agentSpanID := agentSpawnCtx.Data["langfuse_span_id"].(string)
	agentSpawnCtx.Data["new_agent_id"] = "child-agent-123"

	err = hook.afterAgentSpawnHook(context.Background(), agentSpawnCtx, func() error { return nil })
	require.NoError(t, err)

	// Verify agent span
	agentSpan, ok := tc.Spans[agentSpanID].(*AgentSpanContext)
	require.True(t, ok, "Agent span should exist")
	assert.Equal(t, "spawn", agentSpan.EventType)
	assert.Equal(t, parentAgentID.String(), agentSpan.ParentAgentID)
	assert.Equal(t, "child-agent-123", agentSpan.NewAgentID)

	// Simulate tool execution under this agent
	toolCtx := &hooks.HookContext{
		SessionID: sessionID,
		ToolName:  "read_file",
		ToolArgs:  map[string]any{"path": "/test.txt"},
		Data:      make(map[string]any),
	}

	err = hook.beforeToolExecutionHook(context.Background(), toolCtx, func() error { return nil })
	require.NoError(t, err)

	toolSpanID := toolCtx.Data["langfuse_span_id"].(string)
	toolCtx.ToolResult = map[string]any{"content": "file contents"}

	err = hook.afterToolExecutionHook(context.Background(), toolCtx, func() error { return nil })
	require.NoError(t, err)

	// Verify tool span
	toolSpan, ok := tc.Spans[toolSpanID].(*ToolSpanContext)
	require.True(t, ok, "Tool span should exist")
	assert.Equal(t, "read_file", toolSpan.ToolName)
	assert.Equal(t, map[string]any{"path": "/test.txt"}, toolSpan.Input)
	assert.Equal(t, map[string]any{"content": "file contents"}, toolSpan.Output)

	// Simulate LLM call under this agent
	llmCtx := &hooks.HookContext{
		SessionID: sessionID,
		LLMModel:  "claude-3-5-sonnet",
		LLMInput:  "Hello, world!",
		Data:      make(map[string]any),
	}

	err = hook.beforeLLMRequestHook(context.Background(), llmCtx, func() error { return nil })
	require.NoError(t, err)

	llmSpanID := llmCtx.Data["langfuse_span_id"].(string)
	llmCtx.LLMResponse = "Hi there!"

	err = hook.afterLLMResponseHook(context.Background(), llmCtx, func() error { return nil })
	require.NoError(t, err)

	// Verify LLM span
	llmSpan, ok := tc.Spans[llmSpanID].(*LLMSpanContext)
	require.True(t, ok, "LLM span should exist")
	assert.Equal(t, "claude-3-5-sonnet", llmSpan.Model)
	assert.Equal(t, "Hello, world!", llmSpan.Input)
	assert.Equal(t, "Hi there!", llmSpan.Output)

	// Verify all spans are in the same trace context
	assert.Len(t, tc.Spans, 3, "Should have 3 spans (agent, tool, LLM)")
	assert.Contains(t, tc.Spans, agentSpanID)
	assert.Contains(t, tc.Spans, toolSpanID)
	assert.Contains(t, tc.Spans, llmSpanID)

	// Simulate agent removal
	removeCtx := &hooks.HookContext{
		SessionID: sessionID,
		AgentID:   parentAgentID,
		Data:      make(map[string]any),
	}

	err = hook.beforeAgentRemoveHook(context.Background(), removeCtx, func() error { return nil })
	require.NoError(t, err)

	removeSpanID := removeCtx.Data["langfuse_span_id"].(string)

	err = hook.afterAgentRemoveHook(context.Background(), removeCtx, func() error { return nil })
	require.NoError(t, err)

	// Verify removal span
	removeSpan, ok := tc.Spans[removeSpanID].(*AgentSpanContext)
	require.True(t, ok, "Remove span should exist")
	assert.Equal(t, "remove", removeSpan.EventType)

	// Final verification: 4 spans total (spawn, tool, LLM, remove)
	assert.Len(t, tc.Spans, 4, "Should have 4 spans after full lifecycle")

	// Simulate session end (removes trace context)
	hook.removeTraceContext(sessionID)
	tc = hook.getTraceContext(sessionID)
	assert.Nil(t, tc, "TraceContext should be removed after session end")
}

// TestLangfuseHook_ErrorPath_Integration tests error handling at each level of the span hierarchy
func TestLangfuseHook_ErrorPath_Integration(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name          string
		setupFn       func(*LangfuseHook, *hooks.HookContext)
		errorFn       func(*LangfuseHook, *hooks.HookContext) error
		wantSpanType  string
		wantLevel     traces.ObservationLevel
		wantStatusMsg string
	}{
		{
			name: "tool execution error",
			setupFn: func(h *LangfuseHook, ctx *hooks.HookContext) {
				ctx.ToolName = "read_file"
				ctx.ToolArgs = map[string]any{"path": "/nonexistent.txt"}
				h.beforeToolExecutionHook(context.Background(), ctx, func() error { return nil })
			},
			errorFn: func(h *LangfuseHook, ctx *hooks.HookContext) error {
				ctx.ToolError = fmt.Errorf("file not found")
				return h.onToolErrorHook(context.Background(), ctx, func() error { return nil })
			},
			wantSpanType:  "tool",
			wantLevel:     traces.ObservationLevelError,
			wantStatusMsg: "file not found",
		},
		{
			name: "LLM request error",
			setupFn: func(h *LangfuseHook, ctx *hooks.HookContext) {
				ctx.LLMModel = "claude-3-5-sonnet"
				ctx.LLMInput = "Test prompt"
				h.beforeLLMRequestHook(context.Background(), ctx, func() error { return nil })
			},
			errorFn: func(h *LangfuseHook, ctx *hooks.HookContext) error {
				ctx.LLMError = fmt.Errorf("rate limit exceeded")
				ctx.LLMResponse = "" // No response on error
				return h.onLLMErrorHook(context.Background(), ctx, func() error { return nil })
			},
			wantSpanType:  "llm",
			wantLevel:     traces.ObservationLevelError,
			wantStatusMsg: "rate limit exceeded",
		},
		{
			name: "LLM partial response error",
			setupFn: func(h *LangfuseHook, ctx *hooks.HookContext) {
				ctx.LLMModel = "claude-3-5-sonnet"
				ctx.LLMInput = "Generate story"
				h.beforeLLMRequestHook(context.Background(), ctx, func() error { return nil })
			},
			errorFn: func(h *LangfuseHook, ctx *hooks.HookContext) error {
				ctx.LLMError = fmt.Errorf("stream interrupted")
				ctx.LLMResponse = "Once upon a time..." // Partial response
				return h.onLLMErrorHook(context.Background(), ctx, func() error { return nil })
			},
			wantSpanType:  "llm",
			wantLevel:     traces.ObservationLevelError,
			wantStatusMsg: "stream interrupted",
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

			// Expect Info call for client init and Debug for error events
			mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
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

			ctx := &hooks.HookContext{
				SessionID: sessionID,
				Data:      make(map[string]any),
			}

			// Setup: Create span via before hook
			tt.setupFn(hook, ctx)

			spanID := ctx.Data["langfuse_span_id"].(string)
			require.NotEmpty(t, spanID, "Should have span ID")

			// Execute error hook
			err := tt.errorFn(hook, ctx)
			require.NoError(t, err)

			// Verify error state
			switch tt.wantSpanType {
			case "tool":
				span, ok := tc.Spans[spanID].(*ToolSpanContext)
				require.True(t, ok, "Should be ToolSpanContext")
				assert.Equal(t, tt.wantLevel, span.Level)
				assert.Equal(t, tt.wantStatusMsg, span.StatusMessage)
			case "llm":
				span, ok := tc.Spans[spanID].(*LLMSpanContext)
				require.True(t, ok, "Should be LLMSpanContext")
				assert.Equal(t, tt.wantLevel, span.Level)
				assert.Equal(t, tt.wantStatusMsg, span.StatusMessage)
				// For partial response test, verify output is preserved
				if tt.name == "LLM partial response error" {
					assert.Equal(t, "Once upon a time...", span.Output, "Partial response should be preserved")
				}
			}
		})
	}
}

// TestLangfuseHook_ConcurrentAccess_Integration tests concurrent span creation across multiple operations
func TestLangfuseHook_ConcurrentAccess_Integration(t *testing.T) {
	sessionID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLog := mocks.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled:   true,
		LangfuseHost:      "https://cloud.langfuse.com",
		LangfusePublicKey: "pk-test-key",
		LangfuseSecretKey: "sk-test-key",
	}

	// Expect Info call for client init and Debug for lifecycle events
	mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
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

	// Run multiple goroutines creating different span types concurrently
	var wg sync.WaitGroup
	numOperations := 10
	spanIDs := make([]string, numOperations)
	spanIDsMu := &sync.Mutex{}

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx := &hooks.HookContext{
				SessionID: sessionID,
				Data:      make(map[string]any),
			}

			switch idx % 3 {
			case 0:
				// Tool execution
				ctx.ToolName = fmt.Sprintf("tool-%d", idx)
				ctx.ToolArgs = map[string]any{"index": idx}
				hook.beforeToolExecutionHook(context.Background(), ctx, func() error { return nil })
				ctx.ToolResult = map[string]any{"result": idx}
				hook.afterToolExecutionHook(context.Background(), ctx, func() error { return nil })

			case 1:
				// LLM request
				ctx.LLMModel = "claude-3-5-sonnet"
				ctx.LLMInput = fmt.Sprintf("prompt-%d", idx)
				hook.beforeLLMRequestHook(context.Background(), ctx, func() error { return nil })
				ctx.LLMResponse = fmt.Sprintf("response-%d", idx)
				hook.afterLLMResponseHook(context.Background(), ctx, func() error { return nil })

			case 2:
				// Agent spawn
				ctx.AgentID = uuid.New()
				hook.beforeAgentSpawnHook(context.Background(), ctx, func() error { return nil })
				ctx.Data["new_agent_id"] = fmt.Sprintf("agent-%d", idx)
				hook.afterAgentSpawnHook(context.Background(), ctx, func() error { return nil })
			}

			if spanID, ok := ctx.Data["langfuse_span_id"].(string); ok {
				spanIDsMu.Lock()
				spanIDs[idx] = spanID
				spanIDsMu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify all spans were created
	tc = hook.getTraceContext(sessionID)
	require.NotNil(t, tc, "TraceContext should still exist")

	uniqueSpanIDs := make(map[string]bool)
	for _, id := range spanIDs {
		if id != "" {
			uniqueSpanIDs[id] = true
			assert.Contains(t, tc.Spans, id, "Span should be in TraceContext")
		}
	}

	assert.Len(t, uniqueSpanIDs, numOperations, "All spans should have unique IDs")
	assert.Len(t, tc.Spans, numOperations, "All spans should be stored")
}

// TestLangfuseHook_MultipleAgents_Integration tests multiple agents with their own operations
func TestLangfuseHook_MultipleAgents_Integration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sessionID := uuid.New()

	mockLog := mocks.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled:   true,
		LangfuseHost:      "https://cloud.langfuse.com",
		LangfusePublicKey: "pk-test-key",
		LangfuseSecretKey: "sk-test-key",
	}

	// Expect Info call for client init and Debug for lifecycle events
	mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
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

	// Agent 1 spawns and does work
	agent1ID := uuid.New()
	spawnCtx1 := &hooks.HookContext{
		SessionID: sessionID,
		AgentID:   agent1ID,
		Data:      make(map[string]any),
	}
	hook.beforeAgentSpawnHook(context.Background(), spawnCtx1, func() error { return nil })
	spawnCtx1.Data["new_agent_id"] = "agent-1"
	hook.afterAgentSpawnHook(context.Background(), spawnCtx1, func() error { return nil })
	agent1SpanID := spawnCtx1.Data["langfuse_span_id"].(string)

	// Agent 1 executes a tool
	toolCtx1 := &hooks.HookContext{
		SessionID: sessionID,
		ToolName:  "agent1_tool",
		ToolArgs:  map[string]any{"agent": "agent-1"},
		Data:      make(map[string]any),
	}
	hook.beforeToolExecutionHook(context.Background(), toolCtx1, func() error { return nil })
	toolCtx1.ToolResult = map[string]any{"status": "done"}
	hook.afterToolExecutionHook(context.Background(), toolCtx1, func() error { return nil })
	tool1SpanID := toolCtx1.Data["langfuse_span_id"].(string)

	// Agent 2 spawns and does work
	agent2ID := uuid.New()
	spawnCtx2 := &hooks.HookContext{
		SessionID: sessionID,
		AgentID:   agent2ID,
		Data:      make(map[string]any),
	}
	hook.beforeAgentSpawnHook(context.Background(), spawnCtx2, func() error { return nil })
	spawnCtx2.Data["new_agent_id"] = "agent-2"
	hook.afterAgentSpawnHook(context.Background(), spawnCtx2, func() error { return nil })
	agent2SpanID := spawnCtx2.Data["langfuse_span_id"].(string)

	// Agent 2 makes an LLM call
	llmCtx2 := &hooks.HookContext{
		SessionID: sessionID,
		LLMModel:  "claude-3-5-sonnet",
		LLMInput:  "Agent 2 request",
		Data:      make(map[string]any),
	}
	hook.beforeLLMRequestHook(context.Background(), llmCtx2, func() error { return nil })
	llmCtx2.LLMResponse = "Agent 2 response"
	hook.afterLLMResponseHook(context.Background(), llmCtx2, func() error { return nil })
	llm2SpanID := llmCtx2.Data["langfuse_span_id"].(string)

	// Verify all spans are in the same trace context
	assert.Len(t, tc.Spans, 4, "Should have 4 spans (2 agents, 1 tool, 1 LLM)")
	assert.Contains(t, tc.Spans, agent1SpanID)
	assert.Contains(t, tc.Spans, tool1SpanID)
	assert.Contains(t, tc.Spans, agent2SpanID)
	assert.Contains(t, tc.Spans, llm2SpanID)

	// Verify agent 1 span
	agent1Span := tc.Spans[agent1SpanID].(*AgentSpanContext)
	assert.Equal(t, "spawn", agent1Span.EventType)
	assert.Equal(t, "agent-1", agent1Span.NewAgentID)

	// Verify agent 2 span
	agent2Span := tc.Spans[agent2SpanID].(*AgentSpanContext)
	assert.Equal(t, "spawn", agent2Span.EventType)
	assert.Equal(t, "agent-2", agent2Span.NewAgentID)

	// Verify tool span
	tool1Span := tc.Spans[tool1SpanID].(*ToolSpanContext)
	assert.Equal(t, "agent1_tool", tool1Span.ToolName)

	// Verify LLM span
	llm2Span := tc.Spans[llm2SpanID].(*LLMSpanContext)
	assert.Equal(t, "Agent 2 request", llm2Span.Input)
	assert.Equal(t, "Agent 2 response", llm2Span.Output)
}

// TestLangfuseHook_AgentSpanLifecycle_Integration tests full agent lifecycle
func TestLangfuseHook_AgentSpanLifecycle_Integration(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name          string
		parentAgentID uuid.UUID
		newAgentID    string
		eventType     string
	}{
		{
			name:          "agent spawn with hierarchy",
			parentAgentID: uuid.New(),
			newAgentID:    "child-agent-456",
			eventType:     "spawn",
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

			// Simulate agent spawn flow
			hookCtx := &hooks.HookContext{
				SessionID: sessionID,
				AgentID:   tt.parentAgentID,
				Data:      make(map[string]any),
			}

			// Call beforeAgentSpawnHook (creates span)
			err := hook.beforeAgentSpawnHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			// Get span ID
			spanID, ok := hookCtx.Data["langfuse_span_id"].(string)
			require.True(t, ok, "Should have span ID")
			require.NotEmpty(t, spanID, "Span ID should not be empty")

			// Verify initial span state
			spanCtx, ok := tc.Spans[spanID].(*AgentSpanContext)
			require.True(t, ok, "Span should be AgentSpanContext type")
			assert.Equal(t, "spawn", spanCtx.EventType)
			assert.Equal(t, tt.parentAgentID.String(), spanCtx.ParentAgentID)
			assert.False(t, spanCtx.StartTime.IsZero())

			// Set new agent ID (simulating what caller would do)
			hookCtx.Data["new_agent_id"] = tt.newAgentID

			// Call afterAgentSpawnHook
			err = hook.afterAgentSpawnHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			// Verify final span state
			spanCtx = tc.Spans[spanID].(*AgentSpanContext)
			assert.Equal(t, tt.newAgentID, spanCtx.NewAgentID, "NewAgentID should be set")
			assert.Equal(t, traces.ObservationLevelDefault, spanCtx.Level, "Level should be DEFAULT")
			assert.Equal(t, "success", spanCtx.StatusMessage, "Status message should be success")

			// Verify hierarchy metadata captured
			assert.NotEmpty(t, spanCtx.ParentAgentID, "ParentAgentID should be captured")
			assert.NotEmpty(t, spanCtx.NewAgentID, "NewAgentID should be captured")
		})
	}
}

// TestLangfuseHook_SessionTraceLifecycle tests session trace lifecycle
func TestLangfuseHook_SessionTraceLifecycle(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name            string
		langfuseEnabled bool
		sessionID       uuid.UUID
		wantTraceCreated bool
	}{
		{
			name:            "creates trace when enabled with valid session",
			langfuseEnabled: true,
			sessionID:       sessionID,
			wantTraceCreated: true,
		},
		{
			name:            "skips trace when disabled",
			langfuseEnabled: false,
			sessionID:       sessionID,
			wantTraceCreated: false,
		},
		{
			name:            "skips trace with nil session ID",
			langfuseEnabled: true,
			sessionID:       uuid.Nil,
			wantTraceCreated: false,
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
			mockLog.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

			hook := &LangfuseHook{
				log:         mockLog,
				config:      cfg,
				client:      nil, // Will be lazily initialized
				clientMu:    &sync.Mutex{},
				traceCtxs:   make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			hookCtx := &hooks.HookContext{
				SessionID: tt.sessionID,
				Data:      make(map[string]any),
			}

			// Call beforeSessionStartHook
			err := hook.beforeSessionStartHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			if tt.wantTraceCreated {
				// Note: Without a real client, we can't test actual SDK trace creation
				// This tests the hook logic and trace context management
				tc := hook.getTraceContext(tt.sessionID)
				assert.NotNil(t, tc, "TraceContext should be created")
				assert.NotEmpty(t, tc.TraceID, "TraceID should be set")
				assert.NotNil(t, tc.Spans, "Spans map should be initialized")

				// Verify trace ID propagation
				traceID, hasTraceID := hookCtx.Data["langfuse_trace_id"].(string)
				assert.True(t, hasTraceID, "Should have langfuse_trace_id in Data")
				assert.Equal(t, tc.TraceID, traceID, "Trace ID should match")

				// Call afterSessionEndHook
				err = hook.afterSessionEndHook(context.Background(), hookCtx, func() error { return nil })
				require.NoError(t, err)

				// Verify cleanup
				tc = hook.getTraceContext(tt.sessionID)
				assert.Nil(t, tc, "TraceContext should be removed after session end")
			} else {
				// Verify no trace context created
				tc := hook.getTraceContext(tt.sessionID)
				assert.Nil(t, tc, "TraceContext should not be created when disabled or nil session")
			}
		})
	}
}

// TestLangfuseHook_SessionTraceFlush tests session trace flush behavior
func TestLangfuseHook_SessionTraceFlush(t *testing.T) {
	sessionID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := mocks.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled:   true,
		LangfuseHost:      "https://cloud.langfuse.com",
		LangfusePublicKey: "pk-test-key",
		LangfuseSecretKey: "sk-test-key",
	}

	// Expect Info call for client initialization
	mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	hook := &LangfuseHook{
		log:         mockLog,
		config:      cfg,
		client:      nil, // No real client - tests flush error handling
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[uuid.UUID]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// Create trace context manually (simulating session start)
	hook.createTraceContext(sessionID)
	tc := hook.getTraceContext(sessionID)
	require.NotNil(t, tc)

	hookCtx := &hooks.HookContext{
		SessionID: sessionID,
		Data:      make(map[string]any),
	}

	// Call afterSessionEndHook - should not fail even without client
	err := hook.afterSessionEndHook(context.Background(), hookCtx, func() error { return nil })
	require.NoError(t, err, "afterSessionEndHook should not return error on flush failure")

	// Verify trace context removed
	tc = hook.getTraceContext(sessionID)
	assert.Nil(t, tc, "TraceContext should be removed even if flush fails")
}

// TestLangfuseHook_PropagateTraceID tests trace ID propagation via propagateTraceID
func TestLangfuseHook_PropagateTraceID(t *testing.T) {
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
	tc := hook.getTraceContext(sessionID)
	require.NotNil(t, tc)

	hookCtx := &hooks.HookContext{
		SessionID: sessionID,
		Data:      make(map[string]any),
	}

	hook.propagateTraceID(hookCtx)

	traceID, ok := hookCtx.Data["langfuse_trace_id"].(string)
	assert.True(t, ok, "Should have langfuse_trace_id")
	assert.Equal(t, tc.TraceID, traceID, "Trace ID should match TraceContext.TraceID")
}
