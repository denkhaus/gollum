package state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// fileStateManager is the concrete implementation of FileStateManager
type fileStateManager struct {
	mu              sync.RWMutex
	locks           map[string]*LockToken
	stats           map[string]*FileStats
	defaultTimeout  time.Duration
	cleanupInterval time.Duration

	// Read tracking for race condition detection
	readHistory map[string]*ReadRecord // path -> last read by an agent

	// File Watching
	watcher        *fsnotify.Watcher
	watcherEnabled bool
	watchDone      chan struct{}
	cfg            *config.FilesConfig

	// Dependencies
	log logger.LoggerService
}

// NewFileStateManager creates a new file state manager for tracking file locks and stats
func NewFileStateManager(injector do.Injector) (FileStateManager, error) {
	// Get config service
	cfgService, err := do.Invoke[config.ConfigService](injector)
	if err != nil {
		return nil, errs.Wrap(err, errs.TypeInternal, "failed to invoke config service")
	}

	// Get logger service
	logService, err := do.Invoke[logger.LoggerService](injector)
	if err != nil {
		return nil, errs.Wrap(err, errs.TypeInternal, "failed to invoke logger service")
	}

	fsm := &fileStateManager{
		locks:           make(map[string]*LockToken),
		stats:           make(map[string]*FileStats),
		readHistory:     make(map[string]*ReadRecord),
		defaultTimeout:  30 * time.Second,
		cleanupInterval: 1 * time.Minute,
		cfg:             cfgService.GetFilesConfig(),
		log:             logService,
	}

	// Start background cleanup goroutine
	go fsm.cleanupExpiredLocks()

	// Start file watcher if enabled
	if fsm.cfg.WatcherEnabled {
		if err := fsm.StartWatcher(); err != nil {
			logService.Warn("Failed to start file watcher", zap.Error(err))
		}
	}

	return fsm, nil
}

// Prime scans the directory and populates file stats
func (fsm *fileStateManager) Prime(ctx context.Context) error {

	opts := &ScanOptions{
		RootDir:       fsm.cfg.RootDir,
		MaxDepth:      fsm.cfg.MaxDepth,
		IncludeHidden: fsm.cfg.IncludeHidden,
		FilePattern:   fsm.cfg.FilePattern,
		IgnoreDirs:    fsm.cfg.IgnoredDirs,
		Concurrency:   fsm.cfg.Concurrency,
	}
	// Fallback RootDir to cwd if empty
	if opts.RootDir == "" {
		opts.RootDir, _ = os.Getwd()
	}

	// Build ignore set
	ignoreSet := make(map[string]bool)
	for _, dir := range opts.IgnoreDirs {
		ignoreSet[dir] = true
	}

	// Channel for files to process
	fileCh := make(chan string, opts.Concurrency*2)
	errCh := make(chan error, 1)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileCh {
				if err := fsm.scanFile(path); err != nil {
					// Log but don't fail entire scan
					fsm.log.Warn("Failed to scan file", zap.String("path", path), zap.Error(err))
				}
			}
		}()
	}

	// Scanner goroutine
	go func() {
		defer close(fileCh)
		if err := fsm.walkDirectory(ctx, opts.RootDir, opts.MaxDepth, 0, ignoreSet, opts, fileCh); err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	// Wait for completion
	wg.Wait()

	// Check for errors
	select {
	case err := <-errCh:
		return err
	default:
	}

	// Log results
	fsm.mu.RLock()
	count := len(fsm.stats)
	fsm.mu.RUnlock()

	fsm.log.Info("FileStateManager primed", zap.Int("files_indexed", count))

	return nil
}

