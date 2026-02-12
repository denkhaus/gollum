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
// # Phase 4 Update Logic
//
//   - tea.KeyMsg: Handle keyboard input
//   - ctrl+c: Graceful shutdown
//   - esc: Cancel active agent execution or exit search mode
//   - enter: Submit input to agent or add newline in multi-line mode
//   - alt+enter: Toggle multi-line input mode
//   - up/down: Navigate input history or search results
//   - ctrl+r: Start history search
//   - ctrl+s/ctrl+r in search: Navigate search results
//   - All other keys: Delegate to textinput component
//   - tickMsg: Check context cancellation, update timer
//   - agentCompleteMsg: Handle agent response/error
//   - newMessageMsg: Handle new messages from AgentMessenger
//   - WindowSizeMsg: Update terminal dimensions and viewport
//   - exportMsg: Handle conversation export
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)

	case tickMsg:
		return m.handleTickMsg(msg)

	case logTickMsg:
		return m.handleLogTickMsg(msg)

	case agentCompleteMsg:
		return m.handleAgentCompleteMsg(msg)

	case newMessageMsg:
		return m.handleNewMessageMsg(msg)

	case exportMsg:
		return m.handleExport()

	case tea.MouseMsg:
		return m.handleMouseMsg(msg)

	case mouseDebounceMsg:
		return m.handleMouseDebounceMsg(msg)

	case keyDebounceMsg:
		return m.handleKeyDebounceMsg(msg)

	default:
		// Update text input component with remaining messages (unless in search mode)
		if !m.searchState.active {
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}
}
