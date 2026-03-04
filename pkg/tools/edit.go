package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	// EditTool performs exact string replacements in files
	EditTool struct {
		logService  logger.LoggerService
		fsm         state.FileStateManager
		hookManager hooks.HookManager
		agentID     uuid.UUID
	}

	// EditToolProvider creates EditTool instances via DI
	EditToolProvider interface {
		CreateTool(agentID uuid.UUID) *EditTool
	}

	editToolProvider struct {
		logService  logger.LoggerService
		fsm         state.FileStateManager
		hookManager hooks.HookManager
	}
)

// NewEditToolProvider creates a provider for Edit tools
func NewEditToolProvider(injector do.Injector) (EditToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm := do.MustInvoke[state.FileStateManager](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)

	return &editToolProvider{
		logService:  logService,
		fsm:         fsm,
		hookManager: hookManager,
	}, nil
}

// CreateTool creates a new EditTool with agent ID
func (p *editToolProvider) CreateTool(agentID uuid.UUID) *EditTool {
	return &EditTool{
		logService:  p.logService,
		fsm:         p.fsm,
		hookManager: p.hookManager,
		agentID:     agentID,
	}
}

// Spec returns the tool specification for the Edit tool
func (t *EditTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameEdit.String(),
		Description: "Performs exact string replacements in files. Requires the file to be read first. The old_string must be unique in the file. This tool does NOT use regex - it does exact string matching.",
		Parameters: map[string]*gollem.Parameter{
			"file_path": {
				Type:        gollem.TypeString,
				Description: "The absolute path to the file to edit",
			},
			"old_string": {
				Type:        gollem.TypeString,
				Description: "The exact string to replace (must be unique in the file)",
			},
			"new_string": {
				Type:        gollem.TypeString,
				Description: "The new string to replace it with",
			},
			"replace_all": {
				Type:        gollem.TypeBoolean,
				Description: "Replace all occurrences instead of just the first (default: false)",
			},
		},
	}
}

// Run executes the Edit tool to perform string replacements in files
func (t *EditTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameEdit, args,
		func() (map[string]any, error) {
			return t.runEdit(ctx, args)
		})
}

