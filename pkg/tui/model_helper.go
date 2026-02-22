package tui

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// wrapText wraps text to fit within the specified width.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	currentLine := ""
	for _, word := range words {
		testLine := currentLine
		if testLine == "" {
			testLine = word
		} else {
			testLine = testLine + " " + word
		}

		// Handle words longer than width
		if len(word) > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = ""
			}
			// Split long word
			for i := 0; i < len(word); i += width {
				end := min(i+width, len(word))
				lines = append(lines, word[i:end])
			}
			continue
		}

		if len(testLine) <= width {
			currentLine = testLine
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// formatAgentName creates a short display name for agents.
func formatAgentName(agentID uuid.UUID, role string) string {
	if role != "" && len(role) <= 10 {
		return role
	}
	// Use first 4 characters of ID as fallback
	idStr := agentID.String()
	if len(idStr) >= 4 {
		return idStr[:4]
	}
	return "agent"
}

// ansiRegex matches ANSI escape sequences
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[()][AB012]`)

// countLines returns the number of lines in a string.
// Trailing newlines are not counted as additional lines to match
// how lines appear in the viewport content.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	// Remove trailing newline to avoid counting it as an extra line
	// This is important for line-to-message mapping accuracy
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return 0
	}
	count := 1
	for _, r := range s {
		if r == '\n' {
			count++
		}
	}
	return count
}

// visualWidth returns the visual/display width of a string, ignoring ANSI escape codes.
// This is needed because glamour returns text with ANSI color codes that shouldn't
// be counted when calculating padding and alignment.
func visualWidth(s string) int {
	// Strip ANSI escape sequences
	stripped := ansiRegex.ReplaceAllString(s, "")
	// Count runes (not bytes) for proper Unicode handling
	return len([]rune(stripped))
}

// truncateVisual truncates a string to fit within the given visual width.
// Preserves ANSI escape codes at the start of the string.
func truncateVisual(s string, maxVisualWidth int) string {
	if maxVisualWidth <= 0 {
		return ""
	}

	// Find leading ANSI codes (to preserve colors at the start)
	leadingANSI := ansiRegex.FindStringIndex(s)
	ansiPrefix := ""
	if leadingANSI != nil && leadingANSI[0] == 0 {
		ansiPrefix = s[:leadingANSI[1]]
		s = s[leadingANSI[1]:]
	}

	runes := []rune(s)
	if len(runes) <= maxVisualWidth {
		return ansiPrefix + s
	}

	return ansiPrefix + string(runes[:maxVisualWidth])
}
