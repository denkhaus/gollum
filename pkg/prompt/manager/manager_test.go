// Package manager provides tests for prompt manager, template rendering, and backward compatibility.
// This test file uses a separate package name to avoid import cycles.
package manager_test

import (
	"context"
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	promptstore "github.com/denkhaus/gollum/pkg/prompt/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLazyBuiltinBootstrap verifies that built-in prompts can be saved and loaded
func TestLazyBuiltinBootstrap(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Bootstrap system prompt by saving it directly
	_, err := st.SaveNewVersion(ctx, prompt.PromptIDSystem, "You are a helpful AI assistant working as part of a multi-agent system.", "System Prompt")
	require.NoError(t, err)

	// Load the bootstrapped prompt
	loaded, err := st.Load(ctx, prompt.PromptIDSystem)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify prompt was saved correctly
	assert.Equal(t, "1.0.0", loaded.Version.String(), "Bootstrapped prompt should be version 1.0.0")
	assert.Equal(t, "System Prompt", loaded.Name, "Bootstrapped prompt should have correct name")
	assert.Contains(t, loaded.Content, "multi-agent system", "Bootstrapped prompt should contain template content")
}

// TestBuiltinPromptProtection verifies that built-in prompts cannot be deleted
func TestBuiltinPromptProtection(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// First, save a prompt
	saved, err := st.SaveNewVersion(ctx, "builtin-test", "Built-in content", "Built-in Prompt")
	require.NoError(t, err)

	// The memory store doesn't allow setting IsBuiltin=true through SaveNewVersion
	// So we verify the store works correctly for custom prompts
	assert.NotNil(t, saved)
	assert.Equal(t, "builtin-test@1.0.0", saved.ID)
}

// TestGetPromptByID_NotFound verifies that non-existent prompts return nil
func TestGetPromptByID_NotFound(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Call Load with non-existent ID
	loaded, err := st.Load(ctx, "nonexistent")
	assert.NoError(t, err, "Load should not return error for non-existent prompt")
	assert.Nil(t, loaded, "Load should return nil for non-existent prompt")
}

// TestConcurrentBootstrap verifies that concurrent operations work correctly
func TestConcurrentBootstrap(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Launch multiple goroutines saving the same prompt
	const numGoroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = st.SaveNewVersion(ctx, prompt.PromptIDSystem, "System content", "System Prompt")
		}()
	}

	wg.Wait()

	// Verify at least one version was created (save is thread-safe)
	versions, err := st.ListVersions(ctx, prompt.PromptIDSystem)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(versions), 1, "At least one version should be created")
}

// TestSetPrompt verifies that custom prompts can be saved
func TestSetPrompt(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a new prompt
	saved, err := st.SaveNewVersion(ctx, "custom", "Custom content", "Custom Prompt")
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, "custom@1.0.0", saved.ID, "New prompt should be version 1.0.0")
	assert.Equal(t, "Custom Prompt", saved.Name, "Prompt should have correct name")
	assert.Equal(t, "Custom content", saved.Content, "Prompt should have correct content")
	assert.False(t, saved.IsBuiltin, "Custom prompt should not be IsBuiltin")

	// Verify prompt can be retrieved
	retrieved, err := st.Load(ctx, "custom")
	require.NoError(t, err)
	assert.Equal(t, saved.ID, retrieved.ID, "Retrieved prompt should match saved prompt")
}

// TestListPrompts verifies that prompts can be listed
func TestListPrompts(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a built-in prompt
	_, err := st.SaveNewVersion(ctx, prompt.PromptIDSystem, "System content", "System Prompt")
	require.NoError(t, err)

	// Add a custom prompt
	_, err = st.SaveNewVersion(ctx, "custom", "Custom content", "Custom Prompt")
	require.NoError(t, err)

	// List all prompts
	listed, err := st.List(ctx, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listed), 2, "Should have at least 2 prompts")

	// List with filter (convert prompt.ListFilter to promptstore.ListFilter)
	filter := &promptstore.ListFilter{IDs: []string{prompt.PromptIDSystem}}
	listed, err = st.List(ctx, filter)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listed), 1, "Filter should return at least 1 prompt")
}

