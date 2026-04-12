package builtin

import (

	"context"
	"github.com/denkhaus/gollum/pkg/logger"
	"sync"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/git-hulk/langfuse-go/pkg/traces"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

)

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
			mockLog := logger.NewMockLoggerService(ctrl)
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
				traceCtxs:   make(map[string]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			// Create trace context if needed
			if tt.sessionID != uuid.Nil {
				hook.createTraceContext(tt.sessionID.String())
			}

			// Create TypedHookContext with LLMPayload
			hookCtx := hooks.NewTypedHookContext(
				shared.LoggingContext{SessionID: tt.sessionID.String()},
				hooks.LLMPayload{
					Model:   tt.llmModel,
					Input:   tt.llmInput,
					Options: make(map[string]any),
				},
			)

			// Call beforeLLMRequestHook
			err := hook.beforeLLMRequestHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			// Verify span creation via Tracing.SpanID
			spanID := hookCtx.Tracing.SpanID

			if tt.wantSpanCreated {
				assert.NotEmpty(t, spanID, "Span ID should not be empty")

				// Verify span stored in TraceContext
				tc := hook.getTraceContext(tt.sessionID.String())
				require.NotNil(t, tc, "TraceContext should exist")
				assert.Contains(t, tc.Spans, spanID, "Span should be in TraceContext.Spans")

				// Verify span context data
				spanCtx, ok := tc.Spans[spanID].(*LLMSpanContext)
				require.True(t, ok, "Span should be LLMSpanContext type")
				assert.Equal(t, tt.llmModel, spanCtx.Model)
				assert.Equal(t, tt.llmInput, spanCtx.Input)
				assert.False(t, spanCtx.StartTime.IsZero(), "StartTime should be set")
			} else {
				assert.Empty(t, spanID, "Should not have span ID when disabled or nil session")
			}
		})
	}
}

// TestLangfuseHook_LLMSpanUpdate tests span update in afterLLMResponseHook
func TestLangfuseHook_LLMSpanUpdate(t *testing.T) {
	sessionID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLog := logger.NewMockLoggerService(ctrl)
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
		traceCtxs:   make(map[string]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// Create trace context and pre-populate with span
	hook.createTraceContext(sessionID.String())
	tc := hook.getTraceContext(sessionID.String())
	spanID := uuid.New().String()

	startTime := time.Now().Add(-100 * time.Millisecond)
	tc.Spans[spanID] = &LLMSpanContext{
		StartTime: startTime,
		Model:     "claude-3-5-sonnet",
		Input:     "Hello, world!",
	}

	// Create TypedHookContext with span ID in Tracing and response in Payload
	hookCtx := hooks.NewTypedHookContextWithTracing(
		shared.LoggingContext{SessionID: sessionID.String()},
		hooks.LLMPayload{
			Model:    "claude-3-5-sonnet",
			Response: "Hi there!",
			Options:  make(map[string]any),
		},
		hooks.TracingPayload{SpanID: spanID},
	)

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

	hook.createTraceContext(sessionID.String())

	// Create TypedHookContext without span ID in Tracing
	hookCtx := hooks.NewTypedHookContext(
		shared.LoggingContext{SessionID: sessionID.String()},
		hooks.LLMPayload{Response: "Response!"},
	)

	// Should not error, just skip
	err := hook.afterLLMResponseHook(context.Background(), hookCtx, func() error { return nil })
	assert.NoError(t, err, "Should not error when span ID is missing")
}

// TestLangfuseHook_LLMSpanUpdate_NilTraceContext tests afterLLMResponseHook with no trace context
func TestLangfuseHook_LLMSpanUpdate_NilTraceContext(t *testing.T) {
	sessionID := uuid.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	// Don't create trace context

	// Create TypedHookContext with span ID but no trace context
	hookCtx := hooks.NewTypedHookContextWithTracing(
		shared.LoggingContext{SessionID: sessionID.String()},
		hooks.LLMPayload{Response: "Response!"},
		hooks.TracingPayload{SpanID: "nonexistent-span"},
	)

	// Should not error, just skip
	err := hook.afterLLMResponseHook(context.Background(), hookCtx, func() error { return nil })
	assert.NoError(t, err, "Should not error when trace context is missing")
}
