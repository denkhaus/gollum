package skills

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a parsed skill definition following ASOS v1.0 specification
// with Gollum-specific extensions.
type Skill struct {
	// Core ASOS v1.0 fields
	Name        string `yaml:"name"`                  // Required: Unique skill name
	Description string `yaml:"description,omitempty"` // Optional: Brief description
	Version     string `yaml:"version,omitempty"`     // Optional: Semantic version

	// Tool configuration
	Tools      []string  `yaml:"tools,omitempty"`       // Allowed tools (empty = all)
	ToolScope  ToolScope `yaml:"tool_scope,omitempty"`  // Tool access scope
	ToolFilter []string  `yaml:"tool_filter,omitempty"` // Tools to filter out

	// Type and classification
	Type SkillType `yaml:"type,omitempty"` // Skill type (agent, mcp, workflow)

	// Gollum-specific extensions
	Arguments     string `yaml:"arguments,omitempty"`      // Argument specification for slash commands
	UserInvocable bool   `yaml:"user_invocable,omitempty"` // Can be invoked by user via slash command
	Priority      int    `yaml:"priority,omitempty"`       // Execution priority (higher = more important)
	AutoInvoke    bool   `yaml:"auto_invoke,omitempty"`    // Automatically invoke on matching context

	// Metadata
	Author   string   `yaml:"author,omitempty"`
	Tags     []string `yaml:"tags,omitempty"`
	Category string   `yaml:"category,omitempty"`

	// Parsed content (not from YAML)
	FilePath    string `yaml:"-"` // Path to the SKILL.md file
	Content     string `yaml:"-"` // Raw content after frontmatter
	FullContent string `yaml:"-"` // Complete file content including frontmatter
}

// skillYAML is used for YAML unmarshaling with defaults
type skillYAML struct {
	Name          string   `yaml:"name"`
	Description   string   `yaml:"description"`
	Version       string   `yaml:"version"`
	Tools         []string `yaml:"tools"`
	ToolScope     string   `yaml:"tool_scope"`
	ToolFilter    []string `yaml:"tool_filter"`
	Type          string   `yaml:"type"`
	Arguments     string   `yaml:"arguments"`
	UserInvocable bool     `yaml:"user_invocable"`
	Priority      int      `yaml:"priority"`
	AutoInvoke    bool     `yaml:"auto_invoke"`
	Author        string   `yaml:"author"`
	Tags          []string `yaml:"tags"`
	Category      string   `yaml:"category"`
}

