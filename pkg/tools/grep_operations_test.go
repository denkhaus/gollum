package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

// TestGrepTool_Run_ContentMode_Simple tests basic content mode matching
func TestGrepTool_Run_ContentMode_Simple(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &GrepTool{logService: logService, hookManager: mockHookManager}

	tmpDir := t.TempDir()

	// Create test files with content
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world\nfoo bar\ntest line\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern": "hello",
		"path":    tmpDir,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	if count, ok := result["total_count"].(int); !ok || count != 1 {
		t.Errorf("Expected total_count=1, got %v", result["total_count"])
	}

	matches, ok := result["matches"].([]map[string]any)
	if !ok || len(matches) != 1 {
		t.Errorf("Expected 1 match, got %v", result["matches"])
	}
}

// TestGrepTool_Run_ContentMode_NoMatches tests content mode when no matches found
func TestGrepTool_Run_ContentMode_NoMatches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &GrepTool{logService: logService, hookManager: mockHookManager}

	tmpDir := t.TempDir()

	// Create test file with content
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world\nfoo bar\ntest line\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern": "notfound",
		"path":    tmpDir,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected success=false when no matches found")
	}

	if count, ok := result["total_count"].(int); !ok || count != 0 {
		t.Errorf("Expected total_count=0, got %v", result["total_count"])
	}
}

// TestGrepTool_Run_FilesWithMatches tests files_with_matches output mode
func TestGrepTool_Run_FilesWithMatches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &GrepTool{logService: logService, hookManager: mockHookManager}

	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("foo bar"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern":     "hello",
		"path":        tmpDir,
		"output_mode": "files_with_matches",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	matches, ok := result["matches"].([]string)
	if !ok || len(matches) != 1 {
		t.Errorf("Expected 1 matching file, got %v", result["matches"])
	}
}

// TestGrepTool_Run_CountMode tests count output mode
func TestGrepTool_Run_CountMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	tool := &GrepTool{logService: logService, hookManager: mockHookManager}

	tmpDir := t.TempDir()

	// Create test file with multiple matches
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world\nhello there\ntest\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern":     "hello",
		"path":        tmpDir,
		"output_mode": "count",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	counts, ok := result["matches"].(map[string]int)
	if !ok {
		t.Fatal("Expected matches to be map[string]int")
	}

	if len(counts) != 1 {
		t.Errorf("Expected 1 file in counts, got %d", len(counts))
	}

	// Find the count for our test file
	found := false
	for path, count := range counts {
		if filepath.Base(path) == "test.txt" {
			if count != 2 {
				t.Errorf("Expected count=2 for test.txt, got %d", count)
			}
			found = true
		}
	}

	if !found {
		t.Error("Expected to find test.txt in counts")
	}
}

// TestGrepTool_Run_CaseInsensitive tests case-insensitive matching
func TestGrepTool_Run_CaseInsensitive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	tool := &GrepTool{logService: logService, hookManager: mockHookManager}

	tmpDir := t.TempDir()

	// Create test file with mixed case
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "HELLO world\nhello there\nHello Again\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern": "hello",
		"path":    tmpDir,
		"-i":      true,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if count, ok := result["total_count"].(int); !ok || count != 3 {
		t.Errorf("Expected total_count=3 with case insensitive, got %v", result["total_count"])
	}
}

// TestGrepTool_Run_WithGlobPattern tests glob pattern filtering
func TestGrepTool_Run_WithGlobPattern(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	tool := &GrepTool{logService: logService, hookManager: mockHookManager}

	tmpDir := t.TempDir()

	// Create multiple files
	goFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(goFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	txtFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(txtFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern": "hello",
		"path":    tmpDir,
		"glob":    "*.go",
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should only find the .go file
	if count, ok := result["total_count"].(int); !ok || count != 1 {
		t.Errorf("Expected total_count=1 with glob filter, got %v", result["total_count"])
	}
}

// TestGrepTool_Run_WithHeadLimit tests head limit functionality
func TestGrepTool_Run_WithHeadLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	tool := &GrepTool{logService: logService, hookManager: mockHookManager}

	tmpDir := t.TempDir()

	// Create test file with multiple matches
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern":    "line",
		"path":       tmpDir,
		"head_limit": float64(2),
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should only return first 2 matches
	if count, ok := result["total_count"].(int); !ok || count != 2 {
		t.Errorf("Expected total_count=2 with head limit, got %v", result["total_count"])
	}
}

// TestGrepTool_Run_WithLineNumbers tests line number output
func TestGrepTool_Run_WithLineNumbers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	tool := &GrepTool{logService: logService, hookManager: mockHookManager}

	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nhello world\nline 3\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern": "hello",
		"path":    tmpDir,
		"-n":      true,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	matches, ok := result["matches"].([]map[string]any)
	if !ok || len(matches) != 1 {
		t.Fatalf("Expected 1 match, got %v", result["matches"])
	}

	lineNumber, ok := matches[0]["line_number"].(int)
	if !ok || lineNumber != 2 {
		t.Errorf("Expected line_number=2, got %v", matches[0]["line_number"])
	}
}

// TestGrepTool_Run_WithContext tests context output (-C flag)
func TestGrepTool_Run_WithContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	tool := &GrepTool{logService: logService, hookManager: mockHookManager}

	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nhello world\nline 4\nline 5\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"pattern": "hello",
		"path":    tmpDir,
		"-C":      float64(1),
		"-n":      true,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	matches, ok := result["matches"].([]map[string]any)
	if !ok || len(matches) != 1 {
		t.Fatalf("Expected 1 match, got %v", result["matches"])
	}

	context, ok := matches[0]["context"].(string)
	if !ok || context == "" {
		t.Error("Expected context to be non-empty string")
	}
}
