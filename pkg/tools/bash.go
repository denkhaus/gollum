package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

type (
	// BashTool executes bash commands
	BashTool struct {
		logService logger.LoggerService
		agentID    uuid.UUID
	}

	// BashToolProvider creates BashTool instances via DI
	BashToolProvider interface {
		CreateTool(agentID uuid.UUID) *BashTool
	}

	bashToolProvider struct {
		logService logger.LoggerService
	}
)

// NewBashToolProvider creates a provider for Bash tools
func NewBashToolProvider(injector do.Injector) (BashToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	return &bashToolProvider{logService: logService}, nil
}

// CreateBashTool creates a new BashTool with agent ID
func (p *bashToolProvider) CreateTool(agentID uuid.UUID) *BashTool {
	return &BashTool{
		logService: p.logService,
		agentID:    agentID,
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
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return map[string]any{
			"success": false,
			"error":   "command is required and must be a non-empty string",
		}, nil
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

	return result, nil
}
