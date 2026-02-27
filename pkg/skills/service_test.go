package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestSkillType_Validity tests SkillType validation
func TestSkillType_Validity(t *testing.T) {
	tests := []struct {
		name      string
		skillType SkillType
		expected  bool
	}{
		{"agent type", SkillTypeAgent, true},
		{"mcp type", SkillTypeMcp, true},
		{"workflow type", SkillTypeWorkflow, true},
		{"invalid type", SkillType("invalid"), false},
		{"empty type", SkillType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.skillType.IsValid())
		})
	}
}

// TestParseSkillType tests parsing skill type from string
func TestParseSkillType(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  SkillType
		expectErr bool
	}{
		{"agent", "agent", SkillTypeAgent, false},
		{"agent uppercase", "AGENT", SkillTypeAgent, false},
		{"mcp", "mcp", SkillTypeMcp, false},
		{"workflow", "workflow", SkillTypeWorkflow, false},
		{"invalid", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSkillType(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestToolScope_Validity tests ToolScope validation
func TestToolScope_Validity(t *testing.T) {
	tests := []struct {
		name      string
		toolScope ToolScope
		expected  bool
	}{
		{"all scope", ToolScopeAll, true},
		{"read-only scope", ToolScopeReadOnly, true},
		{"none scope", ToolScopeNone, true},
		{"custom scope", ToolScopeCustom, true},
		{"invalid scope", ToolScope("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.toolScope.IsValid())
		})
	}
}

// TestParse_ValidSkill tests parsing a valid skill
func TestParse_ValidSkill(t *testing.T) {
	content := `---
name: test-skill
description: A test skill
version: "1.0"
type: agent
tools:
  - bash
  - read
tool_scope: all
user_invocable: true
tags:
  - test
  - example
---

This is the skill content.
It can be multiple lines.
`

	skill, err := Parse(content, "/test/SKILL.md")
	require.NoError(t, err)

	assert.Equal(t, "test-skill", skill.Name)
	assert.Equal(t, "A test skill", skill.Description)
	assert.Equal(t, "1.0", skill.Version)
	assert.Equal(t, SkillTypeAgent, skill.Type)
	assert.Equal(t, ToolScopeAll, skill.ToolScope)
	assert.True(t, skill.UserInvocable)
	assert.ElementsMatch(t, []string{"bash", "read"}, skill.Tools)
	assert.ElementsMatch(t, []string{"test", "example"}, skill.Tags)
	assert.Contains(t, skill.Content, "This is the skill content")
}

// TestParse_MissingFrontmatter tests parsing without frontmatter
func TestParse_MissingFrontmatter(t *testing.T) {
	content := "This is just content without frontmatter"

	_, err := Parse(content, "/test/SKILL.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no YAML frontmatter")
}

// TestParse_MissingRequiredField tests parsing with missing required field
func TestParse_MissingRequiredField(t *testing.T) {
	content := `---
description: A skill without a name
---
Content
`

	_, err := Parse(content, "/test/SKILL.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field")
}

// TestParse_InvalidType tests parsing with invalid type
func TestParse_InvalidType(t *testing.T) {
	content := `---
name: test-skill
type: invalid-type
---
Content
`

	_, err := Parse(content, "/test/SKILL.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

// TestSkill_Validate tests skill validation
func TestSkill_Validate(t *testing.T) {
	tests := []struct {
		name        string
		skill       *Skill
		expectErr   bool
		errContains string
	}{
		{
			name:      "valid skill",
			skill:     &Skill{Name: "test", Type: SkillTypeAgent, ToolScope: ToolScopeAll, FilePath: "/test"},
			expectErr: false,
		},
		{
			name:        "missing name",
			skill:       &Skill{Type: SkillTypeAgent, ToolScope: ToolScopeAll, FilePath: "/test"},
			expectErr:   true,
			errContains: "name",
		},
		{
			name:        "name with spaces",
			skill:       &Skill{Name: "test skill", Type: SkillTypeAgent, ToolScope: ToolScopeAll, FilePath: "/test"},
			expectErr:   true,
			errContains: "name",
		},
		{
			name:        "invalid type",
			skill:       &Skill{Name: "test", Type: SkillType("bad"), ToolScope: ToolScopeAll, FilePath: "/test"},
			expectErr:   true,
			errContains: "type",
		},
		{
			name:        "invalid tool scope",
			skill:       &Skill{Name: "test", Type: SkillTypeAgent, ToolScope: ToolScope("bad"), FilePath: "/test"},
			expectErr:   true,
			errContains: "tool_scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.skill.Validate()
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSkill_HasTool tests tool access checking
func TestSkill_HasTool(t *testing.T) {
	tests := []struct {
		name     string
		skill    *Skill
		toolName string
		expected bool
	}{
		{
			name:     "no tools specified - all allowed",
			skill:    &Skill{Tools: nil, ToolFilter: nil},
			toolName: "bash",
			expected: true,
		},
		{
			name:     "tool in allowed list",
			skill:    &Skill{Tools: []string{"bash", "read"}, ToolFilter: nil},
			toolName: "bash",
			expected: true,
		},
		{
			name:     "tool not in allowed list",
			skill:    &Skill{Tools: []string{"bash", "read"}, ToolFilter: nil},
			toolName: "write",
			expected: false,
		},
		{
			name:     "tool filtered out",
			skill:    &Skill{Tools: nil, ToolFilter: []string{"write"}},
			toolName: "write",
			expected: false,
		},
		{
			name:     "wildcard in tools",
			skill:    &Skill{Tools: []string{"*"}, ToolFilter: nil},
			toolName: "any-tool",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.skill.HasTool(tt.toolName))
		})
	}
}

// TestSkill_SlashCommand tests slash command generation
func TestSkill_SlashCommand(t *testing.T) {
	tests := []struct {
		name     string
		skill    *Skill
		expected string
	}{
		{
			name:     "user invocable skill",
			skill:    &Skill{Name: "MySkill", UserInvocable: true},
			expected: "/myskill",
		},
		{
			name:     "not user invocable",
			skill:    &Skill{Name: "MySkill", UserInvocable: false},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.skill.SlashCommand())
		})
	}
}

// TestSkills_Collection tests Skills collection methods
func TestSkills_Collection(t *testing.T) {
	skills := Skills{
		{Name: "skill-a", Type: SkillTypeAgent, UserInvocable: true},
		{Name: "skill-b", Type: SkillTypeMcp, UserInvocable: false},
		{Name: "skill-c", Type: SkillTypeAgent, UserInvocable: true},
	}

	t.Run("FindByName", func(t *testing.T) {
		assert.NotNil(t, skills.FindByName("skill-a"))
		assert.NotNil(t, skills.FindByName("SKILL-A")) // case insensitive
		assert.Nil(t, skills.FindByName("nonexistent"))
	})

	t.Run("FilterByType", func(t *testing.T) {
		agentSkills := skills.FilterByType(SkillTypeAgent)
		assert.Len(t, agentSkills, 2)
	})

	t.Run("FilterUserInvocable", func(t *testing.T) {
		invocable := skills.FilterUserInvocable()
		assert.Len(t, invocable, 2)
	})

	t.Run("Names", func(t *testing.T) {
		names := skills.Names()
		assert.ElementsMatch(t, []string{"skill-a", "skill-b", "skill-c"}, names)
	})
}

// TestDiscoverInPath tests skill discovery
func TestDiscoverInPath(t *testing.T) {
	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "skill-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create skill directories
	skillDir1 := filepath.Join(tmpDir, "skill1")
	skillDir2 := filepath.Join(tmpDir, "skill2")
	require.NoError(t, os.MkdirAll(skillDir1, 0755))
	require.NoError(t, os.MkdirAll(skillDir2, 0755))

	// Create SKILL.md files
	skill1Content := `---
name: skill-one
description: First test skill
type: agent
user_invocable: true
---
Content for skill one
`
	skill2Content := `---
name: skill-two
description: Second test skill
type: mcp
---
Content for skill two
`

	require.NoError(t, os.WriteFile(filepath.Join(skillDir1, "SKILL.md"), []byte(skill1Content), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir2, "SKILL.md"), []byte(skill2Content), 0644))

	// Create a file that should be ignored (in .git directory)
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "SKILL.md"), []byte(`---name: ignored-skill---`), 0644))

	// Run discovery
	log := zap.NewNop()
	result, err := DiscoverInPath(context.Background(), tmpDir, log)
	require.NoError(t, err)

	// Verify results
	assert.Len(t, result.Skills, 2, "Should find 2 skills, ignoring .git")
	assert.Equal(t, 0, len(result.Errors), "Should have no errors")

	// Verify skills were found
	names := result.Skills.Names()
	assert.ElementsMatch(t, []string{"skill-one", "skill-two"}, names)
}

