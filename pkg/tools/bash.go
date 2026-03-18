package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/diff"
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
	// bashToolImpl executes bash commands
	bashToolImpl struct {
		logService   logger.LoggerService
		agentID      uuid.UUID
		bashCfg      *config.BashConfig
		fileState    state.FileStateManager
		hookManager  hooks.HookManager
		diffProvider diff.Provider
	}

	// BashToolProvider is an alias for shared.BashToolProvider for convenience

	// BashToolProvider creates BashTool instances via DI
	BashToolProvider interface {
		CreateTool(agentID uuid.UUID) gollem.Tool
	}

	bashToolProvider struct {
		logService   logger.LoggerService
		bashCfg      *config.BashConfig
		fileState    state.FileStateManager
		hookManager  hooks.HookManager
		diffProvider diff.Provider
	}
)

// NewBashToolProvider creates a provider for Bash tools
func NewBashToolProvider(injector do.Injector) (BashToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	cfgService := do.MustInvoke[config.ConfigService](injector)
	fileState := do.MustInvoke[state.FileStateManager](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	diffProvider := do.MustInvoke[diff.Provider](injector)
	bashCfg := cfgService.GetBashConfig()
	return &bashToolProvider{
		logService:   logService,
		bashCfg:      bashCfg,
		fileState:    fileState,
		hookManager:  hookManager,
		diffProvider: diffProvider,
	}, nil
}

// CreateBashTool creates a new BashTool with agent ID
func (p *bashToolProvider) CreateTool(agentID uuid.UUID) gollem.Tool {
	return &bashToolImpl{
		logService:   p.logService,
		agentID:      agentID,
		bashCfg:      p.bashCfg,
		fileState:    p.fileState,
		hookManager:  p.hookManager,
		diffProvider: p.diffProvider,
	}
}

// Spec returns the tool specification for the Bash tool
func (t *bashToolImpl) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameBash.String(),
		Description: "Executes bash commands and returns the output. Useful for running shell commands, scripts, and system operations.",
		Parameters: map[string]*gollem.Parameter{
			"command": {
				Type:        gollem.TypeString,
				Description: "The bash command to execute (e.g., 'ls -la', 'git status', 'echo hello')",
			},
			"timeout": {
				Type:        gollem.TypeNumber,
				Description: "Optional timeout in seconds (default: 60, max: 300 = 5 minutes)",
			},
		},
	}
}

// Run executes the Bash tool to run shell commands
func (t *bashToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameBash, args,
		func() (map[string]any, error) {
			return t.runBashCommand(ctx, args)
		})
}

