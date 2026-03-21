// Package registry provides MCP client registry service.
package registry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

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

	// GetToolNames returns all tool names in "server_name/tool_name" format
	GetToolNames() []string

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
	sem       chan struct{} // semaphore for limiting concurrent connections
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
		sem:       make(chan struct{}, 5), // max 5 parallel MCP connections
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
	p.logger.Info("MCP client initialization starting (parallel)",
		zap.Int("total_servers", len(allConfigs)),
		zap.Int("enabled", enabledCount),
		zap.Int("disabled", len(allConfigs)-enabledCount),
		zap.Duration("init_timeout", mcpConfig.GetClientInitTimeout()),
		zap.Int("max_parallel", cap(p.sem)),
	)

	// Filter to enabled servers only
	configs := make(map[string]mcpconfig.MCPServerConfig)
	for name, cfg := range allConfigs {
		if cfg.Enabled {
			configs[name] = cfg
		}
	}

	// Semaphore-bounded parallel initialization
	var wg sync.WaitGroup
	var mu sync.Mutex // Protects p.clients, p.tools
	errors := make([]error, 0)

	for name, cfg := range configs {
		wg.Add(1)
		go func(name string, cfg mcpconfig.MCPServerConfig) {
			defer wg.Done()

			p.sem <- struct{}{} // Acquire semaphore
			defer func() { <-p.sem }() // Release semaphore

			p.logger.Debug("Attempting to create MCP client (parallel)",
				zap.String("mcp_server", name),
				zap.String("type", cfg.Type),
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
				p.logger.Warn("failed to create MCP client",
					zap.String("mcp_server", name),
					zap.String("type", cfg.Type),
					zap.String("command", cfg.Command),
					zap.Error(err),
				)
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", name, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			p.clients[name] = client
			p.tools = append(p.tools, client) // mcp.Client implements gollem.ToolSet
			mu.Unlock()

			p.logger.Debug("MCP client created successfully",
				zap.String("mcp_server", name))
		}(name, cfg)
	}

	wg.Wait()

	// Log summary
	successCount := len(p.clients)
	failedCount := len(errors)

	p.logger.Info("MCP client initialization completed",
		zap.Int("enabled_attempted", len(configs)),
		zap.Int("success_count", successCount),
		zap.Int("failed_count", failedCount),
	)

	// Log all errors (but don't fail initialization)
	for _, err := range errors {
		p.logger.Warn("MCP client initialization error", zap.Error(err))
	}

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

// GetToolNames returns all tool names in "server_name/tool_name" format
func (p *mcpRegistryImpl) GetToolNames() []string {
	toolNames := make([]string, 0)

	for serverName, client := range p.clients {
		specs, err := client.Specs(context.Background())
		if err != nil {
			continue
		}

		for _, spec := range specs {
			toolNames = append(toolNames, fmt.Sprintf("%s/%s", serverName, spec.Name))
		}
	}

	return toolNames
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
