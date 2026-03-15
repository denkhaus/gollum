// Package registry provides MCP client registry service.
package registry

import (
	"context"
	"fmt"
	"os"
	"strings"

	appconfig "github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	mcpconfig "github.com/denkhaus/gollum/pkg/mcp/config"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mcp"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// MCPRegistry defines the MCP client registry service
type MCPRegistry interface {
	// GetToolSets returns all ToolSets for gollem.WithToolSets()
	GetToolSets() []gollem.ToolSet

	// Close shuts down all active MCP clients
	Close() error
}

// mcpRegistryImpl is the private implementation
type mcpRegistryImpl struct {
	clients   map[string]*mcp.Client
	tools     []gollem.ToolSet
	loader    mcpconfig.ConfigLoader
	appConfig appconfig.ConfigService
	logger    logger.LoggerService
}

// NewMCPRegistry is the DI constructor
func NewMCPRegistry(injector do.Injector) (MCPRegistry, error) {
	loader := do.MustInvoke[mcpconfig.ConfigLoader](injector)
	appConfig := do.MustInvoke[appconfig.ConfigService](injector)
	log := do.MustInvoke[logger.LoggerService](injector)

	p := &mcpRegistryImpl{
		clients:   make(map[string]*mcp.Client),
		tools:     make([]gollem.ToolSet, 0),
		loader:    loader,
		appConfig: appConfig,
		logger:    log,
	}

	// Use context with timeout for MCP client initialization
	mcpConfig := appConfig.GetMCPConfig()
	initTimeout := mcpConfig.GetClientInitTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()

	if err := p.initializeClients(ctx); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *mcpRegistryImpl) initializeClients(ctx context.Context) error {
	allConfigs, err := p.loader.LoadAll()
	if err != nil {
		return err
	}

	// Count enabled/disabled servers
	enabledCount := 0
	for _, cfg := range allConfigs {
		if cfg.Enabled {
			enabledCount++
		}
	}

	mcpConfig := p.appConfig.GetMCPConfig()
	p.logger.Info("MCP client initialization starting",
		zap.Int("total_servers", len(allConfigs)),
		zap.Int("enabled", enabledCount),
		zap.Int("disabled", len(allConfigs)-enabledCount),
		zap.Duration("init_timeout", mcpConfig.GetClientInitTimeout()),
	)

	// Filter to enabled servers only
	configs := make(map[string]mcpconfig.MCPServerConfig)
	for name, cfg := range allConfigs {
		if cfg.Enabled {
			configs[name] = cfg
		}
	}

	for name, cfg := range configs {
		// Log config details (with secrets masked)
		p.logger.Debug("Attempting to create MCP client",
			zap.String("mcp_server", name),
			zap.String("type", cfg.Type),
			zap.String("command", cfg.Command),
			zap.Int("args_count", len(cfg.Args)),
			zap.Int("env_count", len(cfg.Env)),
		)

		// Log env vars (with values masked for security)
		for k, v := range cfg.Env {
			maskedValue := "***" // Mask all values for security
			if len(v) == 0 {
				maskedValue = "(empty)"
			}
			p.logger.Debug("MCP server env var",
				zap.String("mcp_server", name),
				zap.String("key", k),
				zap.String("value", maskedValue),
			)
		}

		client, err := p.createClient(ctx, name, cfg)
		if err != nil {
			// Log warning with structured context for debugging
			p.logger.Warn("failed to create MCP client",
				zap.String("mcp_server", name),
				zap.String("type", cfg.Type),
				zap.String("command", cfg.Command),
				zap.Error(err),
			)
			continue
		}

		p.clients[name] = client
		p.tools = append(p.tools, client) // mcp.Client implements gollem.ToolSet
	}

	p.logger.Info("MCP client initialization completed",
		zap.Int("enabled_attempted", len(configs)),
		zap.Int("success_count", len(p.clients)),
		zap.Int("failed_count", len(configs)-len(p.clients)),
	)

	return nil
}

func (p *mcpRegistryImpl) createClient(ctx context.Context, name string, cfg mcpconfig.MCPServerConfig) (*mcp.Client, error) {
	switch cfg.Type {
	case "stdio", "":
		envVars := p.buildEnvVars(cfg)
		return mcp.NewStdio(ctx, cfg.Command, cfg.Args,
			mcp.WithEnvVars(envVars),
			mcp.WithStdioClientInfo("gollum", "1.0.0"),
		)
	case "sse":
		// SSE (Server-Sent Events) client with headers support
		headers := p.buildHeaders(cfg.Headers)
		return mcp.NewSSE(ctx, cfg.URL,
			mcp.WithSSEClientInfo("gollum", "1.0.0"),
			mcp.WithSSEHeaders(headers),
		)
	case "http":
		// HTTP client (for streaming responses)
		headers := p.buildHeaders(cfg.Headers)
		return mcp.NewStreamableHTTP(ctx, cfg.URL,
			mcp.WithStreamableHTTPClientInfo("gollum", "1.0.0"),
			mcp.WithStreamableHTTPHeaders(headers),
		)
	default:
		return nil, fmt.Errorf("unsupported MCP type: %s (supported: stdio, sse, http)", cfg.Type)
	}
}

func (p *mcpRegistryImpl) buildEnvVars(cfg mcpconfig.MCPServerConfig) []string {
	// Get allowed system env vars from config or use default
	allowedVars := p.getAllowedSystemEnv(cfg)

	result := make([]string, 0, len(os.Environ())+len(cfg.Env))
	systemEnvs := os.Environ()

	// Copy only allowed system environment variables
	for _, e := range systemEnvs {
		// Split on first '=' to get key
		idx := strings.Index(e, "=")
		if idx <= 0 {
			continue
		}
		key := e[:idx]
		if allowedVars[key] {
			result = append(result, e)
		}
	}

	// Add custom env vars from config (these are explicitly configured)
	for k, v := range cfg.Env {
		envVar := fmt.Sprintf("%s=%s", k, v)
		result = append(result, envVar)
	}

	return result
}

// getAllowedSystemEnv returns the allowed system env vars as a map for quick lookup
func (p *mcpRegistryImpl) getAllowedSystemEnv(cfg mcpconfig.MCPServerConfig) map[string]bool {
	// If server-specific whitelist is provided, use it
	if len(cfg.AllowedSystemEnv) > 0 {
		allowed := make(map[string]bool, len(cfg.AllowedSystemEnv))
		for _, v := range cfg.AllowedSystemEnv {
			allowed[v] = true
		}
		return allowed
	}

	// Otherwise use default from config loader
	defaultAllowed := p.loader.GetDefaultAllowedSystemEnv()
	allowed := make(map[string]bool, len(defaultAllowed))
	for _, v := range defaultAllowed {
		allowed[v] = true
	}
	return allowed
}

func (p *mcpRegistryImpl) buildHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return make(map[string]string)
	}
	return headers
}

func (p *mcpRegistryImpl) GetToolSets() []gollem.ToolSet {
	return p.tools
}

func (p *mcpRegistryImpl) Close() error {
	var firstErr error
	for _, client := range p.clients {
		if err := client.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
