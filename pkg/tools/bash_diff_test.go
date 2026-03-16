package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBashTool_Run_WithDiffIntegration_FileModification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	mockDiffProvider := mocks.NewMockProvider(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	agentID := uuid.New()

	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	absPath := testFile

	// Write initial content
	initialContent := "old content"
	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	require.NoError(t, err)

	tool := &bashToolImpl{
		logService:   logService,
		fileState:    mockFSM,
		agentID:      agentID,
		hookManager:  mockHookManager,
		diffProvider: mockDiffProvider,
		bashCfg:      &config.BashConfig{TrackChanges: true},
	}

	ctx := context.Background()

	// Mock GetAllStats to return before stats
	mockFSM.EXPECT().GetAllStats().Return(map[string]*state.FileStats{
		absPath: {
			Path:     absPath,
			Checksum: "checksum1",
			Size:     int64(len(initialContent)),
		},
	})

	// Mock GetWatcherDebounce
	mockFSM.EXPECT().GetWatcherDebounce().Return(time.Millisecond * 100)

	// Mock DetectChanges to return the modified file
	mockFSM.EXPECT().DetectChanges(gomock.Any()).Return([]state.FileChange{
		{
			Path:        absPath,
			Operation:   state.Modified,
			OldChecksum: "oldchecksum",
			NewChecksum: "newchecksum",
		},
	}, nil)

	// Mock diff provider methods
	expectedDiff := "--- " + absPath + "\n+++ " + absPath + "\n@@ -1,1 +1,1 @@\n-old content\n+new content"
	mockDiffProvider.EXPECT().GenerateDiff(absPath, absPath, "", "new content\n").Return(expectedDiff, nil)
	mockDiffProvider.EXPECT().FormatForDisplay(expectedDiff).Return("formatted: " + expectedDiff)
	mockDiffProvider.EXPECT().FormatCompact(expectedDiff).Return("compact: " + expectedDiff)

	// Execute the tool - this will actually run the bash command
	// which modifies the file, then DetectChanges is called, then diff is generated
	result, err := tool.Run(ctx, map[string]any{
		"command": "echo 'new content' > " + absPath,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))

	// Verify diff information is present in file_diffs
	fileDiffs, ok := result["file_diffs"].(map[string]interface{})
	require.True(t, ok)

	fileDiff, ok := fileDiffs[absPath].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "formatted: "+expectedDiff, fileDiff[string(shared.KeyDiff)])
	assert.Equal(t, "compact: "+expectedDiff, fileDiff[string(shared.KeyDiffCompact)])
	assert.False(t, fileDiff[string(shared.KeyIsNewFile)].(bool))
}

func TestBashTool_Run_WithDiffIntegration_FileCreation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	mockDiffProvider := mocks.NewMockProvider(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	agentID := uuid.New()

	// Create temporary directory
	tmpDir := t.TempDir()
	newFile := filepath.Join(tmpDir, "new.txt")
	absPath := newFile

	tool := &bashToolImpl{
		logService:   logService,
		fileState:    mockFSM,
		agentID:      agentID,
		hookManager:  mockHookManager,
		diffProvider: mockDiffProvider,
		bashCfg:      &config.BashConfig{TrackChanges: true},
	}

	ctx := context.Background()

	// Mock GetAllStats to return empty before stats (file doesn't exist yet)
	mockFSM.EXPECT().GetAllStats().Return(map[string]*state.FileStats{})

	// Mock GetWatcherDebounce
	mockFSM.EXPECT().GetWatcherDebounce().Return(time.Millisecond * 100)

	// Mock DetectChanges to return the new file
	mockFSM.EXPECT().DetectChanges(gomock.Any()).Return([]state.FileChange{
		{
			Path:        absPath,
			Operation:   state.Created,
			OldChecksum: "",
			NewChecksum: "newchecksum",
		},
	}, nil)

	// Mock diff provider methods
	// Note: echo adds a trailing newline, so content is "new file content\n"
	newContent := "new file content\n"
	expectedDiff := "--- /dev/null\n+++ " + absPath + "\n@@ -0,0 +1,1 @@\n+" + newContent
	mockDiffProvider.EXPECT().GenerateDiffForNewFile(absPath, newContent).Return(expectedDiff, nil)
	mockDiffProvider.EXPECT().FormatForDisplay(expectedDiff).Return("formatted: " + expectedDiff)
	mockDiffProvider.EXPECT().FormatCompact(expectedDiff).Return("compact: " + expectedDiff)

	// Execute the tool - this will create the file
	result, err := tool.Run(ctx, map[string]any{
		"command": "echo 'new file content' > " + absPath,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))

	// Verify diff information is present in file_diffs
	fileDiffs, ok := result["file_diffs"].(map[string]interface{})
	require.True(t, ok)

	fileDiff, ok := fileDiffs[absPath].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "formatted: "+expectedDiff, fileDiff[string(shared.KeyDiff)])
	assert.Equal(t, "compact: "+expectedDiff, fileDiff[string(shared.KeyDiffCompact)])
	assert.True(t, fileDiff[string(shared.KeyIsNewFile)].(bool))
}

