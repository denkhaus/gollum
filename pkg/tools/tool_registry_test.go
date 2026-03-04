package tools

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolRegistry(t *testing.T) {
	registry, err := NewToolRegistry(nil)
	require.NoError(t, err)
	require.NotNil(t, registry)
}

func TestToolRegistry_IsValidTool(t *testing.T) {
	registry, err := NewToolRegistry(nil)
	require.NoError(t, err)

	tests := []struct {
		name     string
		tool     shared.ToolName
		expected bool
	}{
		// Valid tools
		{"read_file", shared.ToolNameReadFile, true},
		{"write_file", shared.ToolNameWriteFile, true},
		{"edit", shared.ToolNameEdit, true},
		{"glob", shared.ToolNameGlob, true},
		{"grep", shared.ToolNameGrep, true},
		{"bash", shared.ToolNameBash, true},
		{"spawn_agent", shared.ToolNameSpawnAgent, true},
		{"resume_agent", shared.ToolNameResumeAgent, true},
		{"remove_agent", shared.ToolNameRemoveAgent, true},
		{"list_agents", shared.ToolNameListAgents, true},
		{"agent_output", shared.ToolNameAgentOutput, true},
		{"invoke_skill", shared.ToolNameInvokeSkill, true},
		{"change_directory", shared.ToolNameChangeDirectory, true},
		{"current_time", shared.ToolNameCurrentTime, true},
		{"session_logs", shared.ToolNameSessionLogs, true},

		// Invalid tools (common mistakes)
		{"uppercase Read", "Read", false},
		{"uppercase Write", "Write", false},
		{"uppercase Edit", "Edit", false},
		{"uppercase Glob", "Glob", false},
		{"uppercase Grep", "Grep", false},
		{"uppercase Bash", "Bash", false},
		{"camelCase readFile", "readFile", false},
		{"camelCase writeFile", "writeFile", false},
		{"nonexistent", "nonexistent_tool", false},
		{"empty string", "", false},
		{"random string", "foobar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, registry.IsValidTool(tt.tool))
		})
	}
}

func TestToolRegistry_GetValidToolNames(t *testing.T) {
	registry, err := NewToolRegistry(nil)
	require.NoError(t, err)

	names := registry.GetValidToolNames()

	// Should have all 15 tools
	assert.Len(t, names, 15, "Should have exactly 15 valid tools")

	// Should contain expected tools
	assert.Contains(t, names, shared.ToolNameReadFile)
	assert.Contains(t, names, shared.ToolNameWriteFile)
	assert.Contains(t, names, shared.ToolNameEdit)
	assert.Contains(t, names, shared.ToolNameGlob)
	assert.Contains(t, names, shared.ToolNameGrep)
	assert.Contains(t, names, shared.ToolNameBash)
	assert.Contains(t, names, shared.ToolNameSpawnAgent)
	assert.Contains(t, names, shared.ToolNameInvokeSkill)
}

func TestToolRegistry_AllToolsAreValid(t *testing.T) {
	registry, err := NewToolRegistry(nil)
	require.NoError(t, err)

	// Every tool returned by GetValidToolNames should pass IsValidTool
	for _, name := range registry.GetValidToolNames() {
		assert.True(t, registry.IsValidTool(name), "Tool %q should be valid", name)
	}
}
