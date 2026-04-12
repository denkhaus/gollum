// Package app provides the main application service that orchestrates
// the Gollum agent system startup and interactive loop.
package app

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/flows/executor"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
	mcpregistry "github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/denkhaus/gollum/pkg/tui"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/google/uuid"
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
	mcpRegistry := do.MustInvoke[mcpregistry.MCPRegistry](injector)
	channelFacade := do.MustInvoke[channel.ChannelFacade](injector)
	flowExecutorService := do.MustInvoke[executor.FlowExecutorService](injector)
	flowRegistry := do.MustInvoke[flowregistry.FlowRegistry](injector)

	return &applicationServiceImpl{
		sessionID:        uuid.New(),
		logService:       logService,
		fsm:              fsm,
		agentRegistry:    agentRegistry,
		workspaceService: workspaceService,

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
	defaultFlow, err := p.flowRegistry.GetDefaultFlow()
	if err == nil && defaultFlow != nil {
		// Default flow found, execute it using flow.Execute()
		p.logService.Infof("Default flow found: %s", defaultFlow.Name)
		result, err := defaultFlow.Execute(ctx, make(map[string]any))
		if err != nil {
			return fmt.Errorf("default flow execution failed: %w", err)
		}
		// Display flow outputs
		if len(result.Outputs) > 0 {
			p.logService.Info("Flow outputs:")
			for name, value := range result.Outputs {
				p.logService.Infof("  %s: %v", name, value)
			}
		}
		p.logService.Info("Default flow completed successfully")
		return nil
	}
	p.logService.Info("no default flow found -> run tui")

	// No default flow, run TUI
	return p.runChannel(ctx, tui.Identifier,
		tui.WithChannelLogger(p.logService),
		tui.WithChannelRenderer(p.markdownRenderer),
	)
}

// runChannel creates and runs a channel by identifier.
func (p *applicationServiceImpl) runChannel(ctx context.Context, identifier channel.ChannelIdentifier, opts ...channel.ChannelOption) error {
	ch, err := p.channelFacade.CreateChannel(identifier, opts...)
	if err != nil {
		return fmt.Errorf("failed to create channel %s: %w", identifier, err)
	}

	// Ensure cleanup on failure
	defer func() {
		if err != nil {
			p.channelFacade.UnregisterChannel(ch.ID())
		}
	}()

	if err = p.channelFacade.RegisterChannel(ch); err != nil {
		return fmt.Errorf("failed to register channel %s: %w", identifier, err)
	}

	// Start channel lifecycle
	return ch.Start(ctx)
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

// Cleanup performs any necessary cleanup when the application exits.
// Note: Bubbletea handles terminal state restoration automatically.
func (p *applicationServiceImpl) Cleanup() {
	// Bubbletea handles terminal state restoration automatically
	// This method is kept for interface compatibility
}