// walkDirectory walks the directory tree and sends files to the channel
func (fsm *fileStateManager) walkDirectory(
	ctx context.Context,
	dir string,
	maxDepth, currentDepth int,
	ignoreSet map[string]bool,
	opts *ScanOptions,
	fileCh chan<- string,
) error {
	// Check max depth
	if maxDepth >= 0 && currentDepth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files if not included
		if !opts.IncludeHidden && strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := filepath.Join(dir, name)

		// Check if should ignore
		if entry.IsDir() {
			baseName := filepath.Base(fullPath)
			if ignoreSet[baseName] {
				continue
			}
			// Recurse into subdirectory
			if err := fsm.walkDirectory(ctx, fullPath, maxDepth, currentDepth+1, ignoreSet, opts, fileCh); err != nil {
				return err
			}
			continue
		}

		// Check file pattern
		if opts.FilePattern != "*" && opts.FilePattern != "" {
			matched, err := filepath.Match(opts.FilePattern, name)
			if err != nil || !matched {
				continue
			}
		}

		// Send file to be processed
		select {
		case fileCh <- fullPath:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// scanFile scans a single file and updates stats
func (fsm *fileStateManager) scanFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Calculate checksum
	checksum, err := fsm.calculateChecksum(path)
	if err != nil {
		return err
	}

	// Store stats
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	fsm.stats[path] = &FileStats{
		Path:         path,
		Checksum:     checksum,
		ModifiedTime: info.ModTime(),
		Size:         info.Size(),
		LastScanned:  time.Now(),
	}

	return nil
}

// AcquireLock acquires a lock on a file, waiting if necessary
func (fsm *fileStateManager) AcquireLock(ctx context.Context, path string, agentID uuid.UUID, mode LockMode) (*LockToken, error) {
	// Try to acquire lock immediately
	if token, err := fsm.tryAcquireLock(path, agentID, mode); err == nil {
		return token, nil
	}

	// If immediate acquisition failed, wait for the lock
	return fsm.waitForLock(ctx, path, agentID, mode)
}

// tryAcquireLock attempts to acquire a lock without waiting
func (fsm *fileStateManager) tryAcquireLock(path string, agentID uuid.UUID, mode LockMode) (*LockToken, error) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	existingLock, exists := fsm.locks[path]
	if !exists {
		// No existing lock, acquire it
		return fsm.createLock(path, agentID, mode), nil
	}

	// Check if existing lock is expired
	if time.Now().After(existingLock.ExpiresAt) {
		// Lock has expired, remove it and acquire new lock
		delete(fsm.locks, path)
		return fsm.createLock(path, agentID, mode), nil
	}

	// Allow shared locks for multiple readers
	if existingLock.Mode == LockModeShared && mode == LockModeShared {
		// Extend the expiration time for shared locks
		existingLock.ExpiresAt = time.Now().Add(fsm.defaultTimeout)
		return existingLock, nil
	}

	// Same agent re-acquiring lock
	if existingLock.AgentID == agentID {
		// Extend the expiration time
		existingLock.ExpiresAt = time.Now().Add(fsm.defaultTimeout)
		return existingLock, nil
	}

	// Lock is held by another agent
	return nil, errs.Conflictf("file is locked by agent %s", existingLock.AgentID).
		WithContext("path", path).
		WithContext("locked_by", existingLock.AgentID).
		WithContext("requested_by", agentID)
}

// waitForLock waits for a lock to become available
func (fsm *fileStateManager) waitForLock(ctx context.Context, path string, agentID uuid.UUID, mode LockMode) (*LockToken, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, errs.Internal("context canceled while waiting for lock").
				WithContext("path", path).
				WithContext("agent_id", agentID)
		case <-ticker.C:
			if token, err := fsm.tryAcquireLock(path, agentID, mode); err == nil {
				return token, nil
			}
		}
	}
}

// createLock creates a new lock token
func (fsm *fileStateManager) createLock(path string, agentID uuid.UUID, mode LockMode) *LockToken {
	token := &LockToken{
		Path:       path,
		AgentID:    agentID,
		Mode:       mode,
		AcquiredAt: time.Now(),
		ExpiresAt:  time.Now().Add(fsm.defaultTimeout),
	}

	fsm.locks[path] = token
	return token
}

// ReleaseLock releases a lock held by an agent
func (fsm *fileStateManager) ReleaseLock(path string, agentID uuid.UUID) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	lock, exists := fsm.locks[path]
	if !exists {
		return errs.NotFoundf("no lock found for path: %s", path).
			WithContext("path", path).
			WithContext("agent_id", agentID)
	}

	if lock.AgentID != agentID {
		return errs.Permissionf("lock is held by different agent: %s", lock.AgentID).
			WithContext("path", path).
			WithContext("lock_holder", lock.AgentID).
			WithContext("requesting_agent", agentID)
	}

	delete(fsm.locks, path)
	return nil
}

// DoWork executes work under a lock with automatic release.
func (fsm *fileStateManager) DoWork(ctx context.Context, path string, agentID uuid.UUID, mode LockMode, work WorkFunc) (any, error) {
	return fsm.DoWorkWithOptions(ctx, path, agentID, mode, WorkOptions{}, work)
}

