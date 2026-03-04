package app

import (
	"os"
	"path/filepath"
)

// ensureGollumDirectory creates .gollum directory if it doesn't exist
func (p *applicationServiceImpl) ensureGollumDirectory() error {
	p.gollumDir = filepath.Join(p.workspaceService.GetCurrentWorkspace(), gollumDirName)
	if err := os.MkdirAll(p.gollumDir, 0755); err != nil {
		return err
	}
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
	return os.WriteFile(gitignorePath, []byte("/logs\n"), 0644)
}
