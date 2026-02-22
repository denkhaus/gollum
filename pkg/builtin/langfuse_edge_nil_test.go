package builtin

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/git-hulk/langfuse-go/pkg/traces"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestLangfuseHook_NilSessionID(t *testing.T) {
	t.Run("beforeSessionStartHook skips Nil SessionID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := mocks.NewMockLoggerService(ctrl)
		mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

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

		ctx := context.Background()
		hookCtx := &hooks.HookContext{
			SessionID: uuid.Nil, // Nil SessionID
			Data:      make(map[string]interface{}),
		}

		nextCalled := false
		next := func() error {
			nextCalled = true
			return nil
		}

		err := hook.beforeSessionStartHook(ctx, hookCtx, next)

		assert.NoError(t, err)
		assert.True(t, nextCalled, "next() should be called")
		assert.Empty(t, hook.traceCtxs, "No trace context should be created")
	})

	t.Run("afterSessionEndHook handles Nil SessionID", func(t *testing.T) {
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

		ctx := context.Background()
		hookCtx := &hooks.HookContext{
			SessionID: uuid.Nil, // Nil SessionID
			Data:      make(map[string]interface{}),
		}

		nextCalled := false
		next := func() error {
			nextCalled = true
			return nil
		}

		err := hook.afterSessionEndHook(ctx, hookCtx, next)

		assert.NoError(t, err, "Should not error with Nil SessionID")
		assert.True(t, nextCalled, "next() should be called")
	})
}

// TestLangfuseHook_onToolErrorHook_NilError tests onToolErrorHook with nil ToolError
func TestLangfuseHook_onToolErrorHook_NilError(t *testing.T) {
	t.Run("onToolErrorHook handles nil ToolError gracefully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := mocks.NewMockLoggerService(ctrl)
		mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

		cfg := &config.LangfuseConfig{
			LangfuseEnabled: true,
		}

		sessionID := uuid.New()
		spanID := uuid.New().String()

		hook := &LangfuseHook{
			log:         mockLog,
			config:      cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[uuid.UUID]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		// Create trace context with tool span
		tc := hook.createTraceContext(sessionID)
		tc.Spans[spanID] = &ToolSpanContext{
			StartTime: time.Now(),
			ToolName:  "test-tool",
		}

		ctx := context.Background()
		hookCtx := &hooks.HookContext{
			SessionID: sessionID,
			ToolName:  "test-tool",
			ToolError: nil, // nil error
			Data:      make(map[string]interface{}),
		}
		hookCtx.Data["langfuse_span_id"] = spanID

		nextCalled := false
		next := func() error {
			nextCalled = true
			return nil
		}

		err := hook.onToolErrorHook(ctx, hookCtx, next)

		assert.NoError(t, err, "Should not error with nil ToolError")
		assert.True(t, nextCalled, "next() should be called")

		// Verify span was marked with "unknown error"
		hook.traceCtxsMu.RLock()
		spanCtx := tc.Spans[spanID].(*ToolSpanContext)
		assert.Equal(t, traces.ObservationLevelError, spanCtx.Level)
		assert.Equal(t, "unknown error", spanCtx.StatusMessage)
		hook.traceCtxsMu.RUnlock()
	})
}

// TestLangfuseHook_onLLMErrorHook_NilError tests onLLMErrorHook with nil LLMError
func TestLangfuseHook_onLLMErrorHook_NilError(t *testing.T) {
	t.Run("onLLMErrorHook handles nil LLMError gracefully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := mocks.NewMockLoggerService(ctrl)
		mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

		cfg := &config.LangfuseConfig{
			LangfuseEnabled: true,
		}

		sessionID := uuid.New()
		spanID := uuid.New().String()

		hook := &LangfuseHook{
			log:         mockLog,
			config:      cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[uuid.UUID]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		// Create trace context with LLM span
		tc := hook.createTraceContext(sessionID)
		tc.Spans[spanID] = &LLMSpanContext{
			StartTime: time.Now(),
			Model:     "gpt-4",
		}

		ctx := context.Background()
		hookCtx := &hooks.HookContext{
			SessionID: sessionID,
			LLMModel:  "gpt-4",
			LLMError:  nil, // nil error
			Data:      make(map[string]interface{}),
		}
		hookCtx.Data["langfuse_span_id"] = spanID

		nextCalled := false
		next := func() error {
			nextCalled = true
			return nil
		}

		err := hook.onLLMErrorHook(ctx, hookCtx, next)

		assert.NoError(t, err, "Should not error with nil LLMError")
		assert.True(t, nextCalled, "next() should be called")

		// Verify span was marked with "unknown error"
		hook.traceCtxsMu.RLock()
		spanCtx := tc.Spans[spanID].(*LLMSpanContext)
		assert.Equal(t, traces.ObservationLevelError, spanCtx.Level)
		assert.Equal(t, "unknown error", spanCtx.StatusMessage)
		hook.traceCtxsMu.RUnlock()
	})
}

// ============================================================================
// File Operation Hooks - Documentation Test
// These tests document the intentional coverage gap for file operation stubs.
// ============================================================================

// TestLangfuseHook_FileOperationHooks_AreStubs documents that file operation hooks
// are intentionally unimplemented stubs that only propagate trace ID.
// This test documents this intentional coverage gap for future maintainers.
func TestLangfuseHook_FileOperationHooks_AreStubs(t *testing.T) {
	t.Run("all file operation hooks propagate trace ID only", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := mocks.NewMockLoggerService(ctrl)
		mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

		cfg := &config.LangfuseConfig{
			LangfuseEnabled:   true,
			LangfuseHost:      "https://cloud.langfuse.com",
			LangfusePublicKey: "pk-test",
			LangfuseSecretKey: "sk-test",
		}

		sessionID := uuid.New()
		hook := &LangfuseHook{
			log:         mockLog,
			config:      cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[uuid.UUID]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		// Create a trace context for propagation
		tc := hook.createTraceContext(sessionID)

		ctx := context.Background()
		next := func() error { return nil }

		// All file operation hooks should:
		// 1. Call next() (not error)
		// 2. Propagate trace ID if session exists

		fileHooks := []struct {
			name string
			hook func(context.Context, *hooks.HookContext, func() error) error
		}{
			{"beforeFileReadHook", hook.beforeFileReadHook},
			{"afterFileReadHook", hook.afterFileReadHook},
			{"beforeFileWriteHook", hook.beforeFileWriteHook},
			{"afterFileWriteHook", hook.afterFileWriteHook},
			{"beforeFileDeleteHook", hook.beforeFileDeleteHook},
			{"afterFileDeleteHook", hook.afterFileDeleteHook},
			{"beforeFileModifyHook", hook.beforeFileModifyHook},
			{"afterFileModifyHook", hook.afterFileModifyHook},
		}

		for _, fh := range fileHooks {
			t.Run(fh.name, func(t *testing.T) {
				hookCtx := &hooks.HookContext{
					SessionID: sessionID,
					Data:      make(map[string]interface{}),
				}

				err := fh.hook(ctx, hookCtx, next)

				assert.NoError(t, err, "File hook should not error")
				assert.Equal(t, tc.TraceID, hookCtx.Data["langfuse_trace_id"],
					"Trace ID should be propagated")
			})
		}

		t.Log("File operation hooks are stubs - they only propagate trace ID.")
		t.Log("Future implementation may add actual file operation tracing.")
	})
}
