package tui

// This file handles conversation export for the TUI.

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

// handleExport handles conversation export to a file.
func (m Model) handleExport() (tea.Model, tea.Cmd) {
	filename := generateExportFilename()
	content := buildExportContent(m.messages)

	if err := writeExportFile(filename, content); err != nil {
		errorMsg := shared.Message{
			ID:        uuid.New(),
			Type:      shared.MessageTypeError,
			Content:   fmt.Sprintf("Failed to export conversation: %v", err),
			Timestamp: time.Now(),
		}
		m.addMessage(errorMsg)
	} else {
		successMsg := shared.Message{
			ID:        uuid.New(),
			Type:      shared.MessageTypeSystemInfo,
			Content:   fmt.Sprintf("Conversation exported to: %s", filename),
			Timestamp: time.Now(),
		}
		m.addMessage(successMsg)
	}

	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoBottom()
	return m, nil
}

// generateExportFilename creates a timestamped filename for export.
func generateExportFilename() string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("gollum_export_%s.txt", timestamp)
}

// buildExportContent formats messages for export.
func buildExportContent(messages []shared.Message) string {
	var content strings.Builder
	content.WriteString("# Gollum Conversation Export\n")
	fmt.Fprintf(&content, "# Exported: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&content, "# Total Messages: %d\n", len(messages))
	content.WriteString(strings.Repeat("=", 60) + "\n\n")

	for _, msg := range messages {
		timestamp := msg.Timestamp.Format("2006-01-02 15:04:05")
		var prefix string

		switch msg.Type {
		case shared.MessageTypeUserChat:
			prefix = fmt.Sprintf("[%s] You:", timestamp)
		case shared.MessageTypeAgentChat:
			prefix = fmt.Sprintf("[%s] Agent:", timestamp)
		case shared.MessageTypeToolRequest, shared.MessageTypeToolResponse:
			prefix = fmt.Sprintf("[%s] Tool:", timestamp)
		case shared.MessageTypeSystemInfo:
			prefix = fmt.Sprintf("[%s] System:", timestamp)
		case shared.MessageTypeError:
			prefix = fmt.Sprintf("[%s] Error:", timestamp)
		default:
			prefix = fmt.Sprintf("[%s] Unknown:", timestamp)
		}

		content.WriteString(prefix + "\n")
		content.WriteString(msg.Content)
		content.WriteString("\n\n")
	}

	return content.String()
}

// writeExportFile writes content to the export file.
func writeExportFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}
