package tools

import (
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/samber/do/v2"
)

// ToolRegistry provides validation for tool names used in skills.
// It maintains a list of all valid Gollum tool names and allows
// the skills package to validate tool configurations without
// creating a circular dependency.
type ToolRegistry interface {
	// IsValidTool checks if a tool name is a valid registered tool
	IsValidTool(name string) bool

	// GetValidToolNames returns all valid tool names
	GetValidToolNames() []string
}

// toolRegistryImpl implements ToolRegistry using the shared tool name constants
type toolRegistryImpl struct {
	validTools map[string]bool
}

// Ensure toolRegistryImpl implements ToolRegistry
var _ ToolRegistry = (*toolRegistryImpl)(nil)

// Ensure toolRegistryImpl implements skills.ToolNameValidator
var _ skills.ToolNameValidator = (*toolRegistryImpl)(nil)

// NewToolRegistry creates a new ToolRegistry instance via DI
func NewToolRegistry(_ do.Injector) (ToolRegistry, error) {
	// Build the valid tools map from shared constants
	validTools := map[string]bool{
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
func NewToolNameValidatorProvider(injector do.Injector) (skills.ToolNameValidator, error) {
	return NewToolRegistry(injector)
}

// IsValidTool checks if a tool name is a valid registered tool
func (r *toolRegistryImpl) IsValidTool(name string) bool {
	return r.validTools[name]
}

// GetValidToolNames returns all valid tool names
func (r *toolRegistryImpl) GetValidToolNames() []string {
	names := make([]string, 0, len(r.validTools))
	for name := range r.validTools {
		names = append(names, name)
	}
	return names
}
