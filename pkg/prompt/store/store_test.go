// Package store provides tests for prompt store implementations.
package store

import (
	"context"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/stretchr/testify/assert"
)

func createTestPrompt(id, content string) *prompt.Prompt {
	return &prompt.Prompt{
		ID:        id,
		Name:      "test",
		Content:   content,
		Context:   make(map[string]interface{}),
		Tags:      []string{},
		Version:   semver.New(1, 0, 0, "", ""),
		IsBuiltin: false,
	}
}

func setupTestStore(t *testing.T) PromptStore {
	return NewMemoryStore()
}

func setupFileStore(t *testing.T) PromptStore {
	return NewFileStore(t.TempDir(), false)
}

// ==================== MemoryStore Tests ====================

func TestMemoryStore_SaveNewVersion(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("first version creates 1.0.0", func(t *testing.T) {
		p, err := store.SaveNewVersion(ctx, "test", "content", "Test Prompt")
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, "test@1.0.0", p.ID)
		assert.Equal(t, "content", p.Content)
		assert.NotNil(t, p.Version)
		assert.Equal(t, uint64(1), p.Version.Major())
		assert.Equal(t, uint64(0), p.Version.Minor())
		assert.Equal(t, uint64(0), p.Version.Patch())
	})

	t.Run("second version increments patch", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "patchtest", "v1", "Test")
		p2, err := store.SaveNewVersion(ctx, "patchtest", "v2", "Test")
		assert.NoError(t, err)
		assert.NotNil(t, p2)
		assert.Equal(t, "patchtest@1.0.1", p2.ID)
		assert.Equal(t, "v2", p2.Content)
	})

	t.Run("aliases are added to latest version", func(t *testing.T) {
		p, err := store.SaveNewVersion(ctx, "aliastest", "content", "Test")
		assert.NoError(t, err)
		assert.Contains(t, p.Tags, "aliastest")
		assert.Contains(t, p.Tags, "aliastest@latest")
	})

	t.Run("context is copied from previous version", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "ctxtest", "v1", "Test")
		// Manually update to simulate context being set
		store.(*memoryStore).prompts["ctxtest@1.0.0"].Context = map[string]interface{}{"key": "value"}

		p2, err := store.SaveNewVersion(ctx, "ctxtest", "v2", "Test")
		assert.NoError(t, err)
		assert.Equal(t, "value", p2.Context["key"])
	})
}

func TestMemoryStore_Load(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("load existing prompt", func(t *testing.T) {
		created, _ := store.SaveNewVersion(ctx, "loadtest", "content", "Test")
		loaded, err := store.Load(ctx, "loadtest@1.0.0")
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, created.ID, loaded.ID)
		assert.Equal(t, created.Content, loaded.Content)
	})

	t.Run("load non-existent prompt returns nil", func(t *testing.T) {
		loaded, err := store.Load(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("load via alias", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "aliastest", "content", "Test")
		loaded, err := store.Load(ctx, "aliastest")
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, "aliastest@1.0.0", loaded.ID)
	})

	t.Run("load via @latest alias", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "latesttest", "content", "Test")
		loaded, err := store.Load(ctx, "latesttest@latest")
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, "latesttest@1.0.0", loaded.ID)
	})
}

func TestMemoryStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("delete existing prompt", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "deletetest", "content", "Test")
		err := store.Delete(ctx, "deletetest@1.0.0")
		assert.NoError(t, err)

		// Verify deleted
		loaded, _ := store.Load(ctx, "deletetest@1.0.0")
		assert.Nil(t, loaded)
	})

	t.Run("delete non-existent prompt returns nil", func(t *testing.T) {
		err := store.Delete(ctx, "nonexistent")
		assert.NoError(t, err)
	})

	t.Run("delete builtin prompt returns error", func(t *testing.T) {
		builtin := &prompt.Prompt{
			ID:        "builtin@1.0.0",
			Content:   "builtin content",
			IsBuiltin: true,
			Version:   semver.New(1, 0, 0, "", ""),
		}
		store.(*memoryStore).prompts["builtin@1.0.0"] = builtin
		store.(*memoryStore).prompts["builtin"] = builtin
		store.(*memoryStore).prompts["builtin@latest"] = builtin

		err := store.Delete(ctx, "builtin@1.0.0")
		assert.Error(t, err)
		assert.Equal(t, ErrPromptIsBuiltin, err)
	})

	t.Run("delete removes aliases", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "aliastest", "content", "Test")
		err := store.Delete(ctx, "aliastest@1.0.0")
		assert.NoError(t, err)

		_, err1 := store.Load(ctx, "aliastest")
		_, err2 := store.Load(ctx, "aliastest@latest")
		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})
}

