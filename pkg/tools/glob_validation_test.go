package tools

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

func TestGlobTool_Run_MissingPattern(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	// Create mock agent
	agentID := uuid.New()
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().ToSessionContext().Return(*shared.NewSessionContext(uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), agentID, uuid.Nil, "")).AnyTimes()

	tool := &globToolImpl{logService: logService, hookManager: mockHookManager, agent: mockAgent}

	args := map[string]any{
		"path": "/some/path",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false when pattern is missing")
	}

	if result["error"] != "pattern is required and must be a non-empty string" {
		t.Errorf("Expected specific error message, got: %v", result["error"])
	}
}

func TestGlobTool_Run_EmptyPattern(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	// Create mock agent
	agentID := uuid.New()
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().ToSessionContext().Return(*shared.NewSessionContext(uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), agentID, uuid.Nil, "")).AnyTimes()

	tool := &globToolImpl{logService: logService, hookManager: mockHookManager, agent: mockAgent}

	args := map[string]any{
		"pattern": "",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false when pattern is empty")
	}
}

func TestGlobTool_Run_NonStringPattern(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)
	// Create mock agent
	agentID := uuid.New()
	mockAgent := shared.NewMockAgent(ctrl)
	mockAgent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockAgent.EXPECT().ToSessionContext().Return(*shared.NewSessionContext(uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), agentID, uuid.Nil, "")).AnyTimes()

	tool := &globToolImpl{logService: logService, hookManager: mockHookManager, agent: mockAgent}

	args := map[string]any{
		"pattern": 12345,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false when pattern is not a string")
	}
}
