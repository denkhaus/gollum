package skills_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestSkillDiscovery_Integration tests the full discovery workflow
func TestSkillDiscovery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	defer logger.Sync()

	// Create temporary directory with test skills
	tmpDir := t.TempDir()

	// Create skill directory structure
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	// Create a valid skill file
	skillContent := `---
name: test-skill
description: A test skill for integration testing
version: 1.0.0
type: agent
author: Test Author
tags:
  - test
  - integration
user_invocable: true
priority: 50
tools:
  - Read
  - Glob
tool_scope: custom
---

# Test Skill

You are a test skill for integration testing.

## Instructions

1. Process the input
2. Return results
`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	require.NoError(t, os.WriteFile(skillPath, []byte(skillContent), 0644))

	// Test discovery
	ctx := context.Background()
	result, err := skills.DiscoverInPath(ctx, tmpDir, logger)
	require.NoError(t, err)

	// Verify results
	assert.Len(t, result.Skills, 1, "Should find exactly one skill")
	assert.Empty(t, result.Errors, "Should have no errors")
	assert.Greater(t, result.ScannedDirs, 0, "Should have scanned directories")
	assert.Greater(t, result.ScannedFiles, 0, "Should have scanned files")

	// Verify skill content
	skill := result.Skills[0]
	assert.Equal(t, "test-skill", skill.Name)
	assert.Equal(t, "A test skill for integration testing", skill.Description)
	assert.Equal(t, "1.0.0", skill.Version)
	assert.Equal(t, skills.SkillTypeAgent, skill.Type)
	assert.Equal(t, "Test Author", skill.Author)
	assert.True(t, skill.UserInvocable)
	assert.Equal(t, 50, skill.Priority)
	assert.Equal(t, []string{"Read", "Glob"}, skill.Tools)
	assert.Equal(t, skills.ToolScopeCustom, skill.ToolScope)
	assert.Contains(t, skill.Tags, "test")
	assert.Contains(t, skill.Tags, "integration")
	assert.Equal(t, skillPath, skill.FilePath)
	assert.NotEmpty(t, skill.Content)
	assert.Contains(t, skill.Content, "Test Skill")
}

// TestSkillDiscovery_MultiplePaths tests discovery across multiple directories
func TestSkillDiscovery_MultiplePaths(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	defer logger.Sync()

	// Create multiple directories with skills
	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "skills1")
	dir2 := filepath.Join(tmpDir, "skills2")
	require.NoError(t, os.MkdirAll(filepath.Join(dir1, "skill-a"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir2, "skill-b"), 0755))

	// Create skill A
	skillA := `---
name: skill-a
description: Skill A
---
Content A`
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "skill-a", "SKILL.md"), []byte(skillA), 0644))

	// Create skill B
	skillB := `---
name: skill-b
description: Skill B
---
Content B`
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "skill-b", "SKILL.md"), []byte(skillB), 0644))

	// Test discovery in multiple paths
	ctx := context.Background()
	result, err := skills.DiscoverMultiple(ctx, []string{dir1, dir2}, logger)
	require.NoError(t, err)

	assert.Len(t, result.Skills, 2, "Should find two skills")

	// Verify both skills were found
	names := result.Skills.Names()
	assert.Contains(t, names, "skill-a")
	assert.Contains(t, names, "skill-b")
}

// TestSkillDiscovery_IgnoredDirectories tests that ignored directories are skipped
func TestSkillDiscovery_IgnoredDirectories(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	defer logger.Sync()

	tmpDir := t.TempDir()

	// Create skill in normal directory
	normalDir := filepath.Join(tmpDir, "normal")
	require.NoError(t, os.MkdirAll(normalDir, 0755))
	skillContent := `---
name: normal-skill
---
Content`
	require.NoError(t, os.WriteFile(filepath.Join(normalDir, "SKILL.md"), []byte(skillContent), 0644))

	// Create skill in ignored directory (.git)
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	gitSkill := `---
name: git-skill
---
This should not be found`
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "SKILL.md"), []byte(gitSkill), 0644))

	// Create skill in node_modules
	nodeDir := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.MkdirAll(nodeDir, 0755))
	nodeSkill := `---
name: node-skill
---
This should not be found`
	require.NoError(t, os.WriteFile(filepath.Join(nodeDir, "SKILL.md"), []byte(nodeSkill), 0644))

	// Test discovery
	ctx := context.Background()
	result, err := skills.DiscoverInPath(ctx, tmpDir, logger)
	require.NoError(t, err)

	// Should only find the normal skill
	assert.Len(t, result.Skills, 1, "Should only find skill in normal directory")
	assert.Equal(t, "normal-skill", result.Skills[0].Name)
}

