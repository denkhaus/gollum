// Package store provides file-based implementation of PromptStore with JSON persistence.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/shared"
)

type fileStore struct {
	mu          sync.RWMutex
	dir         string
	cache       map[string]*prompt.Prompt
	enableCache bool
}

// NewFileStore creates a new file-based prompt store.
func NewFileStore(dir string, enableCache bool) PromptStore {
	return &fileStore{
		dir:         dir,
		cache:       make(map[string]*prompt.Prompt),
		enableCache: enableCache,
	}
}

// Load retrieves a prompt by versioned ID or alias.
// Returns nil if not found (not an error).
func (f *fileStore) Load(_ context.Context, versionedIDOrAlias prompt.VersionedPromptID) (*prompt.Prompt, error) {
	id := versionedIDOrAlias.String()

	// Check cache first
	if f.enableCache {
		f.mu.RLock()
		if cached, exists := f.cache[id]; exists {
			f.mu.RUnlock()
			return copyPrompt(cached), nil
		}
		f.mu.RUnlock()
	}

	// Read from file
	path := f.getFilePath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	// Unmarshal JSON
	var p prompt.Prompt
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompt: %w", err)
	}

	// Update cache
	if f.enableCache {
		f.mu.Lock()
		f.cache[id] = &p
		f.mu.Unlock()
	}

	return copyPrompt(&p), nil
}

// SaveNewVersion creates a new version with auto-incremented patch version.
// Returns new prompt with versioned ID (e.g., "subagent_system@1.0.1").
func (f *fileStore) SaveNewVersion(ctx context.Context, baseID prompt.PromptID, content string, name string) (*prompt.Prompt, error) {
	_ = ctx // Reserved for future use (cancellation, logging, tracing)
	// Find the latest version
	latestVersion, latestPrompt, err := f.findLatestVersion(baseID)
	if err != nil {
		return nil, err
	}

	// Determine new version
	var newVersion *semver.Version
	if latestVersion == nil {
		// First version: 1.0.0
		newVersion = semver.New(1, 0, 0, "", "")
	} else {
		// Increment patch version
		newVersion = incrementPatchVersion(latestVersion)
	}

	// Create versioned ID
	versionedID := baseID.WithSemVer(newVersion)

	now := time.Now()

	// Create new prompt
	newPrompt := &prompt.Prompt{
		ID:        versionedID.String(),
		Name:      name,
		Content:   content,
		Context:   make(map[string]any),
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		Version:   newVersion,
		IsBuiltin: false,
	}

	// Copy context from previous version if exists
	if latestPrompt != nil && latestPrompt.Context != nil {
		for k, v := range latestPrompt.Context {
			newPrompt.Context[k] = v
		}
	}

	// Add aliases to new version
	newPrompt.Tags = append(newPrompt.Tags, baseID.String(), baseID.String()+"@latest")

	// Write to file with locking
	if err := f.writePromptLocked(newPrompt); err != nil {
		return nil, err
	}

	// Remove aliases from old versions
	if latestPrompt != nil {
		// Remove @latest tag from the old version file
		var newTags []string
		if latestPrompt.Tags != nil {
			for _, tag := range latestPrompt.Tags {
				if tag != baseID.String()+"@latest" {
					newTags = append(newTags, tag)
				}
			}
		}
		latestPrompt.Tags = newTags
		latestPrompt.UpdatedAt = now

		// Write the updated old version
		if err := f.writePromptLocked(latestPrompt); err != nil {
			return nil, fmt.Errorf("failed to update old version: %w", err)
		}

		// Update cache if enabled
		if f.enableCache {
			f.mu.Lock()
			f.cache[latestPrompt.ID] = latestPrompt
			delete(f.cache, baseID.String())
			delete(f.cache, baseID.String()+"@latest")
			f.mu.Unlock()
		}
	}

	// Update cache
	if f.enableCache {
		f.mu.Lock()
		f.cache[versionedID.String()] = newPrompt
		f.cache[baseID.String()] = newPrompt
		f.cache[baseID.String()+"@latest"] = newPrompt
		f.mu.Unlock()
	}

	return copyPrompt(newPrompt), nil
}

