package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStateManager_ChecksumIntegration(t *testing.T) {
	injector := setupTestInjector()
	fsm, err := NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}

	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Calculate initial checksum
	initialChecksum, err := fsm.UpdateChecksum(testFile)
	if err != nil {
		t.Fatalf("failed to update checksum: %v", err)
	}

	if initialChecksum == "" {
		t.Error("checksum should not be empty")
	}

	// Verify checksum
	valid, err := fsm.VerifyChecksum(testFile, initialChecksum)
	if err != nil {
		t.Fatalf("failed to verify checksum: %v", err)
	}

	if !valid {
		t.Error("checksum verification should succeed")
	}

	// Modify file
	newContent := "Modified content"
	err = os.WriteFile(testFile, []byte(newContent), 0644)
	if err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Verify old checksum should fail
	valid, err = fsm.VerifyChecksum(testFile, initialChecksum)
	if err != nil {
		t.Fatalf("failed to verify checksum: %v", err)
	}

	if valid {
		t.Error("checksum verification should fail after file modification")
	}

	// Update checksum to get it in cache
	_, err = fsm.UpdateChecksum(testFile)
	if err != nil {
		t.Fatalf("failed to update checksum: %v", err)
	}

	// Get file status after modification
	status, err := fsm.GetFileStatus(testFile)
	if err != nil {
		t.Fatalf("failed to get file status: %v", err)
	}

	if status.Checksum == initialChecksum {
		t.Error("checksum should have changed after file modification")
	}
}

func TestFileStateManager_ComplexScenario(t *testing.T) {
	injector := setupTestInjector()
	fsm, err := NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}
	ctx := context.Background()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "shared.txt")
	initialContent := "Initial content"

	// Agent1 writes file with lock
	_, err = fsm.AcquireLock(ctx, testFile, TestAgent1, LockModeExclusive)
	if err != nil {
		t.Fatalf("agent1 failed to acquire lock: %v", err)
	}

	err = os.WriteFile(testFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	checksum1, _ := fsm.UpdateChecksum(testFile)
	if err := fsm.ReleaseLock(testFile, TestAgent1); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// Agent2 tries to read with checksum verification
	_, err = fsm.AcquireLock(ctx, testFile, TestAgent2, LockModeExclusive)
	if err != nil {
		t.Fatalf("agent2 failed to acquire lock: %v", err)
	}

	// Agent2 verifies checksum before modifying
	valid, err := fsm.VerifyChecksum(testFile, checksum1)
	if err != nil || !valid {
		t.Error("agent2 should be able to verify checksum")
	}

	// Agent2 modifies file
	newContent := "Modified by agent2"
	err = os.WriteFile(testFile, []byte(newContent), 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	checksum2, _ := fsm.UpdateChecksum(testFile)
	if err := fsm.ReleaseLock(testFile, TestAgent2); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// Agent3 tries to use old checksum (should fail)
	_, err = fsm.AcquireLock(ctx, testFile, TestAgent3, LockModeExclusive)
	if err != nil {
		t.Fatalf("agent3 failed to acquire lock: %v", err)
	}

	valid, err = fsm.VerifyChecksum(testFile, checksum1)
	if err != nil {
		t.Fatalf("failed to verify checksum: %v", err)
	}

	if valid {
		t.Error("agent3 should detect that file has been modified")
	}

	// Agent3 can still proceed with new checksum
	valid, err = fsm.VerifyChecksum(testFile, checksum2)
	if err != nil || !valid {
		t.Error("agent3 should be able to verify current checksum")
	}

	if err := fsm.ReleaseLock(testFile, TestAgent3); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}
}

func TestFileStateManager_ConcurrentAccess(t *testing.T) {
	injector := setupTestInjector()
	fsm, err := NewFileStateManager(injector)
	if err != nil {
		t.Fatalf("Failed to create FileStateManager: %v", err)
	}
	ctx := context.Background()

	testPath := "/test/concurrent.txt"

	// Start multiple goroutines trying to acquire the same lock
	done := make(chan bool, 3)

	go func() {
		token, err := fsm.AcquireLock(ctx, testPath, TestAgent1, LockModeExclusive)
		if err == nil && token.AgentID == TestAgent1 {
			time.Sleep(50 * time.Millisecond)
			if err := fsm.ReleaseLock(testPath, TestAgent1); err != nil {
				t.Logf("warning: failed to release lock: %v", err)
			}
			done <- true
		} else {
			done <- false
		}
	}()

	go func() {
		token, err := fsm.AcquireLock(ctx, testPath, TestAgent2, LockModeExclusive)
		if err == nil && token.AgentID == TestAgent2 {
			time.Sleep(30 * time.Millisecond)
			if err := fsm.ReleaseLock(testPath, TestAgent2); err != nil {
				t.Logf("warning: failed to release lock: %v", err)
			}
			done <- true
		} else {
			done <- false
		}
	}()

	go func() {
		token, err := fsm.AcquireLock(ctx, testPath, TestAgent3, LockModeExclusive)
		if err == nil && token.AgentID == TestAgent3 {
			if err := fsm.ReleaseLock(testPath, TestAgent3); err != nil {
				t.Logf("warning: failed to release lock: %v", err)
			}
			done <- true
		} else {
			done <- false
		}
	}()

	// All agents should eventually acquire and release locks
	successCount := 0
	for i := 0; i < 3; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != 3 {
		t.Errorf("expected all 3 agents to succeed, got %d", successCount)
	}
}
