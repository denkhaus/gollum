package tui

// This file handles periodic timer events for the TUI.

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/denkhaus/gollum/pkg/logger"
)

// handleTickMsg handles periodic context cancellation checks.
func (m Model) handleTickMsg(msg tickMsg) (Model, tea.Cmd) {
	select {
	case <-m.ctx.Done():
		m.quit = true
		return m, tea.Quit
	default:
	}
	return m, tea.Batch(m.tickCmd(), m.logTickCmd())
}

// handleLogTickMsg handles periodic log entry fetching.
func (m Model) handleLogTickMsg(msg logTickMsg) (Model, tea.Cmd) {
	m = m.fetchNewLogEntries()
	return m, m.logTickCmd()
}

// fetchNewLogEntries fetches and appends new log entries from the logger service.
// Only auto-scrolls to bottom if the viewport was already at the bottom (tail -f behavior).
func (m Model) fetchNewLogEntries() Model {
	if m.logService == nil {
		return m
	}

	newLogs := m.logService.GetLogs(logger.LogFilter{
		SinceSeq: m.lastLogFetchSeq,
		Reverse:  false,
	})

	if len(newLogs) == 0 {
		return m
	}

	// Check if viewport is at the bottom before adding new content
	// This implements tail -f behavior: only auto-scroll if already at bottom
	atBottom := m.isLogViewportAtBottom()

	m.lastLogFetchSeq = newLogs[len(newLogs)-1].Sequence

	for _, log := range newLogs {
		formatted := fmt.Sprintf("%s [%s] %s",
			log.Timestamp.Format("15:04:05"),
			strings.ToUpper(log.Level),
			log.Message)
		m.logEntries = append(m.logEntries, formatted)
	}

	m.logViewport.SetContent(strings.Join(m.logEntries, "\n"))

	// Only auto-scroll if the viewport was at the bottom before adding new content
	if atBottom {
		m.logViewport.GotoBottom()
	}

	return m
}

// isLogViewportAtBottom checks if the log viewport is scrolled to the bottom.
func (m Model) isLogViewportAtBottom() bool {
	// The viewport is at the bottom if scrolling down doesn't change the view
	// We check by attempting to scroll and comparing YOffset
	prevYOffset := m.logViewport.YOffset
	m.logViewport.LineDown(1)
	atBottom := m.logViewport.YOffset == prevYOffset
	// Restore the position
	if !atBottom {
		m.logViewport.LineUp(1)
	}
	return atBottom
}
