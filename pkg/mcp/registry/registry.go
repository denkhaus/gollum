// Package registry provides MCP client registry service.
package registry

import (
	"context"
	"fmt"
	"os"

	"github.com/denkhaus/gollum/pkg/mcp/config"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mcp"
	"github.com/samber/do/v2"
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
	clients map[string]*mcp.Client
	tools   []gollem.ToolSet
	loader  config.ConfigLoader
}

// NewMCPRegistry is the DI constructor
func NewMCPRegistry(injector do.Injector) (MCPRegistry, error) {
	loader := do.MustInvoke[config.ConfigLoader](injector)

	p := &mcpRegistryImpl{
		clients: make(map[string]*mcp.Client),
		tools:   make([]gollem.ToolSet, 0),
		loader:  loader,
	}

	if err := p.initializeClients(context.Background()); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *mcpRegistryImpl) initializeClients(ctx context.Context) error {
	configs, err := p.loader.Load()
	if err != nil {
		return err
	}

	for name, cfg := range configs {
		client, err := p.createClient(ctx, name, cfg)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to create MCP client '%s': %v\n", name, err)
			continue
		}

		p.clients[name] = client
		p.tools = append(p.tools, client) // mcp.Client implements gollem.ToolSet
	}

	return nil
}

func (p *mcpRegistryImpl) createClient(ctx context.Context, name string, cfg config.MCPServerConfig) (*mcp.Client, error) {
	switch cfg.Type {
	case "stdio", "":
		envVars := p.buildEnvVars(cfg.Env)
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
	case "streamable-http":
		// Streamable HTTP client (for streaming responses)
		headers := p.buildHeaders(cfg.Headers)
		return mcp.NewStreamableHTTP(ctx, cfg.URL,
			mcp.WithStreamableHTTPClientInfo("gollum", "1.0.0"),
			mcp.WithStreamableHTTPHeaders(headers),
		)
	default:
		return nil, fmt.Errorf("unsupported MCP type: %s", cfg.Type)
	}
}

func (p *mcpRegistryImpl) buildEnvVars(env map[string]string) []string {
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
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