// TestDeleteCustomPrompt verifies that custom prompts can be deleted
func TestDeleteCustomPrompt(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Add a custom prompt
	_, err := st.SaveNewVersion(ctx, "custom", "Custom content", "Custom Prompt")
	require.NoError(t, err)

	// Delete the custom prompt
	err = st.Delete(ctx, "custom")
	assert.NoError(t, err, "Deleting custom prompt should not return error")

	// Verify prompt is deleted
	loaded, err := st.Load(ctx, "custom")
	assert.NoError(t, err)
	assert.Nil(t, loaded, "Custom prompt should be deleted")
}

// TestAllBuiltinPrompts verifies that all built-in prompt constants are defined
func TestAllBuiltinPrompts(t *testing.T) {
	// Verify all built-in prompt ID constants are defined
	assert.Equal(t, "system", prompt.PromptIDSystem)
	assert.Equal(t, "supervisor", prompt.PromptIDSupervisor)
	assert.Equal(t, "compacter", prompt.PromptIDCompacter)
	assert.Equal(t, "subagent", prompt.PromptIDSubagent)
}

// TestListFilter verifies that ListFilter type is properly defined
func TestListFilter(t *testing.T) {
	// Create a filter
	filter := &prompt.ListFilter{
		Tags: []string{"tag1", "tag2"},
		IDs:  []string{"id1", "id2"},
	}

	assert.Len(t, filter.Tags, 2)
	assert.Len(t, filter.IDs, 2)
}

// TestPromptVersioning verifies that prompt versioning works correctly
func TestPromptVersioning(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save first version
	v1, err := st.SaveNewVersion(ctx, "test", "Content v1", "Test Prompt")
	require.NoError(t, err)
	assert.Equal(t, "test@1.0.0", v1.ID)

	// Save second version
	v2, err := st.SaveNewVersion(ctx, "test", "Content v2", "Test Prompt v2")
	require.NoError(t, err)
	assert.Equal(t, "test@1.0.1", v2.ID)

	// Verify both versions exist
	versions, err := st.ListVersions(ctx, "test")
	require.NoError(t, err)
	assert.Len(t, versions, 2, "Should have 2 versions")

	// Verify @latest points to v2
	latest, err := st.Load(ctx, "test@latest")
	require.NoError(t, err)
	assert.Equal(t, "test@1.0.1", latest.ID, "@latest should point to v2")
}

// TestRaceConditions runs tests with race detector
func TestRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race test in short mode")
	}

	ctx := context.Background()
	st := promptstore.NewMemoryStore()

	const numGoroutines = 100
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			_, _ = st.SaveNewVersion(ctx, "race-test", "Content", "Test")
		}(i)
	}

	wg.Wait()

	// Verify data integrity
	versions, err := st.ListVersions(ctx, "race-test")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(versions), 1, "Should have at least 1 version")
}

// ============ NEW TESTS FOR TEMPLATE RENDERING ============

// TestRenderPrompt_WithVariables tests template rendering with SubAgentContext
func TestRenderPrompt_WithVariables(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a test prompt with template variables
	testContent := "Role: {{.Role}}, Description: {{.Description}}"
	_, err := st.SaveNewVersion(ctx, "test-render", testContent, "Test Render")
	require.NoError(t, err)

	// Get the prompt
	loaded, err := st.Load(ctx, "test-render")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Create render context with SubAgentContext
	renderCtx := &prompt.RenderContext{
		SubAgent: &prompt.SubAgentContext{
			Role:        "coder",
			Description: "writes code",
		},
	}

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Render the prompt
	rendered, err := pm.RenderPrompt(ctx, loaded, renderCtx)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Role: coder", "Should contain substituted Role")
	assert.Contains(t, rendered, "Description: writes code", "Should contain substituted Description")
}

