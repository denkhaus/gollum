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
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

type (
	// readFileToolImpl reads file contents with shared locking and automatic stats tracking
	readFileToolImpl struct {
		logService  logger.LoggerService
		fsm         state.FileStateManager
		hookManager hooks.HookManager
		agent       shared.Agent
	}

	// ReadFileToolProvider creates ReadFileTool instances via DI
	ReadFileToolProvider interface {
		CreateTool(agent shared.Agent) gollem.Tool
	}

	readFileToolProvider struct {
		logService  logger.LoggerService
		fsm         state.FileStateManager
		hookManager hooks.HookManager
	}
)

// NewReadFileToolProvider creates a provider for ReadFile tools
func NewReadFileToolProvider(injector do.Injector) (ReadFileToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm := do.MustInvoke[state.FileStateManager](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)

	return &readFileToolProvider{logService: logService, fsm: fsm, hookManager: hookManager}, nil
}

// CreateTool creates a new ReadFileTool with injected dependencies and agent
func (p *readFileToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &readFileToolImpl{
		logService:  p.logService,
		fsm:         p.fsm,
		hookManager: p.hookManager,
		agent:       agent,
	}
}

// Spec returns the tool specification for the ReadFile tool
func (t *readFileToolImpl) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameReadFile.String(),
		Description: "Reads a file from the local filesystem. Returns the file content with line numbers. Automatically updates file stats in FileStateManager with shared locking (allows concurrent readers).",
		Parameters: map[string]*gollem.Parameter{
			"file_path": {
				Type:        gollem.TypeString,
				Description: "The absolute path to the file to read",
			},
			"offset": {
				Type:        gollem.TypeNumber,
				Description: fmt.Sprintf("The line number to start reading from (default: %d)", DefaultReadFileOffset),
			},
			"limit": {
				Type:        gollem.TypeNumber,
				Description: fmt.Sprintf("The maximum number of lines to read (default: %d)", DefaultReadFileLimit),
			},
		},
	}
}

// Run executes the ReadFile tool to read file contents
func (t *readFileToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToLoggingContext(), shared.ToolNameReadFile, args,
		func() (map[string]any, error) {
			return t.runFileRead(ctx, args)
		})
}

