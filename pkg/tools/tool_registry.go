package tools

import (
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
)

// toolRegistryImpl implements ToolRegistry using the shared tool name constants
type toolRegistryImpl struct {
	validTools map[shared.ToolName]bool
}

// Ensure toolRegistryImpl implements ToolRegistry
var _ shared.ToolRegistry = (*toolRegistryImpl)(nil)

// NewToolRegistry creates a new ToolRegistry instance via DI
func NewToolRegistry(_ do.Injector) (shared.ToolRegistry, error) {
	// Build the valid tools map from shared constants
	validTools := map[shared.ToolName]bool{
		// File operations
		shared.ToolNameReadFile:  true,
		shared.ToolNameWriteFile: true,
		shared.ToolNameEdit:      true,
		shared.ToolNameGlob:      true,
		shared.ToolNameGrep:      true,

		// System operations
		shared.ToolNameBash: true,

		// Agent management
		shared.ToolNameSpawnAgent:  true,
		shared.ToolNameResumeAgent: true,
		shared.ToolNameRemoveAgent: true,
		shared.ToolNameListAgents:  true,
		shared.ToolNameAgentOutput: true,

		// Skill operations
		shared.ToolNameInvokeSkill: true,

		// Navigation
		shared.ToolNameChangeDirectory: true,

		// Utilities
		shared.ToolNameCurrentTime: true,
		shared.ToolNameSessionLogs: true,
	}

	return &toolRegistryImpl{
		validTools: validTools,
	}, nil
}

// NewToolNameValidatorProvider creates a skills.ToolNameValidator via DI
// This is a wrapper to satisfy the DI provider signature
func NewToolNameValidatorProvider(injector do.Injector) (shared.ToolRegistry, error) {
	return NewToolRegistry(injector)
}

// IsValidTool checks if a tool name is a valid registered tool
func (r *toolRegistryImpl) IsValidTool(name shared.ToolName) bool {
	return r.validTools[name]
}

// GetValidToolNames returns all valid tool names
func (r *toolRegistryImpl) GetValidToolNames() []shared.ToolName {
	names := make([]shared.ToolName, 0, len(r.validTools))
	for name := range r.validTools {
		names = append(names, name)
	}
	return names
}
