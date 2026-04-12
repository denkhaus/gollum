package builtin

import (
	"sync"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/logger"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestLangfuseHook_TraceContextOperations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := logger.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled:   false,
		LangfusePublicKey: "pk-test",
		LangfuseSecretKey: "sk-test",
		LangfuseHost:      "https://cloud.langfuse.com",
	}

	hook := &LangfuseHook{
		log:         mockLog,
		config:      cfg,
		client:      nil,
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[string]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	sessionID := uuid.New().String()

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
		session1 := uuid.New().String()
		session2 := uuid.New().String()
		session3 := uuid.New().String()

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
			SessionID: uuid.New().String(),
			CreatedAt: time.Now(),
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

		err := RegisterLangfuseHooks(mockHM, hook)

		assert.NoError(t, err)
		// No EXPECT() calls means no hooks should be registered
	})

	t.Run("registers all hooks when Langfuse enabled", func(t *testing.T) {
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

		// Expect 2 session hooks
		mockHM.EXPECT().RegisterSessionHook(gomock.Any(), gomock.Any()).Return(nil).Times(2)
		// Expect 4 agent hooks
		mockHM.EXPECT().RegisterAgentHook(gomock.Any(), gomock.Any()).Return(nil).Times(4)
		// Expect 3 tool hooks
		mockHM.EXPECT().RegisterToolHook(gomock.Any(), gomock.Any()).Return(nil).Times(3)
		// Expect 8 file hooks (read, write, delete, modify - before/after each)
		mockHM.EXPECT().RegisterFileHook(gomock.Any(), gomock.Any()).Return(nil).Times(8)
		// Expect 3 LLM hooks
		mockHM.EXPECT().RegisterLLMHook(gomock.Any(), gomock.Any()).Return(nil).Times(3)

		err := RegisterLangfuseHooks(mockHM, hook)

		assert.NoError(t, err)
	})
}

func TestLangfuseHook_TracingPropagation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := logger.NewMockLoggerService(ctrl)
	cfg := &config.LangfuseConfig{
		LangfuseEnabled: true,
	}
	sessionID := uuid.New().String()

	hook := &LangfuseHook{
		log:         mockLog,
		config:      cfg,
		client:      nil,
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[string]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// Create trace context
	_ = hook.createTraceContext(sessionID)
	requireTraceContext := func(t *testing.T) *TraceContext {
		t.Helper()
		foundTC := hook.getTraceContext(sessionID)
		assert.NotNil(t, foundTC, "TraceContext should exist")
		return foundTC
	}

	t.Run("propagateTracingToContext sets trace ID in TypedHookContext.Tracing", func(t *testing.T) {
		tc := requireTraceContext(t)
		_ = tc // Use tc to avoid unused variable error
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID},
			hooks.ToolPayload{Name: "test"},
		)

		hook.propagateTracingToContext(sessionID, &hookCtx.Tracing)

		assert.NotEmpty(t, hookCtx.Tracing.TraceID, "TraceID should be set in Tracing")
		assert.Equal(t, tc.TraceID, hookCtx.Tracing.TraceID, "TraceID should match trace context")
	})

	t.Run("propagateTracingToContext does nothing when no session", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: uuid.Nil.String()},
			hooks.ToolPayload{Name: "test"},
		)

		hook.propagateTracingToContext(uuid.Nil.String(), &hookCtx.Tracing)

		assert.Empty(t, hookCtx.Tracing.TraceID, "TraceID should not be set for nil session")
	})

	t.Run("propagateTracingToContext does nothing when no trace context", func(t *testing.T) {
		differentSession := uuid.New()
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: differentSession.String()},
			hooks.ToolPayload{Name: "test"},
		)

		hook.propagateTracingToContext(differentSession.String(), &hookCtx.Tracing)

		assert.Empty(t, hookCtx.Tracing.TraceID, "TraceID should not be set for non-existent trace")
	})
}
