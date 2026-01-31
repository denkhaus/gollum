// Package store provides the persistence abstraction for versioned prompts.
package store

import (
	"context"
	"errors"

	"github.com/denkhaus/gollum/pkg/prompt"
)

// PromptStore defines the contract for prompt persistence operations.
type PromptStore interface {
	// SaveNewVersion creates a new version with auto-incremented patch version.
	// Returns new prompt with versioned ID (e.g., "subagent@1.0.1").
	SaveNewVersion(ctx context.Context, baseID string, content string, name string) (*prompt.Prompt, error)

	// Load retrieves a prompt by ID.
	// Returns nil if not found (not an error).
	Load(ctx context.Context, id string) (*prompt.Prompt, error)

	// Delete removes a prompt by ID.
	// Returns nil if not found.
	// Returns error for IsBuiltin prompts.
	Delete(ctx context.Context, id string) error

	// List returns prompts matching the given filter criteria.
	List(ctx context.Context, filter *ListFilter) ([]*prompt.Prompt, error)

	// Exists checks if a prompt exists by ID.
	Exists(ctx context.Context, id string) (bool, error)

	// ListTags returns all unique tags across all prompts.
	ListTags(ctx context.Context) ([]string, error)

	// ResolveAlias resolves shortcuts to versioned IDs.
	// Resolves "subagent" -> "subagent@latest" -> versioned ID.
	ResolveAlias(ctx context.Context, id string) (*prompt.Prompt, error)

	// ListVersions returns all versions of a prompt base ID.
	ListVersions(ctx context.Context, baseID string) ([]*prompt.Prompt, error)

	// SetLatestAlias sets the @latest alias to a specific version.
	// Removes @latest from all other versions of the same base ID.
	SetLatestAlias(ctx context.Context, baseID, versionID string) error
}

// ListFilter provides filtering options for listing prompts.
type ListFilter struct {
	Tags []string // Filter by tags (OR logic)
	IDs  []string // Filter by IDs (OR logic)
}

// Error definitions
var (
	ErrPromptNotFound  = errors.New("prompt not found")
	ErrPromptIsBuiltin = errors.New("cannot delete built-in prompt")
	ErrInvalidPromptID = errors.New("invalid prompt ID format")
	ErrCASFailed       = errors.New("compare-and-swap operation failed")
)
