package skills

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// DiscoveryOptions configures skill discovery behavior
type DiscoveryOptions struct {
	// RootDir is the starting directory for discovery (defaults to current directory)
	RootDir string
	// MaxDepth limits directory traversal depth (0 = unlimited, default 10)
	MaxDepth int
	// IncludeHidden includes hidden directories (starting with .)
	IncludeHidden bool
	// IgnoredDirs is a list of directory names to skip
	IgnoredDirs []string
	// Concurrency is the number of concurrent workers (default 4)
	Concurrency int
}

// DiscoveryResult contains the results of a skill discovery operation
type DiscoveryResult struct {
	// Skills is the list of discovered skills
	Skills Skills
	// Errors is a list of errors encountered during discovery
	Errors []error
	// ScannedDirs is the number of directories scanned
	ScannedDirs int
	// ScannedFiles is the number of files checked
	ScannedFiles int
}

// discoverer handles skill file discovery
type discoverer struct {
	log    *zap.Logger
	config *DiscoveryOptions
}

// newDiscoverer creates a new skill discoverer
func newDiscoverer(log *zap.Logger, opts *DiscoveryOptions) *discoverer {
	if opts == nil {
		opts = &DiscoveryOptions{}
	}

	// Apply defaults
	if opts.MaxDepth == 0 {
		opts.MaxDepth = DefaultMaxDepth
	}
	if opts.Concurrency == 0 {
		opts.Concurrency = DefaultConcurrency
	}
	if len(opts.IgnoredDirs) == 0 {
		opts.IgnoredDirs = DefaultIgnoredDirs
	}
	if opts.RootDir == "" {
		opts.RootDir, _ = os.Getwd()
	}

	return &discoverer{
		log:    log,
		config: opts,
	}
}

// Discover scans directories for SKILL.md files
func (d *discoverer) Discover(ctx context.Context) (*DiscoveryResult, error) {
	result := &DiscoveryResult{
		Skills: make(Skills, 0),
		Errors: make([]error, 0),
	}

	// Build ignore set
	ignoreSet := make(map[string]bool)
	for _, dir := range d.config.IgnoredDirs {
		ignoreSet[dir] = true
	}

	// Channels for concurrent processing
	fileCh := make(chan string, d.config.Concurrency*2)
	skillCh := make(chan *Skill, d.config.Concurrency*2)
	errCh := make(chan error, d.config.Concurrency)

	var wg sync.WaitGroup

	// Start worker goroutines for parsing
	for i := 0; i < d.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileCh {
				skill, err := ParseFile(path)
				if err != nil {
					errCh <- err
					continue
				}
				skillCh <- skill
			}
		}()
	}

	// Collector goroutine
	var collectWg sync.WaitGroup
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
		for skill := range skillCh {
			result.Skills = append(result.Skills, skill)
		}
	}()

	// Error collector goroutine
	var errWg sync.WaitGroup
	errWg.Add(1)
	go func() {
		defer errWg.Done()
		for err := range errCh {
			result.Errors = append(result.Errors, err)
			d.log.Warn("Skill discovery error", zap.Error(err))
		}
	}()

	// Walk directory tree
	go func() {
		defer close(fileCh)
		if err := d.walkDirectory(ctx, d.config.RootDir, 0, ignoreSet, fileCh, result); err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	// Wait for parsing to complete
	wg.Wait()

	// Close channels
	close(skillCh)
	close(errCh)

	// Wait for collectors
	collectWg.Wait()
	errWg.Wait()

	d.log.Info("Skill discovery complete",
		zap.Int("skills_found", len(result.Skills)),
		zap.Int("errors", len(result.Errors)),
		zap.Int("directories_scanned", result.ScannedDirs),
		zap.Int("files_scanned", result.ScannedFiles),
	)

	return result, nil
}

// walkDirectory recursively walks the directory tree
func (d *discoverer) walkDirectory(
	ctx context.Context,
	dir string,
	depth int,
	ignoreSet map[string]bool,
	fileCh chan<- string,
	result *DiscoveryResult,
) error {
	// Check context
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check max depth (negative means unlimited)
	if d.config.MaxDepth > 0 && depth > d.config.MaxDepth {
		return nil
	}

	result.ScannedDirs++

	entries, err := os.ReadDir(dir)
	if err != nil {
		// Log but don't fail entire discovery
		d.log.Debug("Cannot read directory", zap.String("dir", dir), zap.Error(err))
		return nil
	}

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(dir, name)

		// Skip hidden files/directories unless included
		if !d.config.IncludeHidden && strings.HasPrefix(name, ".") {
			continue
		}

		if entry.IsDir() {
			// Skip ignored directories
			if ignoreSet[name] {
				continue
			}

			// Recurse into subdirectory
			if err := d.walkDirectory(ctx, fullPath, depth+1, ignoreSet, fileCh, result); err != nil {
				return err
			}
			continue
		}

		// Check for SKILL.md files
		if name == SkillFileName {
			result.ScannedFiles++
			select {
			case fileCh <- fullPath:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

// DiscoverInPath discovers skills in a specific path with default options
func DiscoverInPath(ctx context.Context, path string, log *zap.Logger) (*DiscoveryResult, error) {
	opts := &DiscoveryOptions{
		RootDir: path,
	}
	d := newDiscoverer(log, opts)
	return d.Discover(ctx)
}

// DiscoverMultiple discovers skills in multiple root directories
func DiscoverMultiple(ctx context.Context, paths []string, log *zap.Logger) (*DiscoveryResult, error) {
	combined := &DiscoveryResult{
		Skills: make(Skills, 0),
		Errors: make([]error, 0),
	}

	seenSkills := make(map[string]string) // skill name -> file path (for duplicate detection)

	for _, path := range paths {
		result, err := DiscoverInPath(ctx, path, log)
		if err != nil {
			combined.Errors = append(combined.Errors, err)
			continue
		}

		// Merge results, checking for duplicates
		for _, skill := range result.Skills {
			if existingPath, exists := seenSkills[skill.ID()]; exists {
				combined.Errors = append(combined.Errors,
					ErrDuplicateSkill(skill.Name, existingPath, skill.FilePath))
				continue
			}
			seenSkills[skill.ID()] = skill.FilePath
			combined.Skills = append(combined.Skills, skill)
		}

		combined.Errors = append(combined.Errors, result.Errors...)
		combined.ScannedDirs += result.ScannedDirs
		combined.ScannedFiles += result.ScannedFiles
	}

	return combined, nil
}