// runBashCommand implements the core bash command logic
func (t *bashToolImpl) runBashCommand(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	command, errResp := args.MustGetString(shared.ParamCommand)
	if errResp != nil {
		return errResp, nil
	}

	// Snapshot file state BEFORE command execution (if tracking enabled)
	var beforeStats map[string]*state.FileStats
	if t.bashCfg.TrackChanges {
		beforeStats = t.fileState.GetAllStats()
	}

	// Get timeout, default to 60 seconds, max 300 (5 minutes)
	timeoutSeconds := max(1.0, min(args.GetFloat(shared.ParamTimeout, 60.0), 300.0))

	// Create context with timeout
	timeout := time.Duration(timeoutSeconds) * time.Second
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	t.logService.InfoWithAgent("Executing bash command", t.agentID, zap.String("command", command))

	cmd := exec.CommandContext(cmdCtx, "bash", "-c", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	stdoutStr := strings.TrimSpace(stdout.String())
	stderrStr := strings.TrimSpace(stderr.String())

	result := map[string]any{
		string(shared.KeyStdout):   stdoutStr,
		string(shared.KeyStderr):   stderrStr,
		string(shared.KeyDuration): fmt.Sprintf("%.2fs", duration.Seconds()),
	}

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			t.logService.WarnWithAgent("Command timed out", t.agentID,
				zap.Float64("timeout_seconds", timeout.Seconds()),
				zap.String("command", command))
			result[string(shared.KeySuccess)] = false
			result[string(shared.KeyError)] = fmt.Sprintf("command timed out after %.2fs", timeout.Seconds())
			result[string(shared.KeyExitCode)] = -1
		} else {
			t.logService.ErrorWithAgent("Command failed", t.agentID,
				zap.String("command", command),
				zap.Error(err))
			result[string(shared.KeySuccess)] = false
			result[string(shared.KeyError)] = err.Error()
			result[string(shared.KeyExitCode)] = cmd.ProcessState.ExitCode()
		}
		return result, nil
	}

	t.logService.InfoWithAgent("Command succeeded", t.agentID,
		zap.Float64("duration_seconds", duration.Seconds()),
		zap.String("command", command))
	result[string(shared.KeySuccess)] = true
	result[string(shared.KeyExitCode)] = 0

	// Detect file changes AFTER command execution (if tracking enabled)
	if t.bashCfg.TrackChanges {
		// Wait for file watcher to process new file creation events.
		// This is only needed for detecting newly created files, since modifications
		// and deletions are now actively verified by DetectChanges() itself.
		watcherDebounce := t.fileState.GetWatcherDebounce()
		time.Sleep(watcherDebounce)

		// Detect changes
		changes, detectErr := t.fileState.DetectChanges(beforeStats)
		if detectErr != nil {
			t.logService.WarnWithAgent("Failed to detect file changes", t.agentID, zap.Error(detectErr))
			result[string(shared.KeyWarning)] = fmt.Sprintf("Failed to detect file changes: %v", detectErr)
		} else if len(changes) > 0 {
			// Log detected changes
			t.logService.InfoWithAgent("Bash command modified files", t.agentID,
				zap.Int("count", len(changes)),
				zap.String("command", command))

			// Generate diffs for modified files
			fileDiffs := make(map[string]interface{})
			for _, change := range changes {
				if change.Operation == state.Modified || change.Operation == state.Created {
					// Get the current content of the file
					currentContent, err := os.ReadFile(change.Path)
					if err != nil {
						t.logService.WarnWithAgent("Failed to get current content for diff", t.agentID,
							zap.String("path", change.Path),
							zap.Error(err))
						continue
					}

					// Generate diff based on change type
					var diffStr string
					if change.Operation == state.Created {
						// For new files, generate diff against empty content
						diffStr, err = t.diffProvider.GenerateDiffForNewFile(change.Path, string(currentContent))
					} else {
						// For modified files, we don't have the old content readily available
						// For now, we'll generate a diff against an empty string to show the new content
						// In a future enhancement, we could store the old content before command execution
						diffStr, err = t.diffProvider.GenerateDiff(change.Path, change.Path, "", string(currentContent))
					}

					if err != nil {
						t.logService.WarnWithAgent("Failed to generate diff", t.agentID,
							zap.String("path", change.Path),
							zap.Error(err))
						continue
					}

					// Format the diff for display
					formattedDiff := t.diffProvider.FormatForDisplay(diffStr)
					compactDiff := t.diffProvider.FormatCompact(diffStr)

					// Add diff information to the result
					fileDiffs[change.Path] = map[string]interface{}{
						string(shared.KeyDiff):        formattedDiff,
						string(shared.KeyDiffCompact): compactDiff,
						string(shared.KeyIsNewFile):   change.Operation == state.Created,
					}
				}
			}

			// Add changes to result (FileChange now has json tags for proper serialization)
			result[string(shared.KeyFileChanges)] = changes

			// Add diffs if any were generated
			if len(fileDiffs) > 0 {
				result[string(shared.KeyFileDiffs)] = fileDiffs
			}

			// Add warning if files were modified
			result[string(shared.KeyWarning)] = fmt.Sprintf("This bash command modified %d file(s). Consider using %s or %s for better file state tracking.",
				len(changes), shared.ToolNameWriteFile, shared.ToolNameEdit)
		}
	}

	return result, nil
}
