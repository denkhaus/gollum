package registry

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/mcp/config"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mcp"
)

func TestMCPRegistry_CreateStdioClient(t *testing.T) {
	// Test with invalid command to verify graceful failure
	// (echo is not a valid MCP server)
	loader := &mockConfigLoader{
		configs: map[string]config.MCPServerConfig{
			"invalid-server": {
				Command: "echo",
				Args:    []string{"not an mcp server"},
				Enabled: true,
			},
		},
	}

	registry, err := NewMCPRegistryWithLoader(loader)
	if err != nil {
		t.Fatalf("NewMCPRegistryWithLoader() error = %v", err)
	}
	defer func() { _ = registry.Close() }()

	// Invalid servers should be skipped, returning 0 ToolSets
	toolSets := registry.GetToolSets()
	if len(toolSets) != 0 {
		t.Errorf("GetToolSets() returned %d ToolSets, want 0 (invalid server skipped)", len(toolSets))
	}
}

func TestMCPRegistry_CreateSSEClient(t *testing.T) {
	// Test SSE client creation with invalid URL (graceful failure)
	loader := &mockConfigLoader{
		configs: map[string]config.MCPServerConfig{
			"invalid-sse": {
				Type:    "sse",
				URL:     "http://invalid-local-url:9999/mcp",
				Enabled: true,
			},
		},
	}

	registry, err := NewMCPRegistryWithLoader(loader)
	if err != nil {
		t.Fatalf("NewMCPRegistryWithLoader() error = %v", err)
	}
	defer func() { _ = registry.Close() }()

	// Invalid SSE servers should be skipped
	toolSets := registry.GetToolSets()
	if len(toolSets) != 0 {
		t.Errorf("GetToolSets() returned %d ToolSets, want 0 (invalid SSE server skipped)", len(toolSets))
	}
}

func TestMCPRegistry_CreateStreamableHTTPClient(t *testing.T) {
	// Test streamable HTTP client creation with invalid URL (graceful failure)
	loader := &mockConfigLoader{
		configs: map[string]config.MCPServerConfig{
			"invalid-streamable": {
				Type:    "streamable-http",
				URL:     "http://invalid-local-url:9999/mcp",
				Enabled: true,
			},
		},
	}

	registry, err := NewMCPRegistryWithLoader(loader)
	if err != nil {
		t.Fatalf("NewMCPRegistryWithLoader() error = %v", err)
	}
	defer func() { _ = registry.Close() }()

	// Invalid streamable HTTP servers should be skipped
	toolSets := registry.GetToolSets()
	if len(toolSets) != 0 {
		t.Errorf("GetToolSets() returned %d ToolSets, want 0 (invalid streamable HTTP server skipped)", len(toolSets))
	}
}

func TestMCPRegistry_UnsupportedType(t *testing.T) {
	// Test unsupported MCP type
	loader := &mockConfigLoader{
		configs: map[string]config.MCPServerConfig{
			"unsupported-type": {
				Type:    "unsupported",
				Enabled: true,
			},
		},
	}

	registry, err := NewMCPRegistryWithLoader(loader)
	if err != nil {
		t.Fatalf("NewMCPRegistryWithLoader() error = %v", err)
	}
	defer func() { _ = registry.Close() }()

	// Unsupported type should be skipped
	toolSets := registry.GetToolSets()
	if len(toolSets) != 0 {
		t.Errorf("GetToolSets() returned %d ToolSets, want 0 (unsupported type skipped)", len(toolSets))
	}
}

// mockConfigLoader is a test double
type mockConfigLoader struct {
	configs map[string]config.MCPServerConfig
}

func (m *mockConfigLoader) Load() (map[string]config.MCPServerConfig, error) {
	return m.configs, nil
}

// NewMCPRegistryWithLoader creates registry with custom loader for testing
func NewMCPRegistryWithLoader(loader config.ConfigLoader) (MCPRegistry, error) {
	p := &mcpRegistryImpl{
		clients: make(map[string]*mcp.Client),
		tools:   make([]gollem.ToolSet, 0),
		loader:  loader,
	}
	// Initialize clients for testing
	if err := p.initializeClients(context.Background()); err != nil {
		return nil, err
	}
	return p, nil
}