func TestMemoryStore_List(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Setup test data
	store.SaveNewVersion(ctx, "prompt1", "content1", "Test1")
	store.SaveNewVersion(ctx, "prompt2", "content2", "Test2")
	p3, _ := store.SaveNewVersion(ctx, "prompt3", "content3", "Test3")
	p3.Tags = append(p3.Tags, "special")
	store.(*memoryStore).prompts["prompt3@1.0.0"] = p3

	t.Run("list all prompts", func(t *testing.T) {
		list, err := store.List(ctx, nil)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 3)
	})

	t.Run("list with tag filter", func(t *testing.T) {
		list, err := store.List(ctx, &ListFilter{Tags: []string{"special"}})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 1)
	})

	t.Run("list with ID filter", func(t *testing.T) {
		list, err := store.List(ctx, &ListFilter{IDs: []string{"prompt1"}})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 1)
	})

	t.Run("list excludes duplicates", func(t *testing.T) {
		list, err := store.List(ctx, nil)
		assert.NoError(t, err)

		// Check for duplicate IDs
		ids := make(map[string]bool)
		for _, p := range list {
			assert.False(t, ids[p.ID], "duplicate ID found: %s", p.ID)
			ids[p.ID] = true
		}
	})
}

func TestMemoryStore_Exists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("existing prompt returns true", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "existstest", "content", "Test")
		exists, err := store.Exists(ctx, "existstest@1.0.0")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("non-existent prompt returns false", func(t *testing.T) {
		exists, err := store.Exists(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestMemoryStore_ResolveAlias(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("resolve shortcut alias", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "aliastest", "content", "Test")
		p, err := store.ResolveAlias(ctx, "aliastest")
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, "aliastest@1.0.0", p.ID)
	})

	t.Run("resolve @latest alias", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "latesttest", "content", "Test")
		p, err := store.ResolveAlias(ctx, "latesttest@latest")
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Contains(t, p.ID, "@")
	})

	t.Run("resolve versioned ID returns itself", func(t *testing.T) {
		created, _ := store.SaveNewVersion(ctx, "versiontest", "content", "Test")
		p, err := store.ResolveAlias(ctx, "versiontest@1.0.0")
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, created.ID, p.ID)
	})

	t.Run("resolve non-existent returns nil", func(t *testing.T) {
		p, err := store.ResolveAlias(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Nil(t, p)
	})
}

func TestMemoryStore_ListVersions(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("list versions of prompt", func(t *testing.T) {
		store.SaveNewVersion(ctx, "versiontest", "v1", "Test")
		store.SaveNewVersion(ctx, "versiontest", "v2", "Test")
		store.SaveNewVersion(ctx, "versiontest", "v3", "Test")

		versions, err := store.ListVersions(ctx, "versiontest")
		assert.NoError(t, err)
		assert.Len(t, versions, 3)
	})

	t.Run("list versions of non-existent prompt", func(t *testing.T) {
		versions, err := store.ListVersions(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Len(t, versions, 0)
	})

	t.Run("versions exclude @latest", func(t *testing.T) {
		store.SaveNewVersion(ctx, "latestcheck", "content", "Test")

		versions, err := store.ListVersions(ctx, "latestcheck")
		assert.NoError(t, err)
		for _, v := range versions {
			assert.NotContains(t, v.ID, "@latest")
		}
	})
}

func TestMemoryStore_SetLatestAlias(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("set latest to specific version", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "settest", "v1", "Test")
		v2, _ := store.SaveNewVersion(ctx, "settest", "v2", "Test")
		_, _ = store.SaveNewVersion(ctx, "settest", "v3", "Test")

		// Set v2 as latest
		err := store.SetLatestAlias(ctx, "settest", v2.ID)
		assert.NoError(t, err)

		// Check v2 has @latest
		p, _ := store.ResolveAlias(ctx, "settest@latest")
		assert.Equal(t, v2.ID, p.ID)
		assert.NotEqual(t, "settest@1.0.2", p.ID)
	})

	t.Run("set latest on non-existent prompt returns error", func(t *testing.T) {
		err := store.SetLatestAlias(ctx, "nonexistent", "nonexistent@1.0.0")
		assert.Error(t, err)
		assert.Equal(t, ErrPromptNotFound, err)
	})
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("concurrent writes", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				_, _ = store.SaveNewVersion(ctx, "concurrent", "content", "Test")
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify store is still functional
		_, err := store.Load(ctx, "concurrent")
		assert.NoError(t, err)
	})

	t.Run("concurrent reads", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "readtest", "content", "Test")
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				_, _ = store.Load(ctx, "readtest")
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestMemoryStore_ListTags(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("list unique tags", func(t *testing.T) {
		p1, _ := store.SaveNewVersion(ctx, "tagtest1", "content1", "Test")
		p1.Tags = append(p1.Tags, "tag1", "tag2")
		store.(*memoryStore).prompts["tagtest1@1.0.0"] = p1

		p2, _ := store.SaveNewVersion(ctx, "tagtest2", "content2", "Test")
		p2.Tags = append(p2.Tags, "tag2", "tag3")
		store.(*memoryStore).prompts["tagtest2@1.0.0"] = p2

		tags, err := store.ListTags(ctx)
		assert.NoError(t, err)
		assert.Contains(t, tags, "tag1")
		assert.Contains(t, tags, "tag2")
		assert.Contains(t, tags, "tag3")
	})
}

