package state

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"

	"github.com/denkhaus/gollum/pkg/errs"
)

// StartWatcher starts the file change watcher
func (fsm *fileStateManager) StartWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errs.Wrap(err, errs.TypeInternal, "failed to create watcher")
	}

	fsm.watcher = watcher
	fsm.watcherEnabled = true
	fsm.watchDone = make(chan struct{})

	// Use root directory from config
	rootDir := fsm.cfg.RootDir
	if rootDir == "" {
		// Fallback to cwd if not configured
		rootDir, err = os.Getwd()
		if err != nil {
			return errs.Wrap(err, errs.TypeInternal, "failed to get working directory")
		}
	}

	// Add root directory to watcher
	if err := watcher.Add(rootDir); err != nil {
		return errs.Wrap(err, errs.TypeInternal, "failed to watch root dir").
			WithContext("root_dir", rootDir)
	}

	// Start watcher loop
	go fsm.watchLoop()

	fsm.log.Info("File watcher started",
		zap.String("root_dir", rootDir),
		zap.Duration("debounce", fsm.cfg.GetDebounceDuration()))

	return nil
}

// StopWatcher stops the file change watcher
func (fsm *fileStateManager) StopWatcher() error {
	if fsm.watcher == nil {
		return nil
	}

	// Signal stop
	close(fsm.watchDone)

	// Close watcher
	if err := fsm.watcher.Close(); err != nil {
		return errs.Wrap(err, errs.TypeInternal, "failed to close watcher")
	}

	fsm.watcher = nil
	fsm.watcherEnabled = false

	fsm.log.Info("File watcher stopped")
	return nil
}

// IsWatcherRunning returns true if watcher is active
func (fsm *fileStateManager) IsWatcherRunning() bool {
	return fsm.watcherEnabled && fsm.watcher != nil
}

// GetWatcherDebounce returns the file watcher's debounce duration
func (fsm *fileStateManager) GetWatcherDebounce() time.Duration {
	return fsm.cfg.GetDebounceDuration()
}

// watchLoop processes file change events
func (fsm *fileStateManager) watchLoop() {
	defer close(fsm.watchDone)

	// Debounce map: path -> last update time
	var debounceMu sync.Mutex
	debounceMap := make(map[string]time.Time)

	ignoredDirs := fsm.cfg.GetIgnoredDirsSet()

	for {
		select {
		case event, ok := <-fsm.watcher.Events:
			if !ok {
				return
			}

			// Check if path should be ignored
			if fsm.shouldIgnorePath(event.Name, ignoredDirs) {
				continue
			}

			// Handle different event types
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				// Debounce: wait for brief pause before update
				debounceMu.Lock()
				debounceMap[event.Name] = time.Now()
				debounceMu.Unlock()

				time.AfterFunc(fsm.cfg.GetDebounceDuration(), func() {
					debounceMu.Lock()
					lastUpdate := debounceMap[event.Name]
					delete(debounceMap, event.Name)
					debounceMu.Unlock()

					// Only update if no new events in meantime
					if time.Since(lastUpdate) >= fsm.cfg.GetDebounceDuration() {
						fsm.updateFileStats(event.Name)
					}
				})
			}

			// On Remove: delete stats
			if event.Has(fsnotify.Remove) {
				fsm.mu.Lock()
				delete(fsm.stats, event.Name)
				fsm.mu.Unlock()
				fsm.log.Debug("File removed, stats deleted", zap.String("path", event.Name))
			}

			// On Rename: delete old path stats
			if event.Has(fsnotify.Rename) {
				fsm.mu.Lock()
				delete(fsm.stats, event.Name)
				fsm.mu.Unlock()
				fsm.log.Debug("File renamed, stats deleted", zap.String("path", event.Name))
			}

		case err, ok := <-fsm.watcher.Errors:
			if !ok {
				return
			}
			fsm.log.Error("File watcher error", zap.Error(err))

		case <-fsm.watchDone:
			return
		}
	}
}

// shouldIgnorePath checks if a path should be ignored
func (fsm *fileStateManager) shouldIgnorePath(path string, ignoredDirs map[string]bool) bool {
	// Skip hidden files
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") {
		return true
	}

	// Skip ignored directories
	for _, dir := range filepath.SplitList(path) {
		if ignoredDirs[dir] {
			return true
		}
	}

	return false
}

// updateFileStats updates stats for a single file
func (fsm *fileStateManager) updateFileStats(path string) {
	info, err := os.Stat(path)
	if err != nil {
		// File deleted or inaccessible
		fsm.mu.Lock()
		delete(fsm.stats, path)
		fsm.mu.Unlock()
		return
	}

	if info.IsDir() {
		// Directory: add to watcher
		if fsm.watcher != nil {
			if err := fsm.watcher.Add(path); err != nil {
				fsm.log.Debug("Failed to watch directory", zap.String("path", path), zap.Error(err))
			}
		}
		return
	}

	// Calculate checksum and update stats
	checksum, err := fsm.calculateChecksum(path)
	if err != nil {
		fsm.log.Debug("Failed to calculate checksum", zap.String("path", path), zap.Error(err))
		return
	}

	fsm.mu.Lock()
	fsm.stats[path] = &FileStats{
		Path:         path,
		Checksum:     checksum,
		ModifiedTime: info.ModTime(),
		Size:         info.Size(),
		LastScanned:  time.Now(),
	}
	fsm.mu.Unlock()

	fsm.log.Debug("File stats updated", zap.String("path", path), zap.String("checksum", checksum))
}
