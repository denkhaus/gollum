// Package app provides the main application service that orchestrates
// the Gollum agent system startup and interactive loop.
package app

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
	mcpregistry "github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/denkhaus/gollum/pkg/middleware"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/denkhaus/gollum/pkg/tui"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

const gollumDirName = ".gollum"

// ApplicationService defines the main application service interface
type ApplicationService interface {
	// Run starts the application, performing all initialization and running the interactive loop
	Run(ctx context.Context) error
	// Cleanup restores terminal state and performs other cleanup
	Cleanup()
}

// applicationServiceImpl implements the ApplicationService interface
type applicationServiceImpl struct {
	gollumDir        string
	logService       logger.LoggerService
	fsm              state.FileStateManager
	agentRegistry    registry.AgentRegistry
	promptMgr        manager.PromptManager
	displayProv      middleware.DisplayMiddlewareProvider
	agentFactory     shared.AgentFactory
	markdownRenderer markdown.Renderer
	workspaceService workspace.Service
	skillsService    skills.SkillService
	mcpRegistry      mcpregistry.MCPRegistry
}

// Ensure implementation satisfies interface
var _ ApplicationService = (*applicationServiceImpl)(nil)

// NewService creates a new application service
func NewService(injector do.Injector) (ApplicationService, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm := do.MustInvoke[state.FileStateManager](injector)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)
	promptMgr := do.MustInvoke[manager.PromptManager](injector)
	displayProv := do.MustInvoke[middleware.DisplayMiddlewareProvider](injector)
	agentFactory := do.MustInvoke[shared.AgentFactory](injector)
	markdownRenderer := do.MustInvoke[markdown.Renderer](injector)
	workspaceService := do.MustInvoke[workspace.Service](injector)
	skillsService := do.MustInvoke[skills.SkillService](injector)
	mcpRegistry := do.MustInvoke[mcpregistry.MCPRegistry](injector)

	return &applicationServiceImpl{
		logService:       logService,
		fsm:              fsm,
		agentRegistry:    agentRegistry,
		workspaceService: workspaceService,
		skillsService:    skillsService,
		promptMgr:        promptMgr,
		displayProv:      displayProv,
		agentFactory:     agentFactory,
		markdownRenderer: markdownRenderer,
		mcpRegistry:      mcpRegistry,
	}, nil
}

// Run starts the application, performing all initialization and running the interactive loop
func (p *applicationServiceImpl) Run(ctx context.Context) error {

	// Create .gollum directory
	if err := p.ensureGollumDirectory(); err != nil {
		return fmt.Errorf("failed to create .gollum directory: %w", err)
	}

	// Prime FileStateManager
	if err := p.primeFileStateManager(ctx); err != nil {
		return err
	}

	// Create and register Supervisor agent
	agent, _, err := p.createSupervisorAgent(ctx)
	if err != nil {
		return err
	}

	// Enable file logging (LoggerService handles logs/ subdir and cleanup)
	if err := p.logService.EnableFileLogging(p.gollumDir, agent.GetID()); err != nil {
		return fmt.Errorf("failed to enable file logging: %w", err)
	}
	defer p.logService.CloseFileLogging()

	// Run interactive loop
	return p.runInteractiveLoop(ctx, agent)
}

// primeFileStateManager primes the file state manager with directory scan
func (p *applicationServiceImpl) primeFileStateManager(ctx context.Context) error {
	p.logService.Infof("Priming FileStateManager - scanning working directory...")
	if err := p.fsm.Prime(ctx); err != nil {
		return fmt.Errorf("failed to prime FileStateManager: %w", err)
	}
	p.logService.Infof("FileStateManager primed successfully")
	return nil
}

// createSupervisorAgent creates and registers the Supervisor agent
func (p *applicationServiceImpl) createSupervisorAgent(ctx context.Context) (shared.Agent, *shared.AgentConfig, error) {

	// Get supervisor prompt from PromptManager
	systemPrompt, err := p.promptMgr.GetPromptWithContext(ctx,
		prompt.PromptIDSupervisorSystem,
		&prompt.RenderContext{
			Workspace: &shared.WorkspaceContext{
				SkillsXML:   p.skillsService.GetSkillsXML(),
				Skills:      p.skillsService.GetSkillInfos(),
				CurrentPath: p.workspaceService.GetCurrentWorkspace(),
			},
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get supervisor prompt: %w", err)
	}

	toolSet := p.mcpRegistry.GetToolSets()
	if len(toolSet) == 0 {
		p.logService.Warn("no mcp servers configured for supervison agent")
	}

	// Create agent config
	agentConfig := &shared.AgentConfig{
		AllowCompaction: true,
		SystemPrompt:    systemPrompt,
		ToolSets:        toolSet,
		Role:            "Supervisor Agent",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/glm-4.7",
		},
	}

	// Create agent
	agent, err := p.agentFactory.CreateAgent(ctx, agentConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create supervisor agent: %v", err)
	}

	// Register in registry
	if err := p.agentRegistry.Register(agent, agentConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to register supervisor agent: %w", err)
	}
	p.logService.Infof("Supervisor agent %s registered", agent.GetID())

	return agent, agentConfig, nil
}

// runInteractiveLoop runs the main CLI interactive loop using Bubbletea TUI
func (p *applicationServiceImpl) runInteractiveLoop(ctx context.Context, agent shared.Agent) error {
	// Create agent executor adapter
	executor := &agentExecutorAdapter{agent: agent}

	// Enable TUI mode to disable stdout logging (logs go to buffer only)
	p.logService.SetTUIMode(true)

	defer p.logService.SetTUIMode(false) // Restore stdout logging on exit

	// Create and run the TUI program with message channel integration, logger service, and markdown renderer
	// This ensures AgentMessenger sends messages to the TUI instead of stdout
	// and logs are displayed in a dedicated panel
	// Markdown is rendered with syntax highlighting for agent responses
	prog := tui.NewProgramWithContext(ctx, executor,
		tui.WithMessageChannel(),
		tui.WithLoggerService(p.logService),
		tui.WithMarkdownRenderer(p.markdownRenderer),
	)

	_, err := prog.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// agentExecutorAdapter adapts shared.Agent to implement tui.AgentExecutor interface
type agentExecutorAdapter struct {
	agent shared.Agent
}

// Execute implements tui.AgentExecutor by delegating to the underlying agent
func (a *agentExecutorAdapter) Execute(ctx context.Context, input string) (*gollem.ExecuteResponse, error) {
	return a.agent.Execute(ctx, gollem.Text(input))
}

// Cleanup performs any necessary cleanup when the application exits.
// Note: Bubbletea handles terminal state restoration automatically.
func (p *applicationServiceImpl) Cleanup() {
	// Bubbletea handles terminal state restoration automatically
	// This method is kept for interface compatibility
}