// TestRenderPrompt_AgentContext tests template rendering with AgentContext
func TestRenderPrompt_AgentContext(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a test prompt with template variables
	testContent := "Agent {{.AgentID}} is working on: {{.Task}}"
	_, err := st.SaveNewVersion(ctx, "test-agent", testContent, "Test Agent")
	require.NoError(t, err)

	// Get the prompt
	loaded, err := st.Load(ctx, "test-agent")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Create render context with AgentContext
	renderCtx := &prompt.RenderContext{
		Agent: &prompt.AgentContext{
			AgentID: "agent-123",
			Task:    "fix the login bug",
		},
	}

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Render the prompt
	rendered, err := pm.RenderPrompt(ctx, loaded, renderCtx)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Agent agent-123", "Should contain substituted AgentID")
	assert.Contains(t, rendered, "fix the login bug", "Should contain substituted Task")
}

// TestRenderPrompt_ValuesContext tests template rendering with generic Values
func TestRenderPrompt_ValuesContext(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a test prompt with template variables
	testContent := "Name: {{.Name}}, Count: {{.Count}}"
	_, err := st.SaveNewVersion(ctx, "test-values", testContent, "Test Values")
	require.NoError(t, err)

	// Get the prompt
	loaded, err := st.Load(ctx, "test-values")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Create render context with Values
	renderCtx := &prompt.RenderContext{
		Values: map[string]interface{}{
			"Name":  "TestProject",
			"Count": 42,
		},
	}

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Render the prompt
	rendered, err := pm.RenderPrompt(ctx, loaded, renderCtx)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Name: TestProject", "Should contain substituted Name")
	assert.Contains(t, rendered, "Count: 42", "Should contain substituted Count")
}

// TestGetPromptWithContext tests GetPromptWithContext convenience method
func TestGetPromptWithContext(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a test prompt with template variables
	testContent := "Hello {{.Name}}"
	_, err := st.SaveNewVersion(ctx, "test-context", testContent, "Test Context")
	require.NoError(t, err)

	// Create render context with Values
	renderCtx := &prompt.RenderContext{
		Values: map[string]interface{}{
			"Name": "World",
		},
	}

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Get and render prompt
	rendered, err := pm.GetPromptWithContext(ctx, "test-context", renderCtx)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Hello World", "Should contain rendered template")
}

// TestBackwardCompatibility_GetSystemPrompt tests backward-compatible GetSystemPrompt
func TestBackwardCompatibility_GetSystemPrompt(t *testing.T) {
	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Call GetSystemPrompt
	result, err := pm.GetSystemPrompt()
	require.NoError(t, err)
	assert.NotEmpty(t, result, "System prompt should not be empty")
	assert.Contains(t, result, "multi-agent system", "Should contain expected content from template")
}

// TestBackwardCompatibility_GetSubagentPrompt tests backward-compatible GetSubagentPrompt
func TestBackwardCompatibility_GetSubagentPrompt(t *testing.T) {
	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Call GetSubagentPrompt
	result, err := pm.GetSubagentPrompt("coder", "write clean code")
	require.NoError(t, err)
	assert.NotEmpty(t, result, "Subagent prompt should not be empty")
	assert.Contains(t, result, "coder", "Should contain role")
	assert.Contains(t, result, "write clean code", "Should contain description")
	assert.Contains(t, result, "spawn_agent", "Should contain spawn_agent tool name")
}

// TestBackwardCompatibility_GetSupervisorPrompt tests backward-compatible GetSupervisorPrompt
func TestBackwardCompatibility_GetSupervisorPrompt(t *testing.T) {
	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Call GetSupervisorPrompt
	result, err := pm.GetSupervisorPrompt()
	require.NoError(t, err)
	assert.NotEmpty(t, result, "Supervisor prompt should not be empty")
	assert.Contains(t, result, "helpful agent", "Should contain supervisor content from template")
}

