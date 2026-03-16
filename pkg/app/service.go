// Package app provides the main application service that orchestrates
// the Gollum agent system startup and interactive loop.
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/flows/executor"
	"github.com/denkhaus/gollum/pkg/flows/parser"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
	mcpregistry "github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/denkhaus/gollum/pkg/tui"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/google/uuid"
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
	gollumDir           string
	sessionID           uuid.UUID
	logService          logger.LoggerService
	fsm                 state.FileStateManager
	agentRegistry       registry.AgentRegistry
	promptMgr           manager.PromptManager
	agentFactory        shared.AgentFactory
	markdownRenderer    markdown.Renderer
	workspaceService    workspace.Service
	skillsService       skills.SkillService
	mcpRegistry         mcpregistry.MCPRegistry
	channelFacade       channel.ChannelFacade
	flowExecutorService executor.FlowExecutorService
	flowRegistry        flowregistry.FlowRegistry
}

// Ensure implementation satisfies interface
var _ ApplicationService = (*applicationServiceImpl)(nil)

// NewService creates a new application service
func NewService(injector do.Injector) (ApplicationService, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm := do.MustInvoke[state.FileStateManager](injector)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)
	promptMgr := do.MustInvoke[manager.PromptManager](injector)
	agentFactory := do.MustInvoke[shared.AgentFactory](injector)
	markdownRenderer := do.MustInvoke[markdown.Renderer](injector)
	workspaceService := do.MustInvoke[workspace.Service](injector)
	skillsService := do.MustInvoke[skills.SkillService](injector)
	mcpRegistry := do.MustInvoke[mcpregistry.MCPRegistry](injector)
	channelFacade := do.MustInvoke[channel.ChannelFacadeService](injector)
	flowExecutorService := do.MustInvoke[executor.FlowExecutorService](injector)
	flowRegistry := do.MustInvoke[flowregistry.FlowRegistry](injector)

	return &applicationServiceImpl{
		sessionID:           uuid.New(),
		logService:          logService,
		fsm:                 fsm,
		agentRegistry:       agentRegistry,
		workspaceService:    workspaceService,
		skillsService:       skillsService,
		promptMgr:           promptMgr,
		agentFactory:        agentFactory,
		markdownRenderer:    markdownRenderer,
		mcpRegistry:         mcpRegistry,
		channelFacade:       channelFacade,
		flowExecutorService: flowExecutorService,
		flowRegistry:        flowRegistry,
	}, nil
}

// Run starts the application, performing all initialization and running the interactive loop
func (p *applicationServiceImpl) Run(ctx context.Context) error {
	// Create .gollum directory first (this initializes p.gollumDir)
	if err := p.ensureGollumDirectory(); err != nil {
		return fmt.Errorf("failed to create .gollum directory: %w", err)
	}

	// Enable file logging (LoggerService handles logs/ subdir and cleanup)
	if err := p.logService.EnableFileLogging(p.gollumDir, p.sessionID); err != nil {
		return fmt.Errorf("failed to enable file logging: %w", err)
	}
	defer func() {
		if err := p.logService.CloseFileLogging(); err != nil {
			p.logService.Warnf("failed to close file logging: %v", err)
		}
	}()

	// Prime FileStateManager
	if err := p.primeFileStateManager(ctx); err != nil {
		return err
	}

	// Check for default flow first
	defaultFlowPath, err := p.resolveDefaultFlowPath()
	if err == nil && defaultFlowPath != "" {
		// Default flow found, execute it
		p.logService.Infof("Default flow found at: %s", defaultFlowPath)
		return p.runDefaultFlow(ctx, defaultFlowPath)
	} else {
		p.logService.Info("no default flow found -> run tui")
	}

	// No default flow, run TUI
	return p.runTUI(ctx)
}

// runTUI runs the terminal user interface
func (p *applicationServiceImpl) runTUI(ctx context.Context) error {

	// Create and register Supervisor agent
	agent, _, err := p.createSupervisorAgent(ctx)
	if err != nil {
		return err
	}

	// Run interactive loop
	return p.runInteractiveLoop(ctx, agent)
}

// resolveDefaultFlowPath checks for default flow in workspace and global locations
func (p *applicationServiceImpl) resolveDefaultFlowPath() (string, error) {
	// Check workspace-local first: .gollum/flows/default/main.xml
	workspace := p.workspaceService.GetCurrentWorkspace()
	workspaceFlowPath := filepath.Join(workspace, ".gollum", "flows", "default", "main.xml")
	if _, err := os.Stat(workspaceFlowPath); err == nil {
		return workspaceFlowPath, nil
	}

	// Check global config: ~/.config/gollum/flows/default/main.xml
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	globalFlowPath := filepath.Join(homeDir, ".config", "gollum", "flows", "default", "main.xml")
	if _, err := os.Stat(globalFlowPath); err == nil {
		return globalFlowPath, nil
	}

	return "", fmt.Errorf("no default flow found")
}

// runDefaultFlow executes the default flow using FlowExecutorService
func (p *applicationServiceImpl) runDefaultFlow(ctx context.Context, flowPath string) error {
	p.logService.Infof("Running default flow: %s", flowPath)

	// Parse flow
	flow, err := parser.Parse(flowPath)
	if err != nil {
		return fmt.Errorf("failed to parse default flow: %w", err)
	}

	// Create executor
	executor := p.flowExecutorService.New(flow)

	// Set empty input (default flow should define required inputs with defaults)
	executor.SetInput(make(map[string]string))

	// Validate and run
	if err := executor.Validate(); err != nil {
		return fmt.Errorf("default flow validation failed: %w", err)
	}

	if err := executor.Run(); err != nil {
		return fmt.Errorf("default flow execution failed: %w", err)
	}

	p.logService.Infof("Default flow completed successfully")
	return nil
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

	// TODO: Convert MCP tool sets to AllowedTools in Task 3
	toolSet := p.mcpRegistry.GetToolSets()
	if len(toolSet) == 0 {
		p.logService.Warn("no mcp servers configured for supervision agent")
	}

	// Create agent config
	agentConfig := &shared.AgentConfig{
		AllowCompaction: true,
		SystemPrompt:    systemPrompt,
		AllowedTools:    nil, // TODO: Configure supervisor tools explicitly
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

	// Create the TUI model with all options
	// This creates the TUIChannel which we need to register with ChannelFacade
	model := tui.NewModel(ctx, executor)
	tui.WithTUIChannel()(&model)                         // Create and set TUIChannel
	tui.WithLoggerService(p.logService)(&model)          // Set logger service
	tui.WithMarkdownRenderer(p.markdownRenderer)(&model) // Set markdown renderer

	// Register the TUIChannel with ChannelFacade
	// This allows the channel system to send messages to the TUI
	tuiChannel := model.GetTUIChannel()
	if tuiChannel != nil {
		if err := p.channelFacade.RegisterChannel(tuiChannel); err != nil {
			return fmt.Errorf("failed to register TUI channel: %w", err)
		}
		p.logService.Infof("TUI channel registered with ChannelFacade")
	}

	// Create and run the TUI program
	prog := tea.NewProgram(model,
		tea.WithContext(ctx),
		tea.WithAltScreen(),
		// Enable mouse cell motion for click detection on tool messages
		// This allows text selection in most terminals while still receiving clicks
		tea.WithMouseCellMotion(),
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
