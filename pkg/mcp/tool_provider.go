package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

// MCPToolProvider creates tools from MCP registry by server/tool name
type MCPToolProvider interface {
	CreateTool(agentID uuid.UUID, serverAndTool string) (gollem.Tool, error)
}

type mcpToolProviderImpl struct {
	registry registry.MCPRegistry
	logger   logger.LoggerService
}

// NewMCPToolProvider creates a new MCPToolProvider
func NewMCPToolProvider(injector do.Injector) (MCPToolProvider, error) {
	reg := do.MustInvoke[registry.MCPRegistry](injector)
	log := do.MustInvoke[logger.LoggerService](injector)

	return &mcpToolProviderImpl{
		registry: reg,
		logger:   log,
	}, nil
}

func (p *mcpToolProviderImpl) CreateTool(agentID uuid.UUID, serverAndTool string) (gollem.Tool, error) {
	parts := strings.SplitN(serverAndTool, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid MCP tool format: %s (expected 'server/tool')", serverAndTool)
	}

	serverName, toolName := parts[0], parts[1]

	// Get tool sets from registry
	toolSets := p.registry.GetToolSets()

	// Search for tool spec in all tool sets
	for _, toolSet := range toolSets {
		specs, err := toolSet.Specs(context.Background())
		if err != nil {
			continue // Skip this tool set on error
		}

		for _, spec := range specs {
			if spec.Name == toolName {
				// Found it - return a wrapper tool
				return &mcpToolWrapper{
					spec:     spec,
					toolSet:  toolSet,
					toolName: toolName,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("tool '%s' not found in server '%s'", toolName, serverName)
}

// mcpToolWrapper wraps an MCP tool from a ToolSet as a Tool
type mcpToolWrapper struct {
	spec     gollem.ToolSpec
	toolSet  gollem.ToolSet
	toolName string
}

func (w *mcpToolWrapper) Spec() gollem.ToolSpec {
	return w.spec
}

func (w *mcpToolWrapper) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return w.toolSet.Run(ctx, w.toolName, args)
}
