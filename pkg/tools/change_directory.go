package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/denkhaus/gollum/pkg/events"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

type (
	// changeDirectoryToolImpl changes the current working directory and publishes an event
	// Services subscribe to the event to update their state (workspace, skills, etc.)
	changeDirectoryToolImpl struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		eventBus    events.Bus
		agent       shared.Agent
	}

	// ChangeDirectoryToolProvider creates ChangeDirectoryTool instances via DI
	ChangeDirectoryToolProvider interface {
		CreateTool(agent shared.Agent) gollem.Tool
	}

	changeDirectoryToolProvider struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		eventBus    events.Bus
	}
)

// NewChangeDirectoryToolProvider creates a provider for ChangeDirectory tools
func NewChangeDirectoryToolProvider(injector do.Injector) (ChangeDirectoryToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	bus := do.MustInvoke[events.Bus](injector)

	return &changeDirectoryToolProvider{
		logService:  logService,
		hookManager: hookManager,
		eventBus:    bus,
	}, nil
}

// CreateTool creates a new ChangeDirectoryTool with agent
func (p *changeDirectoryToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &changeDirectoryToolImpl{
		logService:  p.logService,
		hookManager: p.hookManager,
		eventBus:    p.eventBus,
		agent:       agent,
	}
}

// Run executes the ChangeDirectory tool
func (t *changeDirectoryToolImpl) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToLoggingContext(), shared.ToolNameChangeDirectory, args,
		func() (map[string]any, error) {
			return t.runChangeDirectory(ctx, args)
		})
}

// runChangeDirectory implements the core ChangeDirectory logic
// It only validates, changes the actual directory, and publishes an event.
// Services (WorkspaceService, SkillService) subscribe to the event to update their state.
func (t *changeDirectoryToolImpl) runChangeDirectory(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	// Get path from args
	path, errResp := args.GetFilePath(shared.ParamPath)
	if errResp != nil {
		return nil, fmt.Errorf("path argument is required and must be a non-empty string")
	}

	// Resolve to absolute path
	absPath, err := resolvePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Validate directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory does not exist: %s", absPath)
		}
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absPath)
	}

	// Get previous working directory
	previousPath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Skip if already in the target directory
	if previousPath == absPath {
		return map[string]any{
			"success":       true,
			"previous_path": previousPath,
			"current_path":  absPath,
			"message":       "Already in target directory",
		}, nil
	}

	// Actually change the working directory
	if err := os.Chdir(absPath); err != nil {
		return nil, fmt.Errorf("failed to change directory: %w", err)
	}

	t.logService.InfoWithContext("Changed directory",
		t.agent.ToLoggingContext(),
		zap.String("previous_path", previousPath),
		zap.String("new_path", absPath))

	// Publish directory changed event - services will react to this
	if err := events.PublishTyped(t.eventBus, ctx,
		events.EventDirectoryChanged,
		shared.ToolNameChangeDirectory.String(),
		events.DirectoryChangedPayload{
			OldPath: previousPath,
			NewPath: absPath,
		},
	); err != nil {
		t.logService.Warnf("Failed to publish directory changed event: %v", err)
		// Don't fail the operation - event publishing is non-critical
	}

	// Build result
	result := map[string]any{
		"success":       true,
		"previous_path": previousPath,
		"current_path":  absPath,
	}

	return result, nil
}

// resolvePath converts a path to absolute path, handling relative paths
func resolvePath(path string) (string, error) {
	// Check if path is already absolute
	if os.IsPathSeparator(path[0]) {
		return path, nil
	}

	// Convert relative path to absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}

// Spec returns the tool specification for the ChangeDirectory tool
func (t *changeDirectoryToolImpl) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameChangeDirectory.String(),
		Description: "Changes the current working directory, updates workspace configuration, and triggers automatic skill discovery in the new directory. Use this tool to switch between different project workspaces.",
		Parameters: map[string]*gollem.Parameter{
			"path": {
				Type:        gollem.TypeString,
				Description: "The directory path to switch to. Can be absolute (e.g., '/home/user/projects') or relative (e.g., '../other-project'). The directory must exist.",
			},
		},
	}
}
