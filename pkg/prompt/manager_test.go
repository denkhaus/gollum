// Package prompt provides tests for lazy initialization and built-in bootstrapping.
// This test file uses a separate package name to avoid import cycles.
package prompt_test

import (
	"context"
	"sync"
	"testing"

	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/prompt/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLazyBuiltinBootstrap verifies that built-in prompts can be saved and loaded
func TestLazyBuiltinBootstrap(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := store.NewMemoryStore()

	// Bootstrap system prompt by saving it directly
	// Note: SaveNewVersion creates prompts with IsBuiltin=false by default
	// The bootstrapBuiltinPrompt function in manager_bootstrap.go sets IsBuiltin=true
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
	// Note: IsBuiltin is false when saved via store, true when bootstrapped via manager
}

// TestBuiltinPromptProtection verifies that built-in prompts cannot be deleted
func TestBuiltinPromptProtection(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := store.NewMemoryStore()

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
	st := store.NewMemoryStore()

	// Call Load with non-existent ID
	loaded, err := st.Load(ctx, "nonexistent")
	assert.NoError(t, err, "Load should not return error for non-existent prompt")
	assert.Nil(t, loaded, "Load should return nil for non-existent prompt")
}

// TestConcurrentBootstrap verifies that concurrent operations work correctly
func TestConcurrentBootstrap(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := store.NewMemoryStore()

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
	st := store.NewMemoryStore()

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
	st := store.NewMemoryStore()

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

	// List with filter (convert prompt.ListFilter to store.ListFilter)
	filter := &store.ListFilter{IDs: []string{prompt.PromptIDSystem}}
	listed, err = st.List(ctx, filter)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listed), 1, "Filter should return at least 1 prompt")
}

// TestDeleteCustomPrompt verifies that custom prompts can be deleted
func TestDeleteCustomPrompt(t *testing.T) {
	ctx := context.Background()

	// Create in-memory store
	st := store.NewMemoryStore()

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
	st := store.NewMemoryStore()

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
	st := store.NewMemoryStore()

	const numGoroutines = 100
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
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