// frontmatterRegex matches YAML frontmatter in markdown files
var frontmatterRegex = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n?(.*)`)

// Parse parses a SKILL.md file content into a Skill struct
func Parse(content, filePath string) (*Skill, error) {
	matches := frontmatterRegex.FindStringSubmatch(content)
	if matches == nil {
		return nil, ErrParseFailed(filePath, "no YAML frontmatter found")
	}

	frontmatter := matches[1]
	bodyContent := matches[2]

	var raw skillYAML
	if err := yaml.Unmarshal([]byte(frontmatter), &raw); err != nil {
		return nil, ErrParseFailedWithCause(filePath, err)
	}

	skill := &Skill{
		FilePath:    filePath,
		Content:     bodyContent,
		FullContent: content,
	}

	// Map raw YAML to Skill with validation
	if err := mapSkillFromYAML(raw, skill, filePath); err != nil {
		return nil, err
	}

	// Validate required fields
	if err := skill.Validate(); err != nil {
		return nil, err
	}

	return skill, nil
}

// mapSkillFromYAML maps the raw YAML data to the Skill struct with validation
func mapSkillFromYAML(raw skillYAML, skill *Skill, filePath string) error {
	// Required field
	if raw.Name == "" {
		return ErrMissingRequired(filePath, "name")
	}
	skill.Name = raw.Name

	// Optional string fields
	skill.Description = raw.Description
	skill.Version = raw.Version
	skill.Arguments = raw.Arguments
	skill.Author = raw.Author
	skill.Category = raw.Category

	// Slice fields
	skill.Tools = raw.Tools
	skill.ToolFilter = raw.ToolFilter
	skill.Tags = raw.Tags

	// Boolean fields (with defaults)
	skill.UserInvocable = raw.UserInvocable
	skill.AutoInvoke = raw.AutoInvoke

	// Integer fields
	skill.Priority = raw.Priority

	// Parse skill type
	if raw.Type != "" {
		t, err := ParseSkillType(raw.Type)
		if err != nil {
			return ErrInvalidValue(filePath, "type", raw.Type)
		}
		skill.Type = t
	} else {
		skill.Type = SkillTypeAgent // Default
	}

	// Parse tool scope
	if raw.ToolScope != "" {
		s, err := ParseToolScope(raw.ToolScope)
		if err != nil {
			return ErrInvalidValue(filePath, "tool_scope", raw.ToolScope)
		}
		skill.ToolScope = s
	} else {
		skill.ToolScope = ToolScopeAll // Default
	}

	return nil
}

// ParseFile reads and parses a SKILL.md file from disk
func ParseFile(filePath string) (*Skill, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, ErrSkillLoadFailed(filePath, err)
	}

	return Parse(string(content), filePath)
}

// Validate checks if the skill has all required fields and valid values
func (s *Skill) Validate() error {
	// Name is required
	if s.Name == "" {
		return ErrEmptyField(s.FilePath, "name")
	}

	// Name must not contain spaces (for slash command compatibility)
	if strings.Contains(s.Name, " ") {
		return ErrInvalidValue(s.FilePath, "name", s.Name)
	}

	// Validate skill type
	if !s.Type.IsValid() {
		return ErrInvalidValue(s.FilePath, "type", string(s.Type))
	}

	// Validate tool scope
	if !s.ToolScope.IsValid() {
		return ErrInvalidValue(s.FilePath, "tool_scope", string(s.ToolScope))
	}

	return nil
}

// HasTool checks if a specific tool is allowed for this skill
func (s *Skill) HasTool(toolName string) bool {
	// If no tools specified, all tools are allowed (based on scope)
	if len(s.Tools) == 0 {
		return !s.IsToolFiltered(toolName)
	}

	// Check if tool is in allowed list
	for _, t := range s.Tools {
		if t == toolName || t == "*" {
			return !s.IsToolFiltered(toolName)
		}
	}

	return false
}

// IsToolFiltered checks if a tool is in the filter list
func (s *Skill) IsToolFiltered(toolName string) bool {
	for _, t := range s.ToolFilter {
		if t == toolName || t == "*" {
			return true
		}
	}
	return false
}

// IsUserInvocable checks if the skill can be invoked by users
func (s *Skill) IsUserInvocable() bool {
	return s.UserInvocable
}

// SlashCommand returns the slash command for this skill (if user invocable)
func (s *Skill) SlashCommand() string {
	if !s.UserInvocable {
		return ""
	}
	return "/" + strings.ToLower(s.Name)
}

// DisplayName returns a human-readable name for the skill
func (s *Skill) DisplayName() string {
	if s.Description != "" {
		return s.Description
	}
	return s.Name
}

// String returns a string representation of the skill
func (s *Skill) String() string {
	return fmt.Sprintf("Skill{name=%s, type=%s, path=%s}", s.Name, s.Type, s.FilePath)
}

// ID returns a unique identifier for the skill
func (s *Skill) ID() string {
	return strings.ToLower(s.Name)
}

// Directory returns the directory containing the skill file
func (s *Skill) Directory() string {
	return filepath.Dir(s.FilePath)
}

// ToMap converts the skill to a map for serialization
func (s *Skill) ToMap() map[string]any {
	return map[string]any{
		"name":           s.Name,
		"description":    s.Description,
		"version":        s.Version,
		"type":           s.Type,
		"tools":          s.Tools,
		"tool_scope":     s.ToolScope,
		"tool_filter":    s.ToolFilter,
		"arguments":      s.Arguments,
		"user_invocable": s.UserInvocable,
		"priority":       s.Priority,
		"auto_invoke":    s.AutoInvoke,
		"author":         s.Author,
		"tags":           s.Tags,
		"category":       s.Category,
		"file_path":      s.FilePath,
	}
}

// Skills is a collection of skills with utility methods
type Skills []*Skill

// FindByName finds a skill by name (case-insensitive)
func (s Skills) FindByName(name string) *Skill {
	nameLower := strings.ToLower(name)
	for _, skill := range s {
		if strings.ToLower(skill.Name) == nameLower {
			return skill
		}
	}
	return nil
}

// FilterByType returns skills of a specific type
func (s Skills) FilterByType(skillType SkillType) Skills {
	var result Skills
	for _, skill := range s {
		if skill.Type == skillType {
			result = append(result, skill)
		}
	}
	return result
}

// FilterUserInvocable returns skills that can be invoked by users
func (s Skills) FilterUserInvocable() Skills {
	var result Skills
	for _, skill := range s {
		if skill.UserInvocable {
			result = append(result, skill)
		}
	}
	return result
}

// Names returns a list of skill names
func (s Skills) Names() []string {
	names := make([]string, len(s))
	for i, skill := range s {
		names[i] = skill.Name
	}
	return names
}

// String returns a string representation of the skills collection
func (s Skills) String() string {
	var buf bytes.Buffer
	buf.WriteString("Skills[")
	for i, skill := range s {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(skill.Name)
	}
	buf.WriteString("]")
	return buf.String()
}
