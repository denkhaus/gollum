// Package tui provides Bubbletea-based terminal user interface components for Gollum.
//
// This file contains message formatting and viewport content management methods.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/denkhaus/gollum/pkg/channel"
)

// updateViewportContent updates the viewport with the current messages.
// Uses differential rendering to only append new messages instead of rebuilding everything.
// This is a performance optimization to avoid O(n) re-formatting on every update.
// Also tracks message line positions for mouse click detection.
func (m *Model) updateViewportContent() string {
	currentCount := len(m.messages)

	// Check if we need a full rebuild (cache invalidation scenarios)
	// 1. Width changed - formatting depends on width
	// 2. Messages were removed (e.g., /clear) - need to rebuild from scratch
	// 3. No cached content yet - first call
	// 4. Message collapse state changed - need to rebuild for updated formatting
	needsRebuild := m.cacheWidth != m.width ||
		currentCount < m.lastRenderedCount ||
		m.cachedContent == ""

	if needsRebuild {
		// Full rebuild: iterate through all messages
		var b strings.Builder
		// Reset line-to-message mapping for O(1) click detection
		m.lineToMessage = make([]int, 0)

		for i, msg := range m.messages {
			// Add blank lines BEFORE each message (except the first)
			// Map them to -1 (no message) so viewport line indices match array indices
			if i > 0 {
				b.WriteString("\n")                           // First blank line
				m.lineToMessage = append(m.lineToMessage, -1) // Map blank line to -1
				b.WriteString("\n")                           // Second blank line
				m.lineToMessage = append(m.lineToMessage, -1) // Map blank line to -1
			}

			formatted := m.formatMessage(i, msg)
			b.WriteString(formatted)

			// Build direct line-to-message mapping
			// Each line of this message maps to message index i
			linesInMsg := countLines(formatted)
			for line := 0; line < linesInMsg; line++ {
				m.lineToMessage = append(m.lineToMessage, i)
			}
		}
		m.cachedContent = b.String()
		m.lastRenderedCount = currentCount
		m.cacheWidth = m.width
		return m.cachedContent
	}

	// Differential update: only append new messages
	// This is the hot path for normal operation
	if currentCount > m.lastRenderedCount {
		var b strings.Builder
		// Start with existing content
		b.WriteString(m.cachedContent)

		// Append only new messages
		for i := m.lastRenderedCount; i < currentCount; i++ {
			// Add blank lines BEFORE each message (except if this is the first message)
			// Map them to -1 (no message) so viewport line indices match array indices
			if i > 0 {
				b.WriteString("\n")                           // First blank line
				m.lineToMessage = append(m.lineToMessage, -1) // Map blank line to -1
				b.WriteString("\n")                           // Second blank line
				m.lineToMessage = append(m.lineToMessage, -1) // Map blank line to -1
			}

			formatted := m.formatMessage(i, m.messages[i])
			b.WriteString(formatted)

			// Build direct line-to-message mapping
			// Each line of this message maps to message index i
			linesInMsg := countLines(formatted)
			for line := 0; line < linesInMsg; line++ {
				m.lineToMessage = append(m.lineToMessage, i)
			}
		}

		m.cachedContent = b.String()
		m.lastRenderedCount = currentCount
	}

	return m.cachedContent
}

// clearFormatCache clears the format cache and cached content.
// This is called when cache becomes invalid (e.g., width change, message update).
func (m *Model) clearFormatCache() {
	m.formatCache = make(map[uuid.UUID]string)
	m.cachedContent = ""
	m.lastRenderedCount = 0
	m.lineToMessage = nil
}

// getMessageAtLine finds the message index at the given line number in the viewport content.
// Returns -1 if no message is found at that line.
// The line number is 0-indexed from the top of the viewport content.
// Uses direct O(1) lookup via lineToMessage array.
func (m Model) getMessageAtLine(lineNum int) int {
	// Direct lookup - O(1)
	if lineNum >= 0 && lineNum < len(m.lineToMessage) {
		return m.lineToMessage[lineNum]
	}
	return -1
}

// getMessageStartLine finds the first line number where a given message appears.
// Returns -1 if the message is not found in the lineToMessage mapping.
// This is used for tests that need to find the starting line of a specific message.
func (m Model) getMessageStartLine(msgIdx int) int {
	for line, idx := range m.lineToMessage {
		if idx == msgIdx {
			return line
		}
	}
	return -1
}