// ==================== FileStore Tests ====================

func TestFileStore_SaveNewVersion(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("saves prompt as JSON file", func(t *testing.T) {
		p, err := store.SaveNewVersion(ctx, "filetest", "content", "Test")
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, "filetest@1.0.0", p.ID)
		assert.Equal(t, "content", p.Content)
	})

	t.Run("second version increments patch", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "patchtest", "v1", "Test")
		p2, err := store.SaveNewVersion(ctx, "patchtest", "v2", "Test")
		assert.NoError(t, err)
		assert.Equal(t, "patchtest@1.0.1", p2.ID)
	})
}

func TestFileStore_Load(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("load existing file", func(t *testing.T) {
		created, _ := store.SaveNewVersion(ctx, "loadtest", "content", "Test")
		loaded, err := store.Load(ctx, "loadtest@1.0.0")
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, created.ID, loaded.ID)
		assert.Equal(t, created.Content, loaded.Content)
	})

	t.Run("load non-existent returns nil", func(t *testing.T) {
		loaded, err := store.Load(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Nil(t, loaded)
	})
}

func TestFileStore_Delete(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("delete removes file", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "deletetest", "content", "Test")
		err := store.Delete(ctx, "deletetest@1.0.0")
		assert.NoError(t, err)

		// Verify deleted
		loaded, _ := store.Load(ctx, "deletetest@1.0.0")
		assert.Nil(t, loaded)
	})

	t.Run("delete non-existent returns nil", func(t *testing.T) {
		err := store.Delete(ctx, "nonexistent")
		assert.NoError(t, err)
	})
}

func TestFileStore_List(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("list scans directory", func(t *testing.T) {
		store.SaveNewVersion(ctx, "listtest1", "content1", "Test1")
		store.SaveNewVersion(ctx, "listtest2", "content2", "Test2")

		list, err := store.List(ctx, nil)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 2)
	})
}

func TestFileStore_Exists(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("existing file returns true", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "existstest", "content", "Test")
		exists, err := store.Exists(ctx, "existstest@1.0.0")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("non-existent file returns false", func(t *testing.T) {
		exists, err := store.Exists(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestFileStore_FileLocking(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("concurrent writes with locking", func(t *testing.T) {
		done := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func(id int) {
				_, _ = store.SaveNewVersion(ctx, "locktest", "content", "Test")
				done <- true
			}(i)
		}

		for i := 0; i < 5; i++ {
			<-done
		}

		// Verify store is still functional
		_, err := store.Load(ctx, "locktest")
		assert.NoError(t, err)
	})
}

func TestFileStore_CacheBehavior(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, true)
	ctx := context.Background()

	t.Run("cache hit on second load", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "cachetest", "content", "Test")

		// First load - from file
		_, _ = store.Load(ctx, "cachetest@1.0.0")

		// Second load - from cache
		_, err := store.Load(ctx, "cachetest@1.0.0")
		assert.NoError(t, err)
	})

	t.Run("cache enabled stores in cache", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "cachehit", "content", "Test")
		_, _ = store.Load(ctx, "cachehit@1.0.0")

		// Check cache has the entry
		fs := store.(*fileStore)
		fs.mu.RLock()
		_, exists := fs.cache["cachehit@1.0.0"]
		fs.mu.RUnlock()
		assert.True(t, exists, "cache should have the entry")
	})
}

