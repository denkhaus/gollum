// Package di provides dependency injection container management for the Gollum application.
package di

import (
	"context"

	"github.com/denkhaus/gollum/pkg/agents"
	"github.com/denkhaus/gollum/pkg/app"
	"github.com/denkhaus/gollum/pkg/builtin"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/llm"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/middleware"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/denkhaus/gollum/pkg/ui"
	"github.com/samber/do/v2"
)

// Container holds all dependency injection providers
type Container struct {
	injector do.Injector
}

// NewContainer creates a new dependency injection container
func NewContainer() *Container {
	return &Container{
		injector: do.New(),
	}
}

// GetInjector returns the underlying injector
func (c *Container) GetInjector() do.Injector {
	return c.injector
}

// RegisterServices registers all services in the dependency injection container
func (c *Container) RegisterServices(_ context.Context) do.Injector {
	// Register config service first (other services depend on it)
	do.Provide(c.injector, config.NewService)

	// Register logger service
	do.Provide(c.injector, logger.NewService)
	do.Provide(c.injector, llm.NewClientProvider)

	// Register HookManager
	do.Provide(c.injector, hooks.NewHookManager)

	// Register built-in hooks (must come after HookManager)
	do.Provide(c.injector, builtin.NewBuiltinHooksProvider)

	// Register agent registry before agent provider (agent provider depends on it)
	do.Provide(c.injector, registry.NewAgentRegistry)

	// Register UI components
	do.Provide(c.injector, ui.NewAgentMessenger)

	// Register middleware providers
	do.Provide(c.injector, middleware.NewDisplayMiddlewareProvider)
	do.Provide(c.injector, middleware.NewSummaryMiddlewareProvider)

	// Register FileStateManager
	do.Provide(c.injector, state.NewFileStateManager)

	// Register agent factory
	do.Provide(c.injector, agents.NewAgentFactory)

	// Register agent execution helper
	do.Provide(c.injector, tools.NewAgentExecutionHelper)

	// Register tool providers
	do.Provide(c.injector, tools.NewSpawnAgentToolProvider)
	do.Provide(c.injector, tools.NewAgentOutputToolProvider)
	do.Provide(c.injector, tools.NewRemoveAgentToolProvider)
	do.Provide(c.injector, tools.NewResumeAgentToolProvider)
	do.Provide(c.injector, tools.NewListAgentsToolProvider)
	do.Provide(c.injector, tools.NewCurrentTimeToolProvider)
	do.Provide(c.injector, tools.NewBashToolProvider)
	do.Provide(c.injector, tools.NewWriteFileToolProvider)
	do.Provide(c.injector, tools.NewReadFileToolProvider)
	do.Provide(c.injector, tools.NewGlobToolProvider)
	do.Provide(c.injector, tools.NewGrepToolProvider)
	do.Provide(c.injector, tools.NewEditToolProvider)
	do.Provide(c.injector, tools.NewSessionLogsToolProvider)

	do.Provide(c.injector, prompt.NewPromptManager)

	// Register application service (must be last, depends on all other services)
	do.Provide(c.injector, app.NewService)

	return c.injector
}

// Shutdown gracefully shuts down the container
func (c *Container) Shutdown() error {
	return do.Shutdown[any](c.injector)
}
