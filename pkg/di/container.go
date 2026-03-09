// Package di provides dependency injection container management for the Gollum application.
package di

import (
	"context"

	"github.com/denkhaus/gollum/pkg/agents"
	"github.com/denkhaus/gollum/pkg/app"
	"github.com/denkhaus/gollum/pkg/builtin"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/diff"
	"github.com/denkhaus/gollum/pkg/events"
	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows/executor"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/llm"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
	mcpconfig "github.com/denkhaus/gollum/pkg/mcp/config"
	mcpregistry "github.com/denkhaus/gollum/pkg/mcp/registry"
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
	"github.com/denkhaus/gollum/pkg/workspace"
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

// RegisterServices provides all service factories to the DI container.
// NOTE: do.Provide is lazy - services are only instantiated when invoked.
// The order here does NOT affect instantiation order.
func (p *containerImpl) RegisterServices(_ context.Context) do.Injector {
	// Core services
	do.Provide(p.injector, config.NewService)
	do.Provide(p.injector, logger.NewService)
	do.Provide(p.injector, llm.NewClientProvider)

	// Hooks
	do.Provide(p.injector, hooks.NewHookManager)
	do.Provide(p.injector, builtin.NewBuiltinHooksProvider)

	// Event Bus - must be before services that use it
	do.Provide(p.injector, events.NewBusProvider)

	// Registry
	do.Provide(p.injector, registry.NewAgentRegistry)

	// Workspace and Skills
	do.Provide(p.injector, workspace.NewServiceProvider)
	do.Provide(p.injector, skills.NewService)

	// UI
	do.Provide(p.injector, ui.NewAgentMessenger)
	do.Provide(p.injector, markdown.ProvideRenderer)
	do.Provide(p.injector, diff.NewProvider)

	// Middleware
	do.Provide(p.injector, middleware.NewDisplayMiddlewareProvider)
	do.Provide(p.injector, middleware.NewSummaryMiddlewareProvider)

	// State
	do.Provide(p.injector, state.NewFileStateManager)
	do.Provide(p.injector, profiling.NewProfilingServiceProvider)

	// Agents
	do.Provide(p.injector, agents.NewAgentFactory)
	do.Provide(p.injector, tools.NewAgentExecutionHelper)

	// Tool validation
	do.Provide(p.injector, tools.NewToolNameValidatorProvider)

	// MCP (Model Context Protocol) services
	do.Provide(p.injector, mcpconfig.NewConfigLoader)
	do.Provide(p.injector, mcpregistry.NewMCPRegistry)

	// Flows
	do.Provide(p.injector, flowregistry.NewFlowRegistryService)
	do.Provide(p.injector, executor.NewFlowExecutor)

	// Extensions
	do.Provide(p.injector, extensions.NewGatewayService)
	do.Provide(p.injector, extensions.NewYaegiFuncRunner)
	do.Provide(p.injector, extensions.NewYaegiLoader)
	do.Provide(p.injector, extensions.NewExtensionServiceWithWorkspace)

	// Tools
	do.Provide(p.injector, tools.NewFlowToolsProvider)
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

	// Prompts
	do.Provide(p.injector, store.NewPromptStore)
	do.Provide[manager.PromptManager](p.injector, manager.NewPromptManagerProvider)
	do.Provide(p.injector, optimizer.NewOptimizerProvider)

	// Application
	do.Provide(p.injector, app.NewService)

	return p.injector
}

// Shutdown gracefully shuts down the container
func (p *containerImpl) Shutdown() {
	_ = do.Shutdown[any](p.injector)
}
