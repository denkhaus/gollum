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
//	p := tui.NewProgram()
//	if _, err := p.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Phase 1 Scope
//
// This is Phase 1 of a multi-phase TUI modernization:
//   - Phase 1: Basic Bubbletea model with text input (this package)
//   - Phase 2: Replace stdinReader with Bubbletea input
//   - Phase 3: Integrate AgentMessenger with Bubbletea viewport
//   - Phase 4: Enhanced features and polish
package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the application state for the TUI.
//
// In Bubbletea's architecture, the Model is a pure data structure that
// contains all application state. The Model is immutable between Update cycles.
type Model struct {
	// textInput is the Bubbletea text input component for user entry
	textInput textinput.Model

	// quit indicates when the user wants to exit the application
	quit bool

	// output accumulates text to be displayed in the viewport
	// In Phase 1, this is a simple string buffer
	// Phase 3 will integrate with AgentMessenger for richer output
	output string
}

// NewModel creates a new TUI model with initial state.
//
// The model is initialized with:
//   - A focused text input component
//   - Quit flag set to false
//   - Empty output buffer
//
// Returns a Model ready for use with tea.Program.
func NewModel() Model {
	ti := textinput.New()
	ti.Focus()
	ti.Placeholder = "Enter your message..."
	ti.CharLimit = 256

	return Model{
		textInput: ti,
		quit:      false,
		output:    "",
	}
}

// Init initializes the TUI application.
//
// This function is called once at the start of the program.
// It returns the initial command (usually textinput focus).
//
// In Phase 1, we simply return nil to start with no initial commands.
// Future phases may return commands for initial data loading.
func (m Model) Init() tea.Cmd {
	// Return a command that focuses the text input
	return m.textInput.Focus()
}