// DoWorkWithOptions executes work under a lock with automatic release and options.
func (fsm *fileStateManager) DoWorkWithOptions(ctx context.Context, path string, agentID uuid.UUID, mode LockMode, opts WorkOptions, work WorkFunc) (any, error) {
	// Acquire lock
	token, err := fsm.AcquireLock(ctx, path, agentID, mode)
	if err != nil {
		return nil, errs.Wrap(err, errs.TypeInternal, "failed to acquire lock").
			WithContext("path", path).
			WithContext("agent_id", agentID)
	}

	// Ensure lock is released (even on panic)
	defer func() {
		if err := fsm.ReleaseLock(path, agentID); err != nil {
			fsm.log.Warn("Failed to release lock",
				zap.String("path", path),
				zap.String("agent", agentID.String()),
				zap.Error(err))
		}
	}()

	// Execute work under lock
	result, err := work(ctx, token)
	if err != nil {
		return result, err
	}

	// Track read if requested (for later race condition detection)
	if opts.TrackRead {
		checksum, err := fsm.calculateChecksum(path)
		if err != nil {
			fsm.log.Warn("Failed to calculate checksum for read tracking",
				zap.String("path", path),
				zap.Error(err))
		} else {
			fsm.mu.Lock()
			fsm.readHistory[path] = &ReadRecord{
				AgentID:   agentID,
				Checksum:  checksum,
				Timestamp: time.Now(),
			}
			fsm.mu.Unlock()
		}
	}

	// Update stats immediately after successful work if requested
	if opts.UpdateStatsAfter {
		if _, err := fsm.UpdateStats(path); err != nil {
			fsm.log.Warn("Failed to update stats after work",
				zap.String("path", path),
				zap.Error(err))
			// Don't fail the work if stats update fails
		}
	}

	return result, nil
}

// VerifyChecksum checks if a file's checksum matches the expected value
func (fsm *fileStateManager) VerifyChecksum(path, expected string) (bool, error) {
	if expected == "" {
		// No verification requested
		return true, nil
	}

	// Calculate current checksum
	current, err := fsm.calculateChecksum(path)
	if err != nil {
		return false, errs.Wrap(err, errs.TypeInternal, "failed to calculate checksum").
			WithContext("path", path)
	}

	return current == expected, nil
}

// UpdateChecksum calculates and stores the checksum for a file
func (fsm *fileStateManager) UpdateChecksum(path string) (string, error) {
	checksum, err := fsm.calculateChecksum(path)
	if err != nil {
		return "", err
	}

	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	// Update or create stats
	if stats, exists := fsm.stats[path]; exists {
		stats.Checksum = checksum
	} else {
		// Get file info
		info, err := os.Stat(path)
		if err == nil {
			fsm.stats[path] = &FileStats{
				Path:         path,
				Checksum:     checksum,
				ModifiedTime: info.ModTime(),
				Size:         info.Size(),
				LastScanned:  time.Now(),
			}
		}
	}

	return checksum, nil
}

// UpdateStats updates file stats after modification
func (fsm *fileStateManager) UpdateStats(path string) (*FileStats, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, errs.Wrap(err, errs.TypeInternal, "failed to stat file").
			WithContext("path", path)
	}

	checksum, err := fsm.calculateChecksum(path)
	if err != nil {
		return nil, errs.Wrap(err, errs.TypeInternal, "failed to calculate checksum").
			WithContext("path", path)
	}

	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	stats := &FileStats{
		Path:         path,
		Checksum:     checksum,
		ModifiedTime: info.ModTime(),
		Size:         info.Size(),
		LastScanned:  time.Now(),
	}

	fsm.stats[path] = stats

	return stats, nil
}

// GetFileStats returns cached stats for a file
func (fsm *fileStateManager) GetFileStats(path string) (*FileStats, error) {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	stats, exists := fsm.stats[path]
	if !exists {
		// Try to scan the file
		fsm.mu.RUnlock()
		if err := fsm.scanFile(path); err != nil {
			fsm.mu.RLock()
			return nil, errs.NotFoundf("file not found in cache: %s", path).
				WithContext("path", path)
		}
		fsm.mu.RLock()
		stats = fsm.stats[path]
	}

	// Copy to avoid race conditions
	statsCopy := *stats
	return &statsCopy, nil
}

// GetAllStats returns all tracked file stats
func (fsm *fileStateManager) GetAllStats() map[string]*FileStats {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	result := make(map[string]*FileStats, len(fsm.stats))
	for k, v := range fsm.stats {
		statsCopy := *v
		result[k] = &statsCopy
	}

	return result
}

