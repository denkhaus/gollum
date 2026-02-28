package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

type (
	// ChangeDirectoryTool changes the current working directory and triggers skill discovery
	ChangeDirectoryTool struct {
		logService   logger.LoggerService
		hookManager  hooks.HookManager
		configService config.ConfigService
		skillService skills.SkillService
		agentID      uuid.UUID
	}

	// ChangeDirectoryToolProvider creates ChangeDirectoryTool instances via DI
	ChangeDirectoryToolProvider interface {
		CreateTool(agentID uuid.UUID) *ChangeDirectoryTool
	}

	changeDirectoryToolProvider struct {
		logService   logger.LoggerService
		hookManager  hooks.HookManager
		configService config.ConfigService
		skillService skills.SkillService
	}
)

// NewChangeDirectoryToolProvider creates a provider for ChangeDirectory tools
func NewChangeDirectoryToolProvider(injector do.Injector) (ChangeDirectoryToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	cfgService := do.MustInvoke[config.ConfigService](injector)
	skillSvc := do.MustInvoke[skills.SkillService](injector)

	return &changeDirectoryToolProvider{
		logService:   logService,
		hookManager:  hookManager,
		configService: cfgService,
		skillService: skillSvc,
	}, nil
}

// CreateTool creates a new ChangeDirectoryTool with agent ID
func (p *changeDirectoryToolProvider) CreateTool(agentID uuid.UUID) *ChangeDirectoryTool {
	return &ChangeDirectoryTool{
		logService:   p.logService,
		hookManager:  p.hookManager,
		configService: p.configService,
		skillService: p.skillService,
		agentID:      agentID,
	}
}

// Run executes the ChangeDirectory tool
func (t *ChangeDirectoryTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameChangeDirectory, args,
		func() (map[string]any, error) {
			return t.runChangeDirectory(ctx, args)
		})
}

// runChangeDirectory implements the core ChangeDirectory logic
func (t *ChangeDirectoryTool) runChangeDirectory(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Get path from args
	path, ok := args["path"].(string)
	if !ok || path == "" {
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

	// Get previous workspace
	previousWorkspace := t.configService.GetCurrentWorkspace()

	// Update workspace configuration
	if err := t.configService.SetCurrentWorkspace(absPath); err != nil {
		return nil, fmt.Errorf("failed to set workspace: %w", err)
	}

	// Add new path to skill service search paths and trigger discovery
	if err := t.skillService.AddSearchPathAndDiscover(ctx, absPath); err != nil {
		t.logService.Warnf("Failed to discover skills in new workspace: %v", err)
		// Don't fail the operation, just log the warning
	}

	// Actually change the working directory
	if err := os.Chdir(absPath); err != nil {
		t.logService.Warnf("Failed to change working directory: %v", err)
		// Don't fail the operation, workspace config is updated
	}

	t.logService.Infof("Changed workspace from %s to %s", previousWorkspace, absPath)

	// Build result
	result := map[string]any{
		"success":           true,
		"previous_path":     previousWorkspace,
		"current_path":      absPath,
		"workspace_history": t.configService.GetWorkspaceHistory(),
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
func (t *ChangeDirectoryTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameChangeDirectory,
		Description: "Changes the current working directory, updates workspace configuration, and triggers automatic skill discovery in the new directory. Use this tool to switch between different project workspaces.",
		Parameters: map[string]*gollem.Parameter{
			"path": {
				Type:        gollem.TypeString,
				Description: "The directory path to switch to. Can be absolute (e.g., '/home/user/projects') or relative (e.g., '../other-project'). The directory must exist.",
			},
		},
	}
}
