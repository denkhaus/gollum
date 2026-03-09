package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestEditToolBasicOperation tests successful single replacement
func TestEditToolBasicOperation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	agentID := uuid.New()

	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	absPath := testFile // Already absolute in temp dir

	// Write initial content
	initialContent := "Hello World\nfoo bar baz\nGoodbye"
	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	require.NoError(t, err)

	mockDiffProvider := mocks.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &editToolImpl{
		logService:   logService,
		fsm:          mockFSM,
		agentID:      agentID,
		hookManager:  mockHookManager,
		diffProvider: mockDiffProvider,
	}

	ctx := context.Background()

	// Mock GetFileStats to return valid stats
	oldChecksum := "checksum1"
	mockFSM.EXPECT().GetFileStats(absPath).Return(&state.FileStats{
		Path:     absPath,
		Checksum: oldChecksum,
		Size:     int64(len(initialContent)),
	}, nil)

	// Mock IsFileStaleForAgent to return false (file is fresh)
	mockFSM.EXPECT().IsFileStaleForAgent(agentID, absPath).Return(false, nil)

	// Mock DoWorkWithOptions to execute the edit
	mockFSM.EXPECT().DoWorkWithOptions(ctx, absPath, agentID, state.LockModeExclusive,
		gomock.Any(), // WorkOptions
		gomock.Any(), // WorkFunc
	).Do(func(_ context.Context, path string, _ uuid.UUID, _ state.LockMode, opts state.WorkOptions, workFunc func(context.Context, *state.LockToken) (any, error)) {
		// Verify WorkOptions
		assert.True(t, opts.UpdateStatsAfter)

		// Execute the work function to simulate the edit
		token := &state.LockToken{
			Path:       path,
			AgentID:    agentID,
			Mode:       state.LockModeExclusive,
			AcquiredAt: time.Now(),
			ExpiresAt:  time.Now().Add(5 * time.Minute),
		}

		result, err := workFunc(ctx, token)
		require.NoError(t, err)
		resultMap := result.(map[string]any)
		assert.True(t, resultMap["success"].(bool))
		assert.Equal(t, 1, resultMap["replacements"].(int))
	}).Return(map[string]any{
		"success":      true,
		"file_path":    absPath,
		"replacements": 1,
		"checksum":     "new_checksum",
	}, nil)

	// Mock GetFileStats after edit
	mockFSM.EXPECT().GetFileStats(absPath).Return(&state.FileStats{
		Path:         absPath,
		Checksum:     "new_checksum",
		Size:         int64(len(initialContent) - 3 + 6),
		ModifiedTime: time.Now(),
	}, nil)

	result, err := tool.Run(ctx, map[string]any{
		"file_path":  testFile,
		"old_string": "foo",
		"new_string": "barbar",
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, 1, result["replacements"].(int))
}

// TestEditToolStringNotFound tests error when old_string not found
func TestEditToolStringNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	agentID := uuid.New()

	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	absPath := testFile

	initialContent := "Hello World\nbar baz\nGoodbye"
	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	require.NoError(t, err)

	mockDiffProvider := mocks.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &editToolImpl{
		logService:   logService,
		fsm:          mockFSM,
		agentID:      agentID,
		hookManager:  mockHookManager,
		diffProvider: mockDiffProvider,
	}

	ctx := context.Background()

	mockFSM.EXPECT().GetFileStats(absPath).Return(&state.FileStats{
		Path:     absPath,
		Checksum: "checksum1",
	}, nil)
	mockFSM.EXPECT().IsFileStaleForAgent(agentID, absPath).Return(false, nil)

	mockFSM.EXPECT().DoWorkWithOptions(ctx, absPath, agentID, state.LockModeExclusive,
		gomock.Any(), gomock.Any()).Do(func(_ context.Context, path string, _ uuid.UUID, _ state.LockMode, _ state.WorkOptions, workFunc func(context.Context, *state.LockToken) (any, error)) {
		token := &state.LockToken{
			Path:       path,
			AgentID:    agentID,
			Mode:       state.LockModeExclusive,
			AcquiredAt: time.Now(),
			ExpiresAt:  time.Now().Add(5 * time.Minute),
		}

		result, err := workFunc(ctx, token)
		require.NoError(t, err)
		resultMap := result.(map[string]any)
		assert.False(t, resultMap["success"].(bool))
		assert.Contains(t, resultMap["error"].(string), "not found in file")
	}).Return(map[string]any{
		"success": false,
		"error":   "old_string not found in file",
	}, nil)

	result, err := tool.Run(ctx, map[string]any{
		"file_path":  testFile,
		"old_string": "foo",
		"new_string": "bar",
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "not found in file")
}

