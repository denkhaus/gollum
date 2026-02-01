// Package prompt provides DI provider for PromptManager with PromptStore injection.
package prompt

import (
	"context"

	"github.com/samber/do/v2"
)

// NewPromptManagerProvider creates a PromptManager with PromptStore injection.
// Uses optional injection pattern - falls back to nil if PromptStore not in DI.
// The PromptStore must be provided by the store package's provider.
func NewPromptManagerProvider(ctx context.Context, injector do.Injector) (PromptManager, error) {
	// Try to get PromptStore from DI container using our local PromptStore interface
	// Use recover for optional injection since do v2 doesn't have TryInvoke
	var st PromptStore
	func() {
		defer func() {
			// If MustInvoke panics, store is not available
			_ = recover()
		}()
		st = do.MustInvoke[PromptStore](injector)
	}()

	// Create manager using the original constructor
	pm, err := NewPromptManager(injector)
	if err != nil {
		return nil, err
	}

	// Set the store via SetStore method
	if pmImpl, ok := pm.(*promptManager); ok && st != nil {
		pmImpl.SetStore(st)
	}

	return pm, nil
}
