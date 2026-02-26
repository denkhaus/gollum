package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// WorkspaceConfig manages workspace-aware configuration for tracking current
// working directory and enabling automatic skill discovery in workspace directories.
type WorkspaceConfig struct {
	// CurrentWorkspace is the active workspace directory path
	CurrentWorkspace string `envconfig:"CURRENT_WORKSPACE" default:""`
	// WorkspaceHistory maintains a list of recently used workspace paths
	WorkspaceHistory []string `envconfig:"WORKSPACE_HISTORY" default:""`
	// MaxWorkspaceHistory limits the number of workspaces kept in history
	MaxWorkspaceHistory int `envconfig:"MAX_WORKSPACE_HISTORY" default:"5"`
}

// GetCurrentWorkspace returns the current workspace path.
// If no workspace is set, it returns the current working directory.
func (c *WorkspaceConfig) GetCurrentWorkspace() string {
	if c.CurrentWorkspace == "" {
		if wd, err := os.Getwd(); err == nil {
			return wd
		}
	}
	return c.CurrentWorkspace
}

// SetWorkspace sets the current workspace to the given path.
// It validates that the path exists and converts it to an absolute path.
// The old and new paths are added to the workspace history.
func (c *WorkspaceConfig) SetWorkspace(path string) error {
	// Validate path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("workspace path does not exist: %s", path)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	oldPath := c.CurrentWorkspace
	c.CurrentWorkspace = absPath
	c.addToHistory(oldPath)
	c.addToHistory(absPath)

	return nil
}

// addToHistory adds a path to the workspace history.
// It removes duplicates and maintains the maximum history size.
func (c *WorkspaceConfig) addToHistory(path string) {
	if path == "" {
		return
	}

	// Remove if already exists
	for i, p := range c.WorkspaceHistory {
		if p == path {
			c.WorkspaceHistory = append(c.WorkspaceHistory[:i], c.WorkspaceHistory[i+1:]...)
			break
		}
	}

	// Add to front
	c.WorkspaceHistory = append([]string{path}, c.WorkspaceHistory...)

	// Trim to max size
	if len(c.WorkspaceHistory) > c.MaxWorkspaceHistory {
		c.WorkspaceHistory = c.WorkspaceHistory[:c.MaxWorkspaceHistory]
	}
}

// GetWorkspaceHistory returns the list of recently used workspace paths.
func (c *WorkspaceConfig) GetWorkspaceHistory() []string {
	return c.WorkspaceHistory
}
