package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/diff"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

func TestWriteFileTool_Spec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm, err := state.NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockDiffProvider := diff.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiffForNewFile(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &writeFileToolImpl{logService: logService, fsm: fsm, hookManager: mockHookManager, diffProvider: mockDiffProvider}
	tool.diffProvider = mockDiffProvider

	spec := tool.Spec()

	if spec.Name != "write_file" {
		t.Errorf("Expected tool name 'write_file', got '%s'", spec.Name)
	}

	// Check file_path parameter
	if _, exists := spec.Parameters["file_path"]; !exists {
		t.Error("Missing 'file_path' parameter in spec")
	}

	// Check content parameter
	if _, exists := spec.Parameters["content"]; !exists {
		t.Error("Missing 'content' parameter in spec")
	}

	// Check create_dirs parameter
	if _, exists := spec.Parameters["create_dirs"]; !exists {
		t.Error("Missing 'create_dirs' parameter in spec")
	}
}

func TestWriteFileTool_Run_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm, err := state.NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockDiffProvider := diff.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiffForNewFile(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &writeFileToolImpl{logService: logService, fsm: fsm, hookManager: mockHookManager, diffProvider: mockDiffProvider}
	tool.diffProvider = mockDiffProvider

	// Create a temporary directory
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	args := map[string]any{
		"file_path": testPath,
		"content":   testContent,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	if result[string(shared.KeyFilePath)] != testPath {
		t.Errorf("Expected path=%s, got %v", testPath, result[string(shared.KeyFilePath)])
	}

	if bytes, ok := result["bytes"].(int); !ok || bytes != len(testContent) {
		t.Errorf("Expected bytes=%d, got %v", len(testContent), result["bytes"])
	}

	// Verify checksum is present
	if result["checksum"] == nil {
		t.Error("Expected checksum to be present in result")
	}

	// Verify file was actually created
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected file content '%s', got '%s'", testContent, string(content))
	}
}

func TestWriteFileTool_Run_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm, err := state.NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockDiffProvider := diff.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiffForNewFile(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &writeFileToolImpl{logService: logService, fsm: fsm, hookManager: mockHookManager, diffProvider: mockDiffProvider}
	tool.diffProvider = mockDiffProvider

	tests := []struct {
		name          string
		args          map[string]any
		expectedError string
	}{
		{
			name: "missing path",
			args: map[string]any{
				"content": "test content",
			},
			expectedError: "file_path is required and must be a non-empty string",
		},
		{
			name: "empty path",
			args: map[string]any{
				"file_path": "",
				"content":   "test content",
			},
			expectedError: "file_path is required and must be a non-empty string",
		},
		{
			name: "non-string path",
			args: map[string]any{
				"file_path": 123,
				"content":   "test content",
			},
			expectedError: "file_path is required and must be a non-empty string",
		},
		{
			name: "missing content",
			args: map[string]any{
				"file_path": "test.txt",
			},
			expectedError: "content is required and must be a string",
		},
		{
			name: "non-string content",
			args: map[string]any{
				"file_path": "test.txt",
				"content":   123,
			},
			expectedError: "content is required and must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Run(context.Background(), tt.args)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if success, ok := result["success"].(bool); ok && success {
				t.Error("Expected success=false for invalid input")
			}

			if result["error"] != tt.expectedError {
				t.Errorf("Expected error '%s', got '%v'", tt.expectedError, result["error"])
			}
		})
	}
}

func TestWriteFileTool_Run_CreateDirectories(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm, err := state.NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockDiffProvider := diff.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiffForNewFile(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &writeFileToolImpl{logService: logService, fsm: fsm, hookManager: mockHookManager, diffProvider: mockDiffProvider}
	tool.diffProvider = mockDiffProvider

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "subdir", "nested", "test.txt")
	testContent := "Nested file content"

	args := map[string]any{
		"file_path":   testPath,
		"content":     testContent,
		"create_dirs": true,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	// Verify file was created
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Errorf("Expected file to be created at %s", testPath)
	}

	// Verify content
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected file content '%s', got '%s'", testContent, string(content))
	}
}

func TestWriteFileTool_Run_CreateDirectoriesFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm, err := state.NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockDiffProvider := diff.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiffForNewFile(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &writeFileToolImpl{logService: logService, fsm: fsm, hookManager: mockHookManager, diffProvider: mockDiffProvider}
	tool.diffProvider = mockDiffProvider

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "nonexistent", "test.txt")

	args := map[string]any{
		"file_path":   testPath,
		"content":     "test content",
		"create_dirs": false,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false when directory doesn't exist and create_dirs=false")
	}

	if result["error"] == nil {
		t.Error("Expected error when directory doesn't exist")
	}

	// Verify error message
	if errMsg, ok := result["error"].(string); ok {
		if !strings.Contains(errMsg, "no such file or directory") && !strings.Contains(errMsg, "failed to write file") {
			t.Logf("Warning: Expected error about missing directory, got: %s", errMsg)
		}
	}
}

