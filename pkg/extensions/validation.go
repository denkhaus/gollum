package extensions

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Extension name must be: alphanumeric, hyphen, underscore only
	extNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// ValidateExtensionName validates an extension name for security
func ValidateExtensionName(name string) error {
	if name == "" {
		return fmt.Errorf("extension name cannot be empty")
	}

	// Check for path traversal
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("extension name contains invalid characters")
	}

	// Check for spaces
	if strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("extension name cannot contain whitespace")
	}

	// Check regex
	if !extNameRegex.MatchString(name) {
		return fmt.Errorf("extension name must contain only alphanumeric, hyphen, or underscore characters")
	}

	return nil
}

// ValidateFuncName validates a function name for security
func ValidateFuncName(name string) error {
	return ValidateExtensionName(name) // Same rules apply
}
