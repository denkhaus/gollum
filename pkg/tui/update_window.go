package tui

import tea "github.com/charmbracelet/bubbletea"

// This file handles window resize events for the TUI.

// handleWindowSizeMsg handles terminal resize events.
func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	widthChanged := m.width != msg.Width
	m.width = msg.Width
	m.height = msg.Height

	// Clear format cache when width changes (formatting depends on terminal width)
	// Note: We need to modify m, so we use pointer receiver for cache clearing
	if widthChanged {
		// Get pointer to model for cache operations
		mp := &m
		mp.clearFormatCache()
	}

	// Update viewport size (reserve space for input, footer, status bar, and log panel)
	mainHeight, logHeight := m.calculateViewportDimensions()

	m.viewport.Width = msg.Width
	m.viewport.Height = mainHeight
	m.logViewport.Width = msg.Width
	m.logViewport.Height = logHeight

	return m, nil
}

// calculateViewportDimensions returns the height for main viewport, log panel,
// and the number of reserved lines.
func (m Model) calculateViewportDimensions() (mainHeight int, logHeight int) {
	reservedLines := 6 // status bar + prompt + footer
	if m.config.StatusEnabled {
		reservedLines++ // Extra line for status bar
	}

	logHeight = 6                  // Default log panel height
	reservedLines += logHeight + 1 // +1 for log separator

	mainHeight = m.height - reservedLines
	if mainHeight < 1 {
		mainHeight = 1
		// If we're very short, reduce log panel height
		logHeight = m.height - reservedLines + logHeight
		if logHeight < 3 {
			logHeight = 3 // Minimum log panel height
		}
	}

	return mainHeight, logHeight
}
