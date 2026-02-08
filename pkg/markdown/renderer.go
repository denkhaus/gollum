// Package markdown provides markdown rendering services for terminal UI.
// It uses glamour for rich terminal markdown rendering with syntax highlighting.
package markdown

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
)

// Renderer defines the interface for markdown rendering.
// This allows for different rendering implementations and mock testing.
type Renderer interface {
	// Render converts markdown to formatted terminal output.
	// The width parameter specifies the maximum line width for wrapping.
	Render(ctx context.Context, markdown string, width int) (string, error)
}

// glamourRenderer implements Renderer using the glamour library.
type glamourRenderer struct {
	// cache stores rendered markdown by cache key
	cache map[string]string
	cacheMu sync.RWMutex

	// renderers stores a glamour renderer per width for reuse
	renderers map[int]*glamour.TermRenderer
	renderersMu sync.RWMutex
}

// NewRenderer creates a new markdown renderer with dark theme styling.
// The renderer is configured with:
// - Dark theme for better terminal readability
// - Automatic word wrapping based on provided width
// - Syntax highlighting for code blocks
// - Emoji support
//
// Renderers are created per width and reused for performance.
func NewRenderer() (Renderer, error) {
	return &glamourRenderer{
		cache:     make(map[string]string),
		renderers: make(map[int]*glamour.TermRenderer),
	}, nil
}

// Render converts markdown to formatted terminal output.
// It implements caching to improve performance for repeated renders.
// The cache key is a SHA256 hash of the content and width, ensuring constant size.
//
// Renderers are reused per width to avoid expensive re-initialization.
func (r *glamourRenderer) Render(ctx context.Context, markdown string, width int) (string, error) {
	// Generate cache key using SHA256 hash for constant size regardless of content length
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", markdown, width)))
	cacheKey := hex.EncodeToString(hash[:])

	// Check cache first (fast path)
	r.cacheMu.RLock()
	if cached, ok := r.cache[cacheKey]; ok {
		r.cacheMu.RUnlock()
		return cached, nil
	}
	r.cacheMu.RUnlock()

	// Get or create renderer for this width
	renderer, err := r.getRenderer(width)
	if err != nil {
		return "", fmt.Errorf("failed to get glamour renderer: %w", err)
	}

	// Render the markdown
	result, err := renderer.Render(markdown)
	if err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}

	// Cache the result
	r.cacheMu.Lock()
	r.cache[cacheKey] = result
	r.cacheMu.Unlock()

	return result, nil
}

// getRenderer returns a glamour renderer for the given width, creating one if necessary.
// Renderers are reused to avoid expensive re-initialization on each render call.
func (r *glamourRenderer) getRenderer(width int) (*glamour.TermRenderer, error) {
	// Check if we already have a renderer for this width
	r.renderersMu.RLock()
	if renderer, ok := r.renderers[width]; ok {
		r.renderersMu.RUnlock()
		return renderer, nil
	}
	r.renderersMu.RUnlock()

	// Create a new renderer for this width
	// Use "dark" standard style explicitly to avoid WithAutoStyle() which sends OSC-11 sequences
	// that can leak into the text input field in Bubbletea
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(styles.DarkStyle),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create glamour renderer: %w", err)
	}

	// Store the renderer for reuse
	r.renderersMu.Lock()
	r.renderers[width] = renderer
	r.renderersMu.Unlock()

	return renderer, nil
}

// ClearCache clears the rendering cache and resets all renderers.
// This can be used to free memory or force re-rendering.
func (r *glamourRenderer) ClearCache() {
	r.cacheMu.Lock()
	r.cache = make(map[string]string)
	r.cacheMu.Unlock()

	r.renderersMu.Lock()
	r.renderers = make(map[int]*glamour.TermRenderer)
	r.renderersMu.Unlock()
}
