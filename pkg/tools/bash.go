package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/config"
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
	// BashTool executes bash commands
	BashTool struct {
		logService  logger.LoggerService
		agentID     uuid.UUID
		bashCfg     *config.BashConfig
		fileState   state.FileStateManager
		hookManager hooks.HookManager
	}

	// BashToolProvider creates BashTool instances via DI
	BashToolProvider interface {
		CreateTool(agentID uuid.UUID) *BashTool
	}

	bashToolProvider struct {
		logService  logger.LoggerService
		bashCfg     *config.BashConfig
		fileState   state.FileStateManager
		hookManager hooks.HookManager
	}
)

// NewBashToolProvider creates a provider for Bash tools
func NewBashToolProvider(injector do.Injector) (BashToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	cfgService := do.MustInvoke[config.ConfigService](injector)
	fileState := do.MustInvoke[state.FileStateManager](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	bashCfg := cfgService.GetBashConfig()
	return &bashToolProvider{
		logService:  logService,
		bashCfg:     bashCfg,
		fileState:   fileState,
		hookManager: hookManager,
	}, nil
}

// CreateBashTool creates a new BashTool with agent ID
func (p *bashToolProvider) CreateTool(agentID uuid.UUID) *BashTool {
	return &BashTool{
		logService:  p.logService,
		agentID:     agentID,
		bashCfg:     p.bashCfg,
		fileState:   p.fileState,
		hookManager: p.hookManager,
	}
}

// Spec returns the tool specification for the Bash tool
func (t *BashTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameBash,
		Description: "Executes bash commands and returns the output. Useful for running shell commands, scripts, and system operations.",
		Parameters: map[string]*gollem.Parameter{
			"command": {
				Type:        gollem.TypeString,
				Description: "The bash command to execute (e.g., 'ls -la', 'git status', 'echo hello')",
			},
			"timeout": {
				Type:        gollem.TypeNumber,
				Description: "Optional timeout in seconds (default: 30, max: 120)",
			},
		},
		Required: []string{"command"},
	}
}

// Run executes the Bash tool to run shell commands
func (t *BashTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameBash, args,
		func() (map[string]any, error) {
			return t.runBashCommand(ctx, args)
		})
}

// runBashCommand implements the core bash command logic
func (t *BashTool) runBashCommand(ctx context.Context, args map[string]any) (map[string]any, error) {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return map[string]any{
			"success": false,
			"error":   "command is required and must be a non-empty string",
		}, nil
	}

	// Snapshot file state BEFORE command execution (if tracking enabled)
	var beforeStats map[string]*state.FileStats
	if t.bashCfg.TrackChanges {
		beforeStats = t.fileState.GetAllStats()
	}

	// Get timeout, default to 30 seconds, max 120
	timeoutSeconds := 30.0
	if timeout, exists := args["timeout"].(float64); exists && timeout > 0 {
		timeoutSeconds = timeout
		if timeoutSeconds > 120 {
			timeoutSeconds = 120
		}
	}

	// Create context with timeout
	timeout := time.Duration(timeoutSeconds) * time.Second
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	t.logService.Infof("Executing bash command: %s", command)

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
		"stdout":   stdoutStr,
		"stderr":   stderrStr,
		"duration": fmt.Sprintf("%.2fs", duration.Seconds()),
	}

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			t.logService.Warnf("Command timed out after %.2fs: %s", timeout.Seconds(), command)
			result["success"] = false
			result["error"] = fmt.Sprintf("command timed out after %.2fs", timeout.Seconds())
			result["exit_code"] = -1
		} else {
			t.logService.Errorf("Command failed: %s - %v", command, err)
			result["success"] = false
			result["error"] = err.Error()
			result["exit_code"] = cmd.ProcessState.ExitCode()
		}
		return result, nil
	}

	t.logService.Infof("Command succeeded in %.2fs: %s", duration.Seconds(), command)
	result["success"] = true
	result["exit_code"] = 0

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
			t.logService.Warn("Failed to detect file changes", zap.Error(detectErr))
			result["warning"] = fmt.Sprintf("Failed to detect file changes: %v", detectErr)
		} else if len(changes) > 0 {
			// Log detected changes
			t.logService.Info("Bash command modified files",
				zap.Int("count", len(changes)),
				zap.String("command", command))

			// Add changes to result (FileChange now has json tags for proper serialization)
			result["file_changes"] = changes

			// Add warning if files were modified
			result["warning"] = fmt.Sprintf("This bash command modified %d file(s). Consider using %s or %s for better file state tracking.",
				len(changes), shared.ToolNameWriteFile, shared.ToolNameEdit)
		}
	}

	return result, nil
}
