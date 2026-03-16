package tui

import (
	"strings"

	"go.uber.org/mock/gomock"
)

// containsSubstring checks if a string contains a substring
// This is a test utility function
func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}

// setupMockAgent creates a mock agent executor for testing
func setupMockAgent(ctrl *gomock.Controller) *MockAgentExecutor {
	return NewMockAgentExecutor(ctrl)
}
