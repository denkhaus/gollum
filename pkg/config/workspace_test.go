package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkspaceConfig_GetCurrentWorkspace_Default tests that GetCurrentWorkspace
// returns the current working directory when no workspace is set
func TestWorkspaceConfig_GetCurrentWorkspace_Default(t *testing.T) {
	cfg := &WorkspaceConfig{}
	wd, err := os.Getwd()
	require.NoError(t, err)

	result := cfg.GetCurrentWorkspace()
	assert.Equal(t, wd, result, "Should return current working directory when not set")
}

// TestWorkspaceConfig_GetCurrentWorkspace_Set tests that GetCurrentWorkspace
// returns the set workspace path
func TestWorkspaceConfig_GetCurrentWorkspace_Set(t *testing.T) {
	cfg := &WorkspaceConfig{
		CurrentWorkspace: "/custom/path",
	}

	result := cfg.GetCurrentWorkspace()
	assert.Equal(t, "/custom/path", result, "Should return the set workspace path")
}

// TestWorkspaceConfig_SetWorkspace_ValidPath tests setting a valid workspace path
func TestWorkspaceConfig_SetWorkspace_ValidPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workspace-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	cfg := &WorkspaceConfig{}
	err = cfg.SetWorkspace(tmpDir)
	require.NoError(t, err)

	absPath, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, absPath, cfg.CurrentWorkspace)
}

// TestWorkspaceConfig_SetWorkspace_InvalidPath tests that setting an invalid path returns an error
func TestWorkspaceConfig_SetWorkspace_InvalidPath(t *testing.T) {
	cfg := &WorkspaceConfig{}
	err := cfg.SetWorkspace("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace path does not exist")
}

// TestWorkspaceConfig_SetWorkspace_RelativePath tests that relative paths are converted to absolute
func TestWorkspaceConfig_SetWorkspace_RelativePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workspace-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	parentDir := filepath.Dir(tmpDir)
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(parentDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	cfg := &WorkspaceConfig{}
	relPath := filepath.Base(tmpDir)
	err = cfg.SetWorkspace(relPath)
	require.NoError(t, err)

	absPath, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, absPath, cfg.CurrentWorkspace)
}

// TestWorkspaceConfig_AddToHistory tests the history management
func TestWorkspaceConfig_AddToHistory(t *testing.T) {
	cfg := &WorkspaceConfig{
		MaxWorkspaceHistory: 3,
	}

	cfg.addToHistory("/path/a")
	cfg.addToHistory("/path/b")
	cfg.addToHistory("/path/c")

	assert.Equal(t, []string{"/path/c", "/path/b", "/path/a"}, cfg.WorkspaceHistory)
}

// TestWorkspaceConfig_AddToHistory_Duplicate tests that duplicates are moved to front
func TestWorkspaceConfig_AddToHistory_Duplicate(t *testing.T) {
	cfg := &WorkspaceConfig{
		WorkspaceHistory:    []string{"/path/a", "/path/b", "/path/c"},
		MaxWorkspaceHistory: 5,
	}

	cfg.addToHistory("/path/b")

	assert.Equal(t, []string{"/path/b", "/path/a", "/path/c"}, cfg.WorkspaceHistory)
}

// TestWorkspaceConfig_AddToHistory_MaxSize tests that history is trimmed to max size
func TestWorkspaceConfig_AddToHistory_MaxSize(t *testing.T) {
	cfg := &WorkspaceConfig{
		MaxWorkspaceHistory: 3,
	}

	cfg.addToHistory("/path/a")
	cfg.addToHistory("/path/b")
	cfg.addToHistory("/path/c")
	cfg.addToHistory("/path/d")

	assert.Len(t, cfg.WorkspaceHistory, 3)
	assert.Equal(t, []string{"/path/d", "/path/c", "/path/b"}, cfg.WorkspaceHistory)
}

// TestWorkspaceConfig_AddToHistory_EmptyPath tests that empty paths are ignored
func TestWorkspaceConfig_AddToHistory_EmptyPath(t *testing.T) {
	cfg := &WorkspaceConfig{
		WorkspaceHistory:    []string{"/path/a"},
		MaxWorkspaceHistory: 5,
	}

	cfg.addToHistory("")

	assert.Equal(t, []string{"/path/a"}, cfg.WorkspaceHistory)
}

// TestWorkspaceConfig_GetWorkspaceHistory tests retrieving history
func TestWorkspaceConfig_GetWorkspaceHistory(t *testing.T) {
	cfg := &WorkspaceConfig{
		WorkspaceHistory: []string{"/path/a", "/path/b", "/path/c"},
	}

	history := cfg.GetWorkspaceHistory()
	assert.Equal(t, []string{"/path/a", "/path/b", "/path/c"}, history)
}