// TestDiscoverInPath_InvalidSkill tests discovery with invalid skill files
func TestDiscoverInPath_InvalidSkill(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skill-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create valid skill
	validContent := `---
name: valid-skill
---
Content
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte(validContent), 0644))

	// Create invalid skill (missing name)
	invalidContent := `---
description: No name here
---
Content
`
	invalidDir := filepath.Join(tmpDir, "invalid")
	require.NoError(t, os.MkdirAll(invalidDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(invalidDir, "SKILL.md"), []byte(invalidContent), 0644))

	log := zap.NewNop()
	result, err := DiscoverInPath(context.Background(), tmpDir, log)
	require.NoError(t, err)

	assert.Len(t, result.Skills, 1, "Should find 1 valid skill")
	assert.Len(t, result.Errors, 1, "Should have 1 error for invalid skill")
}

// TestXMLExport tests XML export functionality
func TestXMLExport(t *testing.T) {
	skill := &Skill{
		Name:          "test-skill",
		Description:   "A test skill",
		Type:          SkillTypeAgent,
		Arguments:     "input: string The input to process",
		Content:       "This is the skill content",
		UserInvocable: true,
	}

	xml := skill.ToPromptXML()

	assert.Contains(t, xml, `name="test-skill"`)
	assert.Contains(t, xml, `description="A test skill"`)
	assert.Contains(t, xml, `type="agent"`)
	assert.Contains(t, xml, "<arguments>")
	assert.Contains(t, xml, "<content>")
	assert.Contains(t, xml, "This is the skill content")
}

// TestSkills_ToPromptXML tests collection XML export
func TestSkills_ToPromptXML(t *testing.T) {
	skills := Skills{
		{Name: "skill-a", Description: "Skill A", Type: SkillTypeAgent, Content: "Content A"},
		{Name: "skill-b", Description: "Skill B", Type: SkillTypeMcp, Content: "Content B"},
	}

	xml := skills.ToPromptXML()

	assert.Contains(t, xml, "<skills>")
	assert.Contains(t, xml, "</skills>")
	assert.Contains(t, xml, `name="skill-a"`)
	assert.Contains(t, xml, `name="skill-b"`)
}

// TestToOpenAIFunction tests OpenAI function format conversion
func TestToOpenAIFunction(t *testing.T) {
	skill := &Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Type:        SkillTypeAgent,
		Arguments:   "query: string Search query\nlimit: number Max results",
	}

	fn := skill.ToOpenAIFunction()

	assert.Equal(t, "function", fn["type"])
	functionMap, ok := fn["function"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test-skill", functionMap["name"])
	assert.Equal(t, "A test skill", functionMap["description"])

	params, ok := functionMap["parameters"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, params, "properties")
}

// TestEscapeXML tests XML escaping
func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"a & b", "a &amp; b"},
		{"<tag>", "&lt;tag&gt;"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"'single'", "&apos;single&apos;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, escapeXML(tt.input))
		})
	}
}

// TestSkill_ID tests skill ID generation
func TestSkill_ID(t *testing.T) {
	skill := &Skill{Name: "MySkill"}
	assert.Equal(t, "myskill", skill.ID())
}

// TestSkill_String tests skill string representation
func TestSkill_String(t *testing.T) {
	skill := &Skill{
		Name:     "test",
		Type:     SkillTypeAgent,
		FilePath: "/path/to/SKILL.md",
	}

	str := skill.String()
	assert.Contains(t, str, "test")
	assert.Contains(t, str, "agent")
	assert.Contains(t, str, "/path/to/SKILL.md")
}

// Note: Service tests require DI container setup and are better suited for integration tests
// The following tests focus on unit-testable aspects

func TestNewService(t *testing.T) {
	// Create a minimal injector with required dependencies
	injector := do.New()

	// Provide logger
	do.Provide(injector, func(_ do.Injector) (*zap.Logger, error) {
		return zap.NewNop(), nil
	})

	// Provide config service
	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &mockConfigService{}, nil
	})

	service, err := NewService(injector)
	require.NoError(t, err)
	assert.NotNil(t, service)
}

// mockConfigService for testing
type mockConfigService struct{}

func (m *mockConfigService) GetWorkspaceConfig() *config.WorkspaceConfig {
	return &config.WorkspaceConfig{}
}
func (m *mockConfigService) GetLogLevel() string                                     { return "info" }
func (m *mockConfigService) IsDevMode() bool                                         { return false }
func (m *mockConfigService) GetAnthropicConfig() *config.AnthropicConfig             { return nil }
func (m *mockConfigService) GetGeminiConfig() *config.GeminiConfig                   { return nil }
func (m *mockConfigService) GetOpenAIConfig() *config.OpenAIConfig                   { return nil }
func (m *mockConfigService) GetAgentLimits() *config.AgentLimitsConfig               { return nil }
func (m *mockConfigService) GetFilesConfig() *config.FilesConfig                     { return nil }
func (m *mockConfigService) GetLoggingConfig() *config.LoggingConfig                 { return nil }
func (m *mockConfigService) GetBashConfig() *config.BashConfig                       { return nil }
func (m *mockConfigService) GetHooksConfig() *config.HooksConfig                     { return nil }
func (m *mockConfigService) GetPromptStoreConfig() *config.PromptStoreConfig         { return nil }
func (m *mockConfigService) GetPromptOptimizerConfig() *config.PromptOptimizerConfig { return nil }
func (m *mockConfigService) GetLangfuseConfig() *config.LangfuseConfig               { return nil }
func (m *mockConfigService) SetCurrentWorkspace(_ string) error                      { return nil }
func (m *mockConfigService) GetCurrentWorkspace() string                             { return "" }
func (m *mockConfigService) GetWorkspaceHistory() []string                           { return nil }

// TestSkills_ToOpenAIFunctions tests OpenAI functions conversion for collection
func TestSkills_ToOpenAIFunctions(t *testing.T) {
	skills := Skills{
		{Name: "skill-a", Description: "Skill A"},
		{Name: "skill-b", Description: "Skill B"},
	}

	functions := skills.ToOpenAIFunctions()
	assert.Len(t, functions, 2)
}

// TestSkills_ToUserInvocableXML tests user invocable XML export
func TestSkills_ToUserInvocableXML(t *testing.T) {
	skills := Skills{
		{Name: "skill-a", UserInvocable: true, Content: "Content A"},
		{Name: "skill-b", UserInvocable: false, Content: "Content B"},
		{Name: "skill-c", UserInvocable: true, Content: "Content C"},
	}

	xml := skills.ToUserInvocableXML()
	assert.Contains(t, xml, "skill-a")
	assert.Contains(t, xml, "skill-c")
	assert.NotContains(t, xml, "skill-b")
}

// TestSkill_DisplayName tests display name generation
func TestSkill_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		skill    *Skill
		expected string
	}{
		{
			name:     "with description",
			skill:    &Skill{Name: "test", Description: "Test Skill"},
			expected: "Test Skill",
		},
		{
			name:     "without description",
			skill:    &Skill{Name: "test"},
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.skill.DisplayName())
		})
	}
}

// TestSkill_Directory tests directory extraction
func TestSkill_Directory(t *testing.T) {
	skill := &Skill{FilePath: "/path/to/skills/SKILL.md"}
	assert.Equal(t, "/path/to/skills", skill.Directory())
}

// TestSkill_ToMap tests map conversion
func TestSkill_ToMap(t *testing.T) {
	skill := &Skill{
		Name:          "test",
		Description:   "Test skill",
		Type:          SkillTypeAgent,
		UserInvocable: true,
		FilePath:      "/test/SKILL.md",
	}

	m := skill.ToMap()
	assert.Equal(t, "test", m["name"])
	assert.Equal(t, "Test skill", m["description"])
	assert.Equal(t, SkillTypeAgent, m["type"])
	assert.Equal(t, true, m["user_invocable"])
	assert.Equal(t, "/test/SKILL.md", m["file_path"])
}

// TestSkills_String tests collection string representation
func TestSkills_String(t *testing.T) {
	skills := Skills{
		{Name: "skill-a"},
		{Name: "skill-b"},
	}

	str := skills.String()
	assert.Contains(t, str, "Skills[")
	assert.Contains(t, str, "skill-a")
	assert.Contains(t, str, "skill-b")
}

// TestParseArgumentsToProperties tests argument parsing
func TestParseArgumentsToProperties(t *testing.T) {
	tests := []struct {
		name          string
		args          string
		expectedProps []string
	}{
		{
			name:          "empty args",
			args:          "",
			expectedProps: []string{},
		},
		{
			name:          "single string arg",
			args:          "query: string Search query",
			expectedProps: []string{"query"},
		},
		{
			name:          "multiple args",
			args:          "query: string Search query\nlimit: number Max results",
			expectedProps: []string{"query", "limit"},
		},
		{
			name:          "various types",
			args:          "name: string Name\nage: int Age\nactive: bool Is active\nitems: array Items\nconfig: object Config",
			expectedProps: []string{"name", "age", "active", "items", "config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := parseArgumentsToProperties(tt.args)
			for _, expected := range tt.expectedProps {
				assert.Contains(t, props, expected)
			}
		})
	}
}

// TestSkill_IsToolFiltered tests tool filtering
func TestSkill_IsToolFiltered(t *testing.T) {
	tests := []struct {
		name     string
		skill    *Skill
		tool     string
		expected bool
	}{
		{
			name:     "not filtered",
			skill:    &Skill{ToolFilter: nil},
			tool:     "bash",
			expected: false,
		},
		{
			name:     "filtered",
			skill:    &Skill{ToolFilter: []string{"write", "delete"}},
			tool:     "write",
			expected: true,
		},
		{
			name:     "wildcard filter",
			skill:    &Skill{ToolFilter: []string{"*"}},
			tool:     "any",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.skill.IsToolFiltered(tt.tool))
		})
	}
}

// TestSkillService_SearchPaths tests search path management
func TestSkillService_SearchPaths(t *testing.T) {
	// Create a minimal injector with required dependencies
	injector := do.New()

	do.Provide(injector, func(_ do.Injector) (*zap.Logger, error) {
		return zap.NewNop(), nil
	})

	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &mockConfigService{}, nil
	})

	service, err := NewService(injector)
	require.NoError(t, err)

	// Test AddSearchPath
	service.AddSearchPath("/path/one")
	service.AddSearchPath("/path/two")

	// Adding duplicate should be ignored
	service.AddSearchPath("/path/one")

	paths := service.GetSearchPaths()
	// Note: Service adds current workspace as default search path
	assert.Contains(t, paths, "/path/one")
	assert.Contains(t, paths, "/path/two")

	// Test RemoveSearchPath
	service.RemoveSearchPath("/path/one")
	paths = service.GetSearchPaths()
	assert.Contains(t, paths, "/path/two")
	assert.NotContains(t, paths, "/path/one")

	// Removing non-existent should be no-op
	service.RemoveSearchPath("/nonexistent")
	paths = service.GetSearchPaths()
	assert.Contains(t, paths, "/path/two")
}

// TestSkillService_ListMethods tests service list methods
func TestSkillService_ListMethods(t *testing.T) {
	injector := do.New()

	do.Provide(injector, func(_ do.Injector) (*zap.Logger, error) {
		return zap.NewNop(), nil
	})

	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &mockConfigService{}, nil
	})

	service, err := NewService(injector)
	require.NoError(t, err)

	// Create temp skill directory
	tmpDir, err := os.MkdirTemp("", "skill-list-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create skills
	skillContent := `---
name: %s
type: %s
user_invocable: %v
---
Content
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte(fmt.Sprintf(skillContent, "list-skill", "agent", "true")), 0644))

	service.AddSearchPath(tmpDir)
	err = service.Discover(context.Background())
	require.NoError(t, err)

	// Test List
	skills := service.List()
	assert.Len(t, skills, 1)

	// Test ListByType
	agentSkills := service.ListByType(SkillTypeAgent)
	assert.Len(t, agentSkills, 1)

	mcpSkills := service.ListByType(SkillTypeMcp)
	assert.Len(t, mcpSkills, 0)

	// Test ListUserInvocable
	invocable := service.ListUserInvocable()
	assert.Len(t, invocable, 1)
}

