// Package optimizer provides prompt optimization strategies for improving agent prompts
// through trajectory analysis and feedback processing.
package optimizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/denkhaus/gollum/pkg/prompt"
	promptstore "github.com/denkhaus/gollum/pkg/prompt/store"
)

// OptimizeAndSave orchestrates the optimization workflow and saves the result to store.
// It loads the current prompt, runs optimization, and saves the optimized version with
// incremented SemVer if warranted.
func OptimizeAndSave(ctx context.Context, optimizer PromptOptimizer, store promptstore.PromptStore, promptID prompt.VersionedPromptID, input *OptimizerInput) (*prompt.Prompt, error) {
	// Load current prompt from store
	current, err := store.Load(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to load prompt: %w", err)
	}
	if current == nil {
		return nil, fmt.Errorf("prompt not found: %s", promptID)
	}

	// Set the current prompt content in the input
	input.Prompt = current.Content

	// Run optimization
	result, err := optimizer.Optimize(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("optimization failed: %w", err)
	}

	// If no adjustment is warranted, return original prompt
	if !result.WarrantsAdjustment {
		return current, nil
	}

	// Extract base ID (strip version suffix)
	baseID := ExtractBaseID(current.ID)

	// Save new version to store
	saved, err := store.SaveNewVersion(ctx, prompt.PromptID(baseID), result.NewPrompt, current.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to save optimized prompt: %w", err)
	}

	return saved, nil
}

// ExtractBaseID strips the version suffix from a prompt ID.
// For example: "supervisor@1.0.0" -> "supervisor", "supervisor" -> "supervisor"
func ExtractBaseID(id string) string {
	idx := strings.LastIndex(id, "@")
	if idx == -1 {
		return id
	}
	return id[:idx]
}

// ParseVersion extracts the SemVer from a prompt ID.
// For example: "supervisor@1.0.0" -> 1.0.0, "supervisor" -> error
func ParseVersion(id string) (*semver.Version, error) {
	idx := strings.LastIndex(id, "@")
	if idx == -1 {
		return nil, fmt.Errorf("no version found in ID: %s", id)
	}

	versionStr := id[idx+1:]
	return semver.NewVersion(versionStr)
}
