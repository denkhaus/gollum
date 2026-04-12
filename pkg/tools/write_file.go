package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/diff"
	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

type (
	// writeFileToolImpl writes content to files with automatic locking and checksum verification
	writeFileToolImpl struct {
		logService   logger.LoggerService
		fsm          state.FileStateManager
		hookManager  hooks.HookManager
		diffProvider diff.Provider
		agent        shared.Agent
	}

	// WriteFileToolProvider creates WriteFileTool instances via DI
	WriteFileToolProvider interface {
		CreateTool(agent shared.Agent) gollem.Tool
	}

	writeFileToolProvider struct {
		logService   logger.LoggerService
		fsm          state.FileStateManager
		hookManager  hooks.HookManager
		diffProvider diff.Provider
	}
)

// NewWriteFileToolProvider creates a provider for WriteFile tools
func NewWriteFileToolProvider(injector do.Injector) (WriteFileToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm := do.MustInvoke[state.FileStateManager](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	diffProvider := do.MustInvoke[diff.Provider](injector)

	return &writeFileToolProvider{logService: logService, fsm: fsm, hookManager: hookManager, diffProvider: diffProvider}, nil
}

// CreateWriteFileTool creates a new WriteFileTool with injected dependencies and agent
func (p *writeFileToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &writeFileToolImpl{
		logService:   p.logService,
		fsm:          p.fsm,
		hookManager:  p.hookManager,
		diffProvider: p.diffProvider,
		agent:        agent,
	}
}

// Spec returns the tool specification for the WriteFile tool
func (t *writeFileToolImpl) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameWriteFile.String(),
		Description: "Writes a file to the local filesystem. Automatically prevents overwriting if the file was modified by another agent since you last read it. Creates parent directories if create_dirs is true.",
		Parameters: map[string]*gollem.Parameter{
			"file_path": {
				Type:        gollem.TypeString,
				Description: "The absolute path to the file to write",
			},
			"content": {
				Type:        gollem.TypeString,
				Description: "The content to write to the file",
			},
			"create_dirs": {
				Type:        gollem.TypeBoolean,
				Description: "If true, create parent directories if they don't exist (default: false)",
			},
		},
	}
}

// Run executes the WriteFile tool to write content to files
func (t *writeFileToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToLoggingContext(), shared.ToolNameWriteFile, args,
		func() (map[string]any, error) {
			return t.runFileWrite(ctx, args)
		})
}