func TestBashTool_Run_WithDiffIntegration_NoFileChanges(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	mockDiffProvider := mocks.NewMockProvider(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	agentID := uuid.New()

	tool := &bashToolImpl{
		logService:   logService,
		fileState:    mockFSM,
		agentID:      agentID,
		hookManager:  mockHookManager,
		diffProvider: mockDiffProvider,
		bashCfg:      &config.BashConfig{TrackChanges: true},
	}

	ctx := context.Background()

	// Mock GetAllStats to return empty before stats
	mockFSM.EXPECT().GetAllStats().Return(map[string]*state.FileStats{})

	// Mock GetWatcherDebounce
	mockFSM.EXPECT().GetWatcherDebounce().Return(time.Millisecond * 100)

	// Mock DetectChanges to return no files
	mockFSM.EXPECT().DetectChanges(gomock.Any()).Return([]state.FileChange{}, nil)

	// Execute the tool - command that doesn't modify files
	result, err := tool.Run(ctx, map[string]any{
		"command": "echo hello",
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))

	// Verify no diff information is present
	_, hasFileDiffs := result["file_diffs"]
	assert.False(t, hasFileDiffs)
}

func TestBashTool_Run_WithDiffIntegration_MultipleFileChanges(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := setupTestInjector()
	logService := do.MustInvoke[logger.LoggerService](injector)
	mockFSM := mocks.NewMockFileStateManager(ctrl)
	mockHookManager := hooks.NewMockHookManager(ctrl)
	mockDiffProvider := mocks.NewMockProvider(ctrl)
	setupMockHookManagerPassThrough(mockHookManager)

	agentID := uuid.New()

	// Create temporary test files
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")
	absPath1 := testFile1
	absPath2 := testFile2

	// Write initial content
	err := os.WriteFile(testFile1, []byte("old content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("old content"), 0644)
	require.NoError(t, err)

	tool := &bashToolImpl{
		logService:   logService,
		fileState:    mockFSM,
		agentID:      agentID,
		hookManager:  mockHookManager,
		diffProvider: mockDiffProvider,
		bashCfg:      &config.BashConfig{TrackChanges: true},
	}

	ctx := context.Background()

	// Mock GetAllStats to return before stats for both files
	mockFSM.EXPECT().GetAllStats().Return(map[string]*state.FileStats{
		absPath1: {
			Path:     absPath1,
			Checksum: "checksum1",
			Size:     int64(len("old content")),
		},
		absPath2: {
			Path:     absPath2,
			Checksum: "checksum2",
			Size:     int64(len("old content")),
		},
	})

	// Mock GetWatcherDebounce
	mockFSM.EXPECT().GetWatcherDebounce().Return(time.Millisecond * 100)

	// Mock DetectChanges to return both modified files
	mockFSM.EXPECT().DetectChanges(gomock.Any()).Return([]state.FileChange{
		{
			Path:        absPath1,
			Operation:   state.Modified,
			OldChecksum: "oldchecksum1",
			NewChecksum: "newchecksum1",
		},
		{
			Path:        absPath2,
			Operation:   state.Modified,
			OldChecksum: "oldchecksum2",
			NewChecksum: "newchecksum2",
		},
	}, nil)

	// Mock diff provider methods for both files
	expectedDiff1 := "--- " + absPath1 + "\n+++ " + absPath1 + "\n@@ -1,1 +1,1 @@\n-old content\n+new content"
	expectedDiff2 := "--- " + absPath2 + "\n+++ " + absPath2 + "\n@@ -1,1 +1,1 @@\n-old content\n+new content"

	// Use gomock.Any() for parameters since order may vary
	mockDiffProvider.EXPECT().GenerateDiff(gomock.Any(), gomock.Any(), "", "new content\n").
		DoAndReturn(func(oldPath, newPath, oldContent, newContent string) (string, error) {
			if newPath == absPath1 {
				return expectedDiff1, nil
			}
			return expectedDiff2, nil
		}).Times(2)

	mockDiffProvider.EXPECT().FormatForDisplay(gomock.Any()).
		DoAndReturn(func(diff string) string {
			return "formatted: " + diff
		}).Times(2)

	mockDiffProvider.EXPECT().FormatCompact(gomock.Any()).
		DoAndReturn(func(diff string) string {
			return "compact: " + diff
		}).Times(2)

	// Execute the tool - this will modify both files
	result, err := tool.Run(ctx, map[string]any{
		"command": "echo 'new content' > " + absPath1 + " && echo 'new content' > " + absPath2,
	})

	require.NoError(t, err)
	assert.True(t, result["success"].(bool))

	// Verify diff information is present for both files in file_diffs
	// Note: Map iteration order is non-deterministic, so check both files
	fileDiffs, ok := result["file_diffs"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(fileDiffs), "Expected diffs for 2 files")

	// Helper to verify a file's diff
	verifyDiff := func(absPath string) {
		expectedDiff := "--- " + absPath + "\n+++ " + absPath + "\n@@ -1,1 +1,1 @@\n-old content\n+new content"

		fileDiff, ok := fileDiffs[absPath].(map[string]interface{})
		require.True(t, ok, "Expected diff for file: "+absPath)
		assert.Equal(t, "formatted: "+expectedDiff, fileDiff[string(shared.KeyDiff)])
		assert.Equal(t, "compact: "+expectedDiff, fileDiff[string(shared.KeyDiffCompact)])
		assert.False(t, fileDiff[string(shared.KeyIsNewFile)].(bool))
	}

	// Verify both files have correct diffs
	verifyDiff(absPath1)
	verifyDiff(absPath2)
}
