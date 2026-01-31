// Package app provides the main application service that orchestrates
// the Gollum agent system startup and interactive loop.
package app

import (
	"context"
	"fmt"
	"os"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mcp"
	"github.com/denkhaus/gollum/pkg/middleware"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"golang.org/x/term"
)

// ApplicationService defines the main application service interface
type ApplicationService interface {
	// Run starts the application, performing all initialization and running the interactive loop
	Run(ctx context.Context) error
	// Cleanup restores terminal state and performs other cleanup
	Cleanup()
}

// applicationServiceImpl implements the ApplicationService interface
type applicationServiceImpl struct {
	logService    logger.LoggerService
	fsm           state.FileStateManager
	agentRegistry registry.AgentRegistry
	promptMgr     prompt.PromptManager
	displayProv   middleware.DisplayMiddlewareProvider
	agentFactory  shared.AgentFactory
	currentCancel context.CancelFunc // Current active request cancel function
	oldState      *term.State        // Original terminal state for restoration
}

// Ensure implementation satisfies interface
var _ ApplicationService = (*applicationServiceImpl)(nil)

// NewService creates a new application service
func NewService(injector do.Injector) (ApplicationService, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	fsm := do.MustInvoke[state.FileStateManager](injector)
	agentRegistry := do.MustInvoke[registry.AgentRegistry](injector)
	promptMgr := do.MustInvoke[prompt.PromptManager](injector)
	displayProv := do.MustInvoke[middleware.DisplayMiddlewareProvider](injector)
	agentFactory := do.MustInvoke[shared.AgentFactory](injector)

	return &applicationServiceImpl{
		logService:    logService,
		fsm:           fsm,
		agentRegistry: agentRegistry,
		promptMgr:     promptMgr,
		displayProv:   displayProv,
		agentFactory:  agentFactory,
	}, nil
}

