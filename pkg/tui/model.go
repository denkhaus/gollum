// Package tui provides Bubbletea-based terminal user interface components for Gollum.
//
// This package establishes the foundation for a modern, event-driven TUI that will
// eventually replace the current stdinReader-based input system. It uses Bubbletea's
// Model-Update-View architecture for clean separation of concerns.
//
// # Architecture
//
// The TUI follows Bubbletea's elm architecture:
//   - Model: Application state (textInput, quit flag, output buffer)
//   - Init: Initial command that starts the application
//   - Update: Event handler that returns new model + commands
//   - View: Renderer that converts model to string output
//
// # Usage
//
// To run the TUI:
//
//	p := tui.NewProgramWithContext(ctx, agent)
//	if _, err := p.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Phase Scope
//
//   - Phase 1: Basic Bubbletea model with text input
//   - Phase 2: Replace stdinReader with Bubbletea input (current)
//   - Phase 3: Integrate AgentMessenger with Bubbletea viewport
//   - Phase 4: Enhanced features and polish
package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/m-mizutani/gollem"
)

// AgentExecutor defines the interface for executing agent commands.
// This allows the TUI to interact with different agent implementations.
type AgentExecutor interface {
	Execute(ctx context.Context, input string) (*gollem.ExecuteResponse, error)
}

// tickMsg is sent periodically to update the UI (for agent execution timer).
type tickMsg time.Time

// agentCompleteMsg is sent when agent execution completes.
type agentCompleteMsg struct {
	response *gollem.ExecuteResponse
	err      error
}

// Model represents the application state for the TUI.
//
// In Bubbletea's architecture, the Model is a pure data structure that
// contains all application state. The Model is immutable between Update cycles.
type Model struct {
	// textInput is the Bubbletea text input component for user entry
	textInput textinput.Model

	// quit indicates when the user wants to exit the application
	quit bool

	// messages stores conversation history (agent responses, errors, etc.)
	messages []string

	// agent executes user commands
	agent AgentExecutor

	// ctx is the application context for cancellation
	ctx context.Context

	// agentExecuting indicates whether an agent is currently running
	agentExecuting bool

	// agentStartTime tracks when the current agent execution started
	agentStartTime time.Time

	// currentCancel allows canceling the active agent execution
	currentCancel context.CancelFunc

	// err stores the last agent error
	err error

	// inputHistory stores previous user inputs for up/down navigation
	inputHistory []string

	// inputHistoryIndex tracks the current position in input history
	inputHistoryIndex int

	// preservedInput stores the input when canceling during execution
	preservedInput string

	// cancelRequested indicates if the user requested cancellation
	cancelRequested bool

	// width and height store the terminal dimensions
	width  int
	height int
}

// NewModel creates a new TUI model with initial state.
//
// The model is initialized with:
//   - A focused text input component
//   - Quit flag set to false
//   - Empty messages buffer
//   - Agent executor and context for command execution
//
// Returns a Model ready for use with tea.Program.
func NewModel(ctx context.Context, agent AgentExecutor) Model {
	ti := textinput.New()
	ti.Focus()
	ti.Placeholder = ""
	ti.CharLimit = 0 // No limit for agent input

	return Model{
		textInput:         ti,
		quit:              false,
		messages:          []string{},
		agent:             agent,
		ctx:               ctx,
		inputHistory:      []string{},
		inputHistoryIndex: -1,
	}
}

// Init initializes the TUI application.
//
// This function is called once at the start of the program.
// It returns the initial command (textinput focus and tick for timer updates).
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.tickCmd(),
	)
}

// tickCmd returns a command that sends tick messages for UI updates.
func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