// toggleMessageCollapse toggles the collapsed state of a tool message at the given index.
// Returns true if the message was toggled, false if it wasn't a tool message.
func (m *Model) toggleMessageCollapse(msgIdx int) bool {
	if msgIdx < 0 || msgIdx >= len(m.messages) {
		return false
	}

	msg := &m.messages[msgIdx]
	// Only tool messages can be collapsed
	isTool := msg.Type == channel.MessageTypeToolRequest || msg.Type == channel.MessageTypeToolResponse
	if !isTool {
		return false
	}

	// Toggle collapsed state in metadata
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]any)
	}
	if collapsed, ok := msg.Metadata["collapsed"].(bool); ok {
		msg.Metadata["collapsed"] = !collapsed
	} else {
		msg.Metadata["collapsed"] = true
	}

	// Invalidate cache for this message to force re-render
	delete(m.formatCache, msg.ID)

	// Invalidate cached content since collapse state affects line positions
	m.cachedContent = ""
	m.lineToMessage = nil

	return true
}

// isMessageSelected returns true if the message at the given index is currently selected.
func (m Model) isMessageSelected(msgIdx int) bool {
	return m.selectedMessageIndex >= 0 && msgIdx == m.selectedMessageIndex
}

// selectMessage sets the selected message index and invalidates cache for re-render.
// Pass -1 to deselect all messages.
func (m *Model) selectMessage(msgIdx int) {
	// Invalidate cache for previously selected message
	if m.selectedMessageIndex >= 0 && m.selectedMessageIndex < len(m.messages) {
		delete(m.formatCache, m.messages[m.selectedMessageIndex].ID)
	}

	// Invalidate cache for newly selected message
	if msgIdx >= 0 && msgIdx < len(m.messages) {
		delete(m.formatCache, m.messages[msgIdx].ID)
	}

	m.selectedMessageIndex = msgIdx

	// Invalidate cached content since selection affects border rendering
	m.cachedContent = ""
	// Also invalidate line positions to force recalculation
	// This ensures click detection uses correct positions after selection changes
	m.lineToMessage = nil
}

// invalidateCacheFor removes a specific message from the cache.
// This is designed for future use when messages can be updated or deleted.
// Currently not called in the main code path but kept for API completeness
// and tested in TestFormatCacheInvalidationOnMessageUpdate.
func (m *Model) invalidateCacheFor(msgID uuid.UUID) {
	delete(m.formatCache, msgID)
}

// formatMessage formats a single message for display in the viewport.
// Uses lipgloss styling to match the AgentMessenger appearance.
// For agent messages, it uses the markdown renderer if available.
//
// The message is displayed with a unified frame with 3-column header:
//
//	╭──────────┬──────────┬──────────────────────────────────────╮
//	│ 🤖 7ac5  │ Response │ 23:10:16                            │
//	├──────────┴──────────┴──────────────────────────────────────┤
//	│                                                            │
//	│ Hello! How can I help you today?                           │
//	│                                                            │
//	╰────────────────────────────────────────────────────────────╯
//
// When selected, the message uses double-line borders (╔═╗║╚╝) to indicate selection.
func (m *Model) formatMessage(msgIdx int, msg channel.Message) string {
	// Check cache first - return cached formatted message if available
	// Cache key: message ID
	// Cache invalidation: width change, message update, selection change
	if cached, ok := m.formatCache[msg.ID]; ok && m.cacheWidth == m.width {
		return cached
	}

	// Enforce cache size limit to prevent unbounded growth
	// Check BEFORE formatting to avoid unnecessary work
	if len(m.formatCache) >= m.maxCacheSize {
		// Clear cache when limit reached (simple eviction strategy)
		// Alternative: LRU eviction would be better but more complex
		m.clearFormatCache()
	}

	// Cache miss - format the message and cache the result
	selected := m.isMessageSelected(msgIdx)
	formatted := m.formatMessageImpl(msgIdx, msg, selected)

	// Cache the formatted message
	m.formatCache[msg.ID] = formatted
	m.cacheWidth = m.width

	return formatted
}

