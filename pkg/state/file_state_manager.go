package state

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
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
			// Check if the underlying error is file not found
			if errors.Is(err, os.ErrNotExist) {
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
