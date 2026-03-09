// Package registry provides MCP client registry service.
package registry

import (
	"github.com/denkhaus/gollum/pkg/mcp/config"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mcp"
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
func NewMCPRegistry(injector interface{}) (MCPRegistry, error) {
	return &mcpRegistryImpl{
		clients: make(map[string]*mcp.Client),
		tools:   make([]gollem.ToolSet, 0),
	}, nil
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