// Run starts the application, performing all initialization and running the interactive loop
func (p *applicationServiceImpl) Run(ctx context.Context) error {
	// Prime FileStateManager
	if err := p.primeFileStateManager(ctx); err != nil {
		return err
	}

	// Create and register Supervisor agent
	agent, agentConfig, err := p.createSupervisorAgent(ctx)
	if err != nil {
		return err
	}

	// Display welcome message
	p.displayWelcome(agentConfig)

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
	// Create brain MCP toolset
	brainMCP, err := mcp.NewBrainMCPClient(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create brain MCP client: %w", err)
	}

	// Get supervisor prompt from PromptManager
	systemPrompt, err := p.promptMgr.GetSupervisorPrompt()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get supervisor prompt: %w", err)
	}

	// Create agent config
	agentConfig := &shared.AgentConfig{
		AllowCompaction: true,
		SystemPrompt:    systemPrompt,
		Role:            "Supervisor Agent",
		LLMProvider:     shared.LLMProviderAnthropic,
		ToolSets: []gollem.ToolSet{
			brainMCP,
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

// displayWelcome displays the welcome message
func (p *applicationServiceImpl) displayWelcome(config *shared.AgentConfig) {
	displayMiddleware := p.displayProv.CreateDisplayMiddleware(config.ID, config.Role)
	displayMiddleware.DisplayWelcome()
}

// runInteractiveLoop runs the main CLI interactive loop
func (p *applicationServiceImpl) runInteractiveLoop(ctx context.Context, agent shared.Agent) error {
	inputCh := make(chan string)
	cancelCh := make(chan struct{})
	shutdownCh := make(chan struct{})

	// Read stdin continuously in background
	go p.stdinReader(inputCh, cancelCh, shutdownCh)

	for {
		if _, err := fmt.Fprint(os.Stdout, "\r> "); err != nil {
			p.logService.Debugf("failed to write prompt: %v", err)
		}

		select {
		case <-ctx.Done():
			if _, err := fmt.Fprintln(os.Stdout, "\n👋 Goodbye!"); err != nil {
				p.logService.Debugf("failed to write goodbye message: %v", err)
			}
			return ctx.Err()

		case <-shutdownCh:
			// Ctrl-C was pressed - trigger shutdown
			return context.Canceled

		case <-cancelCh:
			// Escape key was pressed - return to prompt
			continue

		case input, ok := <-inputCh:
			if !ok {
				return nil // stdin closed
			}
			if p.handleExitCommand(input) {
				return nil
			}
			if err := p.executeAgentInput(ctx, agent, input); err != nil {
				p.logService.Infof("❌ Error: %v\n", err)
			}
		}
	}
}

// handleExitCommand checks if user wants to exit
func (p *applicationServiceImpl) handleExitCommand(input string) bool {
	if input == "quit" || input == "exit" {
		if _, err := fmt.Fprintln(os.Stdout, "👋 Goodbye!"); err != nil {
			p.logService.Debugf("failed to write goodbye message: %v", err)
		}
		return true
	}
	return false
}

// executeAgentInput executes the agent with user input
// Creates a per-request context that can be canceled independently of the global context
func (p *applicationServiceImpl) executeAgentInput(globalCtx context.Context, agent shared.Agent, input string) error {
	// Create a per-request context that can be canceled independently
	// This allows canceling just this inference (e.g., via Escape key) without stopping the app
	reqCtx, cancel := context.WithCancel(globalCtx)
	defer cancel()

	// Set the cancel function for the stdinReader
	p.currentCancel = cancel
	defer func() {
		p.currentCancel = nil
	}()

	_, err := agent.Execute(reqCtx, gollem.Text(input))
	return err
}

// stdinReader reads from stdin in raw mode, detecting both regular input (Enter key) and Escape key
// Sends complete lines to inputCh and signals cancelCh when Escape is pressed
func (p *applicationServiceImpl) stdinReader(inputCh chan<- string, cancelCh chan<- struct{}, shutdownCh chan<- struct{}) {
	// Save terminal state and switch to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		p.logService.Errorf("Failed to set raw mode: %v", err)
		close(inputCh)
		return
	}
	// Store state for cleanup
	p.oldState = oldState

	buf := make([]byte, 1)
	var lineBuf []byte

	for {
		// Read single byte
		n, readErr := os.Stdin.Read(buf)
		if n > 0 {
			ch := buf[0]

			switch ch {
			case 3: // Ctrl-C (ASCII 3)
				// Trigger graceful shutdown
				if _, err := fmt.Fprintln(os.Stdout, "^C"); err != nil {
					p.logService.Debugf("failed to write ctrl-c message: %v", err)
				}
				close(inputCh)
				select {
				case shutdownCh <- struct{}{}:
				default:
				}
				return

			case 27: // Escape key
				// Cancel current inference if active
				if p.currentCancel != nil {
					if _, err := fmt.Fprintln(os.Stdout, "\n⚠️  Inference canceled by user"); err != nil {
						p.logService.Debugf("failed to write cancel message: %v", err)
					}
					p.currentCancel()
					p.currentCancel = nil
				}
				// Signal cancel to main loop
				select {
				case cancelCh <- struct{}{}:
				default:
				}
				lineBuf = nil // Clear any pending input
				// Redraw prompt
				if _, err := fmt.Fprint(os.Stdout, "\r> "); err != nil {
					p.logService.Debugf("failed to write prompt: %v", err)
				}

			case 13: // Enter/Return key ( carriage return)
				// Move to next line
				if _, err := fmt.Fprint(os.Stdout, "\n"); err != nil {
					p.logService.Debugf("failed to write newline: %v", err)
				}
				// Send complete line to input channel
				inputCh <- string(lineBuf)
				lineBuf = nil

			case 127, 8: // Backspace/Delete
				// Remove last character from buffer
				if len(lineBuf) > 0 {
					lineBuf = lineBuf[:len(lineBuf)-1]
					// Erase character from screen (backspace, space, backspace)
					if _, err := fmt.Fprint(os.Stdout, "\b \b"); err != nil {
						p.logService.Debugf("failed to write backspace: %v", err)
					}
				}

			default:
				// Regular character - add to buffer and echo
				// Only accept printable ASCII characters
				if ch >= 32 && ch <= 126 {
					lineBuf = append(lineBuf, ch)
					// Echo character to screen
					if _, err := fmt.Fprintf(os.Stdout, "%c", ch); err != nil {
						p.logService.Debugf("failed to write character: %v", err)
					}
				}
			}
		}

		// Check for errors (e.g., stdin closed)
		if readErr != nil {
			close(inputCh)
			return
		}
	}
}

// Cleanup restores terminal state when the application exits
func (p *applicationServiceImpl) Cleanup() {
	if p.oldState != nil {
		if err := term.Restore(int(os.Stdin.Fd()), p.oldState); err != nil {
			p.logService.Debugf("failed to restore terminal state: %v", err)
		}
		if _, err := fmt.Fprintln(os.Stdout); err != nil {
			p.logService.Debugf("failed to write newline: %v", err)
		}
	}
}
