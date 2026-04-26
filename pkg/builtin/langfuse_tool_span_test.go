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

			if tt.sessionID != uuid.Nil {
				hook.createTraceContext(tt.sessionID)
			}

			hookCtx := hooks.NewTypedHookContext(
				shared.SessionContext{SessionID: tt.sessionID},
				hooks.ToolPayload{Name: shared.ToolName(tt.toolName), Args: tt.toolArgs},
			)

			err := hook.beforeToolExecutionHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			spanID := hookCtx.Tracing.SpanID

			if tt.wantSpanCreated {
				assert.NotEmpty(t, spanID, "Span ID should not be empty")

				tc := hook.getTraceContext(tt.sessionID)
				require.NotNil(t, tc, "TraceContext should exist")
				assert.Contains(t, tc.Spans, spanID, "Span should be in TraceContext.Spans")

				spanCtx, ok := tc.Spans[spanID].(*ToolSpanContext)
				require.True(t, ok, "Span should be ToolSpanContext type")
				assert.Equal(t, shared.ToolNameReadFile, spanCtx.ToolName)
				assert.Equal(t, tt.toolArgs, spanCtx.Input)
				assert.False(t, spanCtx.StartTime.IsZero(), "StartTime should be set")
			} else {
				assert.Empty(t, spanID, "Should not have span ID when disabled or nil session")
			}
		})
	}
}

// TestLangfuseHook_ToolSpanUpdate tests span update in afterToolExecutionHook
func TestLangfuseHook_ToolSpanUpdate(t *testing.T) {
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

	hook.createTraceContext(sessionID)
	tc := hook.getTraceContext(sessionID)
	spanID := uuid.New()

	startTime := time.Now().Add(-50 * time.Millisecond)
	tc.Spans[spanID.String()] = &ToolSpanContext{
		StartTime: startTime,
		ToolName:  "read_file",
		Input:     map[string]any{"path": "/test.txt"},
	}

	hookCtx := hooks.NewTypedHookContextWithTracing(
		shared.SessionContext{SessionID: sessionID},
		hooks.ToolPayload{Result: map[string]any{"content": "hello world"}},
		hooks.TracingPayload{SpanID: spanID.String()},
	)

	err := hook.afterToolExecutionHook(context.Background(), hookCtx, func() error { return nil })
	require.NoError(t, err)

	spanCtx, ok := tc.Spans[spanID.String()].(*ToolSpanContext)
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
			mockLog := logger.NewMockLoggerService(ctrl)
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
				traceCtxs:   make(map[string]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			hook.createTraceContext(sessionID)
			tc := hook.getTraceContext(sessionID)
			spanID := uuid.New()

			tc.Spans[spanID.String()] = &ToolSpanContext{
				StartTime: time.Now().Add(-30 * time.Millisecond),
				ToolName:  "read_file",
				Input:     map[string]any{"path": "/test.txt"},
				Level:     traces.ObservationLevelDefault,
			}

			hookCtx := hooks.NewTypedHookContextWithTracing(
				shared.SessionContext{SessionID: sessionID},
				hooks.ToolPayload{Error: tt.toolError},
				hooks.TracingPayload{SpanID: spanID.String()},
			)

			err := hook.onToolErrorHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			spanCtx, ok := tc.Spans[spanID.String()].(*ToolSpanContext)
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
		name       string
		toolName   string
		toolArgs   map[string]any
		toolResult map[string]any
		toolError  error
		wantOutput map[string]any
		wantLevel  traces.ObservationLevel
		wantStatus string
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
			hook.createTraceContext(sessionID)
			tc := hook.getTraceContext(sessionID)

			// Simulate tool flow: BeforeToolExecution -> Tool execution -> AfterToolExecution/OnToolError
			hookCtx := hooks.NewTypedHookContext(
				shared.SessionContext{SessionID: sessionID},
				hooks.ToolPayload{
					Name:   shared.ToolName(tt.toolName),
					Args:   tt.toolArgs,
					Result: tt.toolResult,
					Error:  tt.toolError,
				},
			)

			// Call beforeToolExecutionHook (creates span)
			err := hook.beforeToolExecutionHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			// Get span ID from Tracing
			spanID := hookCtx.Tracing.SpanID
			require.NotEmpty(t, spanID, "Span ID should not be empty")

			// Verify initial span state
			spanCtx, ok := tc.Spans[spanID].(*ToolSpanContext)
			require.True(t, ok, "Span should be ToolSpanContext type")
			assert.Equal(t, shared.ToolName(tt.toolName), spanCtx.ToolName)
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
