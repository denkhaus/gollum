// Package store provides the persistence abstraction for versioned prompts.
package store

import (
	"context"
	"errors"

	"github.com/denkhaus/gollum/pkg/prompt"
)

// PromptStore defines the contract for prompt persistence operations.
//
// # ID Type Conventions
//
// This interface uses two distinct ID types to prevent confusion:
//
//   - **prompt.PromptID** (Base ID): Unversioned identifier like "subagent_system"
//     Used when creating, listing, or managing prompt versions.
//     Methods: SaveNewVersion, SaveBuiltinVersion, ListVersions, SetLatestAlias
//
//   - **prompt.VersionedPromptID** (Versioned ID): Full identifier like "subagent_system@1.0.0"
//     Can also be an alias like "subagent_system@latest" or just "subagent_system".
//     Used when loading or deleting specific prompt versions.
//     Methods: Load, Delete, Exists, ResolveAlias
//
// # Example Usage
//
//	// Create a new version (uses base ID)
//	p, err := store.SaveNewVersion(ctx, prompt.PromptIDSubagentSystem, "content", "name")
//
//	// Load by versioned ID
//	p, err := store.Load(ctx, prompt.VersionedPromptID("subagent_system@1.0.0"))
//
//	// Load by alias
//	p, err := store.Load(ctx, prompt.VersionedPromptID("subagent_system@latest"))
//	p, err := store.Load(ctx, prompt.VersionedPromptID("subagent_system"))  // shorthand for @latest
//
//	// Or use the helper:
//	p, err := store.Load(ctx, prompt.FromBaseID(prompt.PromptIDSubagentSystem, "latest"))
type PromptStore interface {
	// SaveNewVersion creates a new version with auto-incremented patch version.
	// Parameter baseID: the unversioned base identifier (e.g., prompt.PromptIDSubagentSystem).
	// Returns the new prompt with a versioned ID (e.g., "subagent_system@1.0.1").
	SaveNewVersion(ctx context.Context, baseID prompt.PromptID, content string, name string) (*prompt.Prompt, error)

	// SaveBuiltinVersion creates a new version with IsBuiltin=true.
	// Used for bootstrapping built-in prompts that cannot be deleted.
	// Parameter baseID: the unversioned base identifier (e.g., prompt.PromptIDSubagentSystem).
	// Returns the new prompt with a versioned ID (e.g., "subagent_system@1.0.0").
	SaveBuiltinVersion(ctx context.Context, baseID prompt.PromptID, content string, name string) (*prompt.Prompt, error)

	// Load retrieves a prompt by versioned ID or alias.
	// Parameter versionedIDOrAlias can be:
	//   - Full versioned ID: prompt.VersionedPromptID("subagent_system@1.0.0")
	//   - Latest alias: prompt.VersionedPromptID("subagent_system@latest")
	//   - Base ID shorthand: prompt.VersionedPromptID("subagent_system") (resolves to @latest)
	// Returns nil if not found (not an error).
	Load(ctx context.Context, versionedIDOrAlias prompt.VersionedPromptID) (*prompt.Prompt, error)

	// Delete removes a prompt by versioned ID.
	// Parameter versionedID: the full versioned ID (e.g., "subagent_system@1.0.0").
	// Returns nil if not found.
	// Returns ErrPromptIsBuiltin for prompts with IsBuiltin=true.
	Delete(ctx context.Context, versionedID prompt.VersionedPromptID) error

	// List returns prompts matching the given filter criteria.
	List(ctx context.Context, filter *ListFilter) ([]*prompt.Prompt, error)

	// Exists checks if a prompt exists by versioned ID or alias.
	// Parameter versionedIDOrAlias: same format as Load().
	Exists(ctx context.Context, versionedIDOrAlias prompt.VersionedPromptID) (bool, error)

	// ListTags returns all unique tags across all prompts.
	ListTags(ctx context.Context) ([]string, error)

	// ResolveAlias resolves a base ID or alias to the actual versioned prompt.
	// Resolves: "subagent_system" -> "subagent_system@latest" -> prompt with @latest tag.
	// Returns nil if not found.
	ResolveAlias(ctx context.Context, versionedIDOrAlias prompt.VersionedPromptID) (*prompt.Prompt, error)

	// ListVersions returns all versions of a prompt by base ID.
	// Parameter baseID: the unversioned base identifier (e.g., prompt.PromptIDSubagentSystem).
	ListVersions(ctx context.Context, baseID prompt.PromptID) ([]*prompt.Prompt, error)

	// SetLatestAlias sets the @latest alias to point to a specific version.
	// Parameter baseID: the unversioned base identifier.
	// Parameter targetVersionedID: the full versioned ID to point @latest to.
	// Removes @latest tag from all other versions of the same base ID.
	SetLatestAlias(ctx context.Context, baseID prompt.PromptID, targetVersionedID prompt.VersionedPromptID) error
}

// ListFilter provides filtering options for listing prompts.
type ListFilter struct {
	Tags []string                   // Filter by tags (OR logic)
	IDs  []prompt.VersionedPromptID // Filter by versioned IDs (OR logic)
}

// Error definitions
var (
	ErrPromptNotFound  = errors.New("prompt not found")
	ErrPromptIsBuiltin = errors.New("cannot delete built-in prompt")
	ErrInvalidPromptID = errors.New("invalid prompt ID format")
	ErrCASFailed       = errors.New("compare-and-swap operation failed")
)
