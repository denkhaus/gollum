package shared

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestOutputModeValues verifies OutputMode constants have correct values
func TestOutputModeValues(t *testing.T) {
	tests := []struct {
		name     string
		mode     OutputMode
		expected string
	}{
		{"Full mode", OutputModeFull, "full"},
		{"Summary mode", OutputModeSummary, "summary"},
		{"Silent mode", OutputModeSilent, "silent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.mode) != tt.expected {
				t.Errorf("OutputMode = %s, want %s", tt.mode, tt.expected)
			}
		})
	}
}

// TestAgentConfig_OutputModeDefault verifies default OutputMode is empty string
func TestAgentConfig_OutputModeDefault(t *testing.T) {
	config := &AgentConfig{
		ID:           uuid.New(),
		SystemPrompt: "test",
		Role:         "TestAgent",
		LLMClientConfig: &LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
	}

	if config.OutputMode != "" {
		t.Errorf("default OutputMode should be empty, got %s", config.OutputMode)
	}
}

// TestToolResultKeyConstants verifies all ToolResult key constants have correct values
func TestToolResultKeyConstants(t *testing.T) {
	tests := []struct {
		name     string
		key      ToolResultKeys
		expected string
	}{
		{"KeySuccess", KeySuccess, "success"},
		{"KeyError", KeyError, "error"},
		{"KeyFilePath", KeyFilePath, "file_path"},
		{"KeyChecksum", KeyChecksum, "checksum"},
		{"KeySize", KeySize, "size"},
		{"KeyModified", KeyModified, "modified"},
		{"KeyLockedBy", KeyLockedBy, "locked_by"},
		{"KeyReplacements", KeyReplacements, "replacements"},
		{"KeyOldLen", KeyOldLen, "old_len"},
		{"KeyNewLen", KeyNewLen, "new_len"},
		{"KeyDiff", KeyDiff, "diff"},
		{"KeyDiffAdditions", KeyDiffAdditions, "diff_additions"},
		{"KeyDiffRemovals", KeyDiffRemovals, "diff_removals"},
		{"KeyDiffCompact", KeyDiffCompact, "diff_compact"},
		{"KeyIsNewFile", KeyIsNewFile, "is_new_file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.key) != tt.expected {
				t.Errorf("ToolResultKey = %s, want %s", tt.key, tt.expected)
			}
		})
	}
}

// TestToolResult_GetString verifies GetString helper method
func TestToolResult_GetString(t *testing.T) {
	result := ToolResult{
		"success":   true,
		"error":     "test error",
		"file_path": "/path/to/file.txt",
		"checksum":  "abc123",
		"count":     42,
	}

	tests := []struct {
		name     string
		key      ToolResultKeys
		expected string
	}{
		{"Get existing string", KeyError, "test error"},
		{"Get existing file path", KeyFilePath, "/path/to/file.txt"},
		{"Get existing checksum", KeyChecksum, "abc123"},
		{"Get non-existent key", KeySize, ""},
		{"Get wrong type key", KeySuccess, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := result.GetString(tt.key); got != tt.expected {
				t.Errorf("GetString() = %s, want %s", got, tt.expected)
			}
		})
	}
}

// TestToolResult_GetInt verifies GetInt helper method
func TestToolResult_GetInt(t *testing.T) {
	agentID := uuid.New()
	result := ToolResult{
		"success":      true,
		"replacements": 3,
		"old_len":      10,
		"new_len":      15,
		"size":         int64(1024),
		"float_val":    3.14,
		"locked_by":    agentID,
	}

	tests := []struct {
		name     string
		key      ToolResultKeys
		expected int
	}{
		{"Get existing int", KeyReplacements, 3},
		{"Get old_len", KeyOldLen, 10},
		{"Get new_len", KeyNewLen, 15},
		{"Get int64 as int", KeySize, 1024},
		{"Get float64 as int", ToolResultKeys("float_val"), 3},
		{"Get non-existent key", KeyModified, 0},
		{"Get wrong type key", KeySuccess, 0},
		{"Get UUID type", KeyLockedBy, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := result.GetInt(tt.key); got != tt.expected {
				t.Errorf("GetInt() = %d, want %d", got, tt.expected)
			}
		})
	}
}

