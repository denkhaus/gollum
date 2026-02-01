// Package store provides in-memory implementation of PromptStore for testing.
package store

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/denkhaus/gollum/pkg/prompt"
)

type memoryStore struct {
	mu      sync.RWMutex
	prompts map[string]*prompt.Prompt
}

// NewMemoryStore creates a new in-memory prompt store.
func NewMemoryStore() PromptStore {
	return &memoryStore{
		prompts: make(map[string]*prompt.Prompt),
	}
}

// Load retrieves a prompt by ID.
// Returns nil if not found (not an error).
func (m *memoryStore) Load(_ context.Context, id string) (*prompt.Prompt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, exists := m.prompts[id]
	if !exists {
		return nil, nil
	}

	return copyPrompt(p), nil
}

// SaveNewVersion creates a new version with auto-incremented patch version.
// Returns new prompt with versioned ID (e.g., "subagent@1.0.1").
func (m *memoryStore) SaveNewVersion(ctx context.Context, baseID string, content string, name string) (*prompt.Prompt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the latest version of this prompt
	var latestVersion *semver.Version
	var latestPrompt *prompt.Prompt

	// Look for existing prompts with this base ID
	for id, p := range m.prompts {
		// Check if this prompt belongs to the base ID
		if strings.HasPrefix(id, baseID+"@") {
			if latestVersion == nil || p.Version.GreaterThan(latestVersion) {
				latestVersion = p.Version
				latestPrompt = p
			}
		}
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
	versionID := fmt.Sprintf("%s@%s", baseID, newVersion.String())

	now := m.getTime(ctx)

	// Create new prompt
	newPrompt := &prompt.Prompt{
		ID:        versionID,
		Name:      name,
		Content:   content,
		Context:   make(map[string]interface{}),
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

	// Remove aliases from old version and add to new version
	if latestPrompt != nil {
		for _, oldID := range []string{baseID, baseID + "@latest"} {
			if oldPrompt, exists := m.prompts[oldID]; exists {
				// Remove alias tags from old version
				var newTags []string
				for _, tag := range oldPrompt.Tags {
					if tag != oldID {
						newTags = append(newTags, tag)
					}
				}
				oldPrompt.Tags = newTags
				oldPrompt.UpdatedAt = now
			}
		}
	}

	// Add aliases to new version
	newPrompt.Tags = append(newPrompt.Tags, baseID, baseID+"@latest")

	// Store the new prompt
	m.prompts[versionID] = newPrompt
	m.prompts[baseID] = newPrompt
	m.prompts[baseID+"@latest"] = newPrompt

	return copyPrompt(newPrompt), nil
}

// Delete removes a prompt by ID.
// Returns nil if not found.
// Returns error for IsBuiltin prompts.
func (m *memoryStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, exists := m.prompts[id]
	if !exists {
		return nil
	}

	// Check if built-in
	if p.IsBuiltin {
		return ErrPromptIsBuiltin
	}

	now := m.getTime(ctx)

	// Remove all references to this prompt
	for key, prompt := range m.prompts {
		if prompt == p {
			delete(m.prompts, key)
		}
	}

	// If this was the latest version, try to set @latest to the previous version
	baseID := m.extractBaseID(id)
	if baseID != "" {
		var previousVersion *prompt.Prompt

		for key, candidate := range m.prompts {
			if strings.HasPrefix(key, baseID+"@") && key != baseID+"@latest" {
				if previousVersion == nil || candidate.Version.GreaterThan(previousVersion.Version) {
					previousVersion = candidate
				}
			}
		}

		// Update @latest alias
		delete(m.prompts, baseID+"@latest")
		if previousVersion != nil {
			m.prompts[baseID+"@latest"] = previousVersion
			// Add @latest tag
			previousVersion.Tags = append(previousVersion.Tags, baseID+"@latest")
			previousVersion.UpdatedAt = now
		}
	}

	return nil
}

// List returns prompts matching the given filter criteria.
func (m *memoryStore) List(_ context.Context, filter *ListFilter) ([]*prompt.Prompt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Pre-allocate result slice with reasonable capacity
	result := make([]*prompt.Prompt, 0, len(m.prompts))

	// Use a map to avoid duplicates
	seen := make(map[string]bool)

	for _, p := range m.prompts {
		// Skip aliases (non-versioned IDs and @latest)
		if !strings.Contains(p.ID, "@") || strings.HasSuffix(p.ID, "@latest") {
			continue
		}

		// Skip if already seen
		if seen[p.ID] {
			continue
		}

		// Apply filters
		if filter != nil {
			// Filter by tags
			if len(filter.Tags) > 0 && !hasAnyTag(p.Tags, filter.Tags) {
				continue
			}

			// Filter by IDs
			if len(filter.IDs) > 0 && !containsAny(p.ID, filter.IDs) {
				continue
			}
		}

		seen[p.ID] = true
		result = append(result, copyPrompt(p))
	}

	return result, nil
}

// Exists checks if a prompt exists by ID.
func (m *memoryStore) Exists(_ context.Context, id string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.prompts[id]
	return exists, nil
}

// ListTags returns all unique tags across all prompts.
func (m *memoryStore) ListTags(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tagSet := make(map[string]bool)
	for _, p := range m.prompts {
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

// ResolveAlias resolves shortcuts to versioned IDs.
// Resolves "subagent" -> "subagent@latest" -> versioned ID.
func (m *memoryStore) ResolveAlias(_ context.Context, id string) (*prompt.Prompt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Direct lookup
	if p, exists := m.prompts[id]; exists {
		// If this is an alias (no version), return the actual prompt
		if !strings.Contains(id, "@") || strings.HasSuffix(id, "@latest") {
			// Find the versioned ID
			for key, prompt := range m.prompts {
				if prompt == p && strings.Contains(key, "@") && !strings.HasSuffix(key, "@latest") {
					return copyPrompt(prompt), nil
				}
			}
		}
		return copyPrompt(p), nil
	}

	// Try @latest
	latestID := id + "@latest"
	if p, exists := m.prompts[latestID]; exists {
		return copyPrompt(p), nil
	}

	return nil, nil
}

// ListVersions returns all versions of a prompt base ID.
func (m *memoryStore) ListVersions(_ context.Context, baseID string) ([]*prompt.Prompt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*prompt.Prompt

	for id, p := range m.prompts {
		// Match prompts with this base ID (including versioned ones)
		if strings.HasPrefix(id, baseID+"@") && !strings.HasSuffix(id, "@latest") {
			// Use a map to avoid duplicates
			found := false
			for _, existing := range result {
				if existing.ID == p.ID {
					found = true
					break
				}
			}
			if !found {
				result = append(result, copyPrompt(p))
			}
		}
	}

	return result, nil
}

// SetLatestAlias sets the @latest alias to a specific version.
// Removes @latest from all other versions of the same base ID.
func (m *memoryStore) SetLatestAlias(ctx context.Context, baseID, versionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.getTime(ctx)

	// Find the target prompt
	targetPrompt, exists := m.prompts[versionID]
	if !exists {
		return ErrPromptNotFound
	}

	// Remove @latest from all versions of this base ID
	for id, p := range m.prompts {
		if strings.HasPrefix(id, baseID+"@") && id != baseID+"@latest" {
			// Remove @latest tag
			var newTags []string
			if p.Tags != nil {
				for _, tag := range p.Tags {
					if tag != baseID+"@latest" {
						newTags = append(newTags, tag)
					}
				}
			}
			p.Tags = newTags
			p.UpdatedAt = now
		}
	}

	// Add @latest to target prompt
	if targetPrompt.Tags == nil {
		targetPrompt.Tags = []string{}
	}
	targetPrompt.Tags = append(targetPrompt.Tags, baseID+"@latest")
	targetPrompt.UpdatedAt = now

	// Update the @latest alias
	m.prompts[baseID+"@latest"] = targetPrompt

	return nil
}

// Helper functions

// copyPrompt creates a deep copy of a prompt.
func copyPrompt(p *prompt.Prompt) *prompt.Prompt {
	if p == nil {
		return nil
	}

	// Copy context
	contextCopy := make(map[string]interface{})
	for k, v := range p.Context {
		contextCopy[k] = v
	}

	// Copy tags
	tagsCopy := make([]string, len(p.Tags))
	copy(tagsCopy, p.Tags)

	return &prompt.Prompt{
		ID:        p.ID,
		Name:      p.Name,
		Content:   p.Content,
		Context:   contextCopy,
		Tags:      tagsCopy,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		Version:   p.Version,
		IsBuiltin: p.IsBuiltin,
	}
}

// hasAnyTag checks if any of the filter tags exist in the tags list.
func hasAnyTag(tags []string, filter []string) bool {
	filterSet := make(map[string]bool)
	for _, f := range filter {
		filterSet[f] = true
	}

	for _, tag := range tags {
		if filterSet[tag] {
			return true
		}
	}

	return false
}

// containsAny checks if the ID matches any of the filter IDs.
func containsAny(id string, ids []string) bool {
	for _, filterID := range ids {
		if strings.Contains(id, filterID) {
			return true
		}
	}
	return false
}

// incrementPatchVersion increments the patch version of a semver.
func incrementPatchVersion(v *semver.Version) *semver.Version {
	inc := v.IncPatch()
	return &inc
}

// extractBaseID extracts the base ID from a versioned ID.
func (m *memoryStore) extractBaseID(id string) string {
	idx := strings.Index(id, "@")
	if idx == -1 {
		return ""
	}
	return id[:idx]
}

// getTime returns the current time or a time from context if available.
func (m *memoryStore) getTime(_ context.Context) time.Time {
	// In a real implementation, you might extract time from context
	// For now, just return current time
	return time.Now()
}
