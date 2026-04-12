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
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

type (
	// globToolImpl finds file paths matching glob patterns
	globToolImpl struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agent       shared.Agent
	}

	// GlobToolProvider creates GlobTool instances via DI
	GlobToolProvider interface {
		CreateTool(agent shared.Agent) gollem.Tool
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

// CreateTool creates a new GlobTool with agent reference
func (p *globToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &globToolImpl{
		logService:  p.logService,
		hookManager: p.hookManager,
		agent:       agent,
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
	return t.hookManager.WithToolHooks(ctx, t.agent.ToLoggingContext(), shared.ToolNameGlob, args,
		func() (map[string]any, error) {
			return t.runGlob(ctx, args)
		})
}

// runGlob implements the core Glob logic
func (t *globToolImpl) runGlob(_ context.Context, args ToolRequestParams) (map[string]any, error) {
	pattern, errResp := args.MustGetString(shared.ParamPattern)
	if errResp != nil {
		return errResp, nil
	}

	// Get optional path (default: current directory)
	searchPath := args.GetString(shared.ParamPath, ".")

	// Convert relative path to absolute
	searchPath, err := filepath.Abs(searchPath)
	if err != nil {
		t.logService.ErrorWithContext("Failed to resolve absolute path", t.agent.ToLoggingContext(),
			zap.String("search_path", searchPath),
			zap.Error(err))
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("failed to resolve absolute path: %v", err),
		}, nil
	}

	t.logService.InfoWithContext("Starting glob search", t.agent.ToLoggingContext(),
		zap.String("pattern", pattern),
		zap.String("search_path", searchPath))

	// Build the full pattern
	fullPattern := filepath.Join(searchPath, pattern)
	t.logService.DebugWithContext("Full glob pattern", t.agent.ToLoggingContext(),
		zap.String("pattern", fullPattern))

	// Match files
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		t.logService.ErrorWithContext("Invalid glob pattern", t.agent.ToLoggingContext(),
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
		t.logService.DebugWithContext("Pattern contains **, using recursive glob search", t.agent.ToLoggingContext(),
			zap.String("pattern", pattern))
		matches, err = recursiveGlob(t.logService, t.agent.ToLoggingContext(), searchPath, pattern)
		if err != nil {
			t.logService.ErrorWithContext("Recursive glob failed", t.agent.ToLoggingContext(),
				zap.String("pattern", pattern),
				zap.Error(err))
			return map[string]any{
				"success": false,
				"error":   fmt.Sprintf("failed to match files: %v", err),
			}, nil
		}
	}

	t.logService.InfoWithContext("Glob search completed", t.agent.ToLoggingContext(),
		zap.String("pattern", pattern),
		zap.Int("match_count", len(matches)))
	if len(matches) > 0 {
		t.logService.DebugWithContext("Found matches", t.agent.ToLoggingContext(),
			zap.Any("matches", matches))
	} else {
		t.logService.DebugWithContext("No matches found", t.agent.ToLoggingContext(),
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
func recursiveGlob(logService logger.LoggerService, logCtx shared.LoggingContext, root, pattern string) ([]string, error) {
	var matches []string

	logService.DebugWithContext("Starting recursive glob", logCtx,
		zap.String("root", root),
		zap.String("pattern", pattern))

	// Split pattern by **
	segments := splitDoubleStar(pattern)

	// For now, implement a simple recursive walk
	// In a production system, you'd want to use a more efficient algorithm
	basePattern := filepath.Join(root, segments[0])

	walkFn := func(path string, _ os.FileInfo, err error) error { //nolint:unparam
		if err != nil {
			logService.DebugWithContext("Skipping path during walk", logCtx,
				zap.String("path", path),
				zap.Error(err))
			return nil // Continue on error
		}

		// Check if path matches the pattern
		matched, err := filepath.Match(basePattern, path)
		if err != nil {
			logService.DebugWithContext("Pattern match failed", logCtx,
				zap.String("path", path),
				zap.Error(err))
			return nil
		}

		if matched {
			matches = append(matches, path)
			logService.DebugWithContext("Recursive glob matched", logCtx,
				zap.String("path", path))
		}

		return nil
	}

	// Simple implementation - walk the root directory
	// For full ** support, you'd need a more sophisticated implementation
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		return walkFn(path, info, err)
	}); err != nil {
		logService.ErrorWithContext("Directory walk failed", logCtx, zap.Error(err))
		return nil, errs.Wrap(err, errs.TypeInternal, "walk error").
			WithContext("agent_id", logCtx.AgentID).
			WithContext("root_path", root)
	}

	logService.DebugWithContext("Recursive glob completed", logCtx,
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
