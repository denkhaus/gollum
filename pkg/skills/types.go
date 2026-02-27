// Package skills provides skill parsing, validation, and discovery functionality
// for the Gollum agent system following the Agent Skills Open Standard (ASOS) v1.0.
package skills

import (
	"fmt"
	"strings"
)

// SkillType represents the type of skill as defined by ASOS v1.0
type SkillType string

const (
	// SkillTypeAgent is a standard agent skill (default)
	SkillTypeAgent SkillType = "agent"
	// SkillTypeMcp is an MCP server skill
	SkillTypeMcp SkillType = "mcp"
	// SkillTypeWorkflow is a workflow/orchestration skill
	SkillTypeWorkflow SkillType = "workflow"
)

// IsValid checks if the skill type is valid
func (t SkillType) IsValid() bool {
	switch t {
	case SkillTypeAgent, SkillTypeMcp, SkillTypeWorkflow:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (t SkillType) String() string {
	return string(t)
}

// ParseSkillType parses a string into a SkillType
func ParseSkillType(s string) (SkillType, error) {
	t := SkillType(strings.ToLower(s))
	if !t.IsValid() {
		return "", fmt.Errorf("invalid skill type: %s", s)
	}
	return t, nil
}

// ToolScope defines the scope of tool access for a skill
type ToolScope string

const (
	// ToolScopeAll allows access to all tools
	ToolScopeAll ToolScope = "all"
	// ToolScopeReadOnly allows only read-only tools
	ToolScopeReadOnly ToolScope = "read-only"
	// ToolScopeNone allows no tool access
	ToolScopeNone ToolScope = "none"
	// ToolScopeCustom allows custom tool restrictions
	ToolScopeCustom ToolScope = "custom"
)

// IsValid checks if the tool scope is valid
func (s ToolScope) IsValid() bool {
	switch s {
	case ToolScopeAll, ToolScopeReadOnly, ToolScopeNone, ToolScopeCustom:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (s ToolScope) String() string {
	return string(s)
}

// ParseToolScope parses a string into a ToolScope
func ParseToolScope(str string) (ToolScope, error) {
	s := ToolScope(strings.ToLower(str))
	if !s.IsValid() {
		return "", fmt.Errorf("invalid tool scope: %s", str)
	}
	return s, nil
}

// Constants for skill discovery and parsing
const (
	// SkillFileName is the standard filename for skill definitions
	SkillFileName = "SKILL.md"

	// FrontmatterDelimiter is the YAML frontmatter delimiter
	FrontmatterDelimiter = "---"

	// DefaultMaxDepth is the default maximum directory traversal depth
	DefaultMaxDepth = 10

	// DefaultConcurrency is the default number of concurrent discovery workers
	DefaultConcurrency = 4
)

// DefaultIgnoredDirs contains the default list of directories to skip during skill discovery.
var DefaultIgnoredDirs = []string{
	".git",
	".svn",
	".hg",
	"node_modules",
	"vendor",
	"__pycache__",
	".cache",
	"dist",
	"build",
	"target",
	"bin",
	"tmp",
	"temp",
}
