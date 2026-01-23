package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHookPoint_String tests HookPoint string representation
func TestHookPoint_String(t *testing.T) {
	tests := []struct {
		name     string
		point    HookPoint
		expected string
	}{
		{"BeforeSessionStart", BeforeSessionStart, "BeforeSessionStart"},
		{"AfterSessionEnd", AfterSessionEnd, "AfterSessionEnd"},
		{"BeforeAgentSpawn", BeforeAgentSpawn, "BeforeAgentSpawn"},
		{"AfterAgentSpawn", AfterAgentSpawn, "AfterAgentSpawn"},
		{"BeforeAgentRemove", BeforeAgentRemove, "BeforeAgentRemove"},
		{"AfterAgentRemove", AfterAgentRemove, "AfterAgentRemove"},
		{"BeforeToolExecution", BeforeToolExecution, "BeforeToolExecution"},
		{"AfterToolExecution", AfterToolExecution, "AfterToolExecution"},
		{"OnToolError", OnToolError, "OnToolError"},
		{"BeforeFileRead", BeforeFileRead, "BeforeFileRead"},
		{"AfterFileRead", AfterFileRead, "AfterFileRead"},
		{"BeforeFileWrite", BeforeFileWrite, "BeforeFileWrite"},
		{"AfterFileWrite", AfterFileWrite, "AfterFileWrite"},
		{"BeforeFileDelete", BeforeFileDelete, "BeforeFileDelete"},
		{"AfterFileDelete", AfterFileDelete, "AfterFileDelete"},
		{"BeforeFileModify", BeforeFileModify, "BeforeFileModify"},
		{"AfterFileModify", AfterFileModify, "AfterFileModify"},
		{"BeforeLLMRequest", BeforeLLMRequest, "BeforeLLMRequest"},
		{"AfterLLMResponse", AfterLLMResponse, "AfterLLMResponse"},
		{"OnLLMError", OnLLMError, "OnLLMError"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.point.String())
		})
	}
}
