package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

func TestGlobTool_Run_SimplePattern(t *testing.T) {
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
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &globToolImpl{logService: logService, hookManager: mockHookManager, agent: mockAgent}

	tmpDir := t.TempDir()

	// Create some test files
	testFiles := []string{"file1.go", "file2.go", "file3.txt"}
	for _, f := range testFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	args := map[string]any{
		"pattern": "*.go",
		"path":    tmpDir,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	if count, ok := result["count"].(int); !ok || count != 2 {
		t.Errorf("Expected count=2, got %v", result["count"])
	}

	matches, ok := result["matches"].([]string)
	if !ok {
		t.Fatal("Expected matches to be []string")
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}
}

func TestGlobTool_Run_NoMatches(t *testing.T) {
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
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &globToolImpl{logService: logService, hookManager: mockHookManager, agent: mockAgent}

	tmpDir := t.TempDir()

	// Create a file that doesn't match the pattern
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern": "*.go",
		"path":    tmpDir,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false when no matches found")
	}

	if count, ok := result["count"].(int); !ok || count != 0 {
		t.Errorf("Expected count=0, got %v", result["count"])
	}

	matches, ok := result["matches"].([]string)
	if !ok {
		t.Fatal("Expected matches to be []string")
	}

	if len(matches) != 0 {
		t.Errorf("Expected 0 matches, got %d", len(matches))
	}
}

func TestGlobTool_Run_DefaultPath(t *testing.T) {
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
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &globToolImpl{logService: logService, hookManager: mockHookManager, agent: mockAgent}

	// Change to temp directory for this test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	tmpDir := t.TempDir()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create test files
	if err := os.WriteFile("test.go", []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern": "*.go",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	if count, ok := result["count"].(int); !ok || count != 1 {
		t.Errorf("Expected count=1, got %v", result["count"])
	}
}

func TestGlobTool_Run_NestedPath(t *testing.T) {
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
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &globToolImpl{logService: logService, hookManager: mockHookManager, agent: mockAgent}

	tmpDir := t.TempDir()

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "subdir", "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dirs: %v", err)
	}

	// Create file in nested directory
	if err := os.WriteFile(filepath.Join(nestedDir, "file.go"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern": "*.go",
		"path":    nestedDir,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	if count, ok := result["count"].(int); !ok || count != 1 {
		t.Errorf("Expected count=1, got %v", result["count"])
	}
}

func TestGlobTool_Run_SubdirectoryPattern(t *testing.T) {
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
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &globToolImpl{logService: logService, hookManager: mockHookManager, agent: mockAgent}

	tmpDir := t.TempDir()

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create sub dir: %v", err)
	}

	// Create files in different locations
	if err := os.WriteFile(filepath.Join(tmpDir, "root.go"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "pkg.go"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern": "pkg/*.go",
		"path":    tmpDir,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	if count, ok := result["count"].(int); !ok || count != 1 {
		t.Errorf("Expected count=1, got %v", result["count"])
	}

	matches, ok := result["matches"].([]string)
	if !ok {
		t.Fatal("Expected matches to be []string")
	}

	// Verify the match is in pkg subdirectory
	for _, match := range matches {
		if filepath.Base(filepath.Dir(match)) != "pkg" {
			t.Errorf("Expected match in pkg directory, got: %s", match)
		}
	}
}

func TestGlobTool_Run_AllFilesPattern(t *testing.T) {
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
	mockAgent.EXPECT().ToLoggingContext().Return(*shared.NewLoggingContext("test-session", agentID, uuid.Nil)).AnyTimes()

	tool := &globToolImpl{logService: logService, hookManager: mockHookManager, agent: mockAgent}

	tmpDir := t.TempDir()

	// Create various files
	testFiles := []string{"file1.go", "file2.txt", "file3.md"}
	for _, f := range testFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	args := map[string]any{
		"pattern": "*",
		"path":    tmpDir,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if count, ok := result["count"].(int); !ok || count != 3 {
		t.Errorf("Expected count=3, got %v", result["count"])
	}
}