// TestSkillService_Get tests service Get method
func TestSkillService_Get(t *testing.T) {
	injector := do.New()

	do.Provide(injector, func(_ do.Injector) (*zap.Logger, error) {
		return zap.NewNop(), nil
	})

	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &mockConfigService{}, nil
	})

	service, err := NewService(injector)
	require.NoError(t, err)

	// Create temp skill directory
	tmpDir, err := os.MkdirTemp("", "skill-get-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	skillContent := `---
name: get-skill
type: agent
---
Content
`
	skillPath := filepath.Join(tmpDir, "SKILL.md")
	require.NoError(t, os.WriteFile(skillPath, []byte(skillContent), 0644))

	service.AddSearchPath(tmpDir)
	err = service.Discover(context.Background())
	require.NoError(t, err)

	// Test Get
	skill, err := service.Get("get-skill")
	require.NoError(t, err)
	assert.Equal(t, "get-skill", skill.Name)

	// Test Get case insensitive
	skill, err = service.Get("GET-SKILL")
	require.NoError(t, err)
	assert.Equal(t, "get-skill", skill.Name)

	// Test GetByPath
	skill, err = service.GetByPath(skillPath)
	require.NoError(t, err)
	assert.Equal(t, "get-skill", skill.Name)

	// Test Get non-existent
	_, err = service.Get("nonexistent")
	assert.Error(t, err)
}

// TestSkillService_Refresh tests Refresh method
func TestSkillService_Refresh(t *testing.T) {
	injector := do.New()

	do.Provide(injector, func(_ do.Injector) (*zap.Logger, error) {
		return zap.NewNop(), nil
	})

	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &mockConfigService{}, nil
	})

	service, err := NewService(injector)
	require.NoError(t, err)

	// Refresh on empty should not error
	err = service.Refresh(context.Background())
	assert.NoError(t, err)
}

// TestDiscoverMultiple tests multiple path discovery
func TestDiscoverMultiple(t *testing.T) {
	// Create two temp directories
	tmpDir1, err := os.MkdirTemp("", "skill-multi-test-1-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir1) })

	tmpDir2, err := os.MkdirTemp("", "skill-multi-test-2-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir2) })

	// Create skills in each
	skill1Content := `---
name: skill-from-dir1
---
Content 1
`
	skill2Content := `---
name: skill-from-dir2
---
Content 2
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir1, "SKILL.md"), []byte(skill1Content), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir2, "SKILL.md"), []byte(skill2Content), 0644))

	log := zap.NewNop()
	result, err := DiscoverMultiple(context.Background(), []string{tmpDir1, tmpDir2}, log)
	require.NoError(t, err)

	assert.Len(t, result.Skills, 2)
	names := result.Skills.Names()
	assert.ElementsMatch(t, []string{"skill-from-dir1", "skill-from-dir2"}, names)
}

// TestDiscoverMultiple_DuplicateDetection tests duplicate skill detection
func TestDiscoverMultiple_DuplicateDetection(t *testing.T) {
	// Create two temp directories
	tmpDir1, err := os.MkdirTemp("", "skill-dup-test-1-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir1) })

	tmpDir2, err := os.MkdirTemp("", "skill-dup-test-2-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir2) })

	// Create same skill name in both
	skillContent := `---
name: duplicate-skill
---
Content
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir1, "SKILL.md"), []byte(skillContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir2, "SKILL.md"), []byte(skillContent), 0644))

	log := zap.NewNop()
	result, err := DiscoverMultiple(context.Background(), []string{tmpDir1, tmpDir2}, log)
	require.NoError(t, err)

	// Should only have 1 skill, with 1 error for duplicate
	assert.Len(t, result.Skills, 1)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "duplicate")
}
