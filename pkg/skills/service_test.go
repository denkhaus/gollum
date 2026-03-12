package skills_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/events"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// TestSkillType_Validity tests SkillType validation
func TestSkillType_Validity(t *testing.T) {
	tests := []struct {
		name      string
		skillType skills.SkillType
		expected  bool
	}{
		{"agent type", skills.SkillTypeAgent, true},
		{"mcp type", skills.SkillTypeMcp, true},
		{"workflow type", skills.SkillTypeWorkflow, true},
		{"invalid type", skills.SkillType("invalid"), false},
		{"empty type", skills.SkillType(""), false},
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
		expected  skills.SkillType
		expectErr bool
	}{
		{"agent", "agent", skills.SkillTypeAgent, false},
		{"agent uppercase", "AGENT", skills.SkillTypeAgent, false},
		{"mcp", "mcp", skills.SkillTypeMcp, false},
		{"workflow", "workflow", skills.SkillTypeWorkflow, false},
		{"invalid", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := skills.ParseSkillType(tt.input)
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
		toolScope skills.ToolScope
		expected  bool
	}{
		{"all scope", skills.ToolScopeAll, true},
		{"read-only scope", skills.ToolScopeReadOnly, true},
		{"none scope", skills.ToolScopeNone, true},
		{"custom scope", skills.ToolScopeCustom, true},
		{"invalid scope", skills.ToolScope("invalid"), false},
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

	skill, err := skills.Parse(content, "/test/SKILL.md")
	require.NoError(t, err)

	assert.Equal(t, "test-skill", skill.Name)
	assert.Equal(t, "A test skill", skill.Description)
	assert.Equal(t, "1.0", skill.Version)
	assert.Equal(t, skills.SkillTypeAgent, skill.Type)
	assert.Equal(t, skills.ToolScopeAll, skill.ToolScope)
	assert.True(t, skill.UserInvocable)
	assert.ElementsMatch(t, []string{"bash", "read"}, skill.Tools)
	assert.ElementsMatch(t, []string{"test", "example"}, skill.Tags)
	assert.Contains(t, skill.Content, "This is the skill content")
}

// TestParse_MissingFrontmatter tests parsing without frontmatter
func TestParse_MissingFrontmatter(t *testing.T) {
	content := "This is just content without frontmatter"

	_, err := skills.Parse(content, "/test/SKILL.md")
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

	_, err := skills.Parse(content, "/test/SKILL.md")
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

	_, err := skills.Parse(content, "/test/SKILL.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

// TestSkill_Validate tests skill validation
func TestSkill_Validate(t *testing.T) {
	tests := []struct {
		name        string
		skill       *skills.Skill
		expectErr   bool
		errContains string
	}{
		{
			name:      "valid skill",
			skill:     &skills.Skill{Name: "test", Type: skills.SkillTypeAgent, ToolScope: skills.ToolScopeAll, FilePath: "/test"},
			expectErr: false,
		},
		{
			name:        "missing name",
			skill:       &skills.Skill{Type: skills.SkillTypeAgent, ToolScope: skills.ToolScopeAll, FilePath: "/test"},
			expectErr:   true,
			errContains: "name",
		},
		{
			name:        "name with spaces",
			skill:       &skills.Skill{Name: "test skill", Type: skills.SkillTypeAgent, ToolScope: skills.ToolScopeAll, FilePath: "/test"},
			expectErr:   true,
			errContains: "name",
		},
		{
			name:        "invalid type",
			skill:       &skills.Skill{Name: "test", Type: skills.SkillType("bad"), ToolScope: skills.ToolScopeAll, FilePath: "/test"},
			expectErr:   true,
			errContains: "type",
		},
		{
			name:        "invalid tool scope",
			skill:       &skills.Skill{Name: "test", Type: skills.SkillTypeAgent, ToolScope: skills.ToolScope("bad"), FilePath: "/test"},
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
		skill    *skills.Skill
		toolName string
		expected bool
	}{
		{
			name:     "no tools specified - all allowed",
			skill:    &skills.Skill{Tools: nil, ToolFilter: nil},
			toolName: "bash",
			expected: true,
		},
		{
			name:     "tool in allowed list",
			skill:    &skills.Skill{Tools: []string{"bash", "read"}, ToolFilter: nil},
			toolName: "bash",
			expected: true,
		},
		{
			name:     "tool not in allowed list",
			skill:    &skills.Skill{Tools: []string{"bash", "read"}, ToolFilter: nil},
			toolName: "write",
			expected: false,
		},
		{
			name:     "tool filtered out",
			skill:    &skills.Skill{Tools: nil, ToolFilter: []string{"write"}},
			toolName: "write",
			expected: false,
		},
		{
			name:     "wildcard in tools",
			skill:    &skills.Skill{Tools: []string{"*"}, ToolFilter: nil},
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
		skill    *skills.Skill
		expected string
	}{
		{
			name:     "user invocable skill",
			skill:    &skills.Skill{Name: "MySkill", UserInvocable: true},
			expected: "/myskill",
		},
		{
			name:     "not user invocable",
			skill:    &skills.Skill{Name: "MySkill", UserInvocable: false},
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
	testSkills := skills.Skills{
		{Name: "skill-a", Type: skills.SkillTypeAgent, UserInvocable: true},
		{Name: "skill-b", Type: skills.SkillTypeMcp, UserInvocable: false},
		{Name: "skill-c", Type: skills.SkillTypeAgent, UserInvocable: true},
	}

	t.Run("FindByName", func(t *testing.T) {
		assert.NotNil(t, testSkills.FindByName("skill-a"))
		assert.NotNil(t, testSkills.FindByName("SKILL-A")) // case insensitive
		assert.Nil(t, testSkills.FindByName("nonexistent"))
	})

	t.Run("FilterByType", func(t *testing.T) {
		agentSkills := testSkills.FilterByType(skills.SkillTypeAgent)
		assert.Len(t, agentSkills, 2)
	})

	t.Run("FilterUserInvocable", func(t *testing.T) {
		invocable := testSkills.FilterUserInvocable()
		assert.Len(t, invocable, 2)
	})

	t.Run("Names", func(t *testing.T) {
		names := testSkills.Names()
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
	result, err := skills.DiscoverInPath(context.Background(), tmpDir, log)
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

	// Create valid skill in its own folder
	validDir := filepath.Join(tmpDir, "valid-skill")
	require.NoError(t, os.MkdirAll(validDir, 0755))

	validContent := `---
name: valid-skill
---
Content
`
	require.NoError(t, os.WriteFile(filepath.Join(validDir, "SKILL.md"), []byte(validContent), 0644))

	// Create invalid skill (missing name) in its own folder
	invalidContent := `---
description: No name here
---
Content
`
	invalidDir := filepath.Join(tmpDir, "invalid")
	require.NoError(t, os.MkdirAll(invalidDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(invalidDir, "SKILL.md"), []byte(invalidContent), 0644))

	log := zap.NewNop()
	result, err := skills.DiscoverInPath(context.Background(), tmpDir, log)
	require.NoError(t, err)

	assert.Len(t, result.Skills, 1, "Should find 1 valid skill")
	assert.Len(t, result.Errors, 1, "Should have 1 error for invalid skill")
}

// TestXMLExport tests XML export functionality
func TestXMLExport(t *testing.T) {
	skill := &skills.Skill{
		Name:          "test-skill",
		Description:   "A test skill",
		Type:          skills.SkillTypeAgent,
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
	testSkills := skills.Skills{
		{Name: "skill-a", Description: "Skill A", Type: skills.SkillTypeAgent, Content: "Content A"},
		{Name: "skill-b", Description: "Skill B", Type: skills.SkillTypeMcp, Content: "Content B"},
	}

	xml := testSkills.ToPromptXML()

	assert.Contains(t, xml, "<skills>")
	assert.Contains(t, xml, "</skills>")
	assert.Contains(t, xml, `name="skill-a"`)
	assert.Contains(t, xml, `name="skill-b"`)
}

// TestToOpenAIFunction tests OpenAI function format conversion
func TestToOpenAIFunction(t *testing.T) {
	skill := &skills.Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Type:        skills.SkillTypeAgent,
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

// TestSkill_ID tests skill ID generation
func TestSkill_ID(t *testing.T) {
	skill := &skills.Skill{Name: "MySkill"}
	assert.Equal(t, "myskill", skill.ID())
}

// TestSkill_String tests skill string representation
func TestSkill_String(t *testing.T) {
	skill := &skills.Skill{
		Name:     "test",
		Type:     skills.SkillTypeAgent,
		FilePath: "/path/to/SKILL.md",
	}

	str := skill.String()
	assert.Contains(t, str, "test")
	assert.Contains(t, str, "agent")
	assert.Contains(t, str, "/path/to/SKILL.md")
}

// setupTestService creates a skill service with centralized mocks for testing
func setupTestService(t *testing.T) skills.SkillService {
	ctrl := gomock.NewController(t)

	mockWorkspace := mocks.NewMockService(ctrl)
	mockBus := mocks.NewMockBus(ctrl)
	mockLogger := mocks.NewMockLoggerService(ctrl)

	// Setup default expectations
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
	mockWorkspace.EXPECT().GetCurrentWorkspace().Return("").AnyTimes()
	mockWorkspace.EXPECT().GetWorkspaceHistory().Return(nil).AnyTimes()
	mockBus.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Return("test-sub", nil).AnyTimes()
	mockBus.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[workspace.Service](injector, mockWorkspace)
	do.ProvideValue[events.Bus](injector, mockBus)

	service, err := skills.NewService(injector)
	require.NoError(t, err)

	return service
}

// TestNewService tests service creation with centralized mocks
func TestNewService(t *testing.T) {
	service := setupTestService(t)
	require.NotNil(t, service)
}

// TestSkills_ToOpenAIFunctions tests OpenAI functions conversion for collection
func TestSkills_ToOpenAIFunctions(t *testing.T) {
	testSkills := skills.Skills{
		{Name: "skill-a", Description: "Skill A"},
		{Name: "skill-b", Description: "Skill B"},
	}

	functions := testSkills.ToOpenAIFunctions()
	assert.Len(t, functions, 2)
}

// TestSkills_ToUserInvocableXML tests user invocable XML export
func TestSkills_ToUserInvocableXML(t *testing.T) {
	testSkills := skills.Skills{
		{Name: "skill-a", UserInvocable: true, Content: "Content A"},
		{Name: "skill-b", UserInvocable: false, Content: "Content B"},
		{Name: "skill-c", UserInvocable: true, Content: "Content C"},
	}

	xml := testSkills.ToUserInvocableXML()
	assert.Contains(t, xml, "skill-a")
	assert.Contains(t, xml, "skill-c")
	assert.NotContains(t, xml, "skill-b")
}

// TestSkill_DisplayName tests display name generation
func TestSkill_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		skill    *skills.Skill
		expected string
	}{
		{
			name:     "with description",
			skill:    &skills.Skill{Name: "test", Description: "Test Skill"},
			expected: "Test Skill",
		},
		{
			name:     "without description",
			skill:    &skills.Skill{Name: "test"},
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
	skill := &skills.Skill{FilePath: "/path/to/skills/SKILL.md"}
	assert.Equal(t, "/path/to/skills", skill.Directory())
}

// TestSkill_ToMap tests map conversion
func TestSkill_ToMap(t *testing.T) {
	skill := &skills.Skill{
		Name:          "test",
		Description:   "Test skill",
		Type:          skills.SkillTypeAgent,
		UserInvocable: true,
		FilePath:      "/test/SKILL.md",
	}

	m := skill.ToMap()
	assert.Equal(t, "test", m["name"])
	assert.Equal(t, "Test skill", m["description"])
	assert.Equal(t, skills.SkillTypeAgent, m["type"])
	assert.Equal(t, true, m["user_invocable"])
	assert.Equal(t, "/test/SKILL.md", m["file_path"])
}

// TestSkills_String tests collection string representation
func TestSkills_String(t *testing.T) {
	testSkills := skills.Skills{
		{Name: "skill-a"},
		{Name: "skill-b"},
	}

	str := testSkills.String()
	assert.Contains(t, str, "Skills[")
	assert.Contains(t, str, "skill-a")
	assert.Contains(t, str, "skill-b")
}

// TestSkill_IsToolFiltered tests tool filtering
func TestSkill_IsToolFiltered(t *testing.T) {
	tests := []struct {
		name     string
		skill    *skills.Skill
		tool     string
		expected bool
	}{
		{
			name:     "not filtered",
			skill:    &skills.Skill{ToolFilter: nil},
			tool:     "bash",
			expected: false,
		},
		{
			name:     "filtered",
			skill:    &skills.Skill{ToolFilter: []string{"write", "delete"}},
			tool:     "write",
			expected: true,
		},
		{
			name:     "wildcard filter",
			skill:    &skills.Skill{ToolFilter: []string{"*"}},
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
	service := setupTestService(t)

	// Test AddSearchPath
	service.AddSearchPath("/path/one")
	service.AddSearchPath("/path/two")

	// Adding duplicate should be ignored
	service.AddSearchPath("/path/one")

	paths := service.GetSearchPaths()
	assert.Contains(t, paths, "/path/one")
	assert.Contains(t, paths, "/path/two")

	// Test RemoveSearchPath
	service.RemoveSearchPath("/path/one")
	paths = service.GetSearchPaths()
	assert.Contains(t, paths, "/path/two")
	assert.NotContains(t, paths, "/path/one")
}

// TestSkillService_ListMethods tests service list methods
func TestSkillService_ListMethods(t *testing.T) {
	service := setupTestService(t)

	// Create temp skill directory
	tmpDir, err := os.MkdirTemp("", "skill-list-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create skill in its own folder
	skillFolder := filepath.Join(tmpDir, "list-skill")
	require.NoError(t, os.MkdirAll(skillFolder, 0755))

	skillContent := `---
name: list-skill
type: agent
user_invocable: true
---
Content
`
	require.NoError(t, os.WriteFile(filepath.Join(skillFolder, "SKILL.md"), []byte(skillContent), 0644))

	service.AddSearchPath(tmpDir)
	err = service.Discover(context.Background())
	require.NoError(t, err)

	// Test List - should include our test skill (may also include global skills)
	testSkills := service.List()
	assert.NotNil(t, testSkills.FindByName("list-skill"), "test skill should be in the list")

	// Test ListByType - our test skill should be in agent skills
	agentSkills := service.ListByType(skills.SkillTypeAgent)
	assert.NotNil(t, agentSkills.FindByName("list-skill"), "test skill should be in agent skills")

	// Test ListUserInvocable - our test skill should be invocable
	invocable := service.ListUserInvocable()
	assert.NotNil(t, invocable.FindByName("list-skill"), "test skill should be user invocable")
}

// TestSkillService_Get tests service Get method
func TestSkillService_Get(t *testing.T) {
	service := setupTestService(t)

	// Create temp skill directory
	tmpDir, err := os.MkdirTemp("", "skill-get-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create skill in its own folder
	skillFolder := filepath.Join(tmpDir, "get-skill")
	require.NoError(t, os.MkdirAll(skillFolder, 0755))

	skillContent := `---
name: get-skill
type: agent
---
Content
`
	skillPath := filepath.Join(skillFolder, "SKILL.md")
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
	service := setupTestService(t)

	// Refresh on empty should not error
	err := service.Refresh(context.Background())
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

	// Create skills in each (in subfolders)
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

	// Create skill folders
	skillFolder1 := filepath.Join(tmpDir1, "skill1")
	require.NoError(t, os.MkdirAll(skillFolder1, 0755))
	skillFolder2 := filepath.Join(tmpDir2, "skill2")
	require.NoError(t, os.MkdirAll(skillFolder2, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(skillFolder1, "SKILL.md"), []byte(skill1Content), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillFolder2, "SKILL.md"), []byte(skill2Content), 0644))

	log := zap.NewNop()
	result, err := skills.DiscoverMultiple(context.Background(), []string{tmpDir1, tmpDir2}, log)
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

	// Create same skill name in both (in subfolders to pass folder validation)
	skillContent := `---
name: duplicate-skill
---
Content
`

	// Create skill folders
	skillFolder1 := filepath.Join(tmpDir1, "my-skill")
	require.NoError(t, os.MkdirAll(skillFolder1, 0755))
	skillFolder2 := filepath.Join(tmpDir2, "my-skill")
	require.NoError(t, os.MkdirAll(skillFolder2, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(skillFolder1, "SKILL.md"), []byte(skillContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillFolder2, "SKILL.md"), []byte(skillContent), 0644))

	log := zap.NewNop()
	result, err := skills.DiscoverMultiple(context.Background(), []string{tmpDir1, tmpDir2}, log)
	require.NoError(t, err)

	// Should only have 1 skill, with 1 error for duplicate
	assert.Len(t, result.Skills, 1)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "duplicate")
}

// TestSkillService_AutoRemoveEmptySearchPaths tests that search paths with no skills are removed after discovery
func TestSkillService_AutoRemoveEmptySearchPaths(t *testing.T) {
	service := setupTestService(t)

	// Create temp directories
	dirWithSkill, err := os.MkdirTemp("", "skill-with-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dirWithSkill) })

	dirWithoutSkill, err := os.MkdirTemp("", "skill-without-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dirWithoutSkill) })

	// Create skill in one directory (in subfolder)
	skillFolder := filepath.Join(dirWithSkill, "test-skill")
	require.NoError(t, os.MkdirAll(skillFolder, 0755))
	skillContent := `---
name: test-skill
type: agent
---
Content
`
	require.NoError(t, os.WriteFile(filepath.Join(skillFolder, "SKILL.md"), []byte(skillContent), 0644))

	// Add both paths
	service.AddSearchPath(dirWithSkill)
	service.AddSearchPath(dirWithoutSkill)

	// Verify both paths are present before discovery
	pathsBefore := service.GetSearchPaths()
	assert.Contains(t, pathsBefore, dirWithSkill)
	assert.Contains(t, pathsBefore, dirWithoutSkill)

	// Trigger discovery
	err = service.Discover(context.Background())
	require.NoError(t, err)

	// Verify skill was found
	skill, err := service.Get("test-skill")
	require.NoError(t, err)
	assert.Equal(t, "test-skill", skill.Name)

	// Verify empty path was removed, path with skill was kept
	pathsAfter := service.GetSearchPaths()
	assert.Contains(t, pathsAfter, dirWithSkill)
	assert.NotContains(t, pathsAfter, dirWithoutSkill)
}

// TestDiscoverMultiple_PathsWithSkills tests that PathsWithSkills is correctly populated
func TestDiscoverMultiple_PathsWithSkills(t *testing.T) {
	// Create temp directories
	dirWithSkill, err := os.MkdirTemp("", "skill-path-with-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dirWithSkill) })

	dirWithoutSkill, err := os.MkdirTemp("", "skill-path-without-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dirWithoutSkill) })

	// Create skill in one directory (in subfolder)
	skillFolder := filepath.Join(dirWithSkill, "my-skill")
	require.NoError(t, os.MkdirAll(skillFolder, 0755))
	skillContent := `---
name: my-skill
---
Content
`
	require.NoError(t, os.WriteFile(filepath.Join(skillFolder, "SKILL.md"), []byte(skillContent), 0644))

	log := zap.NewNop()
	result, err := skills.DiscoverMultiple(context.Background(), []string{dirWithSkill, dirWithoutSkill}, log)
	require.NoError(t, err)

	// Verify PathsWithSkills is populated correctly
	assert.NotNil(t, result.PathsWithSkills)
	assert.True(t, result.PathsWithSkills[dirWithSkill], "Path with skill should be marked true")
	assert.False(t, result.PathsWithSkills[dirWithoutSkill], "Path without skill should be marked false")

	// Verify skill was found
	assert.Len(t, result.Skills, 1)
	assert.Equal(t, "my-skill", result.Skills[0].Name)
}
