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
	"github.com/google/uuid"
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
		agentID      uuid.UUID
	}

	// WriteFileToolProvider creates WriteFileTool instances via DI
	WriteFileToolProvider interface {
		CreateTool(agentID uuid.UUID) gollem.Tool
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

// CreateWriteFileTool creates a new WriteFileTool with injected dependencies and agent ID
func (p *writeFileToolProvider) CreateTool(agentID uuid.UUID) gollem.Tool {
	return &writeFileToolImpl{
		logService:   p.logService,
		fsm:          p.fsm,
		hookManager:  p.hookManager,
		diffProvider: p.diffProvider,
		agentID:      agentID,
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
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameWriteFile, args,
		func() (map[string]any, error) {
			return t.runFileWrite(ctx, args)
		})
}

// runFileWrite implements the core file write logic
func (t *writeFileToolImpl) runFileWrite(ctx context.Context, args map[string]any) (map[string]any, error) {
	path, ok := args["file_path"].(string)
	if !ok || path == "" {
		t.logService.Debug("Write file failed: invalid file_path parameter",
			zap.String("agent_id", t.agentID.String()),
			zap.String("error", "file_path is required and must be a non-empty string"),
		)
		return map[string]any{
			"success": false,
			"error":   "file_path is required and must be a non-empty string",
		}, nil
	}

	content, ok := args["content"].(string)
	if !ok {
		t.logService.Debug("Write file failed: invalid content parameter",
			zap.String("agent_id", t.agentID.String()),
			zap.String("file_path", path),
			zap.String("error", "content is required and must be a string"),
		)
		return map[string]any{
			"success": false,
			"error":   "content is required and must be a string",
		}, nil
	}

	// Get create_dirs flag, default to false
	createDirs := false
	if createDirsFlag, exists := args["create_dirs"].(bool); exists {
		createDirs = createDirsFlag
	}

	// Convert relative path to absolute
	path, err := filepath.Abs(path)
	if err != nil {
		t.logService.Error("Failed to resolve absolute path",
			zap.String("agent_id", t.agentID.String()),
			zap.String("file_path", path),
			zap.Error(err),
		)
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("failed to resolve absolute path: %v", err),
		}, nil
	}

	t.logService.Info("Writing file",
		zap.String("agent_id", t.agentID.String()),
		zap.String("file_path", path),
		zap.Int("content_bytes", len(content)),
		zap.Bool("create_dirs", createDirs),
	)

	// Create parent directories if requested (before acquiring lock)
	if createDirs {
		dir := filepath.Dir(path)
		t.logService.Debug("Creating parent directories",
			zap.String("agent_id", t.agentID.String()),
			zap.String("file_path", path),
			zap.String("dir", dir),
		)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.logService.Error("Failed to create directories",
				zap.String("agent_id", t.agentID.String()),
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
		isStale, err := t.fsm.IsFileStaleForAgent(t.agentID, path)
		if err != nil {
			t.logService.Error("Failed to check file staleness",
				zap.String("agent_id", t.agentID.String()),
				zap.String("file_path", path),
				zap.Error(err))
			return map[string]any{
				"success": false,
				"error":   fmt.Sprintf("failed to check file state: %v", err),
			}, nil
		}

		if isStale {
			t.logService.Warn("Write operation failed: file must be read before writing existing file",
				zap.String("agent_id", t.agentID.String()),
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
	result, err := t.fsm.DoWorkWithOptions(ctx, path, t.agentID, state.LockModeExclusive,
		state.WorkOptions{
			UpdateStatsAfter: true, // Immediately update stats after write
		},
		func(_ context.Context, token *state.LockToken) (any, error) {
			// Wrap the actual file write with file write hooks
			err := t.hookManager.WithFileWriteHooks(ctx, uuid.Nil, t.agentID, path, content,
				func(finalContent string) error {
					// Write the file
					if err := os.WriteFile(path, []byte(finalContent), 0o644); err != nil {
						t.logService.Error("Failed to write file",
							zap.String("agent_id", t.agentID.String()),
							zap.String("file_path", path),
							zap.Error(err),
						)
						return errs.Wrap(err, errs.TypeInternal, "failed to write file").
							WithContext("agent_id", t.agentID).
							WithContext("file_path", path)
					}

					// Get updated stats (updated by DoWorkWithOptions)
					stats, err := t.fsm.GetFileStats(path)
					if err != nil {
						t.logService.Warn("Failed to get file stats after write",
							zap.String("agent_id", t.agentID.String()),
							zap.String("file_path", path),
							zap.Error(err),
						)
						return errs.Wrap(err, errs.TypeInternal, "failed to get file stats").
							WithContext("agent_id", t.agentID).
							WithContext("file_path", path)
					}

					t.logService.Info("File written successfully",
						zap.String("agent_id", t.agentID.String()),
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
				t.logService.Warn("Failed to get file stats after write",
					zap.String("agent_id", t.agentID.String()),
					zap.String("file_path", path),
					zap.Error(err),
				)
				return nil, errs.Wrap(err, errs.TypeInternal, "failed to get file stats").
					WithContext("agent_id", t.agentID).
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
				t.logService.Warn("Failed to generate diff",
					zap.String("agent_id", t.agentID.String()),
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
		t.logService.Error("File write operation failed",
			zap.String("agent_id", t.agentID.String()),
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
				t.logService.Warn("Race condition detected: file modified since last read",
					zap.String("agent_id", t.agentID.String()),
					zap.String("file_path", path),
					zap.String("error", errMsg),
				)
			}
		}
	}

	return resultMap, nil
}
