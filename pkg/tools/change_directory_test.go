package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

func TestChangeDirectoryTool_Run_ValidDirectory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	// Create mock workspace and skill services
	mockWorkspaceService := mocks.NewMockService(ctrl)
	mockSkillService := mocks.NewMockSkillService(ctrl)

	// Save original directory and restore after test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("Failed to restore working directory: %v", err)
		}
	}()

	// Create temp directory for testing
	tempDir := t.TempDir()

	// Set up expectations
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return("/old/workspace")
	mockWorkspaceService.EXPECT().SetCurrentWorkspace(gomock.Any()).Return(nil)
	mockWorkspaceService.EXPECT().GetWorkspaceHistory().Return([]string{tempDir, "/old/workspace"})
	mockSkillService.EXPECT().AddSearchPathAndDiscover(gomock.Any(), gomock.Any()).Return(nil)

	tool := &ChangeDirectoryTool{
		logService:       logService,
		hookManager:      mockHookManager,
		workspaceService: mockWorkspaceService,
		skillService:     mockSkillService,
		agentID:          uuid.New(),
	}

	args := map[string]any{
		"path": tempDir,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify result fields
	if success, ok := result["success"].(bool); !ok || !success {
		t.Error("Expected success to be true")
	}

	if _, exists := result["current_path"]; !exists {
		t.Error("Expected 'current_path' field in result")
	}

	if _, exists := result["previous_path"]; !exists {
		t.Error("Expected 'previous_path' field in result")
	}

	if _, exists := result["workspace_history"]; !exists {
		t.Error("Expected 'workspace_history' field in result")
	}
}

func TestChangeDirectoryTool_Run_MissingPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &ChangeDirectoryTool{
		logService:  logService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	args := map[string]any{}

	_, err := tool.Run(context.Background(), args)
	if err == nil {
		t.Fatal("Expected error for missing path")
	}
}

func TestChangeDirectoryTool_Run_EmptyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &ChangeDirectoryTool{
		logService:  logService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	args := map[string]any{
		"path": "",
	}

	_, err := tool.Run(context.Background(), args)
	if err == nil {
		t.Fatal("Expected error for empty path")
	}
}

func TestChangeDirectoryTool_Run_NonexistentDirectory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &ChangeDirectoryTool{
		logService:  logService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	args := map[string]any{
		"path": "/nonexistent/directory/that/does/not/exist",
	}

	_, err := tool.Run(context.Background(), args)
	if err == nil {
		t.Fatal("Expected error for nonexistent directory")
	}
}

func TestChangeDirectoryTool_Run_FileNotDirectory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	tool := &ChangeDirectoryTool{
		logService:  logService,
		hookManager: mockHookManager,
		agentID:     uuid.New(),
	}

	// Create a temp file (not directory)
	tempFile := filepath.Join(t.TempDir(), "testfile.txt")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	args := map[string]any{
		"path": tempFile,
	}

	_, err := tool.Run(context.Background(), args)
	if err == nil {
		t.Fatal("Expected error when path is a file, not directory")
	}
}

func TestChangeDirectoryTool_Run_RelativePath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockWorkspaceService := mocks.NewMockService(ctrl)
	mockSkillService := mocks.NewMockSkillService(ctrl)

	// Save original directory and restore after test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("Failed to restore working directory: %v", err)
		}
	}()

	// Create temp directory
	tempDir := t.TempDir()

	// Change to parent directory so we can use relative path
	parentDir := filepath.Dir(tempDir)
	dirName := filepath.Base(tempDir)
	if err := os.Chdir(parentDir); err != nil {
		t.Fatalf("Failed to change to parent dir: %v", err)
	}

	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return("/old/workspace")
	mockWorkspaceService.EXPECT().SetCurrentWorkspace(gomock.Any()).Return(nil)
	mockWorkspaceService.EXPECT().GetWorkspaceHistory().Return([]string{})
	mockSkillService.EXPECT().AddSearchPathAndDiscover(gomock.Any(), gomock.Any()).Return(nil)

	tool := &ChangeDirectoryTool{
		logService:       logService,
		hookManager:      mockHookManager,
		workspaceService: mockWorkspaceService,
		skillService:     mockSkillService,
		agentID:          uuid.New(),
	}

	args := map[string]any{
		"path": dirName, // relative path
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Error("Expected success to be true")
	}
}

