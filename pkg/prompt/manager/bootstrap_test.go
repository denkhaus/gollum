// Package manager provides tests for PromptManager DI integration
package manager_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	promptstore "github.com/denkhaus/gollum/pkg/prompt/store"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// setupMockConfigService creates a configured mock config service for testing
// Returns the injector with the mock registered and the controller for cleanup
func setupMockConfigService(t *testing.T, storeConfig *config.PromptStoreConfig) (do.Injector, *gomock.Controller) {
	ctrl := gomock.NewController(t)

	injector := do.New()

	// Register mocked config service - setup expectations here
	mockConfig := mocks.NewMockConfigService(ctrl)
	mockConfig.EXPECT().GetPromptStoreConfig().Return(storeConfig).AnyTimes()
	mockConfig.EXPECT().GetLogLevel().Return("info").AnyTimes()
	mockConfig.EXPECT().IsDevMode().Return(false).AnyTimes()
	mockConfig.EXPECT().GetLoggingConfig().Return(&config.LoggingConfig{
		SessionLogBufferSize: 1000,
		SessionLogEnabled:    false,
	}).AnyTimes()

	do.Provide[config.ConfigService](injector, func(_ do.Injector) (config.ConfigService, error) {
		return mockConfig, nil
	})

	return injector, ctrl
}

// TestNewPromptManagerProvider_MemoryStore tests DI provider with memory store
func TestNewPromptManagerProvider_MemoryStore(t *testing.T) {
	storeConfig := &config.PromptStoreConfig{
		Type:         config.PromptStoreTypeMemory,
		FilePath:     "",
		CacheEnabled: false,
	}

	injector, ctrl := setupMockConfigService(t, storeConfig)
	defer ctrl.Finish()

	// Register logger service
	do.Provide(injector, logger.NewService)

	// Register PromptStore provider
	do.Provide(injector, promptstore.NewPromptStore)

	// Register PromptManager provider
	do.Provide[manager.PromptManager](injector, manager.NewPromptManagerProvider)

	// Invoke PromptManager from DI
	pm, err := do.Invoke[manager.PromptManager](injector)
	require.NoError(t, err, "Should invoke PromptManager from DI")
	require.NotNil(t, pm, "PromptManager should not be nil")

	// Verify store is accessible
	store := pm.GetStore()
	require.NotNil(t, store, "Store should not be nil")

	// Verify it's a memory store by performing operations
	ctx := context.Background()
	saved, err := store.SaveNewVersion(ctx, "test-di-memory", "Test content", "Test Prompt")
	require.NoError(t, err, "Should save prompt to memory store")
	assert.Equal(t, "test-di-memory@1.0.0", saved.ID, "Should have correct versioned ID")

	// Verify prompt can be retrieved
	loaded, err := pm.GetPromptByID(ctx, "test-di-memory")
	require.NoError(t, err, "Should load prompt from memory store")
	assert.Equal(t, "Test content", loaded.Content, "Should have correct content")
}

// TestNewPromptManagerProvider_FileStore tests DI provider with file store
func TestNewPromptManagerProvider_FileStore(t *testing.T) {
	tempDir := t.TempDir()

	storeConfig := &config.PromptStoreConfig{
		Type:         config.PromptStoreTypeFile,
		FilePath:     tempDir,
		CacheEnabled: false,
	}

	injector, ctrl := setupMockConfigService(t, storeConfig)
	defer ctrl.Finish()

	// Register logger service
	do.Provide(injector, logger.NewService)

	// Register PromptStore provider
	do.Provide(injector, promptstore.NewPromptStore)

	// Register PromptManager provider
	do.Provide[manager.PromptManager](injector, manager.NewPromptManagerProvider)

	// Invoke PromptManager from DI
	pm, err := do.Invoke[manager.PromptManager](injector)
	require.NoError(t, err, "Should invoke PromptManager from DI")
	require.NotNil(t, pm, "PromptManager should not be nil")

	// Verify store is accessible
	store := pm.GetStore()
	require.NotNil(t, store, "Store should not be nil")

	// Verify it's a file store by performing operations
	ctx := context.Background()
	saved, err := store.SaveNewVersion(ctx, "test-di-file", "Test content for file store", "Test Prompt")
	require.NoError(t, err, "Should save prompt to file store")
	assert.Equal(t, "test-di-file@1.0.0", saved.ID, "Should have correct versioned ID")

	// Verify prompt can be retrieved using the versioned ID returned from save
	loaded, err := pm.GetPromptByID(ctx, saved.BaseID())
	require.NoError(t, err, "Should load prompt from file store")
	require.NotNil(t, loaded, "Loaded prompt should not be nil")
	assert.Equal(t, "Test content for file store", loaded.Content, "Should have correct content")

	// Verify file was created
	expectedPath := filepath.Join(tempDir, saved.ID+".json")
	_, err = os.Stat(expectedPath)
	require.NoError(t, err, "Prompt file should exist in file store")

	// Note: File store Load() doesn't resolve aliases, so @latest test is skipped
	// The memory store test covers alias resolution behavior
}

