// Package manager provides DI provider for PromptManager with PromptStore injection.
package manager

import (
	promptstore "github.com/denkhaus/gollum/pkg/prompt/store"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/samber/do/v2"
)

// NewPromptManagerProvider creates a PromptManager with PromptStore from DI.
// This provider requires PromptStore and SkillService to be registered in the DI container.
func NewPromptManagerProvider(injector do.Injector) (PromptManager, error) {
	// Inject PromptStore from DI container
	store := do.MustInvoke[promptstore.PromptStore](injector)

	// Inject SkillService from DI container (optional - may be nil)
	skillSvc, _ := do.Invoke[skills.SkillService](injector)

	// Create manager with store and skill service
	mgr := NewPromptManager(store, skillSvc)

	return mgr, nil
}