func TestChangeDirectoryTool_Run_SkillDiscoveryFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	mockWorkspaceService := mocks.NewMockService(ctrl)
	mockSkillService := mocks.NewMockSkillService(ctrl)

	// Save original directory and restore after test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("Failed to restore working directory: %v", err)
		}
	}()

	tempDir := t.TempDir()

	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return("/old/workspace")
	mockWorkspaceService.EXPECT().SetCurrentWorkspace(gomock.Any()).Return(nil)
	mockWorkspaceService.EXPECT().GetWorkspaceHistory().Return([]string{})
	mockSkillService.EXPECT().AddSearchPathAndDiscover(gomock.Any(), gomock.Any()).Return(skills.ErrSkillNotFound("test")) // Skill discovery fails

	tool := &ChangeDirectoryTool{
		logService:       logService,
		hookManager:      mockHookManager,
		workspaceService: mockWorkspaceService,
		skillService:     mockSkillService,
		agentID:          uuid.New(),
	}

	args := map[string]any{
		"path": tempDir,
	}

	// Should not fail even if skill discovery fails
	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Should not fail when skill discovery fails: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Error("Expected success to be true even when skill discovery fails")
	}
}

func TestChangeDirectoryTool_Spec(t *testing.T) {
	tool := &ChangeDirectoryTool{}

	spec := tool.Spec()

	if spec.Name != shared.ToolNameChangeDirectory {
		t.Errorf("Expected tool name '%s', got '%s'", shared.ToolNameChangeDirectory, spec.Name)
	}

	if spec.Description == "" {
		t.Error("Expected non-empty description")
	}

	// Check path parameter
	if pathParam, exists := spec.Parameters["path"]; !exists {
		t.Error("Missing 'path' parameter in spec")
	} else {
		if pathParam.Type != gollem.TypeString {
			t.Errorf("Expected 'path' parameter type to be String, got %v", pathParam.Type)
		}
		if pathParam.Description == "" {
			t.Error("Expected non-empty description for 'path' parameter")
		}
	}
}

func TestChangeDirectoryToolProvider_CreateTool(t *testing.T) {
	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(nil)
	mockWorkspaceService := mocks.NewMockService(nil)
	mockSkillService := mocks.NewMockSkillService(nil)

	provider := &changeDirectoryToolProvider{
		logService:       logService,
		hookManager:      mockHookManager,
		workspaceService: mockWorkspaceService,
		skillService:     mockSkillService,
	}
	testUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	tool := provider.CreateTool(testUUID)

	if tool == nil {
		t.Fatal("Expected non-nil tool")
	}

	if tool.logService == nil {
		t.Error("Expected tool to have logService")
	}

	if tool.agentID != testUUID {
		t.Errorf("Expected agentID %v, got %v", testUUID, tool.agentID)
	}
}

func TestNewChangeDirectoryToolProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create injector with required services
	injector := do.New()
	do.Provide(injector, config.NewService)
	do.Provide(injector, logger.NewService)
	do.Provide(injector, hooks.NewHookManager)
	do.Provide(injector, workspace.NewServiceProvider)

	// Use mock for SkillService
	mockSkillService := mocks.NewMockSkillService(ctrl)
	do.ProvideValue[skills.SkillService](injector, mockSkillService)

	provider, err := NewChangeDirectoryToolProvider(injector)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}

	// Verify provider can create tool
	testUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tool := provider.CreateTool(testUUID)

	if tool == nil {
		t.Error("Expected provider to create non-nil tool")
	}
}
