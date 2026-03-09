package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/samber/do/v2"
)

// ConfigLoader defines the config loading service
type ConfigLoader interface {
	Load() (map[string]MCPServerConfig, error)
}

// configLoaderImpl is the private implementation
type configLoaderImpl struct {
	projectPath string
	globalPath  string
}

// mcpConfigFile represents the root structure of mcp.json
type mcpConfigFile struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// NewConfigLoader is the DI constructor
func NewConfigLoader(injector do.Injector) ConfigLoader {
	_ = injector // Unused but required for DI signature
	homeDir, _ := os.UserHomeDir()
	return &configLoaderImpl{
		projectPath: ".gollum/mcp.json",
		globalPath:  filepath.Join(homeDir, ".config/gollum/mcp.json"),
	}
}

// Load loads and merges mcp.json from project and global locations
func (p *configLoaderImpl) Load() (map[string]MCPServerConfig, error) {
	servers := make(map[string]MCPServerConfig)

	// Load global config first (if exists)
	if p.globalPath != "" {
		if global, err := p.loadFile(p.globalPath); err == nil && global != nil {
			mergeInto(servers, global)
		} else if err != nil {
			return nil, err // Propagate non-"not found" errors
		}
	}

	// Load project config (if exists) - overrides global
	if p.projectPath != "" {
		if project, err := p.loadFile(p.projectPath); err == nil && project != nil {
			mergeInto(servers, project)
		} else if err != nil {
			return nil, err // Propagate non-"not found" errors
		}
	}

	// Interpolate env vars and shell commands in all configs
	for name, cfg := range servers {
		servers[name] = interpolateConfig(cfg)
	}

	// Filter to enabled servers only
	return filterEnabled(servers), nil
}

func (p *configLoaderImpl) loadFile(path string) (map[string]MCPServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Missing file is not an error - return nil to signal no data
			return nil, nil
		}
		return nil, err
	}

	var cfg mcpConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}

	return cfg.MCPServers, nil
}

func mergeInto(target, source map[string]MCPServerConfig) {
	for k, v := range source {
		target[k] = v
	}
}

func filterEnabled(servers map[string]MCPServerConfig) map[string]MCPServerConfig {
	result := make(map[string]MCPServerConfig)
	for k, v := range servers {
		if v.Enabled {
			result[k] = v
		}
	}
	return result
}
