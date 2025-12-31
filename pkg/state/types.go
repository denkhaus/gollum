package state

import (
	"context"
	"os"
	"time"

	"github.com/google/uuid"
)

// LockMode represents the type of lock (shared for reading, exclusive for writing)
type LockMode string

const (
	// LockModeShared allows multiple concurrent readers
	LockModeShared LockMode = "shared"
	// LockModeExclusive allows only one writer
	LockModeExclusive LockMode = "exclusive"
)

// LockToken represents an active file lock
type LockToken struct {
	Path       string
	AgentID    uuid.UUID
	Mode       LockMode
	AcquiredAt time.Time
	ExpiresAt  time.Time
}

// WorkFunc is the callback type for DoWork.
// The function is executed under lock and receives the lock token for reference.
type WorkFunc func(ctx context.Context, token *LockToken) (any, error)

// WorkOptions contains options for work executed under lock
type WorkOptions struct {
	// UpdateStatsAfter: If true, update file stats immediately after successful work
	// This ensures synchronous stats updates for internal changes (vs watcher for external changes)
	UpdateStatsAfter bool

	// TrackRead: If true, records that the agent has read this file for later verification
	// Used by ReadFileTool to enable automatic race condition detection
	TrackRead bool
}

// ReadRecord tracks when an agent last read a file
type ReadRecord struct {
	AgentID   uuid.UUID
	Checksum  string
	Timestamp time.Time
}

// FileStats represents file statistics tracked by FileStateManager
type FileStats struct {
	Path         string
	Checksum     string
	ModifiedTime time.Time
	Size         int64
	LastScanned  time.Time
	LockedBy     *string
	LockMode     *LockMode
	LockedAt     *time.Time
}

// FileStatus represents the current state of a file (legacy, use FileStats)
type FileStatus struct {
	Path     string
	Checksum string
	LockedBy *string
	LockMode *LockMode
	LockedAt *time.Time
}

// ChangeOperation represents the type of file change detected
type ChangeOperation int

const (
	// Created indicates a new file was created
	Created ChangeOperation = iota
	// Modified indicates an existing file was modified
	Modified
	// Deleted indicates a file was deleted
	Deleted
)

// FileChange represents a detected file modification
type FileChange struct {
	Path        string          // File path that changed
	Operation   ChangeOperation // Type of change
	OldChecksum string          // Checksum before change (empty if Created)
	NewChecksum string          // Checksum after change (empty if Deleted)
}

// String returns a human-readable representation of the change operation
func (o ChangeOperation) String() string {
	switch o {
	case Created:
		return "created"
	case Modified:
		return "modified"
	case Deleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// ScanOptions controls the Prime/Scan behavior
type ScanOptions struct {
	RootDir       string   // Directory to scan (default: current working dir)
	MaxDepth      int      // Maximum depth to scan (-1 for unlimited, default: 10)
	IncludeHidden bool     // Include hidden files (default: false)
	FilePattern   string   // Glob pattern for files (default: "*")
	IgnoreDirs    []string // Directories to ignore (default: ["node_modules", ".git", "vendor"])
	Concurrency   int      // Number of concurrent workers (default: 4)
}

// DefaultScanOptions returns default scan options
func DefaultScanOptions() *ScanOptions {
	wd, _ := os.Getwd()
	return &ScanOptions{
		RootDir:       wd,
		MaxDepth:      10,
		IncludeHidden: false,
		FilePattern:   "*",
		IgnoreDirs:    []string{"node_modules", ".git", "vendor", ".venv", "venv", "target", "build"},
		Concurrency:   4,
	}
}

// FileStateManager manages file state, locks, and checksums
type FileStateManager interface {
	// Prime scans the working directory and populates file stats
	Prime(ctx context.Context) error

	// AcquireLock acquires a lock on a file, waiting if necessary
	AcquireLock(ctx context.Context, path string, agentID uuid.UUID, mode LockMode) (*LockToken, error)

	// ReleaseLock releases a lock held by an agent
	ReleaseLock(path string, agentID uuid.UUID) error

	// DoWork executes work under a lock with automatic release.
	DoWork(ctx context.Context, path string, agentID uuid.UUID, mode LockMode, work WorkFunc) (any, error)

	// DoWorkWithOptions executes work under a lock with options (checksum verification, stats update).
	DoWorkWithOptions(ctx context.Context, path string, agentID uuid.UUID, mode LockMode, opts WorkOptions, work WorkFunc) (any, error)

	// VerifyChecksum checks if a file's checksum matches the expected value
	VerifyChecksum(path, expected string) (bool, error)

	// UpdateChecksum calculates and stores the checksum for a file
	UpdateChecksum(path string) (string, error)

	// UpdateStats updates file stats after modification
	UpdateStats(path string) (*FileStats, error)

	// GetFileStats returns cached stats for a file
	GetFileStats(path string) (*FileStats, error)

	// GetAllStats returns all tracked file stats
	GetAllStats() map[string]*FileStats

	// GetFileStatus returns the current status of a file (legacy)
	GetFileStatus(path string) (*FileStatus, error)

	// IsLocked checks if a file is currently locked
	IsLocked(path string) bool

	// GetLock returns the current lock on a file, if any
	GetLock(path string) (*LockToken, error)

	// StartWatcher starts the file change watcher
	StartWatcher() error

	// StopWatcher stops the file change watcher
	StopWatcher() error

	// IsWatcherRunning returns true if watcher is active
	IsWatcherRunning() bool

	// GetWatcherDebounce returns the file watcher's debounce duration
	// This is the minimum time to wait after a file change before checking for updates
	GetWatcherDebounce() time.Duration

	// IsFileStaleForAgent checks if a file is stale for an agent (not read or modified since last read)
	IsFileStaleForAgent(agentID uuid.UUID, path string) (bool, error)

	// DetectChanges computes file changes between a previous snapshot and current state
	// Typically used to track what files a bash command modified
	DetectChanges(beforeStats map[string]*FileStats) ([]FileChange, error)
}
