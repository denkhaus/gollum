package state

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/denkhaus/gollum/pkg/errs"
)

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
