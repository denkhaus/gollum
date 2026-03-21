package app

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
)

// mockMCPRegistryForTools is a mock that returns predefined tool names
type mockMCPRegistryForTools struct {
	toolNames []string
}

func (m *mockMCPRegistryForTools) GetToolSets() []gollem.ToolSet {
	return nil
}

func (m *mockMCPRegistryForTools) GetToolNames() []string {
	return m.toolNames
}

func (m *mockMCPRegistryForTools) Close() error {
	return nil
}

// Ensure mock implements interface
var _ registry.MCPRegistry = (*mockMCPRegistryForTools)(nil)

// TestConvertToolSetsToAllowedTools tests the conversion function
func TestConvertToolSetsToAllowedTools(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		mcpRegistry    registry.MCPRegistry
		expectedOutput []string
	}{
		{
			name: "empty registry",
			mcpRegistry: &mockMCPRegistryForTools{
				toolNames: []string{},
			},
			expectedOutput: []string{},
		},
		{
			name: "single tool",
			mcpRegistry: &mockMCPRegistryForTools{
				toolNames: []string{"filesystem/read_file"},
			},
			expectedOutput: []string{"filesystem/read_file"},
		},
		{
			name: "multiple tools from same server",
			mcpRegistry: &mockMCPRegistryForTools{
				toolNames: []string{"filesystem/read_file", "filesystem/write_file"},
			},
			expectedOutput: []string{"filesystem/read_file", "filesystem/write_file"},
		},
		{
			name: "multiple tools from different servers",
			mcpRegistry: &mockMCPRegistryForTools{
				toolNames: []string{"filesystem/read_file", "math/add", "math/subtract"},
			},
			expectedOutput: []string{"filesystem/read_file", "math/add", "math/subtract"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToolSetsToAllowedTools(ctx, nil, tt.mcpRegistry)
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}