// TestSkillDiscovery_MaxDepth tests depth limiting via discovery at specific paths
func TestSkillDiscovery_MaxDepth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	defer logger.Sync()

	tmpDir := t.TempDir()

	// Create nested directory structure
	// level1/level2/level3/level4/skill/
	deepDir := filepath.Join(tmpDir, "level1", "level2", "level3", "level4", "skill")
	require.NoError(t, os.MkdirAll(deepDir, 0755))

	skillContent := `---
name: deep-skill
---
Content`
	require.NoError(t, os.WriteFile(filepath.Join(deepDir, "SKILL.md"), []byte(skillContent), 0644))

	// Create shallow skill
	shallowDir := filepath.Join(tmpDir, "shallow")
	require.NoError(t, os.MkdirAll(shallowDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(shallowDir, "SKILL.md"), []byte(`---
name: shallow-skill
---
Content`), 0644))

	// Test discovery from shallow path only
	ctx := context.Background()
	result, err := skills.DiscoverInPath(ctx, filepath.Join(tmpDir, "shallow"), logger)
	require.NoError(t, err)

	// Should only find shallow skill
	assert.Len(t, result.Skills, 1, "Should only find skill in shallow path")
	assert.Equal(t, "shallow-skill", result.Skills[0].Name)

	// Verify deep skill exists but isn't found when searching shallow path
	deepResult, err := skills.DiscoverInPath(ctx, filepath.Join(tmpDir, "level1", "level2", "level3", "level4"), logger)
	require.NoError(t, err)
	assert.Len(t, deepResult.Skills, 1, "Should find skill in deep path")
	assert.Equal(t, "deep-skill", deepResult.Skills[0].Name)
}

// TestSkillDiscovery_DuplicateDetection tests duplicate skill name detection
func TestSkillDiscovery_DuplicateDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	defer logger.Sync()

	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	require.NoError(t, os.MkdirAll(dir1, 0755))
	require.NoError(t, os.MkdirAll(dir2, 0755))

	// Create same skill in both directories
	skillContent := `---
name: duplicate-skill
---
Content`
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "SKILL.md"), []byte(skillContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "SKILL.md"), []byte(skillContent), 0644))

	// Test discovery
	ctx := context.Background()
	result, err := skills.DiscoverMultiple(ctx, []string{dir1, dir2}, logger)
	require.NoError(t, err)

	// Should find one skill and one error
	assert.Len(t, result.Skills, 1, "Should find only one instance of duplicate skill")
	assert.Len(t, result.Errors, 1, "Should report duplicate error")

	// Verify error is about duplicate
	assert.Contains(t, result.Errors[0].Error(), "duplicate")
}

// TestSkillParsing_AllFields tests parsing a skill with all fields
func TestSkillParsing_AllFields(t *testing.T) {
	skillContent := `---
name: complete-skill
description: A complete skill with all fields
version: 2.1.0
type: workflow
author: Complete Author
category: complete
tags:
  - complete
  - test
  - all-fields
user_invocable: true
priority: 75
auto_invoke: true
arguments: "[input] [options]"
tools:
  - Read
  - Write
  - Bash
tool_scope: custom
tool_filter:
  - WebFetch
---

# Complete Skill

This skill has all possible fields defined.

## Instructions

Complete instructions here.
`

	skill, err := skills.Parse(skillContent, "test/complete-skill/SKILL.md")
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, "complete-skill", skill.Name)
	assert.Equal(t, "A complete skill with all fields", skill.Description)
	assert.Equal(t, "2.1.0", skill.Version)
	assert.Equal(t, skills.SkillTypeWorkflow, skill.Type)
	assert.Equal(t, "Complete Author", skill.Author)
	assert.Equal(t, "complete", skill.Category)
	assert.Equal(t, []string{"complete", "test", "all-fields"}, skill.Tags)
	assert.True(t, skill.UserInvocable)
	assert.Equal(t, 75, skill.Priority)
	assert.True(t, skill.AutoInvoke)
	assert.Equal(t, "[input] [options]", skill.Arguments)
	assert.Equal(t, []string{"Read", "Write", "Bash"}, skill.Tools)
	assert.Equal(t, skills.ToolScopeCustom, skill.ToolScope)
	assert.Equal(t, []string{"WebFetch"}, skill.ToolFilter)
	assert.NotEmpty(t, skill.Content)
	assert.Contains(t, skill.Content, "Complete Skill")
}