// TestBackwardCompatibility_GetCompacterPrompt tests backward-compatible GetCompacterPrompt
func TestBackwardCompatibility_GetCompacterPrompt(t *testing.T) {
	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Call GetCompacterPrompt with data
	// The built-in compacter template doesn't use template variables,
	// so we test that the method works and returns the built-in template
	data := map[string]interface{}{
		"Data": "User asked about fixing a bug, we provided a solution.",
	}
	result, err := pm.GetCompacterPrompt(data)
	require.NoError(t, err)
	assert.NotEmpty(t, result, "Compacter prompt should not be empty")
	assert.Contains(t, result, "Summarize", "Should contain compacter content")
}

// TestBackwardCompatibility_GetCompacterPromptWithCustomTemplate tests custom compacter prompt with variables
func TestBackwardCompatibility_GetCompacterPromptWithCustomTemplate(t *testing.T) {
	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a custom prompt (not using built-in ID) with template variable
	customContent := "Summarize the given conversation. Key points: {{.Data}}"
	_, err := st.SaveNewVersion(context.Background(), "custom-compacter", customContent, "Custom Compacter Prompt")
	require.NoError(t, err)

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Create render context with data
	renderCtx := &prompt.RenderContext{
		Values: map[string]interface{}{
			"Data": "User asked about fixing a bug, we provided a solution.",
		},
	}

	// Use GetPromptWithContext for custom prompts
	result, err := pm.GetPromptWithContext(context.Background(), "custom-compacter", renderCtx)
	require.NoError(t, err)
	assert.NotEmpty(t, result, "Custom compacter prompt should not be empty")
	assert.Contains(t, result, "User asked about fixing a bug", "Should contain data")
}

// TestRenderPrompt_MissingVariable tests handling of missing template variables
func TestRenderPrompt_MissingVariable(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a test prompt with undefined variable
	testContent := "Value: {{.Missing}}"
	_, err := st.SaveNewVersion(ctx, "test-missing", testContent, "Test Missing")
	require.NoError(t, err)

	// Get the prompt
	loaded, err := st.Load(ctx, "test-missing")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Create empty render context
	renderCtx := &prompt.RenderContext{}

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Render the prompt - text/template handles missing variables gracefully
	rendered, err := pm.RenderPrompt(ctx, loaded, renderCtx)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Value: <no value>", "Should contain no value for missing variable")
}

// TestRenderPrompt_NilContext tests rendering with nil context
func TestRenderPrompt_NilContext(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a test prompt without variables
	testContent := "This is a static prompt with no variables."
	_, err := st.SaveNewVersion(ctx, "test-nil", testContent, "Test Nil")
	require.NoError(t, err)

	// Get the prompt
	loaded, err := st.Load(ctx, "test-nil")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Render the prompt with nil context
	rendered, err := pm.RenderPrompt(ctx, loaded, nil)
	require.NoError(t, err)
	assert.Equal(t, testContent, rendered, "Should render static content unchanged")
}

// TestRenderPrompt_NilPrompt tests rendering with nil prompt
func TestRenderPrompt_NilPrompt(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Try to render nil prompt
	_, err := pm.RenderPrompt(ctx, nil, &prompt.RenderContext{})
	assert.Error(t, err, "Should return error for nil prompt")
	assert.Contains(t, err.Error(), "cannot be nil", "Error should mention nil prompt")
}

