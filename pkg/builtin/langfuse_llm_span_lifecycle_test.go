package builtin

import (

	"context"
	"fmt"
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

func TestLangfuseHook_OnLLMError(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name            string
		langfuseEnabled bool
		sessionID       uuid.UUID
		llmError        error
		wantErrorLevel  bool
		wantStatusMsg   string
		skipSpanID      bool
	}{
		{
			name:            "marks span as ERROR with error message",
			langfuseEnabled: true,
			sessionID:       sessionID,
			llmError:        fmt.Errorf("rate limit exceeded"),
			wantErrorLevel:  true,
			wantStatusMsg:   "rate limit exceeded",
		},
		{
			name:            "skips marking when disabled",
			langfuseEnabled: false,
			sessionID:       sessionID,
			llmError:        fmt.Errorf("rate limit exceeded"),
			wantErrorLevel:  false,
		},
		{
			name:            "handles nil error gracefully",
			langfuseEnabled: true,
			sessionID:       sessionID,
			llmError:        nil,
			wantErrorLevel:  true, // Still marks ERROR level
			wantStatusMsg:   "unknown error",
		},
		{
			name:            "skips when span ID missing",
			langfuseEnabled: true,
			sessionID:       sessionID,
			llmError:        fmt.Errorf("network error"),
			wantErrorLevel:  false, // Can't mark without span ID
			skipSpanID:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLog := logger.NewMockLoggerService(ctrl)
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
				traceCtxs:   make(map[string]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			// Create trace context and span
			hook.createTraceContext(sessionID.String())
			tc := hook.getTraceContext(sessionID.String())
			spanID := uuid.New().String()

			startTime := time.Now().Add(-50 * time.Millisecond)
			tc.Spans[spanID] = &LLMSpanContext{
				StartTime: startTime,
				Model:     "claude-3-5-sonnet",
				Input:     "Test prompt",
				Level:     traces.ObservationLevelDefault,
			}

			// Create TypedHookContext with LLMPayload
			hookCtx := hooks.NewTypedHookContext(
				shared.LoggingContext{SessionID: tt.sessionID.String()},
				hooks.LLMPayload{
					Error:   tt.llmError,
					Options: make(map[string]any),
				},
			)

			// Add span ID to Tracing (except for missing span ID test)
			if !tt.skipSpanID {
				hookCtx.Tracing.SpanID = spanID
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
		name        string
		llmModel    string
		llmInput    string
		llmResponse string
		llmError    error
		wantOutput  string
		wantLevel   traces.ObservationLevel
		wantStatus  string
	}{
		{
			name:        "successful LLM call",
			llmModel:    "claude-3-5-sonnet",
			llmInput:    "What is 2+2?",
			llmResponse: "4",
			llmError:    nil,
			wantOutput:  "4",
			wantLevel:   traces.ObservationLevelDefault,
			wantStatus:  "success",
		},
		{
			name:        "failed LLM call",
			llmModel:    "claude-3-5-sonnet",
			llmInput:    "What is the meaning of life?",
			llmResponse: "",
			llmError:    fmt.Errorf("timeout"),
			wantOutput:  "",
			wantLevel:   traces.ObservationLevelError,
			wantStatus:  "timeout",
		},
		{
			name:        "LLM call with partial response then error",
			llmModel:    "claude-3-5-sonnet",
			llmInput:    "Generate a long story",
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
			mockLog := logger.NewMockLoggerService(ctrl)
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
				traceCtxs:   make(map[string]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			// Create trace context
			hook.createTraceContext(sessionID.String())
			tc := hook.getTraceContext(sessionID.String())

			// Simulate LLM flow: BeforeLLMRequest -> LLM call -> AfterLLMResponse/OnLLMError
			hookCtx := hooks.NewTypedHookContext(
				shared.LoggingContext{SessionID: sessionID.String()},
				hooks.LLMPayload{
					Model:    tt.llmModel,
					Input:    tt.llmInput,
					Response: tt.llmResponse,
					Error:    tt.llmError,
					Options:  make(map[string]any),
				},
			)

			// Call beforeLLMRequestHook (creates span)
			err := hook.beforeLLMRequestHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			// Get span ID from Tracing
			spanID := hookCtx.Tracing.SpanID
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
