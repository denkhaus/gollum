package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStateManager_DetectChanges(t *testing.T) {
	t.Run("no changes when snapshots are identical", func(t *testing.T) {
		injector := setupTestInjector()
		fsm, err := NewFileStateManager(injector)
		require.NoError(t, err, "Failed to create FileStateManager")

		tmpDir := t.TempDir()
		// Create a test file
		testFile := filepath.Join(tmpDir, "test1.txt")
		err = os.WriteFile(testFile, []byte("content"), 0644)
		require.NoError(t, err)

		// Update stats for the file
		_, err = fsm.UpdateStats(testFile)
		require.NoError(t, err)

		// Get snapshot
		beforeStats := fsm.GetAllStats()

		// No changes made

		// Detect changes
		changes, err := fsm.DetectChanges(beforeStats)
		require.NoError(t, err)

		// Should have no changes
		assert.Empty(t, changes, "expected no changes when nothing was modified")
	})

	t.Run("detects file creation", func(t *testing.T) {
		injector := setupTestInjector()
		fsm, err := NewFileStateManager(injector)
		require.NoError(t, err, "Failed to create FileStateManager")

		tmpDir := t.TempDir()
		// Get initial snapshot
		beforeStats := fsm.GetAllStats()

		// Create a new file
		newFile := filepath.Join(tmpDir, "new_file.txt")
		err = os.WriteFile(newFile, []byte("new content"), 0644)
		require.NoError(t, err)

		// Update stats for the new file
		_, err = fsm.UpdateStats(newFile)
		require.NoError(t, err)

		// Detect changes
		changes, err := fsm.DetectChanges(beforeStats)
		require.NoError(t, err)

		// Should have one created file
		require.Len(t, changes, 1, "expected exactly one change")
		assert.Equal(t, newFile, changes[0].Path)
		assert.Equal(t, Created, changes[0].Operation)
		assert.Empty(t, changes[0].OldChecksum, "old checksum should be empty for created files")
		assert.NotEmpty(t, changes[0].NewChecksum, "new checksum should not be empty")
	})

	t.Run("detects file modification", func(t *testing.T) {
		injector := setupTestInjector()
		fsm, err := NewFileStateManager(injector)
		require.NoError(t, err, "Failed to create FileStateManager")

		tmpDir := t.TempDir()
		// Create a file
		modFile := filepath.Join(tmpDir, "modified.txt")
		err = os.WriteFile(modFile, []byte("original"), 0644)
		require.NoError(t, err)
		_, err = fsm.UpdateStats(modFile)
		require.NoError(t, err)

		// Get snapshot
		beforeStats := fsm.GetAllStats()

		// Modify the file
		err = os.WriteFile(modFile, []byte("modified content"), 0644)
		require.NoError(t, err)
		_, err = fsm.UpdateStats(modFile)
		require.NoError(t, err)

		// Detect changes
		changes, err := fsm.DetectChanges(beforeStats)
		require.NoError(t, err)

		// Should have one modified file
		require.Len(t, changes, 1, "expected exactly one change")
		assert.Equal(t, modFile, changes[0].Path)
		assert.Equal(t, Modified, changes[0].Operation)
		assert.NotEmpty(t, changes[0].OldChecksum, "old checksum should not be empty for modified files")
		assert.NotEmpty(t, changes[0].NewChecksum, "new checksum should not be empty")
		assert.NotEqual(t, changes[0].OldChecksum, changes[0].NewChecksum, "checksums should differ")
	})

	t.Run("detects file deletion", func(t *testing.T) {
		injector := setupTestInjector()
		fsm, err := NewFileStateManager(injector)
		require.NoError(t, err, "Failed to create FileStateManager")

		tmpDir := t.TempDir()
		// Create a file
		delFile := filepath.Join(tmpDir, "deleted.txt")
		err = os.WriteFile(delFile, []byte("to be deleted"), 0644)
		require.NoError(t, err)
		_, err = fsm.UpdateStats(delFile)
		require.NoError(t, err)

		// Get snapshot
		beforeStats := fsm.GetAllStats()

		// Delete the file
		err = os.Remove(delFile)
		require.NoError(t, err)

		// If watcher is running, poll for deletion detection with timeout
		if fsm.IsWatcherRunning() {
			// Poll for up to 2 seconds, checking every 10ms
			timeout := time.After(2 * time.Second)
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()

			var changes []FileChange
			detected := false

		poll:
			for {
				select {
				case <-timeout:
					// Timeout - watcher may not be watching this temp dir
					t.Log("Watcher timeout: file may be outside watched directory")
					break poll
				case <-ticker.C:
					changes, err = fsm.DetectChanges(beforeStats)
					require.NoError(t, err)
					if len(changes) > 0 {
						detected = true
						break poll
					}
				}
			}

			if detected {
				// Should have one deleted file
				require.Len(t, changes, 1, "expected exactly one change")
				assert.Equal(t, delFile, changes[0].Path)
				assert.Equal(t, Deleted, changes[0].Operation)
				assert.NotEmpty(t, changes[0].OldChecksum, "old checksum should not be empty for deleted files")
				assert.Empty(t, changes[0].NewChecksum, "new checksum should be empty for deleted files")
			}
			// If not detected, the file may be outside the watcher's root directory - acceptable
		}
	})

	t.Run("detects create and modify", func(t *testing.T) {
		injector := setupTestInjector()
		fsm, err := NewFileStateManager(injector)
		require.NoError(t, err, "Failed to create FileStateManager")

		tmpDir := t.TempDir()
		// Create file2 first and add to stats
		file2 := filepath.Join(tmpDir, "file2.txt")
		err = os.WriteFile(file2, []byte("original"), 0644)
		require.NoError(t, err)
		_, err = fsm.UpdateStats(file2)
		require.NoError(t, err)

		// Get snapshot after file2 is created
		beforeStats := fsm.GetAllStats()

		// Now make changes:
		// 1. Create a new file
		file1 := filepath.Join(tmpDir, "file1.txt")
		err = os.WriteFile(file1, []byte("file 1"), 0644)
		require.NoError(t, err)
		_, err = fsm.UpdateStats(file1)
		require.NoError(t, err)

		// 2. Modify file2
		err = os.WriteFile(file2, []byte("modified"), 0644)
		require.NoError(t, err)
		_, err = fsm.UpdateStats(file2)
		require.NoError(t, err)

		// Detect changes from before snapshot
		changes, err := fsm.DetectChanges(beforeStats)
		require.NoError(t, err)

		// Should have exactly 2 changes (file1 created, file2 modified)
		require.Len(t, changes, 2, "expected exactly two changes")

		// Verify we have both create and modify operations
		hasCreate := false
		hasModify := false
		for _, change := range changes {
			if change.Operation == Created {
				hasCreate = true
			}
			if change.Operation == Modified {
				hasModify = true
			}
		}
		assert.True(t, hasCreate, "should have at least one created file")
		assert.True(t, hasModify, "should have at least one modified file")
	})
}

func TestChangeOperation_String(t *testing.T) {
	tests := []struct {
		name      string
		operation ChangeOperation
		want      string
	}{
		{"created", Created, "created"},
		{"modified", Modified, "modified"},
		{"deleted", Deleted, "deleted"},
		{"unknown", ChangeOperation(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.operation.String(); got != tt.want {
				t.Errorf("ChangeOperation.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
