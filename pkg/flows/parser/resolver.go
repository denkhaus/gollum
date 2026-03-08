package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
)

// Resolver handles module reference resolution
type Resolver struct {
	// Base paths to search for modules
	ModulePaths []string
}

// NewResolver creates a new resolver with default module paths
func NewResolver(basePaths ...string) *Resolver {
	return &Resolver{
		ModulePaths: basePaths,
	}
}

// DefaultResolver creates a resolver with standard flow paths
func DefaultResolver() *Resolver {
	// Find project root by looking for .gollum directory
	root := findProjectRoot()
	return &Resolver{
		ModulePaths: []string{
			filepath.Join(root, ".gollum", "flows", "modules"),
		},
	}
}

// findProjectRoot attempts to find the project root by looking for .gollum directory
func findProjectRoot() string {
	// Start from current directory and search up
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		gollumPath := filepath.Join(dir, ".gollum")
		if _, err := os.Stat(gollumPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, return current directory
			return "."
		}
		dir = parent
	}
}

// ResolveCall resolves a call reference to an actual flow file path
// Supports:
//   - "module-name" → modules/module-name/main.xml
//   - "module-name/sub-flow" → modules/module-name/sub-flow.xml
//   - "flow-name" → flow-name.xml (current directory if exists, then modules)
//   - "../other-module" → relative path resolution
//   - "/absolute/path" → absolute path
func (r *Resolver) ResolveCall(ref, currentFlowPath string) (string, error) {
	// Absolute path
	if strings.HasPrefix(ref, "/") {
		return ref, nil
	}

	// Parent directory reference
	if strings.HasPrefix(ref, "../") || strings.HasPrefix(ref, "./") {
		// Resolve relative to current flow's directory
		currentDir := filepath.Dir(currentFlowPath)
		resolved := filepath.Join(currentDir, ref)
		if _, err := os.Stat(resolved); err == nil {
			return resolved, nil
		}
		return "", fmt.Errorf("flow not found: %s (resolved to %s)", ref, resolved)
	}

	// Check if ref contains a slash (module/sub-flow format)
	if strings.Contains(ref, "/") {
		parts := strings.Split(ref, "/")
		moduleName := parts[0]
		subFlow := strings.Join(parts[1:], "/")

		// Try to find in module paths
		for _, basePath := range r.ModulePaths {
			modulePath := filepath.Join(basePath, moduleName)
			subFlowPath := filepath.Join(modulePath, subFlow+".xml")

			if _, err := os.Stat(subFlowPath); err == nil {
				return subFlowPath, nil
			}
		}
		return "", fmt.Errorf("sub-flow not found: %s (in module %s)", subFlow, moduleName)
	}

	// Simple reference without slash
	// First check current flow's directory (for same-workflow calls)
	currentDir := filepath.Dir(currentFlowPath)
	localPath := filepath.Join(currentDir, ref+".xml")
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	// Then try as module reference (resolve to main.xml)
	for _, basePath := range r.ModulePaths {
		mainPath := filepath.Join(basePath, ref, "main.xml")

		if _, err := os.Stat(mainPath); err == nil {
			return mainPath, nil
		}
	}

	// Try as direct flow file reference (for backward compatibility)
	if filepath.Ext(ref) == ".xml" {
		if _, err := os.Stat(ref); err == nil {
			return ref, nil
		}
	}

	return "", fmt.Errorf("flow not found: %s (tried: %s.xml in current dir, modules/%s/main.xml)", ref, ref, ref)
}

// ResolveAll resolves all call references in a flow to their actual file paths
func (r *Resolver) ResolveAll(flowPath string, flow *flows.Flow) (map[string]string, error) {
	resolved := make(map[string]string)

	// Collect all call references
	for _, state := range flow.States {
		for _, call := range state.Calls {
			if call.Ref == "" {
				continue
			}

			// Skip if already resolved
			if _, ok := resolved[call.Ref]; ok {
				continue
			}

			resolvedPath, err := r.ResolveCall(call.Ref, flowPath)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve call reference '%s': %w", call.Ref, err)
			}

			resolved[call.Ref] = resolvedPath
		}
	}

	return resolved, nil
}
