package builtin

import (
	"context"
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/git-hulk/langfuse-go/pkg/traces"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestLangfuseHook_AgentSpawnSpanLifecycle(t *testing.T) {
	sessionID := uuid.New()
	parentAgentID := uuid.New()

	tests := []struct {
		name            string
		langfuseEnabled bool
		sessionID       uuid.UUID
		parentAgentID   uuid.UUID
		newAgentID      uuid.UUID
		wantSpanCreated bool
	}{
		{
			name:            "creates spawn span when enabled",
			langfuseEnabled: true,
			sessionID:       sessionID,
			parentAgentID:   parentAgentID,
			newAgentID:      uuid.MustParse("00000000-0000-0000-0000-000000000123"),
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
			mockLog := logger.NewMockLoggerService(ctrl)
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
				traceCtxs:   make(map[string]*TraceContext),
				traceCtxsMu: &sync.RWMutex{},
			}

			if tt.sessionID != uuid.Nil {
				hook.createTraceContext(tt.sessionID.String())
			}

			hookCtx := hooks.NewTypedHookContext(
				shared.LoggingContext{SessionID: tt.sessionID.String(), AgentID: tt.parentAgentID},
				hooks.AgentPayload{Event: hooks.AgentEventSpawn, NewAgentID: tt.newAgentID},
			)

			// Call before hook
			err := hook.beforeAgentSpawnHook(context.Background(), hookCtx, func() error { return nil })
			require.NoError(t, err)

			spanID := hookCtx.Tracing.SpanID

			if tt.wantSpanCreated {
				assert.NotEmpty(t, spanID, "Should have span ID")

				// Call after hook
				err = hook.afterAgentSpawnHook(context.Background(), hookCtx, func() error { return nil })
				require.NoError(t, err)

				tc := hook.getTraceContext(tt.sessionID.String())
				spanCtx, ok := tc.Spans[spanID].(*AgentSpanContext)
				require.True(t, ok, "Span should be AgentSpanContext type")
				assert.Equal(t, "spawn", spanCtx.EventType)
				assert.Equal(t, tt.parentAgentID.String(), spanCtx.ParentAgentID)
				assert.Equal(t, tt.newAgentID.String(), spanCtx.NewAgentID)
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

	mockLog := logger.NewMockLoggerService(ctrl)
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
		traceCtxs:   make(map[string]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	hook.createTraceContext(sessionID.String())

	hookCtx := hooks.NewTypedHookContext(
		shared.LoggingContext{SessionID: sessionID.String(), AgentID: agentID},
		hooks.AgentPayload{Event: hooks.AgentEventRemove},
	)

	// Call before remove hook
	err := hook.beforeAgentRemoveHook(context.Background(), hookCtx, func() error { return nil })
	require.NoError(t, err)

	spanID := hookCtx.Tracing.SpanID
	assert.NotEmpty(t, spanID, "Should have span ID")

	// Call after remove hook
	err = hook.afterAgentRemoveHook(context.Background(), hookCtx, func() error { return nil })
	require.NoError(t, err)

	tc := hook.getTraceContext(sessionID.String())
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
	childAgentID := uuid.MustParse("00000000-0000-0000-0000-000000000123")

	mockLog := logger.NewMockLoggerService(ctrl)
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
		traceCtxs:   make(map[string]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// Simulate session start (creates trace context)
	hook.createTraceContext(sessionID.String())
	tc := hook.getTraceContext(sessionID.String())
	require.NotNil(t, tc, "TraceContext should exist after session start")

	// Simulate agent spawn
	agentSpawnCtx := hooks.NewTypedHookContext(
		shared.LoggingContext{SessionID: sessionID.String(), AgentID: parentAgentID},
		hooks.AgentPayload{Event: hooks.AgentEventSpawn, NewAgentID: childAgentID},
	)

	err := hook.beforeAgentSpawnHook(context.Background(), agentSpawnCtx, func() error { return nil })
	require.NoError(t, err)

	agentSpanID := agentSpawnCtx.Tracing.SpanID

	err = hook.afterAgentSpawnHook(context.Background(), agentSpawnCtx, func() error { return nil })
	require.NoError(t, err)

	// Verify agent span
	agentSpan, ok := tc.Spans[agentSpanID].(*AgentSpanContext)
	require.True(t, ok, "Agent span should exist")
	assert.Equal(t, "spawn", agentSpan.EventType)
	assert.Equal(t, parentAgentID.String(), agentSpan.ParentAgentID)
	assert.Equal(t, childAgentID.String(), agentSpan.NewAgentID)

	// Simulate tool execution under this agent
	toolCtx := hooks.NewTypedHookContext(
		shared.LoggingContext{SessionID: sessionID.String()},
		hooks.ToolPayload{Name: "read_file", Args: map[string]any{"path": "/test.txt"}},
	)

	err = hook.beforeToolExecutionHook(context.Background(), toolCtx, func() error { return nil })
	require.NoError(t, err)

	toolSpanID := toolCtx.Tracing.SpanID
	toolCtx.Payload.Result = map[string]any{"content": "file contents"}

	err = hook.afterToolExecutionHook(context.Background(), toolCtx, func() error { return nil })
	require.NoError(t, err)

	// Verify tool span
	toolSpan, ok := tc.Spans[toolSpanID].(*ToolSpanContext)
	require.True(t, ok, "Tool span should exist")
	assert.Equal(t, shared.ToolNameReadFile, toolSpan.ToolName)
	assert.Equal(t, map[string]any{"path": "/test.txt"}, toolSpan.Input)
	assert.Equal(t, map[string]any{"content": "file contents"}, toolSpan.Output)

	// Simulate LLM call under this agent
	llmCtx := hooks.NewTypedHookContext(
		shared.LoggingContext{SessionID: sessionID.String()},
		hooks.LLMPayload{Model: "claude-3-5-sonnet", Input: "Hello, world!"},
	)

	err = hook.beforeLLMRequestHook(context.Background(), llmCtx, func() error { return nil })
	require.NoError(t, err)

	llmSpanID := llmCtx.Tracing.SpanID
	llmCtx.Payload.Response = "Hi there!"

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
	removeCtx := hooks.NewTypedHookContext(
		shared.LoggingContext{SessionID: sessionID.String(), AgentID: parentAgentID},
		hooks.AgentPayload{Event: hooks.AgentEventRemove},
	)

	err = hook.beforeAgentRemoveHook(context.Background(), removeCtx, func() error { return nil })
	require.NoError(t, err)

	removeSpanID := removeCtx.Tracing.SpanID

	err = hook.afterAgentRemoveHook(context.Background(), removeCtx, func() error { return nil })
	require.NoError(t, err)

	// Verify removal span
	removeSpan, ok := tc.Spans[removeSpanID].(*AgentSpanContext)
	require.True(t, ok, "Remove span should exist")
	assert.Equal(t, "remove", removeSpan.EventType)

	// Final verification: 4 spans total (spawn, tool, LLM, remove)
	assert.Len(t, tc.Spans, 4, "Should have 4 spans after full lifecycle")

	// Simulate session end (removes trace context)
	hook.removeTraceContext(sessionID.String())
	tc = hook.getTraceContext(sessionID.String())
	assert.Nil(t, tc, "TraceContext should be removed after session end")
}
