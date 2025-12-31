// Package state provides file state management with locking and checksum verification
// for concurrent file access in multi-agent environments.
//
// Features:
// - Shared locks for concurrent reading
// - Exclusive locks for writing
// - Automatic lock expiration and cleanup
// - SHA-256 checksum verification
// - Thread-safe operations
//
// Usage:
//
//	fsm := state.NewFileStateManager()
//
//	// Acquire exclusive lock for writing
//	token, err := fsm.AcquireLock(ctx, "/path/to/file", "agent-id", state.LockModeExclusive)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer fsm.ReleaseLock("/path/to/file", "agent-id")
//
//	// Verify file hasn't changed
//	valid, err := fsm.VerifyChecksum("/path/to/file", expectedChecksum)
//	if !valid {
//	    return errors.New("file has been modified")
//	}
//
//	// Write file and update checksum
//	err = os.WriteFile("/path/to/file", content, 0644)
//	newChecksum, _ := fsm.UpdateChecksum("/path/to/file")
package state
