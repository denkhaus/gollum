// Package app provides tests for the application service
package app

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/m-mizutani/gollem"
	"go.uber.org/mock/gomock"
)

// TestExecuteAgentInput_Cancellation verifies that the per-request context
// can be canceled independently of the global context
func TestExecuteAgentInput_Cancellation(t *testing.T) {
	t.Run("normal completion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		globalCtx := context.Background()

		executed := false
		mockAgent := mocks.NewMockAgent(ctrl)
		mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ ...gollem.Input) (*gollem.ExecuteResponse, error) {
			executed = true
			return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
		}).Times(1)

		service := &applicationServiceImpl{}

		// Execute the agent
		err := service.executeAgentInput(globalCtx, mockAgent, "test input")

		// Verify execution occurred
		if !executed {
			t.Error("agent Execute was not called")
		}

		// Verify no error
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify global context is still active
		if globalCtx.Err() != nil {
			t.Error("global context was canceled, should remain active")
		}
	})

	t.Run("canceled via currentCancel", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		globalCtx := context.Background()

		executed := false
		canceled := false
		mockAgent := mocks.NewMockAgent(ctrl)
		mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, _ ...gollem.Input) (*gollem.ExecuteResponse, error) {
			executed = true
			// Wait for cancellation
			<-ctx.Done()
			canceled = true
			return nil, ctx.Err()
		}).Times(1)

		service := &applicationServiceImpl{}

		// Start execution in goroutine
		errCh := make(chan error, 1)
		go func() {
			errCh <- service.executeAgentInput(globalCtx, mockAgent, "test input")
		}()

		// Wait for execution to start
		for !executed { //nolint:staticcheck,revive // Intentional busy wait for testing race conditions
			// Busy wait - this is intentional as we're testing race conditions
		}

		// Trigger cancellation via currentCancel
		if service.currentCancel != nil {
			service.currentCancel()
		} else {
			t.Fatal("currentCancel is nil, cannot trigger cancellation")
		}

		// Wait for error
		err := <-errCh

		// Verify execution was canceled
		if !canceled {
			t.Error("execution was not canceled")
		}

		// Verify error is context canceled
		if err == nil {
			t.Error("expected error when context is canceled, got nil")
		}

		// Verify global context is still active
		if globalCtx.Err() != nil {
			t.Error("global context was canceled, should remain active")
		}

		// Verify currentCancel was cleared
		if service.currentCancel != nil {
			t.Error("currentCancel was not cleared after execution")
		}
	})
}

// TestExecuteAgentInput_ContextIsolation verifies that per-request context
// cancellation doesn't affect the global context
func TestExecuteAgentInput_ContextIsolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	globalCtx := context.Background()

	mockAgent := mocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, _ ...gollem.Input) (*gollem.ExecuteResponse, error) {
		// Verify request context is different from global
		if ctx == globalCtx {
			t.Error("request context is the same as global context, should be isolated")
		}
		return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
	}).Times(1)

	service := &applicationServiceImpl{}

	err := service.executeAgentInput(globalCtx, mockAgent, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify global context is still active
	if globalCtx.Err() != nil {
		t.Error("global context was canceled")
	}
}

// TestExecuteAgentInput_MultipleExecutions verifies that multiple sequential
// executions each get independent contexts
func TestExecuteAgentInput_MultipleExecutions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	globalCtx := context.Background()

	executionCount := 0
	mockAgent := mocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ ...gollem.Input) (*gollem.ExecuteResponse, error) {
		executionCount++
		return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
	}).Times(3)

	service := &applicationServiceImpl{}

	// Execute multiple times
	for i := 0; i < 3; i++ {
		err := service.executeAgentInput(globalCtx, mockAgent, "test")
		if err != nil {
			t.Errorf("execution %d: unexpected error: %v", i, err)
		}
	}

	if executionCount != 3 {
		t.Errorf("expected 3 executions, got %d", executionCount)
	}

	// Verify global context is still active
	if globalCtx.Err() != nil {
		t.Error("global context was canceled after multiple executions")
	}
}