// formatMessageImpl implements the actual message formatting logic.
// This is separated from formatMessage to enable caching.
// The selected parameter determines whether to use bold/double-line borders.
func (m *Model) formatMessageImpl(_ int, msg channel.Message, selected bool) string {
	// For collapsed tool messages, render a compact header with click indicator
	isTool := msg.Type == channel.MessageTypeToolRequest || msg.Type == channel.MessageTypeToolResponse
	collapsed := false
	if msg.Metadata != nil {
		if c, ok := msg.Metadata["collapsed"].(bool); ok {
			collapsed = c
		}
	}
	if isTool && collapsed {
		return m.formatCollapsedToolMessage(msg, selected)
	}

	timestamp := msg.Timestamp.Format("15:04:05")

	// Build header row content with 3 columns
	// Column 1: Agent/User identifier (text format, no emoji)
	// Column 2: Message type (Response, Tool, etc.)
	// Column 3: Timestamp (left-aligned as per plan)
	var col1, col2, col3 string

	switch msg.Type {
	case channel.MessageTypeUserChat:
		col1 = "You"
		col2 = "User"
		col3 = timestamp

	case channel.MessageTypeAgentChat, channel.MessageTypeToolRequest, channel.MessageTypeToolResponse:
		agentName := formatAgentName(msg.AgentID, msg.AgentRole)
		if msg.Type == channel.MessageTypeToolRequest || msg.Type == channel.MessageTypeToolResponse {
			col1 = fmt.Sprintf("Agent: %s", agentName)
			col2 = "Tool"
		} else {
			col1 = fmt.Sprintf("Agent: %s", agentName)
			col2 = "Response"
		}
		col3 = timestamp

	case channel.MessageTypeSystemInfo:
		col1 = "System"
		col2 = "Info"
		col3 = timestamp

	case channel.MessageTypeError:
		col1 = "Error"
		col2 = "Error"
		col3 = timestamp

	default:
		col1 = "Unknown"
		col2 = "Unknown"
		col3 = timestamp
	}

	// Calculate total width and column widths
	// Total width minus margin (2 chars) to avoid edge overflow
	const minWidth = 50
	totalWidth := max(m.width-2, minWidth)

	// Column widths for the 3-column header
	// We need: totalWidth >= col1Width + col2Width + col3Width + 2
	col2Width := 10    // Message type (fixed)
	col3MinWidth := 8  // Minimum for timestamp
	col1MinWidth := 14 // Minimum for icon + agent ID

	// Calculate col3Width first (remaining space after col1 and col2)
	col3Width := totalWidth - col1MinWidth - col2Width - 2
	col3Width = max(col3Width, col3MinWidth)

	// Now calculate col1Width with remaining space
	col1Width := totalWidth - col2Width - col3Width - 2
	if col1Width < col1MinWidth {
		// If still too small, scale down proportionally
		excess := col1MinWidth - col1Width
		col1Width = col1MinWidth
		col3Width = max(col3Width-excess, col3MinWidth)
	}

	// Ensure all widths are positive
	col1Width = max(col1Width, 1)
	col2Width = max(col2Width, 1)
	col3Width = max(col3Width, 1)

	// Content width should use the same width as the border total
	// The border total is col1Width + col2Width + col3Width (for the dashes)
	// Content width = border total + 2 (for the │ on each side)
	contentWidth := col1Width + col2Width + col3Width + 2
	contentWidth = max(contentWidth, 20)

	// For agent messages, use markdown renderer if available
	var contentLines []string
	if msg.Type == channel.MessageTypeAgentChat && m.markdownRenderer != nil {
		rendered, err := m.markdownRenderer.Render(context.Background(), msg.Content, contentWidth)
		if err != nil {
			contentLines = wrapText(msg.Content, contentWidth)
		} else {
			contentLines = strings.Split(rendered, "\n")
		}
	} else {
		contentLines = wrapText(msg.Content, contentWidth)
	}

	// If no content, add an empty line for spacing
	if len(contentLines) == 0 {
		contentLines = []string{""}
	}

	// Build the border string manually to get proper T-junctions
	var b strings.Builder

	// Helper to repeat a string (with safety check)
	repeat := func(s string, count int) string {
		if count <= 0 {
			return ""
		}
		return strings.Repeat(s, count)
	}

	// Select border characters based on selection state
	// Selected messages use double-line borders (bold appearance)
	var (
		topLeft, topMid, topRight string
		midLeft, midMid, midRight string
		bottomLeft, bottomRight   string
		vertical, horizontal      string
	)
	if selected {
		// Double-line borders for selected messages
		topLeft, topMid, topRight = "╔", "╦", "╗"
		midLeft, midMid, midRight = "╠", "╩", "╣"
		bottomLeft, bottomRight = "╚", "╝"
		vertical, horizontal = "║", "═"
	} else {
		// Single-line borders for normal messages
		topLeft, topMid, topRight = "╭", "┬", "╮"
		midLeft, midMid, midRight = "├", "┴", "┤"
		bottomLeft, bottomRight = "╰", "╯"
		vertical, horizontal = "│", "─"
	}

	// Top border with T-junctions for column separators
	fmt.Fprintf(&b, "%s%s%s%s%s%s%s\n",
		topLeft, repeat(horizontal, col1Width), topMid, repeat(horizontal, col2Width), topMid, repeat(horizontal, col3Width), topRight)

	// Pad and truncate columns to fit
	// Adds 1 space of padding on each side of the text
	padCol := func(text string, width int, alignRight bool) string {
		// Use rune count for proper width calculation with emojis
		runes := []rune(text)
		textLen := len(runes)

		// Account for 1 space padding on each side
		availableWidth := max(width-2, 1)

		if textLen > availableWidth {
			// Truncate by runes (not bytes) to preserve emoji
			return " " + string(runes[:availableWidth]) + " "
		}

		padding := max(availableWidth-textLen, 0)

		if alignRight {
			return " " + strings.Repeat(" ", padding) + text + " "
		}
		return " " + text + strings.Repeat(" ", padding) + " "
	}

	// Header row with vertical separators
	// IMPORTANT: Apply padding first, then write the raw string with borders
	// Do NOT use lipgloss styles on the header row as they can interfere with alignment
	paddedCol1 := padCol(col1, col1Width, false)
	paddedCol2 := padCol(col2, col2Width, false)
	paddedCol3 := padCol(col3, col3Width, false) // LEFT-align timestamp (not right)

	// Write the header row with proper borders
	fmt.Fprintf(&b, "%s%s%s%s%s%s%s\n", vertical, paddedCol1, vertical, paddedCol2, vertical, paddedCol3, vertical)

	// Separator line
	fmt.Fprintf(&b, "%s%s%s%s%s%s%s\n",
		midLeft, repeat(horizontal, col1Width), midMid, repeat(horizontal, col2Width), midMid, repeat(horizontal, col3Width), midRight)

	// Content rows with border
	// Add 1 space of padding on each side for content
	const contentPadding = 1
	availableContentWidth := max(contentWidth-2*contentPadding, 1)

	for _, line := range contentLines {
		// Use visual width (ignoring ANSI escape codes) for proper alignment
		// This is critical for markdown-rendered content that contains color codes
		lineVisWidth := visualWidth(line)

		if lineVisWidth > availableContentWidth {
			// Truncate long lines by visual width (preserving ANSI codes at start)
			truncated := truncateVisual(line, availableContentWidth)
			fmt.Fprintf(&b, "%s %s%s %s\n", vertical, truncated, strings.Repeat(" ", availableContentWidth-visualWidth(truncated)), vertical)
		} else {
			// Pad short lines with spaces on the right
			padding := availableContentWidth - lineVisWidth
			fmt.Fprintf(&b, "%s %s%s %s\n", vertical, line, strings.Repeat(" ", padding), vertical)
		}
	}

	// Bottom border - needs to match top border width
	borderLineWidth := col1Width + col2Width + col3Width + 2
	fmt.Fprintf(&b, "%s%s%s\n", bottomLeft, repeat(horizontal, borderLineWidth), bottomRight)

	return b.String()
}

