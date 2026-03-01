package hooks

import (
	"context"
	"errors"
	"testing"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithLLMHooks tests the LLM hook functionality.
func TestWithLLMHooks(t *testing.T) {
	t.Run("executes LLM call without hooks when no hooks registered", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(p string) (string, error) {
			assert.Equal(t, prompt, p, "prompt should be passed unchanged")
			return testLLMResponse, nil
		})

		require.NoError(t, err)
		assert.Equal(t, testLLMResponse, response)
	})

	t.Run("BeforeLLMRequest hook can modify prompt", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, hc *TypedHookContext[LLMPayload], next func() error) error {
			// Modify the prompt
			hc.Payload.Input = "modified prompt"
			return next()
		}

		require.NoError(t, hm.RegisterLLMHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeLLMRequest, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := "original prompt"
		model := testLLMModel

		var receivedPrompt string
		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(p string) (string, error) {
			receivedPrompt = p
			return testLLMResponse, nil
		})

		require.NoError(t, err)
		assert.Equal(t, "modified prompt", receivedPrompt, "work function should receive modified prompt")
		assert.Equal(t, testLLMResponse, response)
	})

	t.Run("AfterLLMResponse hook can modify response", func(t *testing.T) {
		hm := newTestHookManager()

		afterHook := func(_ context.Context, hc *TypedHookContext[LLMPayload], _ func() error) error {
			// Modify the response
			hc.Payload.Response = "modified response"
			return nil
		}

		require.NoError(t, hm.RegisterLLMHook(afterHook, TypedHookMetadata{Name: "after", Point: AfterLLMResponse, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			return "original response", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "modified response", response, "should return modified response from hook")
	})

	t.Run("BeforeLLMRequest hook can block execution by not calling next", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, hc *TypedHookContext[LLMPayload], _ func() error) error {
			// Set a canned response and don't call next
			hc.Payload.Response = "canned response"
			return nil
		}

		require.NoError(t, hm.RegisterLLMHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeLLMRequest, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		workCalled := false
		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			workCalled = true
			return "should not see this", nil
		})

		require.NoError(t, err)
		assert.False(t, workCalled, "work function should not be called")
		assert.Equal(t, "canned response", response, "should return canned response from hook")
	})

	t.Run("OnLLMError hook can recover from error", func(t *testing.T) {
		hm := newTestHookManager()

		errorHook := func(_ context.Context, hc *TypedHookContext[LLMPayload], _ func() error) error {
			// Provide fallback response
			hc.Payload.Response = "fallback response"
			return nil
		}

		require.NoError(t, hm.RegisterLLMHook(errorHook, TypedHookMetadata{Name: "error", Point: OnLLMError, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			return "", errors.New("LLM API error")
		})

		require.NoError(t, err)
		assert.Equal(t, "fallback response", response, "should return fallback from error hook")
	})

	t.Run("LLM error propagates when error hook doesn't recover", func(t *testing.T) {
		hm := newTestHookManager()

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			return "", errors.New("LLM API error")
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "LLM API error")
		assert.Empty(t, response)
	})

	t.Run("BeforeLLMRequest hook with fatal error blocks execution", func(t *testing.T) {
		hm := newTestHookManager()

		beforeHook := func(_ context.Context, _ *TypedHookContext[LLMPayload], _ func() error) error {
			return errs.Validation("prompt validation failed")
		}

		require.NoError(t, hm.RegisterLLMHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeLLMRequest, Priority: 0, FatalError: true}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		workCalled := false
		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			workCalled = true
			return "should not see this", nil
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prompt validation failed")
		assert.False(t, workCalled, "work function should not be called")
		assert.Empty(t, response)
	})

	t.Run("AfterLLMResponse hook with fatal error overrides successful response", func(t *testing.T) {
		hm := newTestHookManager()

		afterHook := func(_ context.Context, _ *TypedHookContext[LLMPayload], next func() error) error {
			_ = next()
			return errs.Validation("response validation failed")
		}

		require.NoError(t, hm.RegisterLLMHook(afterHook, TypedHookMetadata{Name: "after", Point: AfterLLMResponse, Priority: 0, FatalError: true}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		response, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			return testLLMResponse, nil
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "response validation failed")
		assert.Empty(t, response)
	})

	t.Run("hooks receive correct context with sessionID, agentID, model", func(t *testing.T) {
		hm := newTestHookManager()

		var receivedCtx *TypedHookContext[LLMPayload]
		beforeHook := func(_ context.Context, hc *TypedHookContext[LLMPayload], next func() error) error {
			receivedCtx = hc
			return next()
		}

		require.NoError(t, hm.RegisterLLMHook(beforeHook, TypedHookMetadata{Name: "before", Point: BeforeLLMRequest, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := testLLMPrompt
		model := testLLMModel

		_, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(_ string) (string, error) {
			return "response", nil
		})

		require.NoError(t, err)
		require.NotNil(t, receivedCtx)
		assert.Equal(t, sessionID, receivedCtx.SessionID)
		assert.Equal(t, agentID, receivedCtx.AgentID)
		assert.Equal(t, prompt, receivedCtx.Payload.Input)
		assert.Equal(t, model, receivedCtx.Payload.Model)
	})

	t.Run("multiple hooks execute in priority order", func(t *testing.T) {
		hm := newTestHookManager()

		executed := []string{}

		hook1 := func(_ context.Context, hc *TypedHookContext[LLMPayload], next func() error) error {
			executed = append(executed, "hook1")
			hc.Payload.Input += " + hook1"
			return next()
		}

		hook2 := func(_ context.Context, hc *TypedHookContext[LLMPayload], next func() error) error {
			executed = append(executed, "hook2")
			hc.Payload.Input += " + hook2"
			return next()
		}

		require.NoError(t, hm.RegisterLLMHook(hook1, TypedHookMetadata{Name: "hook1", Point: BeforeLLMRequest, Priority: 1, FatalError: false}))
		require.NoError(t, hm.RegisterLLMHook(hook2, TypedHookMetadata{Name: "hook2", Point: BeforeLLMRequest, Priority: 0, FatalError: false}))

		sessionID := uuid.New()
		agentID := uuid.New()
		prompt := "original"
		model := testLLMModel

		var receivedPrompt string
		_, err := hm.WithLLMHooks(context.Background(), sessionID, agentID, prompt, model, func(p string) (string, error) {
			receivedPrompt = p
			return "response", nil
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"hook2", "hook1"}, executed, "hooks should execute in priority order (0 first)")
		assert.Equal(t, "original + hook2 + hook1", receivedPrompt)
	})
}
