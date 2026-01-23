package state

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

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
