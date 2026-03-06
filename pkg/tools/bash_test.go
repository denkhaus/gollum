package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

// createBashToolForTesting creates a BashTool with mocked dependencies for testing.
// This follows the DI pattern guideline: tests create tools directly with their own mock dependencies.
func createBashToolForTesting(t *testing.T, ctrl *gomock.Controller) *BashTool {
	t.Helper()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	mockDiffProvider := mocks.NewMockProvider(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	return &BashTool{
		logService:   logService,
		fileState:    mockFSM,
		agentID:      uuid.New(),
		hookManager:  mockHookManager,
		diffProvider: mockDiffProvider,
		bashCfg: &config.BashConfig{
			TrackChanges: false, // Disable file tracking for basic tests
		},
	}
}

func TestBashTool_Spec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := createBashToolForTesting(t, ctrl)

	spec := tool.Spec()

	if spec.Name != "bash" {
		t.Errorf("Expected tool name 'bash', got '%s'", spec.Name)
	}

	// Check command parameter
	if param, exists := spec.Parameters["command"]; !exists {
		t.Error("Missing 'command' parameter in spec")
	} else if param.Description == "" {
		t.Error("Expected description for 'command' parameter")
	}

	// Check timeout parameter exists
	if _, exists := spec.Parameters["timeout"]; !exists {
		t.Error("Missing 'timeout' parameter in spec")
	}
}

func TestBashTool_Run_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := createBashToolForTesting(t, ctrl)

	args := map[string]any{
		"command": "echo 'hello world'",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	if stdout, ok := result["stdout"].(string); !ok || stdout != "hello world" {
		t.Errorf("Expected stdout='hello world', got '%v'", result["stdout"])
	}

	if exitCode, ok := result["exit_code"].(int); !ok || exitCode != 0 {
		t.Errorf("Expected exit_code=0, got %v", result["exitCode"])
	}

	if result["duration"] == nil {
		t.Error("Expected duration to be set in result")
	}
}

func TestBashTool_Run_InvalidCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := createBashToolForTesting(t, ctrl)

	tests := []struct {
		name          string
		command       any
		expectedError string
	}{
		{
			name:          "missing command",
			command:       nil,
			expectedError: "command is required and must be a non-empty string",
		},
		{
			name:          "non-string command",
			command:       123,
			expectedError: "command is required and must be a non-empty string",
		},
		{
			name:          "empty string command",
			command:       "",
			expectedError: "command is required and must be a non-empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]any{}
			if tt.command != nil {
				args["command"] = tt.command
			}

			result, err := tool.Run(context.Background(), args)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if success, ok := result["success"].(bool); ok && success {
				t.Error("Expected success=false for invalid command")
			}

			if result["error"] != tt.expectedError {
				t.Errorf("Expected error '%s', got '%v'", tt.expectedError, result["error"])
			}
		})
	}
}

func TestBashTool_Run_CommandFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := createBashToolForTesting(t, ctrl)

	args := map[string]any{
		"command": "ls /nonexistent/directory/that/does/not/exist",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false for failing command")
	}

	if result["error"] == nil {
		t.Error("Expected error to be set in result")
	}

	if exitCode, ok := result["exit_code"].(int); !ok || exitCode == 0 {
		t.Errorf("Expected non-zero exit code, got %v", result["exitCode"])
	}

	// Verify stderr contains something
	if stderr, ok := result["stderr"].(string); ok && stderr == "" {
		t.Log("Note: stderr is empty, might be expected for this command")
	}
}

func TestBashTool_Run_WithCustomTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := createBashToolForTesting(t, ctrl)

	// Test with custom timeout
	args := map[string]any{
		"command": "echo 'test'",
		"timeout": float64(10),
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}
}

func TestBashTool_Run_TimeoutExceedsMax(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := createBashToolForTesting(t, ctrl)

	// Test with timeout exceeding max (should be capped at 300 per code)
	args := map[string]any{
		"command": "echo 'test'",
		"timeout": float64(400), // Exceeds max of 300
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true even with excessive timeout, got %v", result["success"])
	}
}

func TestBashTool_Run_Timeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := createBashToolForTesting(t, ctrl)

	// Command that sleeps longer than the timeout
	args := map[string]any{
		"command": "sleep 5",
		"timeout": float64(1), // 1 second timeout
	}

	start := time.Now()
	result, err := tool.Run(context.Background(), args)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false for timed out command")
	}

	if result["error"] == nil {
		t.Error("Expected error to be set for timeout")
	}

	// Verify error message contains "timed out"
	if errMsg, ok := result["error"].(string); ok && !strings.Contains(errMsg, "timed out") {
		t.Errorf("Expected error message to contain 'timed out', got '%s'", errMsg)
	}

	// Verify the command actually timed out reasonably quickly
	if duration > 2*time.Second {
		t.Errorf("Expected command to timeout quickly, but took %v", duration)
	}

	if exitCode, ok := result["exit_code"].(int); !ok || exitCode != -1 {
		t.Errorf("Expected exit_code=-1 for timeout, got %v", result["exitCode"])
	}
}

func TestBashTool_Run_MultiLineCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := createBashToolForTesting(t, ctrl)

	args := map[string]any{
		"command": "echo 'line1' && echo 'line2'",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	if stdout, ok := result["stdout"].(string); !ok || !strings.Contains(stdout, "line1") || !strings.Contains(stdout, "line2") {
		t.Errorf("Expected stdout to contain both lines, got '%v'", result["stdout"])
	}
}

func TestBashTool_Run_CommandWithStderr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := createBashToolForTesting(t, ctrl)

	args := map[string]any{
		"command": "echo 'error message' >&2",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Command succeeds even with stderr output
	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	if stderr, ok := result["stderr"].(string); !ok || stderr != "error message" {
		t.Errorf("Expected stderr='error message', got '%v'", result["stderr"])
	}
}

func TestBashTool_Run_EnvironmentVariables(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := createBashToolForTesting(t, ctrl)

	args := map[string]any{
		"command": "echo $HOME",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	if stdout, ok := result["stdout"].(string); !ok || stdout == "" {
		t.Error("Expected $HOME to be set and non-empty")
	}
}

func TestBashTool_Run_PipeCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tool := createBashToolForTesting(t, ctrl)

	args := map[string]any{
		"command": "echo 'hello world' | grep hello",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	if stdout, ok := result["stdout"].(string); !ok || !strings.Contains(stdout, "hello") {
		t.Errorf("Expected stdout to contain 'hello', got '%v'", result["stdout"])
	}
}
