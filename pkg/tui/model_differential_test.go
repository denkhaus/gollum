package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func BenchmarkUpdateViewportContentDifferential(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Start with 100 messages (simulating existing conversation)
	for i := 0; i < 100; i++ {
		m.messages = append(m.messages, Message{
			ID:        uuid.New(),
			Type:      MessageTypeAgent,
			Content:   fmt.Sprintf("Existing message %d with some content to format", i),
			Timestamp: time.Now(),
		})
	}

	// Initial viewport update (full build)
	_ = m.updateViewportContent()

	b.ResetTimer()
	b.ReportAllocs()

	// Simulate incremental message arrivals
	// Each iteration adds one message and updates viewport
	for i := 0; i < b.N; i++ {
		// Add new message
		m.messages = append(m.messages, Message{
			ID:        uuid.New(),
			Type:      MessageTypeAgent,
			Content:   fmt.Sprintf("New message %d", i),
			Timestamp: time.Now(),
		})

		// Update viewport (should use differential rendering)
		_ = m.updateViewportContent()
	}

	b.StopTimer()
	b.Logf("BenchmarkUpdateViewportContentDifferential: 100 initial messages, %d incremental updates", b.N)
}

// BenchmarkUpdateViewportContentFullRebuild tests full rebuild performance.
// This simulates the scenario where cache is invalidated (e.g., width change)
// and the entire viewport content must be rebuilt.
func BenchmarkUpdateViewportContentFullRebuild(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Create 100 messages (simulating a typical conversation)
	for i := 0; i < 100; i++ {
		m.messages = append(m.messages, Message{
			ID:        uuid.New(),
			Type:      MessageTypeAgent,
			Content:   fmt.Sprintf("Message %d with some content to format", i),
			Timestamp: time.Now(),
		})
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Each iteration simulates a full rebuild (cache invalidation)
	for i := 0; i < b.N; i++ {
		// Simulate width change to force full rebuild
		m.cacheWidth = 0
		m.cachedContent = ""
		m.lastRenderedCount = 0
		_ = m.updateViewportContent()
	}

	b.StopTimer()
	b.Logf("BenchmarkUpdateViewportContentFullRebuild: 100 messages, %d full rebuilds", b.N)
}

// TestDifferentialRenderingCorrectness verifies that differential rendering
// produces the same output as full rebuild.
func TestDifferentialRenderingCorrectness(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Build content incrementally
	for i := 0; i < 10; i++ {
		m.messages = append(m.messages, Message{
			ID:        uuid.New(),
			Type:      MessageTypeAgent,
			Content:   fmt.Sprintf("Message %d", i),
			Timestamp: time.Now(),
		})
		_ = m.updateViewportContent()
	}

	incrementalContent := m.cachedContent

	// Force full rebuild
	m.cacheWidth = 0
	m.cachedContent = ""
	m.lastRenderedCount = 0
	fullRebuildContent := m.updateViewportContent()

	// Both should produce identical output
	if incrementalContent != fullRebuildContent {
		t.Error("Differential rendering should produce same output as full rebuild")
		t.Logf("Incremental length: %d", len(incrementalContent))
		t.Logf("Full rebuild length: %d", len(fullRebuildContent))
	}
}

// TestDifferentialRenderingCacheInvalidation verifies that cache is properly
// invalidated on width change or message removal.
func TestDifferentialRenderingCacheInvalidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Add some messages
	for i := 0; i < 5; i++ {
		m.messages = append(m.messages, Message{
			ID:        uuid.New(),
			Type:      MessageTypeAgent,
			Content:   fmt.Sprintf("Message %d", i),
			Timestamp: time.Now(),
		})
	}
	_ = m.updateViewportContent()

	// Test width change invalidation
	m.width = 100
	_ = m.updateViewportContent()
	// After width change, cache should be rebuilt with new width
	if m.cacheWidth != 100 {
		t.Error("Cache width should be updated after width change")
	}

	// Test message removal invalidation (simulating /clear)
	m.width = 80
	_ = m.updateViewportContent()
	m.messages = m.messages[:3] // Remove last 2 messages
	_ = m.updateViewportContent()
	if m.lastRenderedCount != 3 {
		t.Errorf("Expected lastRenderedCount to be 3, got %d", m.lastRenderedCount)
	}
}