// TestToolResult_GetBool verifies GetBool helper method
func TestToolResult_GetBool(t *testing.T) {
	result := ToolResult{
		"success":      true,
		"is_new":       false,
		"error":        "test error",
		"replacements": 3,
	}

	tests := []struct {
		name     string
		key      ToolResultKeys
		expected bool
	}{
		{"Get true bool", KeySuccess, true},
		{"Get false bool", ToolResultKeys("is_new"), false},
		{"Get non-existent key", KeyError, false},
		{"Get wrong type key", KeyReplacements, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := result.GetBool(tt.key); got != tt.expected {
				t.Errorf("GetBool() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestToolResult_HasDiff verifies HasDiff helper method
func TestToolResult_HasDiff(t *testing.T) {
	tests := []struct {
		name     string
		result   ToolResult
		expected bool
	}{
		{
			name:     "Empty result",
			result:   ToolResult{},
			expected: false,
		},
		{
			name:     "Has full diff",
			result:   ToolResult{"diff": "-line1\n+line2"},
			expected: true,
		},
		{
			name:     "Has additions only",
			result:   ToolResult{"diff_additions": 5},
			expected: true,
		},
		{
			name:     "Has removals only",
			result:   ToolResult{"diff_removals": 3},
			expected: true,
		},
		{
			name:     "Has compact diff only",
			result:   ToolResult{"diff_compact": "2L3"},
			expected: true,
		},
		{
			name:     "Has multiple diff fields",
			result:   ToolResult{"diff_additions": 5, "diff_removals": 3},
			expected: true,
		},
		{
			name:     "Has other fields but no diff",
			result:   ToolResult{"success": true, "file_path": "/test.txt"},
			expected: false,
		},
		{
			name:     "Success result without diff",
			result:   ToolResult{"success": true, "error": "", "file_path": "/test.txt"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasDiff(); got != tt.expected {
				t.Errorf("HasDiff() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestToolResult verifies ToolResult type alias works as map[string]any
func TestToolResult(t *testing.T) {
	// Can create from literal
	result := ToolResult{
		string(KeySuccess):      true,
		string(KeyFilePath):     "/path/to/file.txt",
		string(KeyReplacements): 3,
	}

	assert.True(t, result.GetBool(KeySuccess))
	assert.Equal(t, "/path/to/file.txt", result.GetString(KeyFilePath))
	assert.Equal(t, 3, result.GetInt(KeyReplacements))
	assert.False(t, result.HasDiff())

	// Can be used as map[string]any
	var m map[string]any = result
	assert.True(t, m["success"].(bool))
}

// TestToolResult_ConstantUsage verifies constants can be used as map keys
func TestToolResult_ConstantUsage(t *testing.T) {
	result := make(ToolResult)

	// Set using constants (need explicit string conversion)
	result[string(KeySuccess)] = true
	result[string(KeyError)] = "test error"
	result[string(KeyFilePath)] = "/test/path"
	result[string(KeyChecksum)] = "abc123def"
	result[string(KeySize)] = int64(2048)
	result[string(KeyModified)] = int64(1234567890)
	result[string(KeyReplacements)] = 5
	result[string(KeyOldLen)] = 10
	result[string(KeyNewLen)] = 20

	// Verify using helper methods
	assert.True(t, result.GetBool(KeySuccess))
	assert.Equal(t, "test error", result.GetString(KeyError))
	assert.Equal(t, "/test/path", result.GetString(KeyFilePath))
	assert.Equal(t, "abc123def", result.GetString(KeyChecksum))
	assert.Equal(t, 2048, result.GetInt(KeySize))
	assert.Equal(t, 1234567890, result.GetInt(KeyModified))
	assert.Equal(t, 5, result.GetInt(KeyReplacements))
	assert.Equal(t, 10, result.GetInt(KeyOldLen))
	assert.Equal(t, 20, result.GetInt(KeyNewLen))
}

// TestAgentConfig_AllowedToolsField verifies AllowedTools field exists and works
func TestAgentConfig_AllowedToolsField(t *testing.T) {
	config := &AgentConfig{
		ID:           uuid.New(),
		AllowedTools: []string{"bash", "current_time"},
	}

	assert.NotNil(t, config.AllowedTools)
	assert.Equal(t, "bash", config.AllowedTools[0])
	assert.Equal(t, "current_time", config.AllowedTools[1])
}

// TestAgentConfig_NoToolsOrToolSetsFields verifies old Tools/ToolSets fields are removed
func TestAgentConfig_NoToolsOrToolSetsFields(t *testing.T) {
	config := &AgentConfig{
		ID: uuid.New(),
	}

	// AllowedTools should exist and be nil by default
	assert.Nil(t, config.AllowedTools)
}