// SaveBuiltinVersion creates a new version with IsBuiltin=true.
// Used for bootstrapping built-in prompts that cannot be deleted.
// Returns new prompt with versioned ID (e.g., "subagent_system@1.0.0").
func (f *fileStore) SaveBuiltinVersion(ctx context.Context, baseID prompt.PromptID, content string, name string) (*prompt.Prompt, error) {
	_ = ctx // Reserved for future use (cancellation, logging, tracing)
	// Find the latest version
	latestVersion, latestPrompt, err := f.findLatestVersion(baseID)
	if err != nil {
		return nil, err
	}

	// Determine new version
	var newVersion *semver.Version
	if latestVersion == nil {
		// First version: 1.0.0
		newVersion = semver.New(1, 0, 0, "", "")
	} else {
		// Increment patch version
		newVersion = incrementPatchVersion(latestVersion)
	}

	// Create versioned ID
	versionedID := baseID.WithSemVer(newVersion)

	now := time.Now()

	// Create new prompt with IsBuiltin=true
	newPrompt := &prompt.Prompt{
		ID:        versionedID.String(),
		Name:      name,
		Content:   content,
		Context:   make(map[string]any),
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		Version:   newVersion,
		IsBuiltin: true,
	}

	// Copy context from previous version if exists
	if latestPrompt != nil && latestPrompt.Context != nil {
		for k, v := range latestPrompt.Context {
			newPrompt.Context[k] = v
		}
	}

	// Add aliases to new version
	newPrompt.Tags = append(newPrompt.Tags, baseID.String(), baseID.String()+"@latest")

	// Write to file with locking
	if err := f.writePromptLocked(newPrompt); err != nil {
		return nil, err
	}

	// Remove aliases from old versions
	if latestPrompt != nil {
		// Remove @latest tag from the old version file
		var newTags []string
		if latestPrompt.Tags != nil {
			for _, tag := range latestPrompt.Tags {
				if tag != baseID.String()+"@latest" {
					newTags = append(newTags, tag)
				}
			}
		}
		latestPrompt.Tags = newTags
		latestPrompt.UpdatedAt = now

		// Write the updated old version
		if err := f.writePromptLocked(latestPrompt); err != nil {
			return nil, fmt.Errorf("failed to update old version: %w", err)
		}

		// Update cache if enabled
		if f.enableCache {
			f.mu.Lock()
			f.cache[latestPrompt.ID] = latestPrompt
			delete(f.cache, baseID.String())
			delete(f.cache, baseID.String()+"@latest")
			f.mu.Unlock()
		}
	}

	// Update cache
	if f.enableCache {
		f.mu.Lock()
		f.cache[versionedID.String()] = newPrompt
		f.cache[baseID.String()] = newPrompt
		f.cache[baseID.String()+"@latest"] = newPrompt
		f.mu.Unlock()
	}

	return copyPrompt(newPrompt), nil
}

// Delete removes a prompt by versioned ID.
// Returns nil if not found.
// Returns error for IsBuiltin prompts.
func (f *fileStore) Delete(ctx context.Context, versionedID prompt.VersionedPromptID) error {
	id := versionedID.String()

	// Load the prompt first to check if it's builtin
	p, err := f.Load(ctx, versionedID)
	if err != nil {
		return shared.WrapInternal(err, "failed to load prompt for deletion")
	}
	if p == nil {
		return nil
	}

	// Check if built-in
	if p.IsBuiltin {
		return ErrPromptIsBuiltin
	}

	// Delete all files associated with this prompt
	baseID := versionedID.BaseID()
	if baseID != "" {
		// Delete all versions
		versions, err := f.ListVersions(ctx, baseID)
		if err != nil {
			return shared.WrapInternal(err, "failed to list prompt versions for deletion")
		}

		for _, v := range versions {
			path := f.getFilePath(v.ID)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete prompt file: %w", err)
			}
		}

		// Delete alias files
		for _, aliasID := range []string{baseID.String(), baseID.String() + "@latest"} {
			path := f.getFilePath(aliasID)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				// Ignore errors for aliases
				_ = err
			}
		}

		// Update cache
		if f.enableCache {
			f.mu.Lock()
			for _, v := range versions {
				delete(f.cache, v.ID)
			}
			delete(f.cache, baseID.String())
			delete(f.cache, baseID.String()+"@latest")
			f.mu.Unlock()
		}
	} else {
		// Single file delete
		path := f.getFilePath(id)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete prompt file: %w", err)
		}

		// Update cache
		if f.enableCache {
			f.mu.Lock()
			delete(f.cache, id)
			f.mu.Unlock()
		}
	}

	return nil
}

