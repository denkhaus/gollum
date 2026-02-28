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
	"github.com/denkhaus/gollum/pkg/markdown"
	"github.com/denkhaus/gollum/pkg/middleware"
	"github.com/denkhaus/gollum/pkg/profiling"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/prompt/optimizer"
	"github.com/denkhaus/gollum/pkg/prompt/store"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/denkhaus/gollum/pkg/ui"
	"github.com/samber/do/v2"
)

// Container defines the dependency injection container interface
type Container interface {
	GetInjector() do.Injector
	RegisterServices(ctx context.Context) do.Injector
	Shutdown()
}

// containerImpl implements the Container interface
// IMPORTANT: Implementation is private (containerImpl, not Container)
type containerImpl struct {
	injector do.Injector
}

// Ensure containerImpl implements Container at compile time
var _ Container = (*containerImpl)(nil)

// NewContainer creates a new dependency injection container
func NewContainer() Container {
	return &containerImpl{
		injector: do.New(),
	}
}

// GetInjector returns the underlying injector
func (p *containerImpl) GetInjector() do.Injector {
	return p.injector
}

// RegisterServices registers all services in the dependency injection container
func (p *containerImpl) RegisterServices(_ context.Context) do.Injector {
	// Register config service first (other services depend on it)
	do.Provide(p.injector, config.NewService)

	// Register logger service
	do.Provide(p.injector, logger.NewService)
	do.Provide(p.injector, llm.NewClientProvider)

	// Register HookManager
	do.Provide(p.injector, hooks.NewHookManager)

	// Register built-in hooks
	do.Provide(p.injector, builtin.NewBuiltinHooksProvider)

	// Register agent registry
	do.Provide(p.injector, registry.NewAgentRegistry)

	// Register skill service
	do.Provide(p.injector, skills.NewService)

	// Register UI components
	do.Provide(p.injector, ui.NewAgentMessenger)

	// Register markdown renderer
	do.Provide(p.injector, markdown.ProvideRenderer)

	// Register middleware providers
	do.Provide(p.injector, middleware.NewDisplayMiddlewareProvider)
	do.Provide(p.injector, middleware.NewSummaryMiddlewareProvider)

	// Register FileStateManager
	do.Provide(p.injector, state.NewFileStateManager)

	// Register ProfilingService
	do.Provide(p.injector, profiling.NewProfilingServiceProvider)

	// Register agent factory
	do.Provide(p.injector, agents.NewAgentFactory)

	// Register agent execution helper
	do.Provide(p.injector, tools.NewAgentExecutionHelper)

	// Register tool registry as ToolNameValidator for skill validation
	do.Provide[skills.ToolNameValidator](p.injector, tools.NewToolNameValidatorProvider)

	// Register tool providers
	do.Provide(p.injector, tools.NewSpawnAgentToolProvider)
	do.Provide(p.injector, tools.NewAgentOutputToolProvider)
	do.Provide(p.injector, tools.NewRemoveAgentToolProvider)
	do.Provide(p.injector, tools.NewResumeAgentToolProvider)
	do.Provide(p.injector, tools.NewListAgentsToolProvider)
	do.Provide(p.injector, tools.NewCurrentTimeToolProvider)
	do.Provide(p.injector, tools.NewBashToolProvider)
	do.Provide(p.injector, tools.NewWriteFileToolProvider)
	do.Provide(p.injector, tools.NewReadFileToolProvider)
	do.Provide(p.injector, tools.NewGlobToolProvider)
	do.Provide(p.injector, tools.NewGrepToolProvider)
	do.Provide(p.injector, tools.NewEditToolProvider)
	do.Provide(p.injector, tools.NewSessionLogsToolProvider)
	do.Provide(p.injector, tools.NewChangeDirectoryToolProvider)
	do.Provide(p.injector, tools.NewInvokeSkillToolProvider)

	// Register prompt store
	do.Provide(p.injector, store.NewPromptStore)
	do.Provide[manager.PromptManager](p.injector, manager.NewPromptManagerProvider)

	// Register prompt optimizer
	do.Provide(p.injector, optimizer.NewOptimizerProvider)

	// Register application service
	do.Provide(p.injector, app.NewService)

	return p.injector
}

// Shutdown gracefully shuts down the container
func (p *containerImpl) Shutdown() {
	_ = do.Shutdown[any](p.injector)
}