// formatCollapsedToolMessage renders a collapsed tool message with a click-to-expand indicator.
// Shows a compact header: "⚡ Tool Output [Click to expand]" with agent name and timestamp.
// When selected, uses double-line borders to indicate selection.
func (m *Model) formatCollapsedToolMessage(msg channel.Message, selected bool) string {
	timestamp := msg.Timestamp.Format("15:04:05")
	agentName := formatAgentName(msg.AgentID, msg.AgentRole)

	// Calculate total width matching the full message format
	const minWidth = 50
	totalWidth := max(m.width-2, minWidth)

	// Build a compact single-line header
	// Format: ⚡ Tool (AgentName) [Click to expand] · timestamp
	headerContent := fmt.Sprintf("⚡ Tool (%s) [Click to expand] · %s", agentName, timestamp)

	// Truncate if too long
	headerRunes := []rune(headerContent)
	availableWidth := totalWidth - 2 // Account for borders
	if len(headerRunes) > availableWidth {
		headerContent = string(headerRunes[:availableWidth])
	}

	// Select border characters based on selection state
	var topLeft, topRight, bottomLeft, bottomRight, vertical, horizontal string
	if selected {
		topLeft, topRight = "╔", "╗"
		bottomLeft, bottomRight = "╚", "╝"
		vertical, horizontal = "║", "═"
	} else {
		topLeft, topRight = "╭", "╮"
		bottomLeft, bottomRight = "╰", "╯"
		vertical, horizontal = "│", "─"
	}

	// Build the collapsed border
	var b strings.Builder
	borderWidth := totalWidth - 2

	// Top border
	fmt.Fprintf(&b, "%s%s%s\n", topLeft, strings.Repeat(horizontal, borderWidth), topRight)

	// Header content with padding
	padding := borderWidth - len([]rune(headerContent))
	if padding < 0 {
		padding = 0
	}
	fmt.Fprintf(&b, "%s %s%s %s\n", vertical, headerContent, strings.Repeat(" ", padding), vertical)

	// Bottom border (with trailing newline for consistency)
	fmt.Fprintf(&b, "%s%s%s\n", bottomLeft, strings.Repeat(horizontal, borderWidth), bottomRight)

	return b.String()
}
