// Package diff provides diff generation and formatting services.
package diff

import (
	"github.com/aymanbagabas/go-udiff"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

// getUnifiedDiff generates a unified diff string between two contents.
func getUnifiedDiff(oldPath, newPath, oldContent, newContent string) string {
	return udiff.Unified(oldPath, newPath, oldContent, newContent)
}

// diffStyler handles terminal styling for diff output.
type diffStyler struct {
	addedStyle    lipgloss.Style
	removedStyle  lipgloss.Style
	headerStyle   lipgloss.Style
	metaStyle     lipgloss.Style
	locationStyle lipgloss.Style
}

// newDiffStyler creates a new diff styler with predefined styles.
func newDiffStyler() *diffStyler {
	return &diffStyler{
		addedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC71")), // Green
		removedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E74C3C")), // Red
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB")). // Blue
			Bold(true),
		metaStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9B59B6")), // Purple
		locationStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F39C12")). // Orange
			Bold(true),
	}
}

// styleDiff applies terminal styling to a unified diff string.
func (s *diffStyler) styleDiff(diff string) string {
	if diff == "" {
		return ""
	}
	var result strings.Builder
	lines := strings.Split(diff, "\n")
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		// Handle empty first line
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			// File path headers
			result.WriteString(s.headerStyle.Render(line))
		case strings.HasPrefix(line, "@@ "):
			// Hunk location headers
			result.WriteString(s.locationStyle.Render(line))
		case strings.HasPrefix(line, "+"):
			// Added lines
			result.WriteString(s.addedStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			// Removed lines
			result.WriteString(s.removedStyle.Render(line))
		case strings.HasPrefix(line, "diff "):
			// Diff meta command
			result.WriteString(s.metaStyle.Render(line))
		case strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "new file ") ||
			strings.HasPrefix(line, "deleted file ") || strings.HasPrefix(line, "Binary "):
			// Git diff metadata
			result.WriteString(s.metaStyle.Render(line))
		default:
			// Context lines and other content
			result.WriteString(line)
		}
	}
	return result.String()
}

// styleCompact returns a compact, styled representation of the diff.
// This format is optimized for terminal display with limited vertical space.
func (s *diffStyler) styleCompact(diff string) string {
	if diff == "" {
		return ""
	}
	var result strings.Builder
	lines := strings.Split(diff, "\n")
	// Track state for compact display
	inHunk := false
	skippedContext := 0
	for _, line := range lines {
		// Skip file headers and metadata for compact view
		if strings.HasPrefix(line, "diff ") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "new file ") ||
			strings.HasPrefix(line, "deleted file ") ||
			strings.HasPrefix(line, "Binary ") {
			continue
		}
		// Show file paths
		if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
			if inHunk {
				// End of previous hunk
				inHunk = false
			}
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(s.headerStyle.Render(line))
			result.WriteString("\n")
			continue
		}
		// Hunk headers
		if strings.HasPrefix(line, "@@ ") {
			if skippedContext > 0 {
				result.WriteString(s.metaStyle.Render("..."))
				result.WriteString("\n")
				skippedContext = 0
			}
			inHunk = true
			result.WriteString(s.locationStyle.Render(line))
			result.WriteString("\n")
			continue
		}
		// Changes within hunks
		if inHunk {
			if strings.HasPrefix(line, "+") {
				result.WriteString(s.addedStyle.Render(line))
				result.WriteString("\n")
			} else if strings.HasPrefix(line, "-") {
				result.WriteString(s.removedStyle.Render(line))
				result.WriteString("\n")
			} else if !strings.HasPrefix(line, "\\") {
				// Context lines - skip consecutive context
				if strings.HasPrefix(line, " ") {
					skippedContext++
					if skippedContext <= 2 {
						// Show first 2 context lines
						result.WriteString(line)
						result.WriteString("\n")
					}
				} else {
					skippedContext = 0
					result.WriteString(line)
					result.WriteString("\n")
				}
			}
			// Skip "No newline at end of file" markers
		}
	}
	return result.String()
}
