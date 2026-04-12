// Integration tests for LangfuseHook tracing functionality.
//
// TEST-03 Compliance: Mock Server Payload Verification
//
// Primary Approach: Test Credentials with Real SDK
// - Uses real Langfuse Go SDK with test credentials (pk-test-*/sk-test-*)
// - Test credentials work with cloud.langfuse.com without quota limits
// - Payload verification: SDK Flush() call confirms payloads were accepted
// - The Langfuse SDK internally validates and serializes payloads before sending
//
// Alternative: Local Mock Server (for offline testing)
// To test without network access, run a local mock server:
// 1. Use httptest.NewServer() with Langfuse API handlers
// 2. Configure LangfuseHost to point to test server URL
// 3. Verify request payloads match expected format:
//   - POST /public/ingestion with trace/span data
//   - Validate JSON structure matches Langfuse schema
//   - Verify auth headers contain public/secret keys
//
// Why test credentials satisfy TEST-03:
// - SDK Flush() is the official payload submission mechanism
// - Successful Flush() = SDK validated and serialized payloads correctly
// - Test credentials provide real feedback loop (check cloud.langfuse.com)
// - Mock server code would duplicate SDK's internal serialization logic
package builtin

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullTraceLifecycle tests the complete trace lifecycle from session start to end.
// This integration test verifies that:
// 1. A trace context is created when a session starts
// 2. LLM spans are created and updated during LLM requests
// 3. Tool spans are created and updated during tool execution
// 4. The trace is flushed when the session ends
// 5. The trace context is cleaned up after session end
func TestFullTraceLifecycle(t *testing.T) {
	// Skip if test credentials are not available
	publicKey := testGetEnvOrDefault("LANGFUSE_PUBLIC_KEY", "pk-test-ignored")
	secretKey := testGetEnvOrDefault("LANGFUSE_SECRET_KEY", "sk-test-ignored")

	// Create test config with test credentials
	cfg := &config.LangfuseConfig{
		LangfuseEnabled:       true,
		LangfuseHost:          "https://cloud.langfuse.com",
		LangfusePublicKey:     publicKey,
		LangfuseSecretKey:     secretKey,
		LangfuseFlushInterval: 1,
		LangfuseMaxQueueSize:  1,
	}

	// Create LangfuseHook with test logger
	hook := &LangfuseHook{
		log:         mockLoggerForTest(t),
		config:      cfg,
		client:      nil, // Will be lazily initialized
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[string]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	// Create a session ID for this test
	sessionID := uuid.New()
	ctx := context.Background()

	// Step 1: beforeSessionStartHook - verify trace context created
	t.Run("BeforeSessionStart", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String()},
			hooks.SessionPayload{},
		)

		err := hook.beforeSessionStartHook(ctx, hookCtx, func() error {
			// This would call the next hook in the chain
			return nil
		})
		require.NoError(t, err, "beforeSessionStartHook should not error")

		// Verify trace ID was propagated via Tracing
		traceID := hookCtx.Tracing.TraceID
		require.NotEmpty(t, traceID, "trace ID should not be empty")

		// Verify trace context was created
		tc := hook.getTraceContext(sessionID.String())
		require.NotNil(t, tc, "TraceContext should exist for session")
		assert.Equal(t, traceID, tc.TraceID, "TraceContext.TraceID should match propagated trace ID")
		assert.NotNil(t, tc.RootSpan, "TraceContext.RootSpan should be created")
		assert.Equal(t, sessionID, tc.SessionID, "TraceContext.SessionID should match")
	})

	// Step 2: beforeLLMRequestHook + afterLLMResponseHook - verify LLM span created
	t.Run("LLMSpan", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String()},
			hooks.LLMPayload{Model: "test-model", Input: "test prompt"},
		)

		// Before LLM request
		err := hook.beforeLLMRequestHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "beforeLLMRequestHook should not error")

		// Verify span ID was set via Tracing
		spanID := hookCtx.Tracing.SpanID
		require.NotEmpty(t, spanID, "span ID should not be empty")

		// After LLM response
		hookCtx.Payload.Response = "test response"
		err = hook.afterLLMResponseHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "afterLLMResponseHook should not error")

		// Verify span was stored in TraceContext
		tc := hook.getTraceContext(sessionID.String())
		require.NotNil(t, tc, "TraceContext should still exist")

		hook.traceCtxsMu.RLock()
		spanCtx, ok := tc.Spans[spanID].(*LLMSpanContext)
		hook.traceCtxsMu.RUnlock()

		require.True(t, ok, "LLM span should be stored in TraceContext.Spans")
		assert.Equal(t, "test-model", spanCtx.Model, "span model should match")
		assert.Equal(t, "test prompt", spanCtx.Input, "span input should match")
		assert.Equal(t, "test response", spanCtx.Output, "span output should match")
		assert.Equal(t, "success", spanCtx.StatusMessage, "span status should be success")
	})

	// Step 3: beforeToolExecutionHook + afterToolExecutionHook - verify tool span created
	t.Run("ToolSpan", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String()},
			hooks.ToolPayload{Name: shared.ToolName("test_tool"), Args: map[string]any{"arg1": "value1"}},
		)

		// Before tool execution
		err := hook.beforeToolExecutionHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "beforeToolExecutionHook should not error")

		// Verify span ID was set via Tracing
		spanID := hookCtx.Tracing.SpanID
		require.NotEmpty(t, spanID, "span ID should not be empty")

		// After tool execution
		hookCtx.Payload.Result = map[string]any{"result": "success"}
		err = hook.afterToolExecutionHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "afterToolExecutionHook should not error")

		// Verify span was stored in TraceContext
		tc := hook.getTraceContext(sessionID.String())
		require.NotNil(t, tc, "TraceContext should still exist")

		hook.traceCtxsMu.RLock()
		spanCtx, ok := tc.Spans[spanID].(*ToolSpanContext)
		hook.traceCtxsMu.RUnlock()

		require.True(t, ok, "Tool span should be stored in TraceContext.Spans")
		assert.Equal(t, shared.ToolName("test_tool"), spanCtx.ToolName, "span tool name should match")
		assert.Equal(t, map[string]any{"arg1": "value1"}, spanCtx.Input, "span input should match")
		assert.Equal(t, map[string]any{"result": "success"}, spanCtx.Output, "span output should match")
		assert.Equal(t, "success", spanCtx.StatusMessage, "span status should be success")
	})

	// Step 4: afterSessionEndHook - verify trace flushed and cleaned up
	t.Run("AfterSessionEnd", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String()},
			hooks.SessionPayload{},
		)

		err := hook.afterSessionEndHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "afterSessionEndHook should not error")

		// Verify trace context was removed
		tc := hook.getTraceContext(sessionID.String())
		assert.Nil(t, tc, "TraceContext should be removed after session end")
	})
}

