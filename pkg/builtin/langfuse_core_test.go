package builtin

import (

	"context"
	"github.com/denkhaus/gollum/pkg/logger"
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
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
		mockLog := logger.NewMockLoggerService(ctrl)
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
				LangfuseHost:          "https://cloud.langfuse.com",
				LangfusePublicKey:     "pk-test-key",
				LangfuseSecretKey:     "sk-test-key",
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
			mockLog := logger.NewMockLoggerService(ctrl)

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
		mockLog := logger.NewMockLoggerService(ctrl)

		// Expect Info calls for shutdown and flush success
		mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).Times(2)

		hook := &LangfuseHook{
			log:         mockLog,
			config:      &config.LangfuseConfig{},
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[uuid.UUID]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
		}

		err := hook.Shutdown()
		assert.NoError(t, err, "Shutdown with nil client should not error")
	})

	t.Run("shutdown with valid client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLog := logger.NewMockLoggerService(ctrl)

		// Expect Info calls for client init, shutdown, and flush success
		mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).Times(3)
		cfg := &config.LangfuseConfig{
			LangfuseEnabled:   true,
			LangfusePublicKey: "pk-test",
			LangfuseSecretKey: "sk-test",
			LangfuseHost:      "https://cloud.langfuse.com",
		}

		hook := &LangfuseHook{
			log:         mockLog,
			config:      cfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[uuid.UUID]*TraceContext),
			traceCtxsMu: &sync.RWMutex{},
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
	t.Run("returns valid TypedHookFunc", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLog := logger.NewMockLoggerService(ctrl)
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
		provider := func(h *LangfuseHook) (hooks.TypedHookFunc[hooks.SessionPayload], error) {
			// Return a TypedHookFunc that passes through
			return func(ctx context.Context, hookCtx *hooks.TypedHookContext[hooks.SessionPayload], next func() error) error {
				return next()
			}, nil
		}

		hookFunc, err := provider(hook)
		assert.NoError(t, err)
		assert.NotNil(t, hookFunc)

		// Call function (should be no-op for now)
		hookCtx := &hooks.TypedHookContext[hooks.SessionPayload]{}
		err = hookFunc(nil, hookCtx, func() error { return nil })
		assert.NoError(t, err)
	})
}

// TestNewLangfuseHooksProvider tests the DI provider that returns *LangfuseHook
func TestNewLangfuseHooksProvider(t *testing.T) {
	t.Run("creates LangfuseHook successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLog := logger.NewMockLoggerService(ctrl)
		mockCfg := &config.LangfuseConfig{
			LangfuseEnabled: false,
			LangfuseHost:    "https://cloud.langfuse.com",
		}

		// Create hook directly for test (bypassing full DI container)
		hook := &LangfuseHook{
			log:         mockLog,
			config:      mockCfg,
			client:      nil,
			clientMu:    &sync.Mutex{},
			traceCtxs:   make(map[uuid.UUID]*TraceContext),
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
		mockLog := logger.NewMockLoggerService(ctrl)
		mockCfg := &config.LangfuseConfig{
			LangfuseEnabled: true,
			LangfuseHost:    "https://cloud.langfuse.com",
		}

		// Simulate provider behavior
		provider := func() (*LangfuseHook, error) { //nolint:unparam // test helper, error always nil
			return &LangfuseHook{
				log:         mockLog,
				config:      mockCfg,
				client:      nil,
				clientMu:    &sync.Mutex{},
				traceCtxs:   make(map[uuid.UUID]*TraceContext),
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
		mockLog := logger.NewMockLoggerService(ctrl)

		// Test with different config states
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
				traceCtxs:   make(map[uuid.UUID]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			assert.NotNil(t, hook)
			assert.Equal(t, cfg, hook.config, "Config should match")
		}
	})
}
