// Package manager provides DI provider for PromptManager with PromptStore injection.
package manager

import (
	promptstore "github.com/denkhaus/gollum/pkg/prompt/store"
	"github.com/samber/do/v2"
)

// NewPromptManagerProvider creates a PromptManager with PromptStore from DI.
// This provider requires PromptStore to be registered in the DI container.
func NewPromptManagerProvider(injector do.Injector) (PromptManager, error) {
	// Inject PromptStore from DI container
	store := do.MustInvoke[promptstore.PromptStore](injector)

	// Create manager with store
	mgr := NewPromptManager(store)

	return mgr, nil
}