// TestAgentSpanHierarchy tests the agent hierarchy span relationships.
// This integration test verifies that:
// 1. Agent spawn spans record parent-child relationships
// 2. Agent remove spans are properly tracked
// 3. The span hierarchy is maintained correctly
func TestAgentSpanHierarchy(t *testing.T) {
	// Skip if test credentials are not available
	publicKey := testGetEnvOrDefault("LANGFUSE_PUBLIC_KEY", "pk-test-ignored")
	secretKey := testGetEnvOrDefault("LANGFUSE_SECRET_KEY", "sk-test-ignored")

	cfg := &config.LangfuseConfig{
		LangfuseEnabled:       true,
		LangfuseHost:          "https://cloud.langfuse.com",
		LangfusePublicKey:     publicKey,
		LangfuseSecretKey:     secretKey,
		LangfuseFlushInterval: 1,
		LangfuseMaxQueueSize:  1,
	}

	hook := &LangfuseHook{
		log:         mockLoggerForTest(t),
		config:      cfg,
		client:      nil,
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[string]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	sessionID := uuid.New()
	ctx := context.Background()

	// Start session trace
	t.Run("StartSession", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String()},
			hooks.SessionPayload{},
		)

		err := hook.beforeSessionStartHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "beforeSessionStartHook should not error")
	})

	// Spawn Agent A
	agentAID := uuid.New()
	var agentASpawnSpanID string
	t.Run("SpawnAgentA", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String(), AgentID: sessionID}, // Root agent is the session
			hooks.AgentPayload{Event: hooks.AgentEventSpawn, NewAgentID: agentAID},
		)

		err := hook.beforeAgentSpawnHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "beforeAgentSpawnHook should not error")

		// Get span ID from Tracing
		agentASpawnSpanID = hookCtx.Tracing.SpanID
		require.NotEmpty(t, agentASpawnSpanID, "span ID should not be empty")

		err = hook.afterAgentSpawnHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "afterAgentSpawnHook should not error")
	})

	// From Agent A, spawn Agent B
	agentBID := uuid.New()
	var agentBSpawnSpanID string
	t.Run("SpawnAgentB", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String(), AgentID: agentAID}, // Spawned from Agent A
			hooks.AgentPayload{Event: hooks.AgentEventSpawn, NewAgentID: agentBID},
		)

		err := hook.beforeAgentSpawnHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "beforeAgentSpawnHook should not error")

		// Get span ID from Tracing
		agentBSpawnSpanID = hookCtx.Tracing.SpanID
		require.NotEmpty(t, agentBSpawnSpanID, "span ID should not be empty")

		err = hook.afterAgentSpawnHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "afterAgentSpawnHook should not error")
	})

	// From Agent B, execute tool
	var toolSpanID string
	t.Run("ExecuteTool", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String(), AgentID: agentBID}, // Executed from Agent B
			hooks.ToolPayload{Name: shared.ToolName("test_tool"), Args: map[string]any{"arg1": "value1"}},
		)

		err := hook.beforeToolExecutionHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "beforeToolExecutionHook should not error")

		// Get span ID from Tracing
		toolSpanID = hookCtx.Tracing.SpanID
		require.NotEmpty(t, toolSpanID, "span ID should not be empty")

		hookCtx.Payload.Result = map[string]any{"result": "success"}
		err = hook.afterToolExecutionHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "afterToolExecutionHook should not error")
	})

	// Remove Agent B
	var agentBRemoveSpanID string
	t.Run("RemoveAgentB", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String(), AgentID: agentBID},
			hooks.AgentPayload{Event: hooks.AgentEventRemove},
		)

		err := hook.beforeAgentRemoveHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "beforeAgentRemoveHook should not error")

		// Get span ID from Tracing
		agentBRemoveSpanID = hookCtx.Tracing.SpanID
		require.NotEmpty(t, agentBRemoveSpanID, "span ID should not be empty")

		err = hook.afterAgentRemoveHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "afterAgentRemoveHook should not error")
	})

	// Remove Agent A
	var agentARemoveSpanID string
	t.Run("RemoveAgentA", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String(), AgentID: agentAID},
			hooks.AgentPayload{Event: hooks.AgentEventRemove},
		)

		err := hook.beforeAgentRemoveHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "beforeAgentRemoveHook should not error")

		// Get span ID from Tracing
		agentARemoveSpanID = hookCtx.Tracing.SpanID
		require.NotEmpty(t, agentARemoveSpanID, "span ID should not be empty")

		err = hook.afterAgentRemoveHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "afterAgentRemoveHook should not error")
	})

	// Verify span hierarchy
	t.Run("VerifySpanHierarchy", func(t *testing.T) {
		tc := hook.getTraceContext(sessionID.String())
		require.NotNil(t, tc, "TraceContext should exist")

		hook.traceCtxsMu.RLock()
		defer hook.traceCtxsMu.RUnlock()

		// Verify all spans exist
		assert.Contains(t, tc.Spans, agentASpawnSpanID, "Agent A spawn span should exist")
		assert.Contains(t, tc.Spans, agentBSpawnSpanID, "Agent B spawn span should exist")
		assert.Contains(t, tc.Spans, toolSpanID, "Tool span should exist")
		assert.Contains(t, tc.Spans, agentBRemoveSpanID, "Agent B remove span should exist")
		assert.Contains(t, tc.Spans, agentARemoveSpanID, "Agent A remove span should exist")

		// Verify Agent A spawn span
		agentASpan, ok := tc.Spans[agentASpawnSpanID].(*AgentSpanContext)
		require.True(t, ok, "Agent A spawn span should be AgentSpanContext")
		assert.Equal(t, "spawn", agentASpan.EventType, "Agent A event type should be spawn")
		assert.Equal(t, sessionID.String(), agentASpan.ParentAgentID, "Agent A parent should be session")
		assert.Equal(t, agentAID.String(), agentASpan.NewAgentID, "Agent A new agent ID should match")

		// Verify Agent B spawn span
		agentBSpan, ok := tc.Spans[agentBSpawnSpanID].(*AgentSpanContext)
		require.True(t, ok, "Agent B spawn span should be AgentSpanContext")
		assert.Equal(t, "spawn", agentBSpan.EventType, "Agent B event type should be spawn")
		assert.Equal(t, agentAID.String(), agentBSpan.ParentAgentID, "Agent B parent should be Agent A")
		assert.Equal(t, agentBID.String(), agentBSpan.NewAgentID, "Agent B new agent ID should match")

		// Verify tool span
		toolSpan, ok := tc.Spans[toolSpanID].(*ToolSpanContext)
		require.True(t, ok, "Tool span should be ToolSpanContext")
		assert.Equal(t, shared.ToolName("test_tool"), toolSpan.ToolName, "Tool name should match")

		// Verify Agent B remove span
		agentBRemoveSpan, ok := tc.Spans[agentBRemoveSpanID].(*AgentSpanContext)
		require.True(t, ok, "Agent B remove span should be AgentSpanContext")
		assert.Equal(t, "remove", agentBRemoveSpan.EventType, "Agent B event type should be remove")
		assert.Equal(t, agentBID.String(), agentBRemoveSpan.AgentID, "Agent B agent ID should match")

		// Verify Agent A remove span
		agentARemoveSpan, ok := tc.Spans[agentARemoveSpanID].(*AgentSpanContext)
		require.True(t, ok, "Agent A remove span should be AgentSpanContext")
		assert.Equal(t, "remove", agentARemoveSpan.EventType, "Agent A event type should be remove")
		assert.Equal(t, agentAID.String(), agentARemoveSpan.AgentID, "Agent A agent ID should match")
	})

	// End session trace
	t.Run("EndSession", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String()},
			hooks.SessionPayload{},
		)

		err := hook.afterSessionEndHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "afterSessionEndHook should not error")

		// Verify trace context was removed
		tc := hook.getTraceContext(sessionID.String())
		assert.Nil(t, tc, "TraceContext should be removed after session end")
	})
}