// TestWorkspaceConfig_SetWorkspace_UpdatesHistory tests that SetWorkspace updates history
func TestWorkspaceConfig_SetWorkspace_UpdatesHistory(t *testing.T) {
	tmpDir1, err := os.MkdirTemp("", "workspace-test-1-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir1) })

	tmpDir2, err := os.MkdirTemp("", "workspace-test-2-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir2) })

	cfg := &WorkspaceConfig{
		MaxWorkspaceHistory: 5,
	}

	err = cfg.SetWorkspace(tmpDir1)
	require.NoError(t, err)

	absPath1, err := filepath.Abs(tmpDir1)
	require.NoError(t, err)
	assert.Equal(t, absPath1, cfg.CurrentWorkspace)
	assert.Contains(t, cfg.WorkspaceHistory, absPath1)

	err = cfg.SetWorkspace(tmpDir2)
	require.NoError(t, err)

	absPath2, err := filepath.Abs(tmpDir2)
	require.NoError(t, err)
	assert.Equal(t, absPath2, cfg.CurrentWorkspace)
	assert.Equal(t, absPath2, cfg.WorkspaceHistory[0], "New workspace should be at front")
}

// TestNewService_WorkspaceConfig_Default tests that default workspace config is applied
func TestNewService_WorkspaceConfig_Default(t *testing.T) {
	unsetEnv(t, "GOLLUM_WORKSPACE_CURRENT_WORKSPACE")
	unsetEnv(t, "GOLLUM_WORKSPACE_MAX_WORKSPACE_HISTORY")

	injector := do.New()
	service, err := NewService(injector)
	require.NoError(t, err)

	config := service.GetWorkspaceConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "", config.CurrentWorkspace)
	assert.Equal(t, 5, config.MaxWorkspaceHistory)
}

// TestNewService_WorkspaceConfig_FromEnv tests that environment variables are parsed
func TestNewService_WorkspaceConfig_FromEnv(t *testing.T) {
	unsetEnv(t, "GOLLUM_WORKSPACE_CURRENT_WORKSPACE")
	unsetEnv(t, "GOLLUM_WORKSPACE_MAX_WORKSPACE_HISTORY")

	require.NoError(t, os.Setenv("GOLLUM_WORKSPACE_CURRENT_WORKSPACE", "/custom/workspace"))
	require.NoError(t, os.Setenv("GOLLUM_WORKSPACE_MAX_WORKSPACE_HISTORY", "10"))
	t.Cleanup(func() {
		unsetEnv(t, "GOLLUM_WORKSPACE_CURRENT_WORKSPACE")
		unsetEnv(t, "GOLLUM_WORKSPACE_MAX_WORKSPACE_HISTORY")
	})

	injector := do.New()
	service, err := NewService(injector)
	require.NoError(t, err)

	config := service.GetWorkspaceConfig()
	assert.Equal(t, "/custom/workspace", config.CurrentWorkspace)
	assert.Equal(t, 10, config.MaxWorkspaceHistory)
}

// TestNewService_WorkspaceConfig_MaxHistoryValidation tests that MaxWorkspaceHistory is validated
func TestNewService_WorkspaceConfig_MaxHistoryValidation(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		expectedMax int
	}{
		{
			name:        "Zero_value_gets_default",
			envValue:    "0",
			expectedMax: 5,
		},
		{
			name:        "Negative_value_gets_default",
			envValue:    "-5",
			expectedMax: 5,
		},
		{
			name:        "Valid_value_preserved",
			envValue:    "10",
			expectedMax: 10,
		},
		{
			name:        "One_is_valid",
			envValue:    "1",
			expectedMax: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsetEnv(t, "GOLLUM_WORKSPACE_CURRENT_WORKSPACE")
			unsetEnv(t, "GOLLUM_WORKSPACE_MAX_WORKSPACE_HISTORY")

			require.NoError(t, os.Setenv("GOLLUM_WORKSPACE_MAX_WORKSPACE_HISTORY", tt.envValue))
			t.Cleanup(func() {
				unsetEnv(t, "GOLLUM_WORKSPACE_MAX_WORKSPACE_HISTORY")
			})

			injector := do.New()
			service, err := NewService(injector)
			require.NoError(t, err)

			config := service.GetWorkspaceConfig()
			assert.Equal(t, tt.expectedMax, config.MaxWorkspaceHistory)
		})
	}
}

// TestConfigService_WorkspaceMethods tests the ConfigService workspace methods
func TestConfigService_WorkspaceMethods(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workspace-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	unsetEnv(t, "GOLLUM_WORKSPACE_CURRENT_WORKSPACE")
	unsetEnv(t, "GOLLUM_WORKSPACE_MAX_WORKSPACE_HISTORY")

	injector := do.New()
	service, err := NewService(injector)
	require.NoError(t, err)

	err = service.SetCurrentWorkspace(tmpDir)
	require.NoError(t, err)

	current := service.GetCurrentWorkspace()
	absPath, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, absPath, current)

	history := service.GetWorkspaceHistory()
	assert.Contains(t, history, absPath)
}

// TestConfigService_SetCurrentWorkspace_InvalidPath tests error handling via service
func TestConfigService_SetCurrentWorkspace_InvalidPath(t *testing.T) {
	unsetEnv(t, "GOLLUM_WORKSPACE_CURRENT_WORKSPACE")

	injector := do.New()
	service, err := NewService(injector)
	require.NoError(t, err)

	err = service.SetCurrentWorkspace("/nonexistent/path")
	assert.Error(t, err)
}

// TestGetWorkspaceConfig_ReturnsPointer tests that GetWorkspaceConfig returns a stable pointer
func TestGetWorkspaceConfig_ReturnsPointer(t *testing.T) {
	unsetEnv(t, "GOLLUM_WORKSPACE_CURRENT_WORKSPACE")
	unsetEnv(t, "GOLLUM_WORKSPACE_MAX_WORKSPACE_HISTORY")

	injector := do.New()
	service, err := NewService(injector)
	require.NoError(t, err)

	config1 := service.GetWorkspaceConfig()
	config2 := service.GetWorkspaceConfig()

	assert.Same(t, config1, config2, "GetWorkspaceConfig should return stable pointer")
}
