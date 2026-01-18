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