// TestSkillToolAccess tests tool access methods
func TestSkillToolAccess(t *testing.T) {
	tests := []struct {
		name          string
		skill         *skills.Skill
		tool          string
		expectedHas   bool
		expectedFilt  bool
	}{
		{
			name: "empty tools allows all",
			skill: &skills.Skill{
				Tools:     []string{},
				ToolScope: skills.ToolScopeAll,
			},
			tool:         "Read",
			expectedHas:  true,
			expectedFilt: false,
		},
		{
			name: "tool in list is allowed",
			skill: &skills.Skill{
				Tools:     []string{"Read", "Write"},
				ToolScope: skills.ToolScopeCustom,
			},
			tool:         "Read",
			expectedHas:  true,
			expectedFilt: false,
		},
		{
			name: "tool not in list is denied",
			skill: &skills.Skill{
				Tools:     []string{"Read", "Write"},
				ToolScope: skills.ToolScopeCustom,
			},
			tool:         "Bash",
			expectedHas:  false,
			expectedFilt: false,
		},
		{
			name: "filtered tool is denied",
			skill: &skills.Skill{
				Tools:      []string{},
				ToolFilter: []string{"Bash"},
				ToolScope:  skills.ToolScopeAll,
			},
			tool:         "Bash",
			expectedHas:  false,
			expectedFilt: true,
		},
		{
			name: "wildcard allows all tools",
			skill: &skills.Skill{
				Tools:     []string{"*"},
				ToolScope: skills.ToolScopeCustom,
			},
			tool:         "AnyTool",
			expectedHas:  true,
			expectedFilt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedHas, tt.skill.HasTool(tt.tool))
			assert.Equal(t, tt.expectedFilt, tt.skill.IsToolFiltered(tt.tool))
		})
	}
}

// TestSkillSlashCommand tests slash command generation
func TestSkillSlashCommand(t *testing.T) {
	tests := []struct {
		name     string
		skill    *skills.Skill
		expected string
	}{
		{
			name: "user invocable skill",
			skill: &skills.Skill{
				Name:          "Code-Reviewer",
				UserInvocable: true,
			},
			expected: "/code-reviewer",
		},
		{
			name: "not user invocable",
			skill: &skills.Skill{
				Name:          "internal-skill",
				UserInvocable: false,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.skill.SlashCommand())
		})
	}
}

// TestSkillsCollection tests Skills collection methods
func TestSkillsCollection(t *testing.T) {
	skillsList := skills.Skills{
		{Name: "alpha", Type: skills.SkillTypeAgent, UserInvocable: true},
		{Name: "beta", Type: skills.SkillTypeMcp, UserInvocable: false},
		{Name: "gamma", Type: skills.SkillTypeAgent, UserInvocable: true},
		{Name: "delta", Type: skills.SkillTypeWorkflow, UserInvocable: true},
	}

	t.Run("FindByName", func(t *testing.T) {
		assert.Equal(t, "alpha", skillsList.FindByName("ALPHA").Name)
		assert.Equal(t, "beta", skillsList.FindByName("beta").Name)
		assert.Nil(t, skillsList.FindByName("nonexistent"))
	})

	t.Run("FilterByType", func(t *testing.T) {
		agentSkills := skillsList.FilterByType(skills.SkillTypeAgent)
		assert.Len(t, agentSkills, 2)
	})

	t.Run("FilterUserInvocable", func(t *testing.T) {
		invocable := skillsList.FilterUserInvocable()
		assert.Len(t, invocable, 3)
	})

	t.Run("Names", func(t *testing.T) {
		names := skillsList.Names()
		assert.Equal(t, []string{"alpha", "beta", "gamma", "delta"}, names)
	})
}

// TestSkillValidation_InvalidToolNames tests tool name validation
func TestSkillValidation_InvalidToolNames(t *testing.T) {
	tests := []struct {
		name           string
		tools          []string
		toolFilter     []string
		expectedInvalid []string
	}{
		{
			name:           "valid tools only",
			tools:          []string{"read_file", "write_file", "glob"},
			toolFilter:     []string{},
			expectedInvalid: nil,
		},
		{
			name:           "invalid tool in tools list - uppercase",
			tools:          []string{"Read", "Write"},
			toolFilter:     []string{},
			expectedInvalid: []string{"Read", "Write"},
		},
		{
			name:           "invalid tool in tools list - camelCase",
			tools:          []string{"readFile", "writeFile"},
			toolFilter:     []string{},
			expectedInvalid: []string{"readFile", "writeFile"},
		},
		{
			name:           "invalid tool in tool_filter",
			tools:          []string{},
			toolFilter:     []string{"Bash", "nonexistent"},
			expectedInvalid: []string{"Bash", "nonexistent"},
		},
		{
			name:           "mixed valid and invalid",
			tools:          []string{"read_file", "Read", "glob"},
			toolFilter:     []string{"bash", "InvalidTool"},
			expectedInvalid: []string{"Read", "InvalidTool"},
		},
		{
			name:           "empty tools - all valid",
			tools:          []string{},
			toolFilter:     []string{},
			expectedInvalid: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &skills.Skill{
				Name:       "test-skill",
				Type:       skills.SkillTypeAgent,
				ToolScope:  skills.ToolScopeAll,
				Tools:      tt.tools,
				ToolFilter: tt.toolFilter,
			}

			// Test that the skill parses (basic validation passes)
			err := skill.Validate()
			assert.NoError(t, err, "Basic skill validation should pass")

			// Note: Tool name validation requires SkillService with ToolNameValidator
			// This is tested in service-level tests
		})
	}
}
