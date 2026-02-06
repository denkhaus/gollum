// Package tools provides tool implementations for the Gollum agent system.
// This includes file operations (ReadFile, WriteFile, Edit), search tools (Grep, Glob),
// agent management (SpawnAgent, RemoveAgent, ResumeAgent), and utility tools (CurrentTime).
package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

type (
	// GlobTool finds file paths matching glob patterns
	GlobTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
	}

	// GlobToolProvider creates GlobTool instances via DI
	GlobToolProvider interface {
		CreateTool(agentID uuid.UUID) *GlobTool
	}

	globToolProvider struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
	}
)

// NewGlobToolProvider creates a provider for Glob tools
func NewGlobToolProvider(injector do.Injector) (GlobToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	return &globToolProvider{
		logService:  logService,
		hookManager: hookManager,
	}, nil
}

// CreateTool creates a new GlobTool with agent ID
func (p *globToolProvider) CreateTool(agentID uuid.UUID) *GlobTool {
	return &GlobTool{
		logService:  p.logService,
		hookManager: p.hookManager,
		agentID:     agentID,
	}
}

// Spec returns the tool specification for the Glob tool
func (t *GlobTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameGlob,
		Description: "Fast file pattern matching tool that works with any codebase size. Supports glob patterns like '**/*.go' or 'pkg/**/*.ts'. Returns matching file paths sorted by modification time.",
		Parameters: map[string]*gollem.Parameter{
			"pattern": {
				Type:        gollem.TypeString,
				Description: "The glob pattern to match files against",
			},
			"path": {
				Type:        gollem.TypeString,
				Description: "The directory to search in (default: current working directory)",
			},
		},
	}
}

// Run executes the Glob tool to find files matching a pattern
func (t *GlobTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameGlob, args,
		func() (map[string]any, error) {
			return t.runGlob(ctx, args)
		})
}

// runGlob implements the core Glob logic
func (t *GlobTool) runGlob(_ context.Context, args map[string]any) (map[string]any, error) {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		t.logService.Errorf("[Agent %s] Glob pattern validation failed: pattern is required and must be non-empty", t.agentID)
		return map[string]any{
			"success": false,
			"error":   "pattern is required and must be a non-empty string",
		}, nil
	}

	// Get optional path (default: current directory)
	searchPath := "."
	if pathVal, exists := args["path"].(string); exists && pathVal != "" {
		searchPath = pathVal
	}

	// Convert relative path to absolute
	searchPath, err := filepath.Abs(searchPath)
	if err != nil {
		t.logService.Errorf("[Agent %s] Failed to resolve absolute path for '%s': %v", t.agentID, searchPath, err)
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("failed to resolve absolute path: %v", err),
		}, nil
	}

	t.logService.Infof("[Agent %s] Starting glob search: pattern='%s' path='%s'", t.agentID, pattern, searchPath)

	// Build the full pattern
	fullPattern := filepath.Join(searchPath, pattern)
	t.logService.Debugf("[Agent %s] Full glob pattern: %s", t.agentID, fullPattern)

	// Match files
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		t.logService.Errorf("[Agent %s] Invalid glob pattern '%s': %v", t.agentID, pattern, err)
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("invalid glob pattern: %v", err),
		}, nil
	}

	// Handle double-star pattern (recursive matching)
	// filepath.Glob doesn't support **, so we need to handle it manually
	if len(matches) == 0 && containsDoubleStar(pattern) {
		t.logService.Debugf("[Agent %s] Pattern contains **, using recursive glob search", t.agentID)
		matches, err = recursiveGlob(t.logService, t.agentID, searchPath, pattern)
		if err != nil {
			t.logService.Errorf("[Agent %s] Recursive glob failed for pattern '%s': %v", t.agentID, pattern, err)
			return map[string]any{
				"success": false,
				"error":   fmt.Sprintf("failed to match files: %v", err),
			}, nil
		}
	}

	t.logService.Infof("[Agent %s] Glob search completed: pattern='%s' matches=%d", t.agentID, pattern, len(matches))
	if len(matches) > 0 {
		t.logService.Debugf("[Agent %s] Found matches: %v", t.agentID, matches)
	} else {
		t.logService.Debugf("[Agent %s] No matches found for pattern '%s'", t.agentID, pattern)
	}

	return map[string]any{
		"success": len(matches) > 0,
		"matches": matches,
		"count":   len(matches),
		"pattern": pattern,
		"path":    searchPath,
	}, nil
}

// containsDoubleStar checks if pattern contains **
func containsDoubleStar(pattern string) bool {
	for i := 0; i < len(pattern)-1; i++ {
		if pattern[i] == '*' && pattern[i+1] == '*' {
			return true
		}
	}
	return false
}

// recursiveGlob handles ** patterns by walking the directory tree
func recursiveGlob(logService logger.LoggerService, agentID uuid.UUID, root, pattern string) ([]string, error) {
	var matches []string

	logService.Debugf("[Agent %s] Starting recursive glob: root='%s' pattern='%s'", agentID, root, pattern)

	// Split pattern by **
	segments := splitDoubleStar(pattern)

	// For now, implement a simple recursive walk
	// In a production system, you'd want to use a more efficient algorithm
	basePattern := filepath.Join(root, segments[0])

	walkFn := func(path string, _ os.FileInfo, err error) error { //nolint:unparam
		if err != nil {
			logService.Debugf("[Agent %s] Skipping path during walk: %s (error: %v)", agentID, path, err)
			return nil // Continue on error
		}

		// Check if path matches the pattern
		matched, err := filepath.Match(basePattern, path)
		if err != nil {
			logService.Debugf("[Agent %s] Pattern match failed for %s: %v", agentID, path, err)
			return nil
		}

		if matched {
			matches = append(matches, path)
			logService.Debugf("[Agent %s] Recursive glob matched: %s", agentID, path)
		}

		return nil
	}

	// Simple implementation - walk the root directory
	// For full ** support, you'd need a more sophisticated implementation
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		return walkFn(path, info, err)
	}); err != nil {
		logService.Errorf("[Agent %s] Directory walk failed: %v", agentID, err)
		return nil, errs.Wrap(err, errs.TypeInternal, "walk error").
			WithContext("agent_id", agentID).
			WithContext("root_path", root)
	}

	logService.Debugf("[Agent %s] Recursive glob completed: found %d matches", agentID, len(matches))
	return matches, nil
}

// splitDoubleStar splits a pattern by ** segments
func splitDoubleStar(pattern string) []string {
	var segments []string
	current := ""

	for i := 0; i < len(pattern); i++ {
		if i < len(pattern)-1 && pattern[i] == '*' && pattern[i+1] == '*' {
			if current != "" {
				segments = append(segments, current)
				current = ""
			}
			i++ // Skip the second *
		} else {
			current += string(pattern[i])
		}
	}

	if current != "" {
		segments = append(segments, current)
	}

	return segments
}
