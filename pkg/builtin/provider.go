// Package builtin provides production-ready built-in hooks for common use cases.
// These hooks can be registered out-of-the-box and configured via environment variables.
package builtin

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/samber/do/v2"
)

// NewBuiltinHooksProvider is a DI provider that registers all built-in hooks with the HookManager.
// This provider must be called after the HookManager is registered in the DI container.
func NewBuiltinHooksProvider(injector do.Injector) (func() error, error) {
	// Get required services
	hm := do.MustInvoke[hooks.HookManager](injector)
	log := do.MustInvoke[logger.LoggerService](injector)

	// Create hooks
	loggingHook, err := NewLoggingHook(injector)
	if err != nil {
		return nil, fmt.Errorf("failed to create logging hook: %w", err)
	}

	securityHook, err := NewSecurityHook(injector)
	if err != nil {
		return nil, fmt.Errorf("failed to create security hook: %w", err)
	}

	langfuseHook, err := NewLangfuseHooksProvider(injector)
	if err != nil {
		return nil, fmt.Errorf("failed to create Langfuse hook: %w", err)
	}

	// Register logging hooks
	if err := RegisterLoggingHooks(hm, loggingHook); err != nil {
		return nil, fmt.Errorf("failed to register logging hooks: %w", err)
	}
	log.Info("Builtin hooks: Logging registered")

	// Register security hooks
	if err := RegisterSecurityHooks(hm, securityHook); err != nil {
		return nil, fmt.Errorf("failed to register security hooks: %w", err)
	}
	if securityHook.isStrictMode() {
		log.Info("Builtin hooks: Security registered (strict mode)")
	} else {
		log.Info("Builtin hooks: Security not registered (lenient mode)")
	}

	// Register Langfuse hooks
	if err := RegisterLangfuseHooks(hm, langfuseHook); err != nil {
		return nil, fmt.Errorf("failed to register Langfuse hooks: %w", err)
	}
	log.Info("Builtin hooks: Langfuse registered")

	// Return a no-op shutdown function
	return func() error {
		return nil
	}, nil
}