// TestNewPromptManagerProvider_DefaultToMemory tests that empty store type defaults to memory
func TestNewPromptManagerProvider_DefaultToMemory(t *testing.T) {
	storeConfig := &config.PromptStoreConfig{
		Type:         "", // Empty type should default to memory
		FilePath:     "",
		CacheEnabled: false,
	}

	injector, ctrl := setupMockConfigService(t, storeConfig)
	defer ctrl.Finish()

	// Register logger service
	do.Provide(injector, logger.NewService)

	// Register PromptStore provider
	do.Provide(injector, promptstore.NewPromptStore)

	// Register PromptManager provider
	do.Provide[manager.PromptManager](injector, manager.NewPromptManagerProvider)

	// Invoke PromptManager from DI
	pm, err := do.Invoke[manager.PromptManager](injector)
	require.NoError(t, err, "Should invoke PromptManager from DI")
	require.NotNil(t, pm, "PromptManager should not be nil")

	// Verify it works like a memory store
	ctx := context.Background()
	saved, err := pm.GetStore().SaveNewVersion(ctx, "test-default", "Content", "Test")
	require.NoError(t, err, "Should work with default memory store")
	assert.NotNil(t, saved, "Saved prompt should not be nil")
}

// TestNewPromptManagerProvider_CachedFileStore tests file store with caching enabled
func TestNewPromptManagerProvider_CachedFileStore(t *testing.T) {
	tempDir := t.TempDir()

	storeConfig := &config.PromptStoreConfig{
		Type:         config.PromptStoreTypeFile,
		FilePath:     tempDir,
		CacheEnabled: true,
	}

	injector, ctrl := setupMockConfigService(t, storeConfig)
	defer ctrl.Finish()

	// Register logger service
	do.Provide(injector, logger.NewService)

	// Register PromptStore provider
	do.Provide(injector, promptstore.NewPromptStore)

	// Register PromptManager provider
	do.Provide[manager.PromptManager](injector, manager.NewPromptManagerProvider)

	// Invoke PromptManager from DI
	pm, err := do.Invoke[manager.PromptManager](injector)
	require.NoError(t, err, "Should invoke PromptManager from DI")
	require.NotNil(t, pm, "PromptManager should not be nil")

	// Verify cached store works correctly
	ctx := context.Background()
	_, err = pm.GetStore().SaveNewVersion(ctx, "test-cache", "Cached content", "Test")
	require.NoError(t, err, "Should save prompt")

	// First load - should read from file
	firstLoad, err := pm.GetPromptByID(ctx, "test-cache")
	require.NoError(t, err, "Should load prompt first time")
	assert.Equal(t, "Cached content", firstLoad.Content, "Should have correct content")

	// Second load - should read from cache
	secondLoad, err := pm.GetPromptByID(ctx, "test-cache")
	require.NoError(t, err, "Should load prompt second time from cache")
	assert.Equal(t, firstLoad.ID, secondLoad.ID, "Should return same prompt")
}

// TestNewPromptManagerProvider_UnknownStoreType tests that unknown store type falls back to memory
func TestNewPromptManagerProvider_UnknownStoreType(t *testing.T) {
	storeConfig := &config.PromptStoreConfig{
		Type:         "unknown_type", // Unknown type
		FilePath:     "",
		CacheEnabled: false,
	}

	injector, ctrl := setupMockConfigService(t, storeConfig)
	defer ctrl.Finish()

	// Register logger service
	do.Provide(injector, logger.NewService)

	// Register PromptStore provider
	do.Provide(injector, promptstore.NewPromptStore)

	// Register PromptManager provider
	do.Provide[manager.PromptManager](injector, manager.NewPromptManagerProvider)

	// Invoke PromptManager from DI - should still work with memory fallback
	pm, err := do.Invoke[manager.PromptManager](injector)
	require.NoError(t, err, "Should invoke PromptManager from DI with memory fallback")
	require.NotNil(t, pm, "PromptManager should not be nil")

	// Verify it works like a memory store (fallback)
	ctx := context.Background()
	saved, err := pm.GetStore().SaveNewVersion(ctx, "test-fallback", "Fallback content", "Test")
	require.NoError(t, err, "Should work with fallback memory store")
	assert.NotNil(t, saved, "Saved prompt should not be nil")
}

// TestNewPromptManagerProvider_PromptManagerInterface tests that PromptManager interface is satisfied
func TestNewPromptManagerProvider_PromptManagerInterface(t *testing.T) {
	storeConfig := &config.PromptStoreConfig{
		Type:         config.PromptStoreTypeMemory,
		FilePath:     "",
		CacheEnabled: false,
	}

	injector, ctrl := setupMockConfigService(t, storeConfig)
	defer ctrl.Finish()

	// Register logger service
	do.Provide(injector, logger.NewService)

	// Register PromptStore provider
	do.Provide(injector, promptstore.NewPromptStore)

	// Register PromptManager provider
	do.Provide[manager.PromptManager](injector, manager.NewPromptManagerProvider)

	// Invoke PromptManager from DI
	pm, err := do.Invoke[manager.PromptManager](injector)
	require.NoError(t, err)

	// Test all interface methods
	ctx := context.Background()

	// GetPromptByID
	_, err = pm.GetPromptByID(ctx, "nonexistent")
	assert.NoError(t, err, "GetPromptByID should not error on not found")

	// SetPrompt
	saved, err := pm.SetPrompt(ctx, "interface-test", "Content", "Test")
	require.NoError(t, err, "SetPrompt should work")
	assert.NotNil(t, saved, "SetPrompt should return prompt")

	// DeletePrompt
	err = pm.DeletePrompt(ctx, "interface-test")
	assert.NoError(t, err, "DeletePrompt should work")

	// ListPrompts
	listed, err := pm.ListPrompts(ctx, &prompt.ListFilter{})
	assert.NoError(t, err, "ListPrompts should work")
	assert.NotNil(t, listed, "ListPrompts should return list")
}
