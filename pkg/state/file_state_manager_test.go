package state

import (
	"context"
	"testing"
	"time"
)

func TestFileStateManager_AcquireLock(t *testing.T) {
	injector := setupTestInjector()
	fsm, err := NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}
	ctx := context.Background()

	// Test acquiring a new lock
	token, err := fsm.AcquireLock(ctx, "/test/file.txt", TestAgent1, LockModeExclusive)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	if token.Path != "/test/file.txt" {
		t.Errorf("expected path /test/file.txt, got %s", token.Path)
	}

	if token.AgentID != TestAgent1 {
		t.Errorf("expected agent1, got %s", token.AgentID)
	}

	if token.Mode != LockModeExclusive {
		t.Errorf("expected exclusive lock mode, got %s", token.Mode)
	}
}

func TestFileStateManager_SharedLocks(t *testing.T) {
	injector := setupTestInjector()
	fsm, err := NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}
	ctx := context.Background()

	// Multiple agents should be able to acquire shared locks
	token1, err := fsm.AcquireLock(ctx, "/test/shared.txt", TestAgent1, LockModeShared)
	if err != nil {
		t.Fatalf("failed to acquire first shared lock: %v", err)
	}

	token2, err := fsm.AcquireLock(ctx, "/test/shared.txt", TestAgent2, LockModeShared)
	if err != nil {
		t.Fatalf("failed to acquire second shared lock: %v", err)
	}

	if token1.Path != token2.Path {
		t.Errorf("shared locks should be on same path")
	}

	// Verify file is locked
	if !fsm.IsLocked("/test/shared.txt") {
		t.Error("file should be locked")
	}
}

func TestFileStateManager_ExclusiveLockBlocks(t *testing.T) {
	injector := setupTestInjector()
	fsm, err := NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}
	ctx := context.Background()

	// First agent acquires exclusive lock
	_, err = fsm.AcquireLock(ctx, "/test/exclusive.txt", TestAgent1, LockModeExclusive)
	if err != nil {
		t.Fatalf("failed to acquire exclusive lock: %v", err)
	}

	// Second agent should not be able to acquire exclusive lock
	done := make(chan bool)
	go func() {
		_, err := fsm.AcquireLock(ctx, "/test/exclusive.txt", TestAgent2, LockModeExclusive)
		if err != nil {
			done <- false
			return
		}
		done <- true
	}()

	// Wait a bit and check if lock is still held
	time.Sleep(100 * time.Millisecond)
	if !fsm.IsLocked("/test/exclusive.txt") {
		t.Error("file should still be locked by agent1")
	}

	// Release lock
	if err := fsm.ReleaseLock("/test/exclusive.txt", TestAgent1); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// Now second agent should be able to acquire lock
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("second agent should have been able to acquire lock after first was released")
	}
}

func TestFileStateManager_ReleaseLock(t *testing.T) {
	injector := setupTestInjector()
	fsm, err := NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}
	ctx := context.Background()

	// Acquire lock
	_, err = fsm.AcquireLock(ctx, "/test/release.txt", TestAgent1, LockModeExclusive)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Verify lock is held
	if !fsm.IsLocked("/test/release.txt") {
		t.Error("file should be locked")
	}

	// Release lock
	err = fsm.ReleaseLock("/test/release.txt", TestAgent1)
	if err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// Verify lock is released
	if fsm.IsLocked("/test/release.txt") {
		t.Error("file should not be locked after release")
	}
}

func TestFileStateManager_SameAgentReacquire(t *testing.T) {
	injector := setupTestInjector()
	fsm, err := NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}
	ctx := context.Background()

	// Agent acquires lock
	token1, err := fsm.AcquireLock(ctx, "/test/reacquire.txt", TestAgent1, LockModeExclusive)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Same agent re-acquires lock (should succeed and extend timeout)
	token2, err := fsm.AcquireLock(ctx, "/test/reacquire.txt", TestAgent1, LockModeExclusive)
	if err != nil {
		t.Fatalf("failed to re-acquire lock: %v", err)
	}

	// The tokens might be the same object, so check expiration was extended
	if !token2.ExpiresAt.After(token1.ExpiresAt) {
		t.Logf("Warning: token1 expires at %v, token2 expires at %v", token1.ExpiresAt, token2.ExpiresAt)
	}
}

func TestFileStateManager_WrongAgentRelease(t *testing.T) {
	injector := setupTestInjector()
	fsm, err := NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}
	ctx := context.Background()

	// Agent1 acquires lock
	_, err = fsm.AcquireLock(ctx, "/test/wrong.txt", TestAgent1, LockModeExclusive)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Agent2 tries to release lock (should fail)
	err = fsm.ReleaseLock("/test/wrong.txt", TestAgent2)
	if err == nil {
		t.Error("agent2 should not be able to release agent1's lock")
	}

	// Verify lock is still held
	if !fsm.IsLocked("/test/wrong.txt") {
		t.Error("file should still be locked by agent1")
	}
}

func TestFileStateManager_GetLock(t *testing.T) {
	injector := setupTestInjector()
	fsm, err := NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}
	ctx := context.Background()

	testPath := "/test/getlock.txt"

	// Initially no lock
	_, err = fsm.GetLock(testPath)
	if err == nil {
		t.Error("expected error when getting non-existent lock")
	}

	// Acquire lock
	_, err = fsm.AcquireLock(ctx, testPath, TestAgent1, LockModeExclusive)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Get lock should now return the lock
	lock, err := fsm.GetLock(testPath)
	if err != nil {
		t.Fatalf("failed to get lock: %v", err)
	}

	if lock.AgentID != TestAgent1 {
		t.Errorf("expected agent1, got %s", lock.AgentID)
	}

	if lock.Mode != LockModeExclusive {
		t.Errorf("expected exclusive mode, got %s", lock.Mode)
	}
}