// runFileRead implements the core file read logic
func (t *readFileToolImpl) runFileRead(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	path, errResp := args.GetFilePath(shared.ParamFilePath)
	if errResp != nil {
		return errResp, nil
	}

	// Get optional offset (line number, default: DefaultReadFileOffset)
	offset := max(1, args.GetInt(shared.ParamOffset, DefaultReadFileOffset))

	// Get optional limit (max lines, default: DefaultReadFileLimit)
	limit := max(1, args.GetInt(shared.ParamLimit, DefaultReadFileLimit))

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

	t.logService.InfoWithContext("Reading file",
		t.agent.ToLoggingContext(),
		zap.String("file_path", path),
		zap.Int("offset", offset),
		zap.Int("limit", limit),
	)

	// Execute file read under shared lock with read tracking for race condition detection
	result, err := t.fsm.DoWorkWithOptions(ctx, path, t.agent.GetID(), state.LockModeShared, state.WorkOptions{
		TrackRead: true, // Track that this agent read this file (for automatic race condition detection on write)
	},
		func(_ context.Context, token *state.LockToken) (any, error) {
			// Wrap the file read with file read hooks
			content, err := t.hookManager.WithFileReadHooks(ctx, t.agent.ToLoggingContext(), path,
				func() (string, error) {
					// Check if file exists
					_, err := os.Stat(path)
					if err != nil {
						if os.IsNotExist(err) {
							t.logService.DebugWithContext("File not found",
								t.agent.ToLoggingContext(),
								zap.String("file_path", path),
							)
							return "", nil // Return empty content, will be handled below
						}
						t.logService.ErrorWithContext("Failed to stat file",
							t.agent.ToLoggingContext(),
							zap.String("file_path", path),
							zap.Error(err),
						)
						return "", errs.Wrap(err, errs.TypeInternal, "failed to stat file").
							WithContext("agent_id", t.agent.GetID()).
							WithContext("file_path", path)
					}

					// Read entire file
					contentBytes, err := os.ReadFile(path)
					if err != nil {
						t.logService.ErrorWithContext("Failed to read file",
							t.agent.ToLoggingContext(),
							zap.String("file_path", path),
							zap.Error(err),
						)
						return "", errs.Wrap(err, errs.TypeInternal, "failed to read file").
							WithContext("agent_id", t.agent.GetID()).
							WithContext("file_path", path)
					}

					return string(contentBytes), nil
				})

			// Handle file not found case (content is empty string)
			if content == "" && err == nil {
				// Check if file actually doesn't exist
				if _, statErr := os.Stat(path); statErr != nil && os.IsNotExist(statErr) {
					return map[string]any{
						"success": false,
						"error":   "file not found",
						"path":    path,
					}, nil
				}
			}

			if err != nil {
				return nil, err
			}

			// Split into lines
			allLines := strings.Split(content, "\n")

			// Validate offset
			if offset > len(allLines) {
				t.logService.WarnWithContext("Offset exceeds line count",
					t.agent.ToLoggingContext(),
					zap.String("file_path", path),
					zap.Int("offset", offset),
					zap.Int("total_lines", len(allLines)),
				)
				return nil, errs.Validationf("offset (%d) exceeds line count (%d)", offset, len(allLines)).
					WithContext("agent_id", t.agent.GetID()).
					WithContext("offset", offset).
					WithContext("line_count", len(allLines))
			}

			// Calculate end line (exclusive)
			endLine := min(offset+limit, len(allLines))

			// Extract the requested lines (convert to 0-based index)
			selectedLines := allLines[offset-1 : endLine]

			// Join with newlines and add trailing newline if original had it
			processedContent := strings.Join(selectedLines, "\n")
			if len(content) > 0 && content[len(content)-1] == '\n' {
				processedContent += "\n"
			}

			// Get/update file stats
			stats, err := t.fsm.GetFileStats(path)
			if err != nil {
				t.logService.WarnWithContext("Failed to get file stats",
					t.agent.ToLoggingContext(),
					zap.String("file_path", path),
					zap.Error(err),
				)
				// Continue anyway, just don't include stats in response
				stats = nil
			}

			// Build response
			response := map[string]any{
				"success":   true,
				"content":   processedContent,
				"offset":    offset,
				"limit":     limit,
				"locked_by": token.AgentID,
			}

			// Add stats if available
			if stats != nil {
				response["checksum"] = stats.Checksum
				response["size"] = stats.Size
				response["modified"] = stats.ModifiedTime
				response["last_scanned"] = stats.LastScanned

				t.logService.InfoWithContext("File read successfully",
					t.agent.ToLoggingContext(),
					zap.String("file_path", path),
					zap.Int64("size", stats.Size),
					zap.String("checksum", stats.Checksum),
					zap.Int("lines_returned", len(selectedLines)),
					zap.Int("total_lines", len(allLines)),
				)
			} else {
				t.logService.InfoWithContext("File read successfully (no stats)",
					t.agent.ToLoggingContext(),
					zap.String("file_path", path),
					zap.Int("lines_returned", len(selectedLines)),
					zap.Int("total_lines", len(allLines)),
				)
			}

			return response, nil
		})
	if err != nil {
		t.logService.ErrorWithContext("File read operation failed",
			t.agent.ToLoggingContext(),
			zap.String("file_path", path),
			zap.Error(err),
		)
		return map[string]any{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	return result.(map[string]any), nil
}

// ReadFileLines reads a file and returns lines (helper for common use case)
func (t *readFileToolImpl) ReadFileLines(ctx context.Context, path string) ([]string, error) {
	t.logService.DebugWithContext("ReadFileLines helper called",
		t.agent.ToLoggingContext(),
		zap.String("file_path", path),
	)

	result, err := t.Run(ctx, map[string]any{
		"path": path,
	})
	if err != nil {
		t.logService.ErrorWithContext("ReadFileLines helper failed",
			t.agent.ToLoggingContext(),
			zap.String("file_path", path),
			zap.Error(err),
		)
		return nil, err
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.logService.WarnWithContext("ReadFileLines helper returned unsuccessful",
			t.agent.ToLoggingContext(),
			zap.String("file_path", path),
			zap.Any("error", result["error"]),
		)
		return nil, errs.Internalf("failed to read file: %v", result["error"]).
			WithContext("agent_id", t.agent.GetID()).
			WithContext("file_path", path)
	}

	content, ok := result["content"].(string)
	if !ok {
		t.logService.ErrorWithContext("ReadFileLines helper: invalid content type in response",
			t.agent.ToLoggingContext(),
			zap.String("file_path", path),
			zap.String("content_type", fmt.Sprintf("%T", result["content"])),
		)
		return nil, errs.Internal("invalid content type in response").
			WithContext("agent_id", t.agent.GetID()).
			WithContext("file_path", path).
			WithContext("content_type", fmt.Sprintf("%T", result["content"]))
	}

	lines := strings.Split(content, "\n")
	t.logService.DebugWithContext("ReadFileLines helper completed",
		t.agent.ToLoggingContext(),
		zap.String("file_path", path),
		zap.Int("lines_count", len(lines)),
	)
	return lines, nil
}