// GetFileStatus returns the current status of a file (legacy compatibility)
func (fsm *fileStateManager) GetFileStatus(path string) (*FileStatus, error) {
	stats, err := fsm.GetFileStats(path)
	if err != nil {
		return nil, err
	}

	return &FileStatus{
		Path:     stats.Path,
		Checksum: stats.Checksum,
		LockedBy: stats.LockedBy,
		LockMode: stats.LockMode,
		LockedAt: stats.LockedAt,
	}, nil
}

// IsLocked checks if a file is currently locked
func (fsm *fileStateManager) IsLocked(path string) bool {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	lock, exists := fsm.locks[path]
	if !exists {
		return false
	}

	// Check if lock is expired
	return time.Now().Before(lock.ExpiresAt)
}

// GetLock returns the current lock on a file, if any
func (fsm *fileStateManager) GetLock(path string) (*LockToken, error) {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	lock, exists := fsm.locks[path]
	if !exists {
		return nil, errs.NotFoundf("no lock found for path: %s", path).
			WithContext("path", path)
	}

	return lock, nil
}

// calculateChecksum calculates the SHA-256 checksum of a file
func (fsm *fileStateManager) calculateChecksum(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", errs.Wrap(err, errs.TypeInternal, "failed to read file").
			WithContext("path", path)
	}

	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}

// cleanupExpiredLocks removes expired locks from the manager
func (fsm *fileStateManager) cleanupExpiredLocks() {
	ticker := time.NewTicker(fsm.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		fsm.mu.Lock()

		now := time.Now()
		for path, lock := range fsm.locks {
			if now.After(lock.ExpiresAt) {
				delete(fsm.locks, path)
			}
		}

		fsm.mu.Unlock()
	}
}

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

// IsFileStaleForAgent checks if a file is stale for an agent (either not read or modified since last read)
// Returns (true, nil) if the file is stale (agent must read it first before writing)
// Returns (false, nil) if the file is fresh (agent has read it and it hasn't been modified)
func (fsm *fileStateManager) IsFileStaleForAgent(agentID uuid.UUID, path string) (bool, error) {
	fsm.mu.RLock()
	lastRead, exists := fsm.readHistory[path]
	fsm.mu.RUnlock()

	// Agent has never read this file
	if !exists || lastRead.AgentID != agentID {
		return true, nil
	}

	// Check if file has been modified since agent last read it
	currentChecksum, err := fsm.calculateChecksum(path)
	if err != nil {
		return false, errs.Wrap(err, errs.TypeInternal, "failed to calculate checksum for staleness check").
			WithContext("path", path)
	}

	// true = stale (file has been modified), false = fresh (file unchanged)
	return lastRead.Checksum != currentChecksum, nil
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

// DetectChanges computes file changes between a previous snapshot and current state
// This is typically used to track what files a bash command modified
//
// For modifications and deletions, this method actively checks files on disk rather
// than relying on the async watcher's cache. This makes detection reliable even if
// the watcher is slow or disabled. New file creation still uses the watcher's cache
// to avoid expensive full-directory scans.
func (fsm *fileStateManager) DetectChanges(beforeStats map[string]*FileStats) ([]FileChange, error) {
	var changes []FileChange

	// Check for modifications and deletions by actively checking files from the
	// 'before' state. This is more reliable than depending on the async watcher.
	for path, before := range beforeStats {
		currentChecksum, err := fsm.calculateChecksum(path)
		if err != nil {
			// Check if the underlying error is file not found (unwrap our error wrapper)
			if os.IsNotExist(errors.Unwrap(err)) {
				// File was deleted
				changes = append(changes, FileChange{
					Path:        path,
					Operation:   Deleted,
					OldChecksum: before.Checksum,
				})
				continue
			}
			return nil, err // Some other error
		}

		// File exists, check for modification
		if before.Checksum != currentChecksum {
			changes = append(changes, FileChange{
				Path:        path,
				Operation:   Modified,
				OldChecksum: before.Checksum,
				NewChecksum: currentChecksum,
			})
		}
	}

	// Check for creations using the watcher's current state to avoid a full scan
	afterStats := fsm.GetAllStats()
	for path, after := range afterStats {
		if _, ok := beforeStats[path]; !ok {
			// This path was not in beforeStats, so it's a new file
			changes = append(changes, FileChange{
				Path:        path,
				Operation:   Created,
				NewChecksum: after.Checksum,
			})
		}
	}

	return changes, nil
}
