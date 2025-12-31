package shared

import (
	"testing"

	"github.com/google/uuid"
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
		LLMProvider:  LLMProviderAnthropic,
	}

	if config.OutputMode != "" {
		t.Errorf("default OutputMode should be empty, got %s", config.OutputMode)
	}
}
