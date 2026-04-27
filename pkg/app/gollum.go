package app

import (
	"os"
	"path/filepath"
)

// PrepareWorkspace creates the .gollum directory and performs basic workspace setup.
// Should be called before any services that need the workspace directory.
func (p *applicationServiceImpl) PrepareWorkspace() error {
	return p.ensureGollumDirectory()
}

// ensureGollumDirectory creates .gollum directory if it doesn't exist
func (p *applicationServiceImpl) ensureGollumDirectory() error {
	gollumDir := filepath.Join(p.workspaceService.GetCurrentWorkspace(), gollumDirName)
	if err := os.MkdirAll(gollumDir, 0755); err != nil {
		return err
	}

	p.gollumDir = gollumDir
	return p.ensureGollumGitignore()
}

// ensureGollumGitignore creates a .gitignore file in the .gollum directory
// that ignores the log subfolder
func (p *applicationServiceImpl) ensureGollumGitignore() error {
	gitignorePath := filepath.Join(p.gollumDir, ".gitignore")

	// Check if .gitignore already exists
	if _, err := os.Stat(gitignorePath); err == nil {
		// File exists, no need to overwrite
		return nil
	}

	// Create .gitignore with log directory exclusion
	return os.WriteFile(gitignorePath, []byte("/logs\nmcp.json\n"), 0644)
}
