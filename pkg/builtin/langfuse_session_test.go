package builtin

import (
	"context"
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/git-hulk/langfuse-go/pkg/traces"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

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
		name             string
		langfuseEnabled  bool
		sessionID        uuid.UUID
		wantTraceCreated bool
	}{
		{
			name:             "creates trace when enabled with valid session",
			langfuseEnabled:  true,
			sessionID:        sessionID,
			wantTraceCreated: true,
		},
		{
			name:             "skips trace when disabled",
			langfuseEnabled:  false,
			sessionID:        sessionID,
			wantTraceCreated: false,
		},
		{
			name:             "skips trace with nil session ID",
			langfuseEnabled:  true,
			sessionID:        uuid.Nil,
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

// TestLangfuseHook_FullTraceLifecycle tests complete trace lifecycle from session start to shutdown
func TestLangfuseHook_FullTraceLifecycle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sessionID := uuid.New()

	mockLog := mocks.NewMockLoggerService(ctrl)
	mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()

	cfg := &config.LangfuseConfig{
		LangfuseEnabled:   true,
		LangfuseHost:      "https://cloud.langfuse.com",
		LangfusePublicKey: "pk-test-key",
		LangfuseSecretKey: "sk-test-key",
	}

	hook := &LangfuseHook{
		log:         mockLog,
		config:      cfg,
		client:      nil, // Will be initialized lazily
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[uuid.UUID]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// Step 1: Create trace context manually (simulating session start)
	// Note: beforeSessionStartHook requires a real client, so we simulate
	// the post-session-start state where trace context exists.
	hook.createTraceContext(sessionID)

	tc := hook.getTraceContext(sessionID)
	require.NotNil(t, tc, "TraceContext should exist after session start")
	assert.NotEmpty(t, tc.TraceID, "TraceID should be set")

	// Step 2: LLM span
	llmCtx := &hooks.HookContext{
		SessionID: sessionID,
		LLMModel:  "claude-3-5-sonnet",
		LLMInput:  "Hello",
		Data:      make(map[string]any),
	}
	err := hook.beforeLLMRequestHook(context.Background(), llmCtx, func() error { return nil })
	require.NoError(t, err)

	llmSpanID, ok := llmCtx.Data["langfuse_span_id"].(string)
	require.True(t, ok, "Should have LLM span ID")
	assert.Contains(t, tc.Spans, llmSpanID, "LLM span should be in TraceContext")

	// Complete LLM span
	llmCtx.LLMResponse = "Hi there!"
	err = hook.afterLLMResponseHook(context.Background(), llmCtx, func() error { return nil })
	require.NoError(t, err)

	// Step 3: Tool span
	toolCtx := &hooks.HookContext{
		SessionID: sessionID,
		ToolName:  "read_file",
		ToolArgs:  map[string]any{"path": "/test.txt"},
		Data:      make(map[string]any),
	}
	err = hook.beforeToolExecutionHook(context.Background(), toolCtx, func() error { return nil })
	require.NoError(t, err)

	toolSpanID, ok := toolCtx.Data["langfuse_span_id"].(string)
	require.True(t, ok, "Should have tool span ID")
	assert.Contains(t, tc.Spans, toolSpanID, "Tool span should be in TraceContext")

	// Complete tool span
	toolCtx.ToolResult = map[string]any{"content": "hello"}
	err = hook.afterToolExecutionHook(context.Background(), toolCtx, func() error { return nil })
	require.NoError(t, err)

	// Verify span count
	assert.GreaterOrEqual(t, len(tc.Spans), 2, "Should have at least LLM and tool spans")

	// Step 4: Agent span
	agentCtx := &hooks.HookContext{
		SessionID: sessionID,
		AgentID:   uuid.New(),
		Data:      make(map[string]any),
	}
	err = hook.beforeAgentSpawnHook(context.Background(), agentCtx, func() error { return nil })
	require.NoError(t, err)

	agentSpanID, ok := agentCtx.Data["langfuse_span_id"].(string)
	require.True(t, ok, "Should have agent span ID")
	assert.Contains(t, tc.Spans, agentSpanID, "Agent span should be in TraceContext")

	// Step 5: Session end - simulate by removing trace context
	// (afterSessionEndHook would do this after flushing)
	hook.removeTraceContext(sessionID)

	// Verify cleanup
	tc = hook.getTraceContext(sessionID)
	assert.Nil(t, tc, "TraceContext should be removed after session end")

	// Step 6: Shutdown
	err = hook.Shutdown()
	require.NoError(t, err, "Shutdown should not return error")
}
