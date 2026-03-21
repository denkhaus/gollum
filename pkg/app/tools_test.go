package app

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
)

// mockToolSet is a simple mock ToolSet for testing
type mockToolSet struct {
	serverName string
	tools      []gollem.ToolSpec
}

func (m *mockToolSet) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return m.tools, nil
}

func (m *mockToolSet) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return nil, nil
}

// TestConvertToolSetsToAllowedTools tests the conversion function
func TestConvertToolSetsToAllowedTools(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		toolSets       []gollem.ToolSet
		expectedOutput []string
	}{
		{
			name:           "empty tool sets",
			toolSets:       []gollem.ToolSet{},
			expectedOutput: []string{},
		},
		{
			name: "single tool set with one tool",
			toolSets: []gollem.ToolSet{
				&mockToolSet{
					serverName: "filesystem",
					tools: []gollem.ToolSpec{
						{Name: "read_file"},
					},
				},
			},
			expectedOutput: []string{"filesystem/read_file"},
		},
		{
			name: "single tool set with multiple tools",
			toolSets: []gollem.ToolSet{
				&mockToolSet{
					serverName: "filesystem",
					tools: []gollem.ToolSpec{
						{Name: "read_file"},
						{Name: "write_file"},
					},
				},
			},
			expectedOutput: []string{"filesystem/read_file", "filesystem/write_file"},
		},
		{
			name: "multiple tool sets",
			toolSets: []gollem.ToolSet{
				&mockToolSet{
					serverName: "filesystem",
					tools: []gollem.ToolSpec{
						{Name: "read_file"},
					},
				},
				&mockToolSet{
					serverName: "math",
					tools: []gollem.ToolSpec{
						{Name: "add"},
						{Name: "subtract"},
					},
				},
			},
			expectedOutput: []string{"filesystem/read_file", "math/add", "math/subtract"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToolSetsToAllowedTools(ctx, tt.toolSets)
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}