// runFileWrite implements the core file write logic
func (t *writeFileToolImpl) runFileWrite(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	path, errResp := args.GetFilePath(shared.ParamFilePath)
	if errResp != nil {
		return errResp, nil
	}

	// Content must be provided and must be a string type (but can be empty string for creating empty files)
	content, errResp := args.GetStringAllowEmpty(shared.ParamContent)
	if errResp != nil {
		return errResp, nil
	}

	// Get create_dirs flag, default to false
	createDirs := args.GetBool(shared.ParamCreateDirs, false)

	// Convert relative path to absolute
	path, err := filepath.Abs(path)
	if err != nil {
		t.logService.ErrorWithContext("Failed to resolve absolute path",
			t.agent.ToLoggingContext(),
			zap.String("file_path", path),
			zap.Error(err),
		)
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("failed to resolve absolute path: %v", err),
		}, nil
	}

	t.logService.InfoWithContext("Writing file",
		t.agent.ToLoggingContext(),
		zap.String("file_path", path),
		zap.Int("content_bytes", len(content)),
		zap.Bool("create_dirs", createDirs),
	)

	// Create parent directories if requested (before acquiring lock)
	if createDirs {
		dir := filepath.Dir(path)
		t.logService.DebugWithContext("Creating parent directories",
			t.agent.ToLoggingContext(),
			zap.String("file_path", path),
			zap.String("dir", dir),
		)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.logService.ErrorWithContext("Failed to create directories",
				t.agent.ToLoggingContext(),
				zap.String("file_path", path),
				zap.String("dir", dir),
				zap.Error(err),
			)
			return map[string]any{
				"success": false,
				"error":   fmt.Sprintf("failed to create directories: %v", err),
			}, nil
		}
	}

	// Check if file exists and if agent has read it (for race condition detection)
	var oldContent string
	isNewFile := false
	if _, err := os.Stat(path); err == nil {
		// File exists - verify it's not stale for this agent
		isStale, err := t.fsm.IsFileStaleForAgent(t.agent.GetID(), path)
		if err != nil {
			t.logService.ErrorWithContext("Failed to check file staleness",
				t.agent.ToLoggingContext(),
				zap.String("file_path", path),
				zap.Error(err))
			return map[string]any{
				"success": false,
				"error":   fmt.Sprintf("failed to check file state: %v", err),
			}, nil
		}

		if isStale {
			t.logService.WarnWithContext("Write operation failed: file must be read before writing existing file",
				t.agent.ToLoggingContext(),
				zap.String("file_path", path),
				zap.String("required_tool", shared.ToolNameReadFile.String()))
			return map[string]any{
				"success": false,
				"error":   fmt.Sprintf("file was modified since you last read it on %s. Please use the %s tool to re-read the file before writing to get the latest content.", "unknown", shared.ToolNameReadFile),
				"path":    path,
			}, nil
		}

		// Read existing content for diff generation
		if contentBytes, err := os.ReadFile(path); err == nil {
			oldContent = string(contentBytes)
		}
	} else {
		// File doesn't exist - this is a new file
		isNewFile = true
	}
	// If file doesn't exist, agent can write without reading first (new file)

	// Execute file write under exclusive lock (staleness already checked for existing files)
	result, err := t.fsm.DoWorkWithOptions(ctx, path, t.agent.GetID(), state.LockModeExclusive,
		state.WorkOptions{
			UpdateStatsAfter: true, // Immediately update stats after write
		},
		func(_ context.Context, token *state.LockToken) (any, error) {
			// Wrap the actual file write with file write hooks
			err := t.hookManager.WithFileWriteHooks(ctx, t.agent.ToLoggingContext(), path, content,
				func(finalContent string) error {
					// Write the file
					if err := os.WriteFile(path, []byte(finalContent), 0o644); err != nil {
						t.logService.ErrorWithContext("Failed to write file",
							t.agent.ToLoggingContext(),
							zap.String("file_path", path),
							zap.Error(err),
						)
						return errs.Wrap(err, errs.TypeInternal, "failed to write file").
							WithContext("agent_id", t.agent.GetID().String()).
							WithContext("file_path", path)
					}

					// Get updated stats (updated by DoWorkWithOptions)
					stats, err := t.fsm.GetFileStats(path)
					if err != nil {
						t.logService.WarnWithContext("Failed to get file stats after write",
							t.agent.ToLoggingContext(),
							zap.String("file_path", path),
							zap.Error(err),
						)
						return errs.Wrap(err, errs.TypeInternal, "failed to get file stats").
							WithContext("agent_id", t.agent.GetID().String()).
							WithContext("file_path", path)
					}

					t.logService.InfoWithContext("File written successfully",
						t.agent.ToLoggingContext(),
						zap.String("file_path", path),
						zap.Int64("size", stats.Size),
						zap.String("checksum", stats.Checksum),
						zap.String("modified_time", stats.ModifiedTime.Format(time.RFC3339)),
					)

					return nil
				})

			if err != nil {
				return nil, err
			}

			// Get stats for the result
			stats, err := t.fsm.GetFileStats(path)
			if err != nil {
				t.logService.WarnWithContext("Failed to get file stats after write",
					t.agent.ToLoggingContext(),
					zap.String("file_path", path),
					zap.Error(err),
				)
				return nil, errs.Wrap(err, errs.TypeInternal, "failed to get file stats").
					WithContext("agent_id", t.agent.GetID().String()).
					WithContext("file_path", path)
			}

			// Generate diff based on whether this is a new file or existing file
			var diffStr string
			var diffErr error
			if isNewFile {
				diffStr, diffErr = t.diffProvider.GenerateDiffForNewFile(path, content)
			} else {
				diffStr, diffErr = t.diffProvider.GenerateDiff(path, path, oldContent, content)
			}

			if diffErr != nil {
				t.logService.WarnWithContext("Failed to generate diff",
					t.agent.ToLoggingContext(),
					zap.String("file_path", path),
					zap.Error(diffErr))
			}

			result := map[string]any{
				string(shared.KeySuccess):  true,
				string(shared.KeyFilePath): path,
				"bytes":                    len(content),
				string(shared.KeySize):     stats.Size,
				string(shared.KeyChecksum): stats.Checksum,
				string(shared.KeyModified): stats.ModifiedTime,
				string(shared.KeyLockedBy): token.AgentID,
			}

			// Add diff information if generation was successful
			if diffStr != "" {
				result[string(shared.KeyDiff)] = t.diffProvider.FormatForDisplay(diffStr)
				result[string(shared.KeyDiffCompact)] = t.diffProvider.FormatCompact(diffStr)
				result[string(shared.KeyIsNewFile)] = isNewFile
			}

			return result, nil
		})
	if err != nil {
		t.logService.ErrorWithContext("File write operation failed",
			t.agent.ToLoggingContext(),
			zap.String("file_path", path),
			zap.Error(err),
		)
		return map[string]any{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	// Check for race condition warning in result
	resultMap, ok := result.(map[string]any)
	if ok && resultMap["success"] == false {
		if errMsg, ok := resultMap["error"].(string); ok {
			if strings.Contains(errMsg, "modified since you last read it") ||
				strings.Contains(errMsg, "file has been modified since read") {
				t.logService.WarnWithContext("Race condition detected: file modified since last read",
					t.agent.ToLoggingContext(),
					zap.String("file_path", path),
					zap.String("error", errMsg),
				)
			}
		}
	}

	return resultMap, nil
}