// TestEditToolMultipleOccurrences tests error when old_string appears multiple times without replace_all
func TestEditToolMultipleOccurrences(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	agentID := uuid.New()

	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	absPath := testFile

	initialContent := "foo bar foo baz foo"
	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	require.NoError(t, err)

	mockDiffProvider := mocks.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &editToolImpl{
		logService:   logService,
		fsm:          mockFSM,
		agentID:      agentID,
		hookManager:  mockHookManager,
		diffProvider: mockDiffProvider,
	}

	ctx := context.Background()

	mockFSM.EXPECT().GetFileStats(absPath).Return(&state.FileStats{
		Path:     absPath,
		Checksum: "checksum1",
	}, nil)
	mockFSM.EXPECT().IsFileStaleForAgent(agentID, absPath).Return(false, nil)

	mockFSM.EXPECT().DoWorkWithOptions(ctx, absPath, agentID, state.LockModeExclusive,
		gomock.Any(), gomock.Any()).Do(func(_ context.Context, path string, _ uuid.UUID, _ state.LockMode, _ state.WorkOptions, workFunc func(context.Context, *state.LockToken) (any, error)) {
		token := &state.LockToken{
			Path:       path,
			AgentID:    agentID,
			Mode:       state.LockModeExclusive,
			AcquiredAt: time.Now(),
			ExpiresAt:  time.Now().Add(5 * time.Minute),
		}

		result, err := workFunc(ctx, token)
		require.NoError(t, err)
		resultMap := result.(map[string]any)
		assert.False(t, resultMap["success"].(bool))
		assert.Contains(t, resultMap["error"].(string), "appears 3 times")
		assert.Equal(t, 3, resultMap["replacements"].(int))
	}).Return(map[string]any{
		"success":      false,
		"error":        "old_string appears 3 times in the file. For safety, it must be unique unless replace_all is set to true",
		"replacements": 3,
	}, nil)

	result, err := tool.Run(ctx, map[string]any{
		"file_path":  testFile,
		"old_string": "foo",
		"new_string": "bar",
	})

	require.NoError(t, err)
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "appears 3 times")
}

// TestEditToolReplaceAll tests successful replace_all operation
func TestEditToolReplaceAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := mocks.NewMockHookManager(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	agentID := uuid.New()

	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	absPath := testFile

	initialContent := "foo bar foo baz foo"
	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	require.NoError(t, err)

	mockDiffProvider := mocks.NewMockProvider(ctrl)
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
	tool := &editToolImpl{
		logService:   logService,
		fsm:          mockFSM,
		agentID:      agentID,
		hookManager:  mockHookManager,
		diffProvider: mockDiffProvider,
	}

	ctx := context.Background()

	mockFSM.EXPECT().GetFileStats(absPath).Return(&state.FileStats{
		Path:     absPath,
		Checksum: "checksum1",
	}, nil)
	mockFSM.EXPECT().IsFileStaleForAgent(agentID, absPath).Return(false, nil)

	mockFSM.EXPECT().DoWorkWithOptions(ctx, absPath, agentID, state.LockModeExclusive,
		gomock.Any(), gomock.Any()).Do(func(_ context.Context, path string, _ uuid.UUID, _ state.LockMode, opts state.WorkOptions, workFunc func(context.Context, *state.LockToken) (any, error)) {
		assert.True(t, opts.UpdateStatsAfter)

		token := &state.LockToken{
			Path:       path,
			AgentID:    agentID,
			Mode:       state.LockModeExclusive,
			AcquiredAt: time.Now(),
			ExpiresAt:  time.Now().Add(5 * time.Minute),
		}

		result, err := workFunc(ctx, token)
		require.NoError(t, err)
		resultMap := result.(map[string]any)
		assert.True(t, resultMap["success"].(bool))
		assert.Equal(t, 3, resultMap["replacements"].(int))
	}).Return(map[string]any{
		"success":      true,
		"file_path":    absPath,
		"replacements": 3,
		"checksum":     "new_checksum",
	}, nil)

	mockFSM.EXPECT().GetFileStats(absPath).Return(&state.FileStats{
		Path:         absPath,
		Checksum:     "new_checksum",
		Size:         int64(len("bar bar bar baz bar")),
		ModifiedTime: time.Now(),
	}, nil)

	result, err := tool.Run(ctx, map[string]any{
		"file_path":   testFile,
		"old_string":  "foo",
		"new_string":  "bar",
		"replace_all": true,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, 3, result["replacements"].(int))
}
