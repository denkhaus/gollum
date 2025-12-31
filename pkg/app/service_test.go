// Package app provides tests for the application service
package app

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// mockAgent is a minimal mock for testing
type mockAgent struct {
	id      uuid.UUID
	config  *shared.AgentConfig
	session gollem.Session
	execute func(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error)
}

func (m *mockAgent) GetID() uuid.UUID {
	return m.id
}

func (m *mockAgent) GetConfig() *shared.AgentConfig {
	return m.config
}

func (m *mockAgent) Session() gollem.Session {
	return m.session
}

func (m *mockAgent) Execute(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
	if m.execute != nil {
		return m.execute(ctx, input...)
	}
	return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
}

// TestExecuteAgentInput_Cancellation verifies that the per-request context
// can be canceled independently of the global context
func TestExecuteAgentInput_Cancellation(t *testing.T) {
	t.Run("normal completion", func(t *testing.T) {
		globalCtx := context.Background()

		executed := false

		agent := &mockAgent{
			id:     uuid.New(),
			config: &shared.AgentConfig{},
			execute: func(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
				executed = true
				return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
			},
		}

		service := &applicationServiceImpl{}

		// Execute the agent
		err := service.executeAgentInput(globalCtx, agent, "test input")

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
		globalCtx := context.Background()

		executed := false
		canceled := false

		agent := &mockAgent{
			id:     uuid.New(),
			config: &shared.AgentConfig{},
			execute: func(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
				executed = true
				// Wait for cancellation
				<-ctx.Done()
				canceled = true
				return nil, ctx.Err()
			},
		}

		service := &applicationServiceImpl{}

		// Start execution in goroutine
		errCh := make(chan error, 1)
		go func() {
			errCh <- service.executeAgentInput(globalCtx, agent, "test input")
		}()

		// Wait for execution to start
		for !executed {
			// Small sleep to avoid busy waiting
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
	globalCtx := context.Background()

	agent := &mockAgent{
		id:     uuid.New(),
		config: &shared.AgentConfig{},
		execute: func(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
			// Verify request context is different from global
			if ctx == globalCtx {
				t.Error("request context is the same as global context, should be isolated")
			}
			return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
		},
	}

	service := &applicationServiceImpl{}

	err := service.executeAgentInput(globalCtx, agent, "test")
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
	globalCtx := context.Background()

	executionCount := 0
	agent := &mockAgent{
		id:     uuid.New(),
		config: &shared.AgentConfig{},
		execute: func(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
			executionCount++
			return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
		},
	}

	service := &applicationServiceImpl{}

	// Execute multiple times
	for i := 0; i < 3; i++ {
		err := service.executeAgentInput(globalCtx, agent, "test")
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