// List returns prompts matching the given filter criteria.
func (f *fileStore) List(_ context.Context, filter *ListFilter) ([]*prompt.Prompt, error) {
	// Read directory
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*prompt.Prompt{}, nil
		}
		return nil, fmt.Errorf("failed to read store directory: %w", err)
	}

	// Use a map to avoid duplicates
	seen := make(map[string]bool)
	result := make([]*prompt.Prompt, 0, len(entries))

	for _, entry := range entries {
		// Skip non-JSON files
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Extract ID from filename
		id := strings.TrimSuffix(entry.Name(), ".json")

		// Skip aliases (non-versioned IDs and @latest)
		if !strings.Contains(id, "@") || strings.HasSuffix(id, "@latest") {
			continue
		}

		// Skip if already seen
		if seen[id] {
			continue
		}

		// Load prompt
		path := filepath.Join(f.dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip files that can't be read
		}

		var p prompt.Prompt
		if err := json.Unmarshal(data, &p); err != nil {
			continue // Skip files that can't be unmarshaled
		}

		// Apply filters
		if filter != nil {
			// Filter by tags
			if len(filter.Tags) > 0 && !hasAnyTag(p.Tags, filter.Tags) {
				continue
			}

			// Filter by IDs
			if len(filter.IDs) > 0 && !containsVersionedID(p.ID, filter.IDs) {
				continue
			}
		}

		seen[id] = true
		result = append(result, copyPrompt(&p))
	}

	return result, nil
}

// Exists checks if a prompt exists by versioned ID or alias.
func (f *fileStore) Exists(_ context.Context, versionedIDOrAlias prompt.VersionedPromptID) (bool, error) {
	id := versionedIDOrAlias.String()

	// Check cache first
	if f.enableCache {
		f.mu.RLock()
		_, exists := f.cache[id]
		f.mu.RUnlock()
		if exists {
			return true, nil
		}
	}

	// Check file
	path := f.getFilePath(id)
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// ListTags returns all unique tags across all prompts.
func (f *fileStore) ListTags(ctx context.Context) ([]string, error) {
	prompts, err := f.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	tagSet := make(map[string]bool)
	for _, p := range prompts {
		for _, tag := range p.Tags {
			tagSet[tag] = true
		}
	}

	result := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		result = append(result, tag)
	}

	return result, nil
}

// ResolveAlias resolves a base ID or alias to the actual versioned prompt.
// Returns nil if not found.
func (f *fileStore) ResolveAlias(ctx context.Context, versionedIDOrAlias prompt.VersionedPromptID) (*prompt.Prompt, error) {
	id := versionedIDOrAlias.String()

	// Direct lookup first
	p, err := f.Load(ctx, versionedIDOrAlias)
	if err != nil {
		return nil, err
	}
	if p != nil {
		// If this is an alias (no version or @latest suffix), find the versioned ID
		if !strings.Contains(id, "@") || strings.HasSuffix(id, "@latest") {
			// For file store, we need to scan files to find the one with @latest tag
			baseID := versionedIDOrAlias.BaseID()
			if strings.HasSuffix(id, "@latest") {
				baseID = prompt.PromptID(strings.TrimSuffix(id, "@latest"))
			}
			return f.findLatestWithTag(ctx, baseID, baseID.String()+"@latest")
		}
		return copyPrompt(p), nil
	}

	// If still not found and input has @latest suffix, scan for it
	if strings.HasSuffix(id, "@latest") {
		baseID := prompt.PromptID(strings.TrimSuffix(id, "@latest"))
		return f.findLatestWithTag(ctx, baseID, id)
	}

	// Try @latest
	latestID := prompt.VersionedPromptID(id + "@latest")
	p, err = f.Load(ctx, latestID)
	if err != nil {
		return nil, err
	}
	if p != nil {
		return copyPrompt(p), nil
	}

	// If still not found, try to scan for it
	if !strings.Contains(id, "@") {
		baseID := prompt.PromptID(id)
		return f.findLatestWithTag(ctx, baseID, baseID.String()+"@latest")
	}

	return nil, nil
}

// findLatestWithTag scans files to find the prompt with the specified tag.
func (f *fileStore) findLatestWithTag(_ context.Context, baseID prompt.PromptID, tag string) (*prompt.Prompt, error) {
	// Read directory
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read store directory: %w", err)
	}

	for _, entry := range entries {
		// Check if file matches baseID@version.json pattern
		if !strings.HasPrefix(entry.Name(), baseID.String()+"@") {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "@latest.json") {
			continue
		}

		// Load prompt
		path := filepath.Join(f.dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var p prompt.Prompt
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}

		// Check if this prompt has the tag we're looking for
		if p.Tags != nil {
			for _, t := range p.Tags {
				if t == tag {
					return copyPrompt(&p), nil
				}
			}
		}
	}

	return nil, nil
}

