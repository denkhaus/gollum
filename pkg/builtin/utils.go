// Package builtin provides production-ready built-in hooks for common use cases.
// These hooks can be registered out-of-the-box and configured via environment variables.
package builtin

import "fmt"

// truncateString truncates a string to a maximum length and adds ellipsis if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return fmt.Sprintf("%s...", s[:maxLen])
}

// truncateMap truncates a map to a maximum string length and adds ellipsis if truncated.
// This is useful for logging large map structures like tool args or results.
func truncateMap(m map[string]any, maxLen int) string {
	if m == nil {
		return ""
	}
	// Convert map to string representation
	s := fmt.Sprintf("%v", m)
	if len(s) <= maxLen {
		return s
	}
	return fmt.Sprintf("%s...", s[:maxLen])
}
