package hooks

import (
	"context"
	"testing"
	"time"

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
