// Package manager provides DI provider for PromptManager with PromptStore injection.
package manager

import (
	"github.com/denkhaus/gollum/pkg/prompt/store"
	"github.com/denkhaus/gollum/pkg/startup"
	"github.com/samber/do/v2"
)

// NewPromptManagerProvider creates a PromptManager with PromptStore from DI.
// This provider requires PromptStore to be registered in the DI container.
func NewPromptManagerProvider(injector do.Injector) (PromptManager, error) {
	// Inject PromptStore from DI container
	store := do.MustInvoke[store.PromptStore](injector)
	// Inject StartupContextService from DI container
	startupContextService := do.MustInvoke[startup.StartupContextService](injector)

	// Create manager with store and startup context service
	mgr := NewPromptManager(store, startupContextService)

	return mgr, nil
}