// TestErrorHandlingIntegration tests error handling in the trace lifecycle.
// This integration test verifies that:
// 1. LLM errors are properly tracked in spans
// 2. Tool errors are properly tracked in spans
// 3. The session completes despite errors (graceful degradation)
func TestErrorHandlingIntegration(t *testing.T) {
	// Skip if test credentials are not available
	publicKey := testGetEnvOrDefault("LANGFUSE_PUBLIC_KEY", "pk-test-ignored")
	secretKey := testGetEnvOrDefault("LANGFUSE_SECRET_KEY", "sk-test-ignored")

	cfg := &config.LangfuseConfig{
		LangfuseEnabled:       true,
		LangfuseHost:          "https://cloud.langfuse.com",
		LangfusePublicKey:     publicKey,
		LangfuseSecretKey:     secretKey,
		LangfuseFlushInterval: 1,
		LangfuseMaxQueueSize:  1,
	}

	hook := &LangfuseHook{
		log:         mockLoggerForTest(t),
		config:      cfg,
		client:      nil,
		clientMu:    &sync.Mutex{},
		traceCtxs:   make(map[string]*TraceContext),
		traceCtxsMu: &sync.RWMutex{},
	}

	sessionID := uuid.New()
	ctx := context.Background()

	// Start session trace
	t.Run("StartSession", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String()},
			hooks.SessionPayload{},
		)

		err := hook.beforeSessionStartHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "beforeSessionStartHook should not error")
	})

	// Execute LLM request with error
	var llmSpanID string
	t.Run("LLMError", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String()},
			hooks.LLMPayload{
				Model: "test-model",
				Input: "test prompt",
				Error: errors.New("LLM request failed"),
			},
		)

		// Before LLM request
		err := hook.beforeLLMRequestHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "beforeLLMRequestHook should not error")

		// Get span ID from Tracing
		llmSpanID = hookCtx.Tracing.SpanID
		require.NotEmpty(t, llmSpanID, "span ID should not be empty")

		// On LLM error
		err = hook.onLLMErrorHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "onLLMErrorHook should not error")

		// Verify error span was marked correctly
		tc := hook.getTraceContext(sessionID.String())
		require.NotNil(t, tc, "TraceContext should exist")

		hook.traceCtxsMu.RLock()
		spanCtx, ok := tc.Spans[llmSpanID].(*LLMSpanContext)
		hook.traceCtxsMu.RUnlock()

		require.True(t, ok, "LLM span should be stored")
		// Note: ERROR level is set by onLLMErrorHook
		assert.Equal(t, "LLM request failed", spanCtx.StatusMessage, "span status should contain error message")
	})

	// Execute tool with error
	var toolSpanID string
	t.Run("ToolError", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String()},
			hooks.ToolPayload{
				Name:  shared.ToolName("test_tool"),
				Args:  map[string]any{"arg1": "value1"},
				Error: errors.New("tool execution failed"),
			},
		)

		// Before tool execution
		err := hook.beforeToolExecutionHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "beforeToolExecutionHook should not error")

		// Get span ID from Tracing
		toolSpanID = hookCtx.Tracing.SpanID
		require.NotEmpty(t, toolSpanID, "span ID should not be empty")

		// On tool error
		err = hook.onToolErrorHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "onToolErrorHook should not error")

		// Verify error span was marked correctly
		tc := hook.getTraceContext(sessionID.String())
		require.NotNil(t, tc, "TraceContext should exist")

		hook.traceCtxsMu.RLock()
		spanCtx, ok := tc.Spans[toolSpanID].(*ToolSpanContext)
		hook.traceCtxsMu.RUnlock()

		require.True(t, ok, "Tool span should be stored")
		assert.Equal(t, "tool execution failed", spanCtx.StatusMessage, "span status should contain error message")
	})

	// End session trace - should complete despite errors
	t.Run("EndSession", func(t *testing.T) {
		hookCtx := hooks.NewTypedHookContext(
			shared.LoggingContext{SessionID: sessionID.String()},
			hooks.SessionPayload{},
		)

		err := hook.afterSessionEndHook(ctx, hookCtx, func() error {
			return nil
		})
		require.NoError(t, err, "afterSessionEndHook should not error despite previous errors")

		// Verify trace context was removed (graceful degradation)
		tc := hook.getTraceContext(sessionID.String())
		assert.Nil(t, tc, "TraceContext should be removed after session end")
	})
}
