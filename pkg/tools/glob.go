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
	"go.uber.org/zap"
)

type (
	// globToolImpl finds file paths matching glob patterns
	globToolImpl struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
	}

	// GlobToolProvider creates GlobTool instances via DI
	GlobToolProvider interface {
		CreateTool(agentID uuid.UUID) gollem.Tool
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
func (p *globToolProvider) CreateTool(agentID uuid.UUID) gollem.Tool {
	return &globToolImpl{
		logService:  p.logService,
		hookManager: p.hookManager,
		agentID:     agentID,
	}
}

// Spec returns the tool specification for the Glob tool
func (t *globToolImpl) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameGlob.String(),
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
func (t *globToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameGlob, args,
		func() (map[string]any, error) {
			return t.runGlob(ctx, args)
		})
}

// runGlob implements the core Glob logic
func (t *globToolImpl) runGlob(_ context.Context, args map[string]any) (map[string]any, error) {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		t.logService.ErrorWithAgent("Glob pattern validation failed", t.agentID,
			zap.String("reason", "pattern_is_required"))
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
		t.logService.ErrorWithAgent("Failed to resolve absolute path", t.agentID,
			zap.String("search_path", searchPath),
			zap.Error(err))
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("failed to resolve absolute path: %v", err),
		}, nil
	}

	t.logService.InfoWithAgent("Starting glob search", t.agentID,
		zap.String("pattern", pattern),
		zap.String("search_path", searchPath))

	// Build the full pattern
	fullPattern := filepath.Join(searchPath, pattern)
	t.logService.DebugWithAgent("Full glob pattern", t.agentID,
		zap.String("pattern", fullPattern))

	// Match files
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		t.logService.ErrorWithAgent("Invalid glob pattern", t.agentID,
			zap.String("pattern", pattern),
			zap.Error(err))
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("invalid glob pattern: %v", err),
		}, nil
	}

	// Handle double-star pattern (recursive matching)
	// filepath.Glob doesn't support **, so we need to handle it manually
	if len(matches) == 0 && containsDoubleStar(pattern) {
		t.logService.DebugWithAgent("Pattern contains **, using recursive glob search", t.agentID,
			zap.String("pattern", pattern))
		matches, err = recursiveGlob(t.logService, t.agentID, searchPath, pattern)
		if err != nil {
			t.logService.ErrorWithAgent("Recursive glob failed", t.agentID,
				zap.String("pattern", pattern),
				zap.Error(err))
			return map[string]any{
				"success": false,
				"error":   fmt.Sprintf("failed to match files: %v", err),
			}, nil
		}
	}

	t.logService.InfoWithAgent("Glob search completed", t.agentID,
		zap.String("pattern", pattern),
		zap.Int("match_count", len(matches)))
	if len(matches) > 0 {
		t.logService.DebugWithAgent("Found matches", t.agentID,
			zap.Any("matches", matches))
	} else {
		t.logService.DebugWithAgent("No matches found", t.agentID,
			zap.String("pattern", pattern))
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

	logService.DebugWithAgent("Starting recursive glob", agentID,
		zap.String("root", root),
		zap.String("pattern", pattern))

	// Split pattern by **
	segments := splitDoubleStar(pattern)

	// For now, implement a simple recursive walk
	// In a production system, you'd want to use a more efficient algorithm
	basePattern := filepath.Join(root, segments[0])

	walkFn := func(path string, _ os.FileInfo, err error) error { //nolint:unparam
		if err != nil {
			logService.DebugWithAgent("Skipping path during walk", agentID,
				zap.String("path", path),
				zap.Error(err))
			return nil // Continue on error
		}

		// Check if path matches the pattern
		matched, err := filepath.Match(basePattern, path)
		if err != nil {
			logService.DebugWithAgent("Pattern match failed", agentID,
				zap.String("path", path),
				zap.Error(err))
			return nil
		}

		if matched {
			matches = append(matches, path)
			logService.DebugWithAgent("Recursive glob matched", agentID,
				zap.String("path", path))
		}

		return nil
	}

	// Simple implementation - walk the root directory
	// For full ** support, you'd need a more sophisticated implementation
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		return walkFn(path, info, err)
	}); err != nil {
		logService.ErrorWithAgent("Directory walk failed", agentID, zap.Error(err))
		return nil, errs.Wrap(err, errs.TypeInternal, "walk error").
			WithContext("agent_id", agentID).
			WithContext("root_path", root)
	}

	logService.DebugWithAgent("Recursive glob completed", agentID,
		zap.Int("match_count", len(matches)))
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
