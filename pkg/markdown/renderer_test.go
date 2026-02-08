// Package markdown provides tests for markdown rendering services.
package markdown

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRenderer(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)
	require.NotNil(t, renderer)
}

func TestRenderer_Render_BasicMarkdown(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	ctx := context.Background()
	markdown := "# Hello World\n\nThis is **bold** text."
	result, err := renderer.Render(ctx, markdown, 80)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	// Check for content (ANSI codes will be present but content should be there)
	assert.Contains(t, result, "Hello")
	assert.Contains(t, result, "World")
	assert.Contains(t, result, "bold")
}

func TestRenderer_Render_CodeBlock(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	ctx := context.Background()
	markdown := "```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```"
	result, err := renderer.Render(ctx, markdown, 80)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "func")
}

func TestRenderer_Render_EmptyString(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	ctx := context.Background()
	result, err := renderer.Render(ctx, "", 80)
	require.NoError(t, err)
	assert.NotEmpty(t, result) // glamour returns empty string with styling
}

func TestRenderer_Render_WordWrapping(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	ctx := context.Background()
	longText := "This is a very long line of text that should be wrapped at the specified width"
	result, err := renderer.Render(ctx, longText, 40)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestRenderer_Caching(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	ctx := context.Background()
	markdown := "# Cached Content\n\nThis should be cached."
	width := 80

	// First render
	result1, err := renderer.Render(ctx, markdown, width)
	require.NoError(t, err)

	// Second render with same parameters should use cache
	result2, err := renderer.Render(ctx, markdown, width)
	require.NoError(t, err)

	// Results should be identical
	assert.Equal(t, result1, result2)
}

func TestRenderer_CacheKey(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	ctx := context.Background()
	markdown := "# Content"
	width1 := 80
	width2 := 100

	// Render with different widths
	result1, err := renderer.Render(ctx, markdown, width1)
	require.NoError(t, err)

	result2, err := renderer.Render(ctx, markdown, width2)
	require.NoError(t, err)

	// Results should differ due to different wrapping
	assert.NotEqual(t, result1, result2)
}

func TestRenderer_ClearCache(t *testing.T) {
	r, err := NewRenderer()
	require.NoError(t, err)

	// Ensure r is *glamourRenderer type
	glamourRenderer, ok := r.(*glamourRenderer)
	require.True(t, ok, "Renderer should be *glamourRenderer type")

	ctx := context.Background()
	markdown := "# Cache Test"

	// First render
	result1, err := r.Render(ctx, markdown, 80)
	require.NoError(t, err)

	// Clear cache
	glamourRenderer.ClearCache()

	// Render again after cache clear
	result2, err := r.Render(ctx, markdown, 80)
	require.NoError(t, err)

	// Results should be identical even after cache clear
	assert.Equal(t, result1, result2)
}

func TestRenderer_Render_Lists(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	ctx := context.Background()
	markdown := "- Item 1\n- Item 2\n- Item 3"
	result, err := renderer.Render(ctx, markdown, 80)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	// Check for items (ANSI codes and bullet points will be present)
	assert.Contains(t, result, "Item")
}

func TestRenderer_Render_Links(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	ctx := context.Background()
	markdown := "[Link Text](https://example.com)"
	result, err := renderer.Render(ctx, markdown, 80)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Link Text")
}

func TestRenderer_Render_ContextCancellation(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	markdown := "# Test"
	_, err = renderer.Render(ctx, markdown, 80)
	// Note: glamour doesn't currently support context cancellation,
	// but the renderer should handle it gracefully
	// This test documents current behavior
	assert.NoError(t, err) // glamour doesn't check context
}

func TestRenderer_CacheKeySize(t *testing.T) {
	r, err := NewRenderer()
	require.NoError(t, err)

	// Ensure r is *glamourRenderer type
	glamourRenderer, ok := r.(*glamourRenderer)
	require.True(t, ok, "Renderer should be *glamourRenderer type")

	ctx := context.Background()

	// Test with a very long markdown content
	longContent := strings.Repeat("# This is a very long heading\n", 1000)

	// First render - cache miss
	_, err = r.Render(ctx, longContent, 80)
	require.NoError(t, err)

	// Second render - cache hit
	_, err = r.Render(ctx, longContent, 80)
	require.NoError(t, err)

	// Verify cache keys are constant size (SHA256 = 64 hex chars)
	glamourRenderer.cacheMu.RLock()
	defer glamourRenderer.cacheMu.RUnlock()

	for key := range glamourRenderer.cache {
		// SHA256 hex encoding is always 64 characters
		assert.Equal(t, 64, len(key), "Cache key should be 64 characters (SHA256 hex)")
	}
}

func TestRenderer_NoOSCSequences(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	ctx := context.Background()
	markdown := "# Test\n\n**Bold** text."
	result, err := renderer.Render(ctx, markdown, 80)
	require.NoError(t, err)

	// Check that no OSC-11 sequences (background color) are in the output
	// OSC-11 format: \x1b]11;rgb:XXXX/XXXX/XXXX...
	assert.NotContains(t, result, "\x1b]11", "Output should not contain OSC-11 sequences")
	assert.NotContains(t, result, "rgb:", "Output should not contain RGB color sequences (from OSC)")

	// Check for any other OSC sequences (general OSC format: \x1b]N;...)
	assert.NotRegexp(t, `\x1b\][0-9]+;`, result, "Output should not contain any OSC sequences")
}

