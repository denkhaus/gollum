package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	appconfig "github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/samber/do/v2"
)

// ConfigLoader defines the config loading service
type ConfigLoader interface {
	Load() (map[string]MCPServerConfig, error)
	LoadAll() (map[string]MCPServerConfig, error)
	GetDefaultAllowedSystemEnv() []string
}

// configLoaderImpl is the private implementation
type configLoaderImpl struct {
	projectPath             string
	globalPath              string
	appConfig               appconfig.ConfigService
	logger                  logger.LoggerService
	defaultAllowedSystemEnv []string
}

// mcpConfigFile represents the root structure of mcp.json
type mcpConfigFile struct {
	MCPServers              map[string]MCPServerConfig `json:"mcpServers"`
	DefaultAllowedSystemEnv []string                   `json:"defaultAllowedSystemEnv"`
}

// NewConfigLoader is the DI constructor
func NewConfigLoader(injector do.Injector) (ConfigLoader, error) {
	appConfig := do.MustInvoke[appconfig.ConfigService](injector)
	log := do.MustInvoke[logger.LoggerService](injector)
	homeDir, _ := os.UserHomeDir()
	return &configLoaderImpl{
		projectPath: ".gollum/mcp.json",
		globalPath:  filepath.Join(homeDir, ".config/gollum/mcp.json"),
		appConfig:   appConfig,
		logger:      log,
	}, nil
}

// Load loads and merges mcp.json from project and global locations, returning only enabled servers
func (p *configLoaderImpl) Load() (map[string]MCPServerConfig, error) {
	servers, err := p.LoadAll()
	if err != nil {
		return nil, err
	}
	// Filter to enabled servers only
	return filterEnabled(servers), nil
}

// LoadAll loads and merges mcp.json from project and global locations, returning all servers
func (p *configLoaderImpl) LoadAll() (map[string]MCPServerConfig, error) {
	servers := make(map[string]MCPServerConfig)

	// Load global config first (if exists)
	if p.globalPath != "" {
		if global, defaultEnv, err := p.loadFile(p.globalPath); err == nil && global != nil {
			mergeInto(servers, global)
			// Use global default if not yet set
			if p.defaultAllowedSystemEnv == nil && len(defaultEnv) > 0 {
				p.defaultAllowedSystemEnv = defaultEnv
			}
		} else if err != nil {
			return nil, err // Propagate non-"not found" errors
		}
	}

	// Load project config (if exists) - overrides global
	if p.projectPath != "" {
		if project, defaultEnv, err := p.loadFile(p.projectPath); err == nil && project != nil {
			mergeInto(servers, project)
			// Project default overrides global default
			if len(defaultEnv) > 0 {
				p.defaultAllowedSystemEnv = defaultEnv
			}
		} else if err != nil {
			return nil, err // Propagate non-"not found" errors
		}
	}

	// Interpolate env vars and shell commands in all configs
	mcpConfig := p.appConfig.GetMCPConfig()
	timeout := mcpConfig.GetCommandTimeout()
	for name, cfg := range servers {
		servers[name] = interpolateConfigWithTimeout(cfg, timeout, p.logger)
	}

	return servers, nil
}

// GetDefaultAllowedSystemEnv returns the default allowed system environment variables
func (p *configLoaderImpl) GetDefaultAllowedSystemEnv() []string {
	if p.defaultAllowedSystemEnv != nil {
		return p.defaultAllowedSystemEnv
	}
	// Return built-in safe default
	return []string{"PATH", "HOME", "USER", "LOGNAME", "SHELL", "LANG", "LC_ALL", "LC_CTYPE", "TERM", "NODE", "NODE_PATH"}
}

func (p *configLoaderImpl) loadFile(path string) (map[string]MCPServerConfig, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Missing file is not an error - return nil to signal no data
			return nil, nil, nil
		}
		return nil, nil, err
	}

	var cfg mcpConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}

	// Set default enabled=true for servers that don't have explicit enabled field
	for name, server := range cfg.MCPServers {
		// Check if "enabled" key exists in the raw JSON for this server
		var rawCfg map[string]json.RawMessage
		if err := json.Unmarshal(data, &rawCfg); err == nil {
			if serversMsg, ok := rawCfg["mcpServers"]; ok {
				var serversMap map[string]json.RawMessage
				if err := json.Unmarshal(serversMsg, &serversMap); err == nil {
					if serverRaw, ok := serversMap[name]; ok {
						// Check if "enabled" key exists in this server's config
						if !strings.Contains(string(serverRaw), `"enabled"`) {
							// No explicit enabled field, default to true
							server.Enabled = true
							cfg.MCPServers[name] = server
						}
					}
				}
			}
		}
	}

	return cfg.MCPServers, cfg.DefaultAllowedSystemEnv, nil
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
