package builtin

import (
	"context"
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestLangfuseHook_DisabledConfig(t *testing.T) {
	t.Run("beforeSessionStartHook is no-op when disabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := mocks.NewMockLoggerService(ctrl)

		cfg := &config.LangfuseConfig{
			LangfuseEnabled: false, // Disabled
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
		sessionID := uuid.New()
		hookCtx := &hooks.HookContext{
			SessionID: sessionID,
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

	t.Run("beforeLLMRequestHook is no-op when disabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := mocks.NewMockLoggerService(ctrl)
		mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

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

		ctx := context.Background()
		hookCtx := &hooks.HookContext{
			SessionID: uuid.New(),
			LLMModel:  "gpt-4",
			LLMInput:  "test input",
			Data:      make(map[string]interface{}),
		}

		nextCalled := false
		next := func() error {
			nextCalled = true
			return nil
		}

		err := hook.beforeLLMRequestHook(ctx, hookCtx, next)

		assert.NoError(t, err)
		assert.True(t, nextCalled, "next() should be called")
	})

	t.Run("beforeToolExecutionHook is no-op when disabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := mocks.NewMockLoggerService(ctrl)
		mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

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

		ctx := context.Background()
		hookCtx := &hooks.HookContext{
			SessionID: uuid.New(),
			ToolName:  "test-tool",
			ToolArgs:  map[string]interface{}{"arg": "value"},
			Data:      make(map[string]interface{}),
		}

		nextCalled := false
		next := func() error {
			nextCalled = true
			return nil
		}

		err := hook.beforeToolExecutionHook(ctx, hookCtx, next)

		assert.NoError(t, err)
		assert.True(t, nextCalled, "next() should be called")
	})
}

// TestLangfuseHook_MissingSpanID tests hooks when span ID is missing
func TestLangfuseHook_MissingSpanID(t *testing.T) {
	t.Run("afterLLMResponseHook handles missing span ID", func(t *testing.T) {
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
		sessionID := uuid.New()
		hookCtx := &hooks.HookContext{
			SessionID:   sessionID,
			LLMModel:    "gpt-4",
			LLMResponse: "test response",
			Data:        make(map[string]interface{}),
			// No span_id set
		}

		nextCalled := false
		next := func() error {
			nextCalled = true
			return nil
		}

		err := hook.afterLLMResponseHook(ctx, hookCtx, next)

		assert.NoError(t, err, "Should not error with missing span ID")
		assert.True(t, nextCalled, "next() should be called")
	})

	t.Run("afterToolExecutionHook handles missing span ID", func(t *testing.T) {
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
		sessionID := uuid.New()
		hookCtx := &hooks.HookContext{
			SessionID:  sessionID,
			ToolName:   "test-tool",
			ToolResult: map[string]interface{}{"result": "value"},
			Data:       make(map[string]interface{}),
			// No span_id set
		}

		nextCalled := false
		next := func() error {
			nextCalled = true
			return nil
		}

		err := hook.afterToolExecutionHook(ctx, hookCtx, next)

		assert.NoError(t, err, "Should not error with missing span ID")
		assert.True(t, nextCalled, "next() should be called")
	})

	t.Run("afterAgentSpawnHook handles missing span ID", func(t *testing.T) {
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
		sessionID := uuid.New()
		hookCtx := &hooks.HookContext{
			SessionID: sessionID,
			AgentID:   uuid.New(),
			Data:      make(map[string]interface{}),
			// No span_id set
		}

		nextCalled := false
		next := func() error {
			nextCalled = true
			return nil
		}

		err := hook.afterAgentSpawnHook(ctx, hookCtx, next)

		assert.NoError(t, err, "Should not error with missing span ID")
		assert.True(t, nextCalled, "next() should be called")
	})
}

// TestLangfuseHook_NilContextHandling tests hooks with nil HookContext
func TestLangfuseHook_NilContextHandling(t *testing.T) {
	t.Run("beforeLLMRequestHook handles nil HookContext.Data", func(t *testing.T) {
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
			SessionID: uuid.New(),
			LLMModel:  "gpt-4",
			LLMInput:  "test input",
			Data:      nil, // nil Data
		}

		nextCalled := false
		next := func() error {
			nextCalled = true
			return nil
		}

		err := hook.beforeLLMRequestHook(ctx, hookCtx, next)

		assert.NoError(t, err, "Should not error with nil Data")
		assert.True(t, nextCalled, "next() should be called")
	})

	t.Run("propagateTraceID handles nil HookContext.Data", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := mocks.NewMockLoggerService(ctrl)

		cfg := &config.LangfuseConfig{
			LangfuseEnabled: true,
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

		// Create a trace context
		hook.createTraceContext(sessionID)

		// propagateTraceID with nil Data should not panic
		hookCtx := &hooks.HookContext{
			SessionID: sessionID,
			Data:      nil, // nil Data
		}

		// This should not panic
		hook.propagateTraceID(hookCtx)

		assert.Nil(t, hookCtx.Data, "Data should remain nil")
	})
}