func TestWriteFileTool_Run_OverwriteExisting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm, err := state.NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	agentID := uuid.New()
	agent := shared.NewMockAgent(ctrl)
	agent.EXPECT().GetID().Return(agentID).AnyTimes()
	mockDiffProvider := diff.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiffForNewFile(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &writeFileToolImpl{logService: logService, fsm: fsm, agent: agent, hookManager: mockHookManager, diffProvider: mockDiffProvider}
	tool.diffProvider = mockDiffProvider

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.txt")

	// Create initial file
	initialContent := "Initial content"
	if err := os.WriteFile(testPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Update file stats to make it visible to FSM
	if _, err := fsm.UpdateStats(testPath); err != nil {
		t.Fatalf("Failed to update file stats: %v", err)
	}

	// Track that agent has read this file (simulate read before write)
	// First update checksum so FSM knows about the file
	if _, err := fsm.UpdateChecksum(testPath); err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}

	// Manually mark file as read by agent (simulate ReadFileTool behavior)
	_, err = fsm.DoWorkWithOptions(context.Background(), testPath, agentID, state.LockModeShared,
		state.WorkOptions{TrackRead: true},
		func(_ context.Context, _ *state.LockToken) (any, error) {
			return nil, nil
		})
	if err != nil {
		t.Fatalf("Failed to mark file as read: %v", err)
	}

	// Now overwrite with new content (should succeed since agent has read it)
	newContent := "New content"
	args := map[string]any{
		"file_path": testPath,
		"content":   newContent,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	// Verify file was overwritten
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != newContent {
		t.Errorf("Expected file content '%s', got '%s'", newContent, string(content))
	}

	if string(content) == initialContent {
		t.Error("File was not overwritten")
	}
}

func TestWriteFileTool_Run_AutomaticRaceConditionDetection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm, err := state.NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Create tools for the same agent
	testAgentID := state.TestAgent1
	agent := shared.NewMockAgent(ctrl)
	agent.EXPECT().GetID().Return(testAgentID).AnyTimes()
	writeTool := &writeFileToolImpl{logService: logService, fsm: fsm, agent: agent, hookManager: mockHookManager}
	readTool := &readFileToolImpl{logService: logService, fsm: fsm, agent: agent, hookManager: mockHookManager}

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.txt")

	// Create initial file
	initialContent := "Initial content"
	if err := os.WriteFile(testPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Agent reads the file (this tracks the read for race condition detection)
	readResult, err := readTool.Run(context.Background(), map[string]any{
		"file_path": testPath,
	})
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if success, ok := readResult["success"].(bool); !ok || !success {
		t.Fatalf("Read failed: %v", readResult["error"])
	}

	// Verify the read was tracked
	storedChecksum := readResult["checksum"].(string)
	if storedChecksum == "" {
		t.Fatal("Expected checksum to be tracked")
	}

	// File is modified externally (simulating another agent changing it)
	modifiedContent := "Modified by another agent"
	if err := os.WriteFile(testPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Agent tries to write (should fail due to race condition detection)
	writeResult, err := writeTool.Run(context.Background(), map[string]any{
		"file_path": testPath,
		"content":   "New content from this agent",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := writeResult["success"].(bool); ok && success {
		t.Error("Expected write to fail due to race condition detection")
	}

	// Verify the error message mentions file was modified and references read_file tool
	if errMsg, ok := writeResult["error"].(string); ok {
		if !strings.Contains(errMsg, "modified since you last read") {
			t.Errorf("Expected error about file modification, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "read_file tool") {
			t.Errorf("Expected error to mention read_file tool, got: %s", errMsg)
		}
	} else {
		t.Error("Expected error message in result")
	}
}

func TestWriteFileTool_Run_WriteCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm, err := state.NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockDiffProvider := diff.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiffForNewFile(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &writeFileToolImpl{logService: logService, fsm: fsm, hookManager: mockHookManager, diffProvider: mockDiffProvider}
	tool.diffProvider = mockDiffProvider

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "main.go")
	codeContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	args := map[string]any{
		"file_path":   testPath,
		"content":     codeContent,
		"create_dirs": true,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	// Verify the code file was written correctly
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read code file: %v", err)
	}

	if string(content) != codeContent {
		t.Errorf("Code content mismatch")
	}
}

func TestWriteFileTool_Run_EmptyContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm, err := state.NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockDiffProvider := diff.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiffForNewFile(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &writeFileToolImpl{logService: logService, fsm: fsm, hookManager: mockHookManager, diffProvider: mockDiffProvider}
	tool.diffProvider = mockDiffProvider

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "empty.txt")

	args := map[string]any{
		"file_path": testPath,
		"content":   "",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true with empty content, got %v", result["success"])
	}

	// Verify empty file was created
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(content) != 0 {
		t.Errorf("Expected empty file, got %d bytes", len(content))
	}
}

func TestWriteFileTool_Run_MultiLineContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm, err := state.NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	mockHookManager := hooks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockDiffProvider := diff.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiffForNewFile(gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &writeFileToolImpl{logService: logService, fsm: fsm, hookManager: mockHookManager, diffProvider: mockDiffProvider}
	tool.diffProvider = mockDiffProvider

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "multiline.txt")
	multilineContent := "Line 1\nLine 2\nLine 3\n"

	args := map[string]any{
		"file_path": testPath,
		"content":   multilineContent,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	// Verify content is preserved exactly
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != multilineContent {
		t.Errorf("Content mismatch:\nExpected: %q\nGot: %q", multilineContent, string(content))
	}
}

func TestWriteFileToolProvider_CreateWriteFileTool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm, err := state.NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	testUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	agent := shared.NewMockAgent(ctrl)
	agent.EXPECT().GetID().Return(testUUID).AnyTimes()
	provider := &writeFileToolProvider{logService: logService, fsm: fsm}
	tool := provider.CreateTool(agent)
	toolImpl := tool.(*writeFileToolImpl)

	if tool == nil {
		t.Fatal("Expected non-nil tool")
	}

	if toolImpl.fsm == nil {
		t.Error("Expected tool to have FileStateManager")
	}
	if toolImpl.logService == nil {
		t.Error("Expected tool to have logService")
	}
}
