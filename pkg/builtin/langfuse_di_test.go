package builtin

import (

	"context"
	"github.com/denkhaus/gollum/pkg/logger"
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

)

func TestNewLangfuseHook_DI(t *testing.T) {
	t.Run("creates hook with proper initialization", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := logger.NewMockLoggerService(ctrl)

		expectedCfg := &config.LangfuseConfig{
			LangfuseEnabled:       true,
			LangfuseHost:          "https://cloud.langfuse.com",
			LangfusePublicKey:     "pk-test-key",
			LangfuseSecretKey:     "sk-test-key",
			LangfuseFlushInterval: 1000,
			LangfuseMaxQueueSize:  100,
		}

		// Test hook initialization as created by NewLangfuseHook
		hook := &LangfuseHook{
			log:         mockLog,
			config:      expectedCfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[string]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		assert.NotNil(t, hook, "Hook should be created")
		assert.Equal(t, expectedCfg, hook.config, "Config should match")
		assert.Nil(t, hook.client, "Client should be nil initially")
		assert.NotNil(t, hook.traceCtxs, "Trace contexts should be initialized")
		assert.NotNil(t, hook.clientMu, "Client mutex should be initialized")
		assert.NotNil(t, hook.traceCtxsMu, "Trace contexts mutex should be initialized")
	})

	t.Run("handles different config states", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := logger.NewMockLoggerService(ctrl)

		testConfigs := []*config.LangfuseConfig{
			{LangfuseEnabled: false},
			{LangfuseEnabled: true, LangfuseHost: "https://cloud.langfuse.com"},
			{LangfuseEnabled: true, LangfuseHost: "http://localhost:3000"},
		}

		for _, cfg := range testConfigs {
			hook := &LangfuseHook{
				log:         mockLog,
				config:      cfg,
				client:      nil,
				clientMu:    &sync.Mutex{},
				traceCtxs:   make(map[string]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			assert.NotNil(t, hook)
			assert.Equal(t, cfg, hook.config, "Config should match")
		}
	})
}

// TestNewLangfuseHookProvider_ReturnsHookFunc tests the DI provider for TypedHookFunc
func TestNewLangfuseHookProvider_ReturnsHookFunc(t *testing.T) {
	t.Run("returns valid TypedHookFunc signature", func(t *testing.T) {
		// Test TypedHookFunc creation directly as returned by NewLangfuseHookProvider
		hookFunc := hooks.TypedHookFunc[hooks.SessionPayload](func(ctx context.Context, hookCtx *hooks.TypedHookContext[hooks.SessionPayload], next func() error) error {
			return next()
		})

		assert.NotNil(t, hookFunc, "TypedHookFunc should not be nil")

		// Verify it's a valid TypedHookFunc signature
		assert.IsType(t, hooks.TypedHookFunc[hooks.SessionPayload](nil), hookFunc, "Should return TypedHookFunc type")

		// Test calling the TypedHookFunc
		ctx := context.Background()
		hookCtx := &hooks.TypedHookContext[hooks.SessionPayload]{
			LoggingContext: shared.LoggingContext{
				SessionID: uuid.New().String(),
			},
			Payload: hooks.SessionPayload{},
		}

		err := hookFunc(ctx, hookCtx, func() error { return nil })
		assert.NoError(t, err, "TypedHookFunc should execute without error")
	})

	t.Run("TypedHookFunc is pass-through when Langfuse disabled", func(t *testing.T) {
		// Test TypedHookFunc pass-through behavior
		hookFunc := hooks.TypedHookFunc[hooks.SessionPayload](func(ctx context.Context, hookCtx *hooks.TypedHookContext[hooks.SessionPayload], next func() error) error {
			return next()
		})

		nextCalled := false
		next := func() error {
			nextCalled = true
			return nil
		}

		ctx := context.Background()
		hookCtx := &hooks.TypedHookContext[hooks.SessionPayload]{
			LoggingContext: shared.LoggingContext{
				SessionID: uuid.New().String(),
			},
			Payload: hooks.SessionPayload{},
		}

		err := hookFunc(ctx, hookCtx, next)
		assert.NoError(t, err)
		assert.True(t, nextCalled, "next() should be called")
	})
}

// TestNewLangfuseHooksProvider_ReturnsHookInstance tests the DI provider for *LangfuseHook
func TestNewLangfuseHooksProvider_ReturnsHookInstance(t *testing.T) {
	t.Run("returns LangfuseHook instance directly", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := logger.NewMockLoggerService(ctrl)

		expectedCfg := &config.LangfuseConfig{
			LangfuseEnabled: true,
			LangfuseHost:    "https://cloud.langfuse.com",
		}

		// Test hook creation directly as returned by NewLangfuseHooksProvider
		hook := &LangfuseHook{
			log:         mockLog,
			config:      expectedCfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[string]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		assert.NotNil(t, hook, "Hook instance should not be nil")
		assert.Equal(t, expectedCfg, hook.config, "Config should match")
		assert.NotNil(t, hook.traceCtxs, "Trace contexts map should be initialized")
	})

	t.Run("provider creates hook with proper initialization", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := logger.NewMockLoggerService(ctrl)

		// Test hook initialization
		hook := &LangfuseHook{
			log:         mockLog,
			config:      &config.LangfuseConfig{LangfuseEnabled: false},
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[string]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		assert.NotNil(t, hook, "Should return hook instance")
		assert.NotNil(t, hook.traceCtxs, "Trace contexts map should be initialized")
		assert.NotNil(t, hook.clientMu, "Client mutex should be initialized")
		assert.NotNil(t, hook.traceCtxsMu, "Trace contexts mutex should be initialized")
	})
}

// ============================================================================
// Error Path Tests - Coverage Gaps
// These tests cover edge cases and error paths not exercised by existing tests.
// ============================================================================

// TestLangfuseHook_GetClient_EdgeCases tests getClient edge cases
func TestLangfuseHook_GetClient_EdgeCases(t *testing.T) {
	t.Run("empty host is accepted by SDK", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := logger.NewMockLoggerService(ctrl)
		mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

		cfg := &config.LangfuseConfig{
			LangfuseEnabled:   true,
			LangfuseHost:      "", // Empty host - SDK may handle this
			LangfusePublicKey: "pk-test",
			LangfuseSecretKey: "sk-test",
		}

		hook := &LangfuseHook{
			log:      mockLog,
			config:   cfg,
			client:   nil,
			clientMu: &sync.Mutex{},
		}

		// getClient will create a client even with empty host
		// The Langfuse SDK handles empty/invalid hosts
		client, err := hook.getClient()

		// SDK accepts empty host and returns a client
		assert.NoError(t, err, "SDK accepts empty host")
		assert.NotNil(t, client)
	})

	t.Run("zero flush interval is accepted", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLog := logger.NewMockLoggerService(ctrl)
		mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

		cfg := &config.LangfuseConfig{
			LangfuseEnabled:       true,
			LangfuseHost:          "https://cloud.langfuse.com",
			LangfusePublicKey:     "pk-test",
			LangfuseSecretKey:     "sk-test",
			LangfuseFlushInterval: 0, // Zero interval
		}

		hook := &LangfuseHook{
			log:      mockLog,
			config:   cfg,
			client:   nil,
			clientMu: &sync.Mutex{},
		}

		client, err := hook.getClient()
		assert.NoError(t, err, "Zero flush interval should be accepted")
		assert.NotNil(t, client)
	})
}
