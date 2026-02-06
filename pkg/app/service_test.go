// Package app provides tests for the application service
package app

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/m-mizutani/gollem"
	"go.uber.org/mock/gomock"
)

// TestAgentExecutorAdapter_Execute verifies that the adapter correctly
// delegates to the underlying agent
func TestAgentExecutorAdapter_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockAgent := mocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ ...gollem.Input) (*gollem.ExecuteResponse, error) {
		return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
	}).Times(1)

	adapter := &agentExecutorAdapter{agent: mockAgent}

	response, err := adapter.Execute(ctx, "test input")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("response should not be nil")
	}

	if len(response.Texts) != 1 {
		t.Errorf("expected 1 text, got %d", len(response.Texts))
	}

	if response.Texts[0] != "response" {
		t.Errorf("expected 'response', got '%s'", response.Texts[0])
	}
}

// TestAgentExecutorAdapter_MultipleExecutions verifies that multiple sequential
// executions work correctly
func TestAgentExecutorAdapter_MultipleExecutions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	executionCount := 0
	mockAgent := mocks.NewMockAgent(ctrl)
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ ...gollem.Input) (*gollem.ExecuteResponse, error) {
		executionCount++
		return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
	}).Times(3)

	adapter := &agentExecutorAdapter{agent: mockAgent}

	// Execute multiple times
	for i := 0; i < 3; i++ {
		_, err := adapter.Execute(ctx, "test")
		if err != nil {
			t.Errorf("execution %d: unexpected error: %v", i, err)
		}
	}

	if executionCount != 3 {
		t.Errorf("expected 3 executions, got %d", executionCount)
	}

	// Verify global context is still active
	if ctx.Err() != nil {
		t.Error("context was canceled after multiple executions")
	}
}
