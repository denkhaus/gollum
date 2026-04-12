package hooks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestExecutorPayload_FlowStepContext(t *testing.T) {
	flowID := uuid.New()
	sessionID := uuid.New()

	payload := &ExecutorPayload{
		FlowID:       flowID,
		FlowName:     "test-flow",
		SessionID:    sessionID,
		CurrentState: "process",
		StepType:     "llm",
		StepIndex:    0,
		StateName:    "process",
		StepResult:   map[string]any{"output": "test"},
		StepError:    nil,
		Duration:     100 * time.Millisecond,
	}

	assert.Equal(t, flowID, payload.FlowID)
	assert.Equal(t, "test-flow", payload.FlowName)
	assert.Equal(t, sessionID, payload.SessionID)
	assert.Equal(t, "process", payload.CurrentState)
	assert.Equal(t, "llm", payload.StepType)
	assert.Equal(t, 0, payload.StepIndex)
	assert.Equal(t, "process", payload.StateName)
	assert.NotNil(t, payload.StepResult)
	assert.NoError(t, payload.StepError)
	assert.Equal(t, 100*time.Millisecond, payload.Duration)
}

func TestExecutorPayload_WithStepError(t *testing.T) {
	payload := &ExecutorPayload{
		FlowID:       uuid.New(),
		SessionID:    uuid.New(),
		CurrentState: "failed",
		StepType:     "shell",
		StateName:    "failed",
		StepError:    assert.AnError,
		Duration:     50 * time.Millisecond,
	}

	assert.Error(t, payload.StepError)
	assert.Equal(t, 50*time.Millisecond, payload.Duration)
}

func TestHookManager_RegisterExecutorHook(t *testing.T) {
	hm := newTestHookManager()

	hookFn := func(ctx context.Context, hookCtx *TypedHookContext[ExecutorPayload], next func() error) error {
		return nil
	}

	meta := TypedHookMetadata{
		Name:     "test-executor-hook",
		Point:    BeforeFlowStep,
		Priority: 50,
	}

	err := hm.RegisterExecutorHook(hookFn, meta)
	assert.NoError(t, err, "Hook registration should succeed")
}