func TestFileStore_ListVersions(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("list versions from directory", func(t *testing.T) {
		store.SaveNewVersion(ctx, "verlisttest", "v1", "Test")
		store.SaveNewVersion(ctx, "verlisttest", "v2", "Test")
		store.SaveNewVersion(ctx, "verlisttest", "v3", "Test")

		versions, err := store.ListVersions(ctx, "verlisttest")
		assert.NoError(t, err)
		assert.Len(t, versions, 3)
	})
}

func TestFileStore_SetLatestAlias(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("set latest updates file", func(t *testing.T) {
		v1, _ := store.SaveNewVersion(ctx, "setlatest", "v1", "Test")
		v2, _ := store.SaveNewVersion(ctx, "setlatest", "v2", "Test")

		// Set v1 as latest
		err := store.SetLatestAlias(ctx, "setlatest", v1.ID)
		assert.NoError(t, err)

		// Check v1 is now latest
		p, err := store.ResolveAlias(ctx, "setlatest@latest")
		assert.NoError(t, err)
		assert.NotNil(t, p, "ResolveAlias returned nil")
		if p != nil {
			assert.Equal(t, v1.ID, p.ID)
			assert.NotEqual(t, v2.ID, p.ID)
		}
	})
}

func TestFileStore_ResolveAlias(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("resolve shortcut from file", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "filealiastest", "content", "Test")
		p, err := store.ResolveAlias(ctx, "filealiastest")
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, "filealiastest@1.0.0", p.ID)
	})
}

func TestFileStore_ListTags(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("list tags from files", func(t *testing.T) {
		_, _ = store.SaveNewVersion(ctx, "filetag1", "content1", "Test")

		tags, err := store.ListTags(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, tags)
	})
}

// ==================== SaveBuiltinVersion Tests ====================

func TestMemoryStore_SaveBuiltinVersion_SetsIsBuiltinTrue(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("first version has IsBuiltin=true", func(t *testing.T) {
		p, err := store.SaveBuiltinVersion(ctx, "builtin-test", "builtin content", "Builtin Prompt")
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, "builtin-test@1.0.0", p.ID)
		assert.True(t, p.IsBuiltin, "SaveBuiltinVersion should set IsBuiltin=true")
	})

	t.Run("second version also has IsBuiltin=true", func(t *testing.T) {
		_, _ = store.SaveBuiltinVersion(ctx, "builtin-v2", "v1", "Builtin")
		p2, err := store.SaveBuiltinVersion(ctx, "builtin-v2", "v2", "Builtin")
		assert.NoError(t, err)
		assert.True(t, p2.IsBuiltin, "SaveBuiltinVersion should set IsBuiltin=true on subsequent versions")
	})
}

func TestMemoryStore_DeleteBuiltinPrompt_ReturnsError(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("delete prompt saved with SaveBuiltinVersion fails", func(t *testing.T) {
		_, _ = store.SaveBuiltinVersion(ctx, "protected", "protected content", "Protected Prompt")

		err := store.Delete(ctx, "protected@1.0.0")
		assert.Error(t, err)
		assert.Equal(t, ErrPromptIsBuiltin, err)

		// Verify prompt still exists
		loaded, _ := store.Load(ctx, "protected@1.0.0")
		assert.NotNil(t, loaded, "Builtin prompt should still exist after failed delete")
	})
}

func TestFileStore_SaveBuiltinVersion_SetsIsBuiltinTrue(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("file store saves with IsBuiltin=true", func(t *testing.T) {
		p, err := store.SaveBuiltinVersion(ctx, "file-builtin", "file builtin content", "File Builtin")
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.True(t, p.IsBuiltin, "SaveBuiltinVersion should set IsBuiltin=true in FileStore")
	})

	t.Run("IsBuiltin persists across load", func(t *testing.T) {
		saved, _ := store.SaveBuiltinVersion(ctx, "persist-builtin", "content", "Persist")
		loaded, err := store.Load(ctx, saved.ID)
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.True(t, loaded.IsBuiltin, "IsBuiltin should persist when loaded from file")
	})
}

func TestFileStore_DeleteBuiltinPrompt_ReturnsError(t *testing.T) {
	store := setupFileStore(t)
	ctx := context.Background()

	t.Run("delete builtin prompt from file store fails", func(t *testing.T) {
		_, _ = store.SaveBuiltinVersion(ctx, "file-protected", "protected content", "Protected")

		err := store.Delete(ctx, "file-protected@1.0.0")
		assert.Error(t, err)
		assert.Equal(t, ErrPromptIsBuiltin, err)

		// Verify prompt still exists
		loaded, _ := store.Load(ctx, "file-protected@1.0.0")
		assert.NotNil(t, loaded, "Builtin prompt should still exist in file store after failed delete")
	})
}
