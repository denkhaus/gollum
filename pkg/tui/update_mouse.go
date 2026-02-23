package tui

// This file handles mouse events for the TUI.

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Debug log file for click detection debugging
const debugLogFile = "/tmp/gollum_click_debug.log"

func debugLog(format string, args ...interface{}) {
	f, err := os.OpenFile(debugLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()                      //nolint:errcheck // Debug logging errors are acceptable to ignore
	fmt.Fprintf(f, format+"\n", args...) //nolint:errcheck // Debug logging errors are acceptable to ignore
}

// handleMouseMsg handles mouse events with debouncing for smooth scrolling.
//
// This function implements the debounce pattern from Charmbracelet's example:
// https://github.com/charmbracelet/bubbletea/blob/main/examples/debounce/main.go
//
// The pattern works by:
// 1. Incrementing a tag on each mouse event
// 2. Scheduling a debounce command that includes the current tag
// 3. Only processing the scroll if the tag matches when the command fires
//
// This prevents rapid mouse wheel events from overwhelming the UI.
//
// For mouse clicks (tea.MouseLeft), this function detects clicks on tool message
// headers to toggle collapse/expand state.
func (m Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseWheelUp, tea.MouseWheelDown:
		// Increment tag to invalidate any pending debounce commands
		m.mouseDebounceTag++

		// Determine scroll direction (-1 for up, 1 for down)
		direction := -1
		if msg.Type == tea.MouseWheelDown {
			direction = 1
		}

		// Return a debounce command with the current tag
		return m, m.createDebounceCommand(m.mouseDebounceTag, direction, m.activeViewport)

	case tea.MouseLeft:
		// Handle click on tool message header to toggle collapse state
		return m.handleClickOnToolMessage(msg)
	}

	// For other mouse events, pass to text input (for clicks, etc.)
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// handleClickOnToolMessage handles mouse clicks for message selection and collapse toggle.
//
// Click behavior:
//   - Single click: Select the message (shows bold border)
//   - Double click (within 500ms on same Y): Select AND toggle collapse state
//
// This allows users to always see which message is currently selected via the bold border,
// while still being able to quickly toggle collapse state with a double-click.
func (m Model) handleClickOnToolMessage(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Get the viewport's Y offset (how far the content is scrolled)
	yOffset := m.viewport.YOffset
	viewportHeight := m.viewport.Height

	// The viewport starts at terminal Y=0 (top of screen)
	// Mouse Y coordinate is 0-indexed from top of terminal
	clickY := msg.Y

	debugLog("=== CLICK ===")
	debugLog("clickY=%d, yOffset=%d, viewportHeight=%d", clickY, yOffset, viewportHeight)
	debugLog("messages count: %d", len(m.messages))
	for i, msg := range m.messages {
		debugLog("  msg[%d]: type=%s, isTool=%v, collapsed=%v", i, msg.Type.String(), msg.IsTool, msg.Collapsed)
	}

	// Only process clicks within the viewport bounds
	if clickY < 0 || clickY >= viewportHeight {
		// Click is outside viewport, pass to text input
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	// Calculate the line number in the content (0-indexed from top of content)
	contentLine := yOffset + clickY
	debugLog("contentLine = yOffset + clickY = %d + %d = %d", yOffset, clickY, contentLine)

	// ALWAYS rebuild content and mapping fresh on every click
	// This ensures lineToMessage is 100% in sync with actual content
	content := m.updateViewportContent()
	m.viewport.SetContent(content)

	// NOW log the CORRECT data (after rebuild)
	debugLog("lineToMessage (first 30): %v", m.lineToMessage[:min(30, len(m.lineToMessage))])
	debugLog("lineToMessage length: %d", len(m.lineToMessage))

	// Log actual viewport content (lines around the click)
	contentLines := strings.Split(content, "\n")
	startLine := max(0, yOffset-2)
	endLine := min(len(contentLines), yOffset+viewportHeight+2)
	debugLog("=== VIEWPORT CONTENT (lines %d to %d of %d) ===", startLine, endLine-1, len(contentLines))
	for i := startLine; i < endLine; i++ {
		if i < len(contentLines) {
			// Clean up ANSI codes for readability
			cleanLine := ansiRegex.ReplaceAllString(contentLines[i], "")
			prefix := "  "
			if i == contentLine {
				prefix = "> " // Mark the clicked line
			}
			debugLog("%sLine %d: %s", prefix, i, cleanLine)
		}
	}
	debugLog("=== END VIEWPORT CONTENT ===")

	// Find which message is at this line using direct O(1) lookup
	msgIdx := m.getMessageAtLine(contentLine)
	debugLog("getMessageAtLine(%d) = %d", contentLine, msgIdx)

	if msgIdx < 0 {
		// Clicked on a blank gap between messages, pass to text input
		debugLog("Clicked on blank gap - passing to text input")
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	// Check for double-click: same message index within threshold time
	now := time.Now()
	isDoubleClick := false
	timeSinceLastClick := now.Sub(m.lastClickTime)
	threshold := m.doubleClickThreshold

	debugLog("DOUBLE-CLICK CHECK:")
	debugLog("  lastClickedMessageIndex=%d, currentMsgIdx=%d", m.lastClickedMessageIndex, msgIdx)
	debugLog("  timeSinceLastClick=%v, threshold=%v", timeSinceLastClick, threshold)
	debugLog("  sameMessage=%v, withinThreshold=%v",
		m.lastClickedMessageIndex == msgIdx,
		timeSinceLastClick < threshold)

	if m.lastClickedMessageIndex == msgIdx && timeSinceLastClick < threshold {
		isDoubleClick = true
		debugLog("  >>> DOUBLE-CLICK DETECTED! <<<")
	}

	// Update click tracking for next potential double-click
	m.lastClickTime = now
	m.lastClickedMessageIndex = msgIdx

	// Select the message (always)
	m.selectMessage(msgIdx)
	debugLog("SELECTED message %d (type=%s, isTool=%v)", msgIdx, m.messages[msgIdx].Type.String(), m.messages[msgIdx].IsTool)

	// On double-click, also toggle collapse state if it's a tool message
	if isDoubleClick {
		debugLog("DOUBLE-CLICK: Attempting toggle on message %d", msgIdx)
		debugLog("  message type=%s, isTool=%v", m.messages[msgIdx].Type.String(), m.messages[msgIdx].IsTool)

		if m.messages[msgIdx].IsTool || m.messages[msgIdx].Type == MessageTypeTool {
			oldCollapsed := m.messages[msgIdx].Collapsed
			if m.toggleMessageCollapse(msgIdx) {
				debugLog("  TOGGLE SUCCESS: collapsed %v -> %v", oldCollapsed, !oldCollapsed)
				// Message was toggled, rebuild viewport content
				m.viewport.SetContent(m.updateViewportContent())
				return m, nil
			}
			debugLog("  TOGGLE FAILED - toggleMessageCollapse returned false")
		} else {
			debugLog("  SKIPPED - not a tool message")
		}
	}

	// Rebuild viewport to show selection (bold border)
	m.viewport.SetContent(m.updateViewportContent())
	return m, nil
}

// handleMouseDebounceMsg processes debounced mouse scroll events.
func (m Model) handleMouseDebounceMsg(msg mouseDebounceMsg) (tea.Model, tea.Cmd) {
	// Only apply if the tag matches (this is the latest scroll event)
	if msg.tag == m.mouseDebounceTag {
		m = m.scrollViewport(msg.viewport, msg.direction)
	}
	return m, nil
}

// scrollViewport scrolls the specified viewport by the given direction.
func (m Model) scrollViewport(viewportName Viewport, direction int) Model {
	scrollLines := 3 // Scroll 3 lines per debounced event for smooth scrolling

	if viewportName == ViewportLogs {
		m = m.scrollLogViewport(direction, scrollLines)
	} else {
		m = m.scrollMainViewport(direction, scrollLines)
	}

	return m
}

// scrollMainViewport scrolls the main viewport.
func (m Model) scrollMainViewport(direction int, scrollLines int) Model {
	if direction < 0 {
		m.viewport.LineUp(scrollLines)
	} else {
		m.viewport.LineDown(scrollLines)
	}
	return m
}

// scrollLogViewport scrolls the log viewport.
func (m Model) scrollLogViewport(direction int, scrollLines int) Model {
	if direction < 0 {
		m.logViewport.LineUp(scrollLines)
	} else {
		m.logViewport.LineDown(scrollLines)
	}
	return m
}

// incrementDebounceTag increments the debounce tag and returns the new value.
func (m Model) incrementDebounceTag() (Model, int) {
	m.mouseDebounceTag++
	return m, m.mouseDebounceTag
}

// createDebounceCommand creates a debounce command for mouse scrolling.
func (m Model) createDebounceCommand(tag int, direction int, viewportName Viewport) tea.Cmd {
	return tea.Tick(m.mouseDebounceDuration, func(_ time.Time) tea.Msg {
		return mouseDebounceMsg{
			tag:       tag,
			direction: direction,
			viewport:  viewportName,
		}
	})
}