func TestHookManager_RegisterExecutorHook_DuplicateName(t *testing.T) {
	hm := newTestHookManager()

	hookFn := func(ctx context.Context, hookCtx *TypedHookContext[ExecutorPayload], next func() error) error {
		return nil
	}

	meta := TypedHookMetadata{
		Name:  "duplicate-hook",
		Point: BeforeFlowStep,
	}

	// First registration should succeed
	err := hm.RegisterExecutorHook(hookFn, meta)
	assert.NoError(t, err)

	// Second registration with same name should fail
	err = hm.RegisterExecutorHook(hookFn, meta)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestHookManager_TriggerExecutorHooks_BeforeFlowStep(t *testing.T) {
	hm := newTestHookManager()

	executed := false
	var capturedPayload *ExecutorPayload

	hookFn := func(_ context.Context, hookCtx *TypedHookContext[ExecutorPayload], next func() error) error {
		executed = true
		capturedPayload = &hookCtx.Payload
		return next()
	}

	err := hm.RegisterExecutorHook(hookFn, TypedHookMetadata{
		Name:  "test-trigger",
		Point: BeforeFlowStep,
	})
	assert.NoError(t, err)

	payload := ExecutorPayload{
		FlowID:       uuid.New(),
		FlowName:     "trigger-test",
		SessionID:    uuid.New(),
		CurrentState: "initial",
		StepType:     "func",
		StateName:    "initial",
	}

	hookCtx := &TypedHookContext[ExecutorPayload]{
		LoggingContext: shared.LoggingContext{
			SessionID: payload.SessionID.String(),
		},
		Payload: payload,
	}

	result := hm.TriggerExecutorHooks(context.Background(), BeforeFlowStep, hookCtx)

	assert.True(t, executed)
	assert.NotNil(t, capturedPayload)
	assert.Equal(t, "trigger-test", capturedPayload.FlowName)
	assert.Equal(t, "func", capturedPayload.StepType)
	assert.False(t, result.Stopped)
	assert.Nil(t, result.Error)
}

func TestHookManager_TriggerExecutorHooks_Blocking(t *testing.T) {
	hm := newTestHookManager()

	hookFn := func(_ context.Context, hookCtx *TypedHookContext[ExecutorPayload], _ func() error) error {
		// Stop the chain by not calling next()
		return nil
	}

	err := hm.RegisterExecutorHook(hookFn, TypedHookMetadata{
		Name:     "blocking-hook",
		Point:    BeforeFlowStep,
		Priority: 100,
	})
	assert.NoError(t, err)

	payload := ExecutorPayload{
		FlowID:    uuid.New(),
		StepType:  "shell",
		StateName: "blocked",
	}

	hookCtx := &TypedHookContext[ExecutorPayload]{
		Payload: payload,
	}

	result := hm.TriggerExecutorHooks(context.Background(), BeforeFlowStep, hookCtx)

	// When a hook doesn't call next(), the chain is stopped
	assert.True(t, result.Stopped, "Hook should stop execution")
}

func TestHookManager_WithFlowStepHooks_BeforeAndAfter(t *testing.T) {
	hm := newTestHookManager()

	var beforePayload, afterPayload *ExecutorPayload

	// Before hook
	_ = hm.RegisterExecutorHook(func(_ context.Context, hookCtx *TypedHookContext[ExecutorPayload], next func() error) error {
		beforePayload = &hookCtx.Payload
		return next()
	}, TypedHookMetadata{Name: "before", Point: BeforeFlowStep})

	// After hook
	_ = hm.RegisterExecutorHook(func(_ context.Context, hookCtx *TypedHookContext[ExecutorPayload], next func() error) error {
		afterPayload = &hookCtx.Payload
		return next()
	}, TypedHookMetadata{Name: "after", Point: AfterFlowStep})

	flowID := uuid.New()
	sessionID := uuid.New()

	result, err := hm.WithFlowStepHooks(
		context.Background(),
		sessionID,
		flowID,
		"test-flow",
		"llm",
		"test-state",
		func() (map[string]any, error) {
			return map[string]any{"response": "test"}, nil
		},
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test", result["response"])

	// Verify before hook was called
	assert.NotNil(t, beforePayload)
	assert.Equal(t, "llm", beforePayload.StepType)
	assert.Equal(t, "test-state", beforePayload.StateName)
	assert.Nil(t, beforePayload.StepResult) // Before hook has no result

	// Verify after hook was called
	assert.NotNil(t, afterPayload)
	assert.Equal(t, "llm", afterPayload.StepType)
	assert.NotNil(t, afterPayload.StepResult)
	assert.Equal(t, "test", afterPayload.StepResult["response"])
	assert.Greater(t, afterPayload.Duration, time.Duration(0))
}

func TestHookManager_WithFlowStepHooks_ExecutionBlocked(t *testing.T) {
	hm := newTestHookManager()

	// Blocking before hook
	_ = hm.RegisterExecutorHook(func(_ context.Context, _ *TypedHookContext[ExecutorPayload], _ func() error) error {
		// Stop the chain by not calling next()
		return nil
	}, TypedHookMetadata{Name: "blocker", Point: BeforeFlowStep})

	workCalled := false
	_, err := hm.WithFlowStepHooks(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"test-flow",
		"func",
		"blocked-state",
		func() (map[string]any, error) {
			workCalled = true
			return nil, nil
		},
	)

	assert.Error(t, err)
	assert.False(t, workCalled, "Work should not be called when blocked")
	assert.Contains(t, err.Error(), "blocked")
}

func TestHookManager_WithFlowStepHooks_WorkError(t *testing.T) {
	hm := newTestHookManager()

	var afterPayload *ExecutorPayload
	_ = hm.RegisterExecutorHook(func(_ context.Context, hookCtx *TypedHookContext[ExecutorPayload], next func() error) error {
		afterPayload = &hookCtx.Payload
		return next()
	}, TypedHookMetadata{Name: "after", Point: AfterFlowStep})

	workErr := errors.New("work failed")

	_, err := hm.WithFlowStepHooks(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"test-flow",
		"shell",
		"error-state",
		func() (map[string]any, error) {
			return nil, workErr
		},
	)

	assert.Error(t, err)
	assert.Equal(t, workErr, err)

	// After hook should still be called even on error
	assert.NotNil(t, afterPayload)
	assert.Error(t, afterPayload.StepError)
	assert.Equal(t, workErr, afterPayload.StepError)
}