// ListVersions returns all versions of a prompt by base ID.
func (f *fileStore) ListVersions(_ context.Context, baseID prompt.PromptID) ([]*prompt.Prompt, error) {
	// Read directory
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*prompt.Prompt{}, nil
		}
		return nil, fmt.Errorf("failed to read store directory: %w", err)
	}

	result := make([]*prompt.Prompt, 0, len(entries))
	seen := make(map[string]bool)

	for _, entry := range entries {
		// Check if file matches baseID@version.json pattern
		if !strings.HasPrefix(entry.Name(), baseID.String()+"@") {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "@latest.json") {
			continue
		}

		// Extract ID from filename
		id := strings.TrimSuffix(entry.Name(), ".json")

		// Skip duplicates
		if seen[id] {
			continue
		}

		// Load prompt
		path := filepath.Join(f.dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var p prompt.Prompt
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}

		seen[id] = true
		result = append(result, copyPrompt(&p))
	}

	return result, nil
}

// SetLatestAlias sets the @latest alias to point to a specific version.
// Removes @latest from all other versions of the same base ID.
func (f *fileStore) SetLatestAlias(ctx context.Context, baseID prompt.PromptID, targetVersionedID prompt.VersionedPromptID) error {
	// Find the target prompt
	targetPrompt, err := f.Load(ctx, targetVersionedID)
	if err != nil {
		return shared.WrapInternal(err, "failed to load prompt for setting latest alias")
	}
	if targetPrompt == nil {
		return ErrPromptNotFound
	}

	now := time.Now()

	// List all versions
	versions, err := f.ListVersions(ctx, baseID)
	if err != nil {
		return shared.WrapInternal(err, "failed to list prompt versions for setting latest alias")
	}

	// Remove @latest tag from all versions
	for _, v := range versions {
		var newTags []string
		if v.Tags != nil {
			for _, tag := range v.Tags {
				if tag != baseID.String()+"@latest" {
					newTags = append(newTags, tag)
				}
			}
		}
		v.Tags = newTags
		v.UpdatedAt = now

		// Write updated file
		if err := f.writePromptLocked(v); err != nil {
			return shared.WrapInternal(err, "failed to write prompt version")
		}

		// Update cache
		if f.enableCache {
			f.mu.Lock()
			f.cache[v.ID] = v
			f.mu.Unlock()
		}
	}

	// Add @latest to target prompt
	if targetPrompt.Tags == nil {
		targetPrompt.Tags = []string{}
	}
	targetPrompt.Tags = append(targetPrompt.Tags, baseID.String()+"@latest")
	targetPrompt.UpdatedAt = now

	// Write target prompt
	if err := f.writePromptLocked(targetPrompt); err != nil {
		return shared.WrapInternal(err, "failed to write target prompt with latest alias")
	}

	// Update cache
	if f.enableCache {
		f.mu.Lock()
		f.cache[targetVersionedID.String()] = targetPrompt
		f.cache[baseID.String()+"@latest"] = targetPrompt
		f.mu.Unlock()
	}

	return nil
}

// =============================================================================
// Helper functions
// =============================================================================

// getFilePath returns the file path for a prompt ID.
func (f *fileStore) getFilePath(id string) string {
	return filepath.Join(f.dir, id+".json")
}

// writePromptLocked writes a prompt to file with exclusive locking.
func (f *fileStore) writePromptLocked(p *prompt.Prompt) error {
	path := f.getFilePath(p.ID)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file with locking
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open prompt file: %w", err)
	}
	defer func() { _ = file.Close() }() // Error from Close ignored in defer

	// Acquire exclusive lock
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock file: %w", err)
	}
	defer func() { _ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN) }() // Error from unlock ignored in defer

	// Marshal JSON
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal prompt: %w", err)
	}

	// Write data
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write prompt: %w", err)
	}

	return nil
}

// findLatestVersion finds the latest version of a prompt by base ID.
func (f *fileStore) findLatestVersion(baseID prompt.PromptID) (*semver.Version, *prompt.Prompt, error) {
	versions, err := f.ListVersions(context.Background(), baseID)
	if err != nil {
		return nil, nil, err
	}

	var latestVersion *semver.Version
	var latestPrompt *prompt.Prompt

	for _, p := range versions {
		if latestVersion == nil || p.Version.GreaterThan(latestVersion) {
			latestVersion = p.Version
			latestPrompt = p
		}
	}

	return latestVersion, latestPrompt, nil
}