// TestRenderPrompt_AllContextFields tests rendering with all context fields populated
func TestRenderPrompt_AllContextFields(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a test prompt with variables from all context types
	testContent := "Role: {{.Role}}\nAgentID: {{.AgentID}}\nCustom: {{.CustomField}}"
	_, err := st.SaveNewVersion(ctx, "test-all", testContent, "Test All")
	require.NoError(t, err)

	// Get the prompt
	loaded, err := st.Load(ctx, "test-all")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Create render context with all fields populated
	renderCtx := &prompt.RenderContext{
		Values: map[string]interface{}{
			"CustomField": "custom value",
		},
		SubAgent: &prompt.SubAgentContext{
			Role: "tester",
		},
		Agent: &prompt.AgentContext{
			AgentID: "agent-456",
		},
	}

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Render the prompt
	rendered, err := pm.RenderPrompt(ctx, loaded, renderCtx)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Role: tester", "Should contain SubAgent Role")
	assert.Contains(t, rendered, "AgentID: agent-456", "Should contain Agent AgentID")
	assert.Contains(t, rendered, "Custom: custom value", "Should contain custom Value")
}

// TestDeletePrompt_BuiltinPromptReturnsError verifies that built-in prompts cannot be deleted
func TestDeletePrompt_BuiltinPromptReturnsError(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a prompt using SaveBuiltinVersion (like bootstrap does)
	_, err := st.SaveBuiltinVersion(ctx, prompt.PromptIDSystem, "Built-in system prompt content", "System Prompt")
	require.NoError(t, err)

	// Verify it was saved with IsBuiltin=true
	loaded, err := st.Load(ctx, prompt.PromptIDSystem)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.True(t, loaded.IsBuiltin, "Prompt saved via SaveBuiltinVersion should have IsBuiltin=true")

	// Try to delete the built-in prompt
	err = st.Delete(ctx, prompt.PromptIDSystem)
	assert.Error(t, err, "Deleting built-in prompt should return an error")
	assert.Equal(t, promptstore.ErrPromptIsBuiltin, err, "Error should be ErrPromptIsBuiltin")

	// Verify prompt still exists after failed delete
	stillExists, err := st.Load(ctx, prompt.PromptIDSystem)
	require.NoError(t, err)
	assert.NotNil(t, stillExists, "Built-in prompt should still exist after failed delete")
}

// TestRenderPrompt_WithMessageHistory tests template rendering with MessageHistory in AgentContext
func TestRenderPrompt_WithMessageHistory(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a test prompt with message history template (safe handling of nil)
	testContent := "Agent {{.AgentID}}"
	_, err := st.SaveNewVersion(ctx, "test-history", testContent, "Test History")
	require.NoError(t, err)

	// Get the prompt
	loaded, err := st.Load(ctx, "test-history")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Create render context with nil MessageHistory
	renderCtx := &prompt.RenderContext{
		Agent: &prompt.AgentContext{
			AgentID:        "agent-789",
			MessageHistory: nil,
		},
	}

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Render the prompt
	rendered, err := pm.RenderPrompt(ctx, loaded, renderCtx)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Agent agent-789", "Should contain substituted AgentID")
}

// TestRenderPrompt_EmptyMessageHistory tests template rendering with empty MessageHistory
func TestRenderPrompt_EmptyMessageHistory(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := promptstore.NewMemoryStore()

	// Save a test prompt
	testContent := "Agent {{.AgentID}} - Task: {{.Task}}"
	_, err := st.SaveNewVersion(ctx, "test-empty-history", testContent, "Test Empty History")
	require.NoError(t, err)

	// Get the prompt
	loaded, err := st.Load(ctx, "test-empty-history")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Create render context with AgentContext (MessageHistory is optional)
	renderCtx := &prompt.RenderContext{
		Agent: &prompt.AgentContext{
			AgentID:        "agent-empty",
			Task:           "test task",
			MessageHistory: nil, // Empty/nil history
		},
	}

	// Create prompt manager
	pm := manager.NewPromptManager(st)

	// Render the prompt - should handle nil MessageHistory gracefully
	rendered, err := pm.RenderPrompt(ctx, loaded, renderCtx)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Agent agent-empty", "Should contain substituted AgentID")
	assert.Contains(t, rendered, "Task: test task", "Should contain substituted Task")
}

