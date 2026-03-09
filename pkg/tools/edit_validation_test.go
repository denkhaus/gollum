package tools

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestEditToolValidation tests invalid input handling
func TestEditToolValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	mockFSM := mocks.NewMockFileStateManager(ctrl)

	agentID := uuid.New()
	tool := &editToolImpl{
		logService:  logService,
		fsm:         mockFSM,
		hookManager: mockHookManager,
		agentID:     agentID,
	}

	// Set up mock hookManager to pass through calls (no hooks registered)
	setupMockHookManagerPassThrough(mockHookManager)

	ctx := context.Background()

	t.Run("Missing file_path", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"old_string": "foo",
			"new_string": "bar",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "file_path is required")
	})

	t.Run("Empty file_path", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"file_path":  "",
			"old_string": "foo",
			"new_string": "bar",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "file_path is required")
	})

	t.Run("Missing old_string", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"file_path":  "/tmp/test.txt",
			"new_string": "bar",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "old_string is required")
	})

	t.Run("Empty old_string", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"file_path":  "/tmp/test.txt",
			"old_string": "",
			"new_string": "bar",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "old_string is required")
	})

	t.Run("Missing new_string", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"file_path":  "/tmp/test.txt",
			"old_string": "foo",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "new_string is required")
	})

	t.Run("Non-string file_path", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"file_path":  123,
			"old_string": "foo",
			"new_string": "bar",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "file_path is required")
	})

	t.Run("Non-string old_string", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"file_path":  "/tmp/test.txt",
			"old_string": 123,
			"new_string": "bar",
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "old_string is required")
	})

	t.Run("Non-string new_string", func(t *testing.T) {
		result, err := tool.Run(ctx, map[string]any{
			"file_path":  "/tmp/test.txt",
			"old_string": "foo",
			"new_string": 123,
		})

		require.NoError(t, err)
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "new_string is required")
	})
}

// TestEditToolFileNotRead tests error when file was not read first
func TestEditToolFileNotRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	agentID := uuid.New()
	testFile := "/tmp/test_edit.txt"
	absPath, _ := filepath.Abs(testFile)

	tool := &editToolImpl{
		logService:  logService,
		fsm:         mockFSM,
		agentID:     agentID,
		hookManager: mockHookManager,
	}

	ctx := context.Background()

	// Mock GetFileStats to return nil (file not tracked)
	mockFSM.EXPECT().GetFileStats(absPath).Return(nil, nil)

	result, err := tool.Run(ctx, map[string]any{
		"file_path":  testFile,
		"old_string": "foo",
		"new_string": "bar",
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "must be read before editing")
	assert.Contains(t, result["error"].(string), "read_file")
}

// TestEditToolStaleFile tests error when file is stale for agent
func TestEditToolStaleFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	agentID := uuid.New()
	testFile := "/tmp/test_edit.txt"
	absPath, _ := filepath.Abs(testFile)

	tool := &editToolImpl{
		logService:  logService,
		fsm:         mockFSM,
		agentID:     agentID,
		hookManager: mockHookManager,
	}

	ctx := context.Background()

	// Mock GetFileStats to return valid stats
	mockFSM.EXPECT().GetFileStats(absPath).Return(&state.FileStats{
		Path:     absPath,
		Checksum: "abc123",
	}, nil)

	// Mock IsFileStaleForAgent to return true (file is stale)
	mockFSM.EXPECT().IsFileStaleForAgent(agentID, absPath).Return(true, nil)

	result, err := tool.Run(ctx, map[string]any{
		"file_path":  testFile,
		"old_string": "foo",
		"new_string": "bar",
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "must read this file before editing")
}