// runEdit implements the core edit logic
func (t *EditTool) runEdit(ctx context.Context, args map[string]any) (map[string]any, error) {
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		t.logService.Error("Edit operation failed: file_path is required and must be a non-empty string",
			zap.String("agent_id", t.agentID.String()))
		return map[string]any{
			"success": false,
			"error":   "file_path is required and must be a non-empty string",
		}, nil
	}

	oldString, ok := args["old_string"].(string)
	if !ok || oldString == "" {
		t.logService.Error("Edit operation failed: old_string is required and must be a non-empty string",
			zap.String("agent_id", t.agentID.String()),
			zap.String("file_path", filePath))
		return map[string]any{
			"success": false,
			"error":   "old_string is required and must be a non-empty string",
		}, nil
	}

	newString, ok := args["new_string"].(string)
	if !ok {
		t.logService.Error("Edit operation failed: new_string is required and must be a string",
			zap.String("agent_id", t.agentID.String()),
			zap.String("file_path", filePath))
		return map[string]any{
			"success": false,
			"error":   "new_string is required and must be a string",
		}, nil
	}

	// Get replace_all flag, default to false
	replaceAll := false
	if flagVal, exists := args["replace_all"].(bool); exists {
		replaceAll = flagVal
	}

	// Convert relative path to absolute
	originalPath := filePath
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		t.logService.Error("Edit operation failed: failed to resolve absolute path",
			zap.String("agent_id", t.agentID.String()),
			zap.String("file_path", originalPath),
			zap.Error(err))
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("failed to resolve absolute path: %v", err),
		}, nil
	}

	t.logService.Info("Edit operation started",
		zap.String("agent_id", t.agentID.String()),
		zap.String("file_path", filePath),
		zap.Int("old_string_length", len(oldString)),
		zap.Int("new_string_length", len(newString)),
		zap.Bool("replace_all", replaceAll))

	// Verify file was read first (check FileStateManager)
	fileStats, err := t.fsm.GetFileStats(filePath)
	if err != nil || fileStats == nil {
		t.logService.Warn("Edit operation failed: file must be read before editing",
			zap.String("agent_id", t.agentID.String()),
			zap.String("file_path", filePath),
			zap.String("required_tool", shared.ToolNameReadFile.String()))
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("file must be read before editing. Use the %s tool first.", shared.ToolNameReadFile),
		}, nil
	}

	// Check if file is stale for this agent (not read or modified since last read)
	isStale, err := t.fsm.IsFileStaleForAgent(t.agentID, filePath)
	if err != nil {
		t.logService.Error("Failed to check file staleness",
			zap.String("agent_id", t.agentID.String()),
			zap.String("file_path", filePath),
			zap.Error(err))
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("failed to check file state: %v", err),
		}, nil
	}

	if isStale {
		t.logService.Warn("Edit operation failed: file must be read before editing",
			zap.String("agent_id", t.agentID.String()),
			zap.String("file_path", filePath),
			zap.String("required_tool", shared.ToolNameReadFile.String()))
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("you must read this file before editing it. Use the %s tool first to get the latest content.", shared.ToolNameReadFile),
		}, nil
	}

	t.logService.Debug("Acquiring exclusive lock for edit operation",
		zap.String("agent_id", t.agentID.String()),
		zap.String("file_path", filePath))

	// Perform edit under exclusive lock (staleness already checked above)
	result, err := t.fsm.DoWorkWithOptions(ctx, filePath, t.agentID, state.LockModeExclusive,
		state.WorkOptions{
			UpdateStatsAfter: true, // Update stats after write
		},
		func(_ context.Context, token *state.LockToken) (any, error) {
			t.logService.Debug("Edit lock acquired, reading file content",
				zap.String("agent_id", t.agentID.String()),
				zap.String("file_path", filePath),
				zap.String("lock_agent_id", token.AgentID.String()),
				zap.String("lock_mode", string(token.Mode)))

			// Read the current file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.logService.Error("Failed to read file for editing",
					zap.String("agent_id", t.agentID.String()),
					zap.String("file_path", filePath),
					zap.Error(err))
				return nil, errs.Wrap(err, errs.TypeInternal, "failed to read file").
					WithContext("agent_id", t.agentID).
					WithContext("file_path", filePath)
			}

			contentStr := string(content)

			// Count occurrences of old_string
			count := strings.Count(contentStr, oldString)

			if count == 0 {
				t.logService.Warn("Edit operation failed: old_string not found in file",
					zap.String("agent_id", t.agentID.String()),
					zap.String("file_path", filePath),
					zap.Int("old_string_length", len(oldString)))
				return map[string]any{
					"success": false,
					"error":   "old_string not found in file",
				}, nil
			}

			// If not replacing all, verify uniqueness
			if !replaceAll && count > 1 {
				t.logService.Warn("Edit operation failed: old_string appears multiple times",
					zap.String("agent_id", t.agentID.String()),
					zap.String("file_path", filePath),
					zap.Int("occurrence_count", count))
				return map[string]any{
					"success": false,
					"error":   fmt.Sprintf("old_string appears %d times in the file. For safety, it must be unique unless replace_all is set to true", count),
					"count":   count,
				}, nil
			}

			// Perform replacement
			var newContent string
			if replaceAll {
				newContent = strings.ReplaceAll(contentStr, oldString, newString)
			} else {
				newContent = strings.Replace(contentStr, oldString, newString, 1)
			}

			t.logService.Debug("Writing modified content to file",
				zap.String("agent_id", t.agentID.String()),
				zap.String("file_path", filePath),
				zap.Int("replacements", count),
				zap.Int("old_size", len(contentStr)),
				zap.Int("new_size", len(newContent)))

			// Wrap the file write with file write hooks (edit is a write operation)
			err = t.hookManager.WithFileWriteHooks(ctx, uuid.Nil, t.agentID, filePath, newContent,
				func(finalContent string) error {
					// Write the modified content back
					if err := os.WriteFile(filePath, []byte(finalContent), 0o644); err != nil {
						t.logService.Error("Failed to write modified file",
							zap.String("agent_id", t.agentID.String()),
							zap.String("file_path", filePath),
							zap.Error(err))
						return errs.Wrap(err, errs.TypeInternal, "failed to write file").
							WithContext("agent_id", t.agentID).
							WithContext("file_path", filePath)
					}
					return nil
				})

			if err != nil {
				return nil, err
			}

			// Get updated stats
			stats, err := t.fsm.GetFileStats(filePath)
			if err != nil {
				t.logService.Error("Failed to get file stats after edit",
					zap.String("agent_id", t.agentID.String()),
					zap.String("file_path", filePath),
					zap.Error(err))
				return nil, errs.Wrap(err, errs.TypeInternal, "failed to get file stats").
					WithContext("agent_id", t.agentID).
					WithContext("file_path", filePath)
			}

			t.logService.Info("Edit operation completed successfully",
				zap.String("agent_id", t.agentID.String()),
				zap.String("file_path", filePath),
				zap.Int("replacements", count),
				zap.String("old_checksum", fileStats.Checksum),
				zap.String("new_checksum", stats.Checksum),
				zap.Int64("new_size", stats.Size))

			return map[string]any{
				"success":      true,
				"file_path":    filePath,
				"replacements": count,
				"old_len":      len(oldString),
				"new_len":      len(newString),
				"checksum":     stats.Checksum,
				"size":         stats.Size,
				"modified":     stats.ModifiedTime,
				"locked_by":    token.AgentID,
			}, nil
		})
	if err != nil {
		t.logService.Error("Edit operation failed with error",
			zap.String("agent_id", t.agentID.String()),
			zap.String("file_path", filePath),
			zap.Error(err))
		return map[string]any{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	return result.(map[string]any), nil
}
