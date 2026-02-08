// Package markdown provides DI provider for markdown rendering services.
package markdown

import (
	"github.com/samber/do/v2"
)

// ProvideRenderer is a DI provider function that creates and provides
// the markdown renderer service as a singleton in the DI container.
//
// Usage:
//
//	injector := do.New()
//	renderer := do.MustInvoke[markdown.Renderer](injector)
func ProvideRenderer(injector do.Injector) (Renderer, error) {
	return NewRenderer()
}
