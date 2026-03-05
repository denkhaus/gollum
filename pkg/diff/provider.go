// Package diff provides diff generation and formatting services.
package diff

import (
	"github.com/samber/do/v2"
)

// Provider defines the interface for diff generation and formatting services.
type Provider interface {
	// GenerateDiff creates a unified diff between two file contents.
	GenerateDiff(oldPath, newPath, oldContent, newContent string) (string, error)

	// GenerateDiffForNewFile creates a diff for a newly created file.
	GenerateDiffForNewFile(path, content string) (string, error)

	// FormatForDisplay formats a diff string with terminal styling for display.
	FormatForDisplay(diff string) string

	// FormatCompact returns a compact representation of the diff.
	FormatCompact(diff string) string
}

// providerImpl is the private implementation of the Provider interface.
type providerImpl struct {
	styler *diffStyler
}

// Ensure providerImpl implements Provider at compile time.
var _ Provider = (*providerImpl)(nil)

// NewProvider creates a new diff provider for dependency injection.
func NewProvider(_ do.Injector) (Provider, error) {
	return &providerImpl{
		styler: newDiffStyler(),
	}, nil
}

// GenerateDiff creates a unified diff between two file contents.
func (p *providerImpl) GenerateDiff(oldPath, newPath, oldContent, newContent string) (string, error) {
	diff := getUnifiedDiff(oldPath, newPath, oldContent, newContent)
	return diff, nil
}

// GenerateDiffForNewFile creates a diff for a newly created file.
func (p *providerImpl) GenerateDiffForNewFile(path, content string) (string, error) {
	diff := getUnifiedDiff("/dev/null", path, "", content)
	return diff, nil
}

// FormatForDisplay formats a diff string with terminal styling for display.
func (p *providerImpl) FormatForDisplay(diff string) string {
	if diff == "" {
		return ""
	}
	return p.styler.styleDiff(diff)
}

// FormatCompact returns a compact representation of the diff.
func (p *providerImpl) FormatCompact(diff string) string {
	if diff == "" {
		return ""
	}
	return p.styler.styleCompact(diff)
}
