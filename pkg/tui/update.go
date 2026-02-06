package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles incoming messages and updates the model state.
//
// This is the core event loop of the TUI. Each message (key press, mouse event,
// custom message) is processed to produce a new model state and optional commands.
//
// In Bubbletea's architecture, Update must be a pure function:
//   - Same input always produces same output
//   - No side effects (use Cmd for those)
//   - Returns new Model, never modifies in place
//
// # Phase 1 Update Logic
//
//   - tea.KeyMsg: Handle keyboard input
//   - ctrl+c / esc / "quit": Set quit flag to exit
//   - enter: Capture text input and add to output
//   - All other keys: Delegate to textinput component
//   - All other msgs: No-op, return model unchanged
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle quit conditions first
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c", "esc":
			m.quit = true
			return m, tea.Quit

		case "enter":
			// Capture the input and add to output
			input := m.textInput.Value()
			if input != "" {
				if m.output != "" {
					m.output += "\n"
				}
				m.output += "You: " + input
				m.textInput.Reset()
			}
			return m, nil
		}
	}

	// Update text input component
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	return m, cmd
}
