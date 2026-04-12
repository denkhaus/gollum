package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	// editToolImpl performs exact string replacements in files
	editToolImpl struct {
		logService   logger.LoggerService
		fsm          state.FileStateManager
		hookManager  hooks.HookManager
		diffProvider diff.Provider
		agent        shared.Agent
	}

	// EditToolProvider creates EditTool instances via DI
	EditToolProvider interface {
		CreateTool(agent shared.Agent) gollem.Tool
	}

	editToolProvider struct {
		logService   logger.LoggerService
		fsm          state.FileStateManager
		hookManager  hooks.HookManager
		diffProvider diff.Provider
	}
)

// NewEditToolProvider creates a provider for Edit tools
func NewEditToolProvider(injector do.Injector) (EditToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm := do.MustInvoke[state.FileStateManager](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	diffProvider := do.MustInvoke[diff.Provider](injector)

	return &editToolProvider{
		logService:   logService,
		fsm:          fsm,
		hookManager:  hookManager,
		diffProvider: diffProvider,
	}, nil
}

// CreateTool creates a new EditTool with agent
func (p *editToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &editToolImpl{
		logService:   p.logService,
		fsm:          p.fsm,
		hookManager:  p.hookManager,
		diffProvider: p.diffProvider,
		agent:        agent,
	}
}

// Spec returns the tool specification for the Edit tool
func (t *editToolImpl) Spec() gollem.ToolSpec {
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
func (t *editToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToLoggingContext(), shared.ToolNameEdit, args,
		func() (map[string]any, error) {
			return t.runEdit(ctx, args)
		})
}

// runEdit implements the core edit logic
func (t *editToolImpl) runEdit(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	filePath, errResp := args.GetFilePath(shared.ParamFilePath)
	if errResp != nil {
		return errResp, nil
	}

	oldString, errResp := args.MustGetString(shared.ParamOldString)
	if errResp != nil {
		return errResp, nil
	}

	newString, errResp := args.MustGetString(shared.ParamNewString)
	if errResp != nil {
		return errResp, nil
	}

	// Get replace_all flag, default to false
	replaceAll := args.GetBool(shared.ParamReplaceAll, false)

	// Convert relative path to absolute
	originalPath := filePath
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		t.logService.ErrorWithContext("Edit operation failed: failed to resolve absolute path",
			t.agent.ToLoggingContext(),
			zap.String("file_path", originalPath),
			zap.Error(err),
		)
		return map[string]any{
			string(shared.KeySuccess): false,
			string(shared.KeyError):   fmt.Sprintf("failed to resolve absolute path: %v", err),
		}, nil
	}

	t.logService.InfoWithContext("Edit operation started",
		t.agent.ToLoggingContext(),
		zap.String("file_path", filePath),
		zap.Int("old_string_length", len(oldString)),
		zap.Int("new_string_length", len(newString)),
		zap.Bool("replace_all", replaceAll),
	)

	// Verify file was read first (check FileStateManager)
	fileStats, err := t.fsm.GetFileStats(filePath)
	if err != nil || fileStats == nil {
		t.logService.WarnWithContext("Edit operation failed: file must be read before editing",
			t.agent.ToLoggingContext(),
			zap.String("file_path", filePath),
			zap.String("required_tool", shared.ToolNameReadFile.String()),
		)
		return map[string]any{
			string(shared.KeySuccess): false,
			string(shared.KeyError):   fmt.Sprintf("file must be read before editing. Use the %s tool first.", shared.ToolNameReadFile),
		}, nil
	}

	// Check if file is stale for this agent (not read or modified since last read)
	isStale, err := t.fsm.IsFileStaleForAgent(t.agent.GetID(), filePath)
	if err != nil {
		t.logService.ErrorWithContext("Failed to check file staleness",
			t.agent.ToLoggingContext(),
			zap.String("file_path", filePath),
			zap.Error(err),
		)
		return map[string]any{
			string(shared.KeySuccess): false,
			string(shared.KeyError):   fmt.Sprintf("failed to check file state: %v", err),
		}, nil
	}

	if isStale {
		t.logService.WarnWithContext("Edit operation failed: file must be read before editing",
			t.agent.ToLoggingContext(),
			zap.String("file_path", filePath),
			zap.String("required_tool", shared.ToolNameReadFile.String()),
		)
		return map[string]any{
			string(shared.KeySuccess): false,
			string(shared.KeyError):   fmt.Sprintf("you must read this file before editing it. Use the %s tool first to get the latest content.", shared.ToolNameReadFile),
		}, nil
	}

	t.logService.DebugWithContext("Acquiring exclusive lock for edit operation",
		t.agent.ToLoggingContext(),
		zap.String("file_path", filePath),
	)

	// Perform edit under exclusive lock (staleness already checked above)
	result, err := t.fsm.DoWorkWithOptions(ctx, filePath, t.agent.GetID(), state.LockModeExclusive,
		state.WorkOptions{
			UpdateStatsAfter: true, // Update stats after write
		},
		func(_ context.Context, token *state.LockToken) (any, error) {
			t.logService.DebugWithContext("Edit lock acquired, reading file content",
				t.agent.ToLoggingContext(),
				zap.String("file_path", filePath),
				zap.String("lock_agent_id", token.AgentID.String()),
				zap.String("lock_mode", string(token.Mode)),
			)

			// Read the current file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.logService.ErrorWithContext("Failed to read file for editing",
					t.agent.ToLoggingContext(),
					zap.String("file_path", filePath),
					zap.Error(err),
				)
				return nil, errs.Wrap(err, errs.TypeInternal, "failed to read file").
					WithContext("agent_id", t.agent.GetID().String()).
					WithContext("file_path", filePath)
			}

			contentStr := string(content)

			// Count occurrences of old_string
			count := strings.Count(contentStr, oldString)

			if count == 0 {
				t.logService.WarnWithContext("Edit operation failed: old_string not found in file",
					t.agent.ToLoggingContext(),
					zap.String("file_path", filePath),
					zap.Int("old_string_length", len(oldString)),
				)
				return map[string]any{
					string(shared.KeySuccess): false,
					string(shared.KeyError):   "old_string not found in file",
				}, nil
			}

			// If not replacing all, verify uniqueness
			if !replaceAll && count > 1 {
				t.logService.WarnWithContext("Edit operation failed: old_string appears multiple times",
					t.agent.ToLoggingContext(),
					zap.String("file_path", filePath),
					zap.Int("occurrence_count", count),
				)
				return map[string]any{
					string(shared.KeySuccess):      false,
					string(shared.KeyError):        fmt.Sprintf("old_string appears %d times in the file. For safety, it must be unique unless replace_all is set to true", count),
					string(shared.KeyReplacements): count,
				}, nil
			}

			// Perform replacement
			var newContent string
			if replaceAll {
				newContent = strings.ReplaceAll(contentStr, oldString, newString)
			} else {
				newContent = strings.Replace(contentStr, oldString, newString, 1)
			}

			t.logService.DebugWithContext("Writing modified content to file",
				t.agent.ToLoggingContext(),
				zap.String("file_path", filePath),
				zap.Int("replacements", count),
				zap.Int("old_size", len(contentStr)),
				zap.Int("new_size", len(newContent)),
			)

			// Wrap the file write with file write hooks (edit is a write operation)
			err = t.hookManager.WithFileWriteHooks(ctx, t.agent.ToLoggingContext(), filePath, newContent,
				func(finalContent string) error {
					// Write the modified content back
					if err := os.WriteFile(filePath, []byte(finalContent), 0o644); err != nil {
						t.logService.ErrorWithContext("Failed to write modified file",
							t.agent.ToLoggingContext(),
							zap.String("file_path", filePath),
							zap.Error(err),
						)
						return errs.Wrap(err, errs.TypeInternal, "failed to write file").
							WithContext("agent_id", t.agent.GetID().String()).
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
				t.logService.ErrorWithContext("Failed to get file stats after edit",
					t.agent.ToLoggingContext(),
					zap.String("file_path", filePath),
					zap.Error(err),
				)
				return nil, errs.Wrap(err, errs.TypeInternal, "failed to get file stats").
					WithContext("agent_id", t.agent.GetID().String()).
					WithContext("file_path", filePath)
			}

			t.logService.InfoWithContext("Edit operation completed successfully",
				t.agent.ToLoggingContext(),
				zap.String("file_path", filePath),
				zap.Int("replacements", count),
				zap.String("old_checksum", fileStats.Checksum),
				zap.String("new_checksum", stats.Checksum),
				zap.Int64("new_size", stats.Size),
			)

			// Generate diff between original and modified content
			diffStr, diffErr := t.diffProvider.GenerateDiff(filePath, filePath, contentStr, newContent)
			if diffErr != nil {
				t.logService.WarnWithContext("Failed to generate diff",
					t.agent.ToLoggingContext(),
					zap.String("file_path", filePath),
					zap.Error(diffErr),
				)
			}

			result := map[string]any{
				string(shared.KeySuccess):      true,
				string(shared.KeyFilePath):     filePath,
				string(shared.KeyReplacements): count,
				string(shared.KeyOldLen):       len(oldString),
				string(shared.KeyNewLen):       len(newString),
				string(shared.KeyChecksum):     stats.Checksum,
				string(shared.KeySize):         stats.Size,
				string(shared.KeyModified):     stats.ModifiedTime,
				string(shared.KeyLockedBy):     token.AgentID,
			}

			// Add diff information if generation was successful
			if diffStr != "" {
				result[string(shared.KeyDiff)] = t.diffProvider.FormatForDisplay(diffStr)
				result[string(shared.KeyDiffCompact)] = t.diffProvider.FormatCompact(diffStr)
				result[string(shared.KeyIsNewFile)] = false
			}

			return result, nil
		})
	if err != nil {
		t.logService.ErrorWithContext("Edit operation failed with error",
			t.agent.ToLoggingContext(),
			zap.String("file_path", filePath),
			zap.Error(err),
		)
		return map[string]any{
			string(shared.KeySuccess): false,
			string(shared.KeyError):   err.Error(),
		}, nil
	}

	return result.(map[string]any), nil
}
