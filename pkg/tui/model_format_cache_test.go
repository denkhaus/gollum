package tui

import (
	"github.com/m-mizutani/gollem"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestFormatCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Create a test message
	msg := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleAssistant,
		Content:   "Test message with **markdown**",
		Timestamp: time.Now(),
	}

	// First call should cache the result
	result1 := m.formatMessage(0, msg)
	if result1 == "" {
		t.Fatal("First formatMessage call should return non-empty string")
	}

	// Verify cache was populated (should have 1 entry)
	if len(m.formatCache) != 1 {
		t.Errorf("Expected cache size 1, got %d", len(m.formatCache))
	}

	// Second call should return cached result (same width)
	result2 := m.formatMessage(0, msg)
	if result1 != result2 {
		t.Errorf("Cache hit should return same result. First: %q, Second: %q", result1, result2)
	}

	// Verify cache size is still 1 (no new entries)
	if len(m.formatCache) != 1 {
		t.Errorf("Expected cache size 1 after cache hit, got %d", len(m.formatCache))
	}
}

// TestFormatCacheInvalidationOnWidthChange verifies that cache is cleared on width change.
func TestFormatCacheInvalidationOnWidthChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Create and cache a message
	msg := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleAssistant,
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	_ = m.formatMessage(0, msg)

	// Verify cache has entry
	if len(m.formatCache) != 1 {
		t.Errorf("Expected cache size 1, got %d", len(m.formatCache))
	}

	// Simulate window resize by changing width
	m.width = 100
	m.clearFormatCache()

	// Verify cache was cleared
	if len(m.formatCache) != 0 {
		t.Errorf("Expected cache size 0 after width change, got %d", len(m.formatCache))
	}

	// New format call should populate cache again
	_ = m.formatMessage(0, msg)
	if len(m.formatCache) != 1 {
		t.Errorf("Expected cache size 1 after re-format, got %d", len(m.formatCache))
	}
}

// TestFormatCacheInvalidationOnMessageUpdate verifies that individual messages can be invalidated.
func TestFormatCacheInvalidationOnMessageUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Create two messages and cache them
	msg1 := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleAssistant,
		Content:   "shared.Message 1",
		Timestamp: time.Now(),
	}
	msg2 := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleAssistant,
		Content:   "shared.Message 2",
		Timestamp: time.Now(),
	}

	_ = m.formatMessage(0, msg1)
	_ = m.formatMessage(1, msg2)

	// Verify both are cached
	if len(m.formatCache) != 2 {
		t.Errorf("Expected cache size 2, got %d", len(m.formatCache))
	}

	// Invalidate first message
	m.invalidateCacheFor(msg1.ID)

	// Verify only one entry remains
	if len(m.formatCache) != 1 {
		t.Errorf("Expected cache size 1 after invalidation, got %d", len(m.formatCache))
	}

	// Verify the correct entry remains
	if _, ok := m.formatCache[msg2.ID]; !ok {
		t.Error("Second message should still be cached")
	}

	// Verify first message is not cached
	if _, ok := m.formatCache[msg1.ID]; ok {
		t.Error("First message should be invalidated")
	}
}

// TestFormatCacheSizeLimit verifies that cache respects max size limit.
func TestFormatCacheSizeLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.maxCacheSize = 5 // Set low limit for testing

	// Add messages up to the limit
	for i := 0; i < 5; i++ {
		msg := shared.Message{
			ID:        uuid.New(),
			Role: gollem.RoleAssistant,
			Content:   fmt.Sprintf("shared.Message %d", i),
			Timestamp: time.Now(),
		}
		_ = m.formatMessage(i, msg)
	}

	// Verify cache has 5 entries
	if len(m.formatCache) != 5 {
		t.Errorf("Expected cache size 5, got %d", len(m.formatCache))
	}

	// Add one more message (should trigger cache clear)
	msg := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleAssistant,
		Content:   "shared.Message 6",
		Timestamp: time.Now(),
	}
	_ = m.formatMessage(0, msg)

	// Cache should be cleared and have only 1 entry (the new message)
	if len(m.formatCache) != 1 {
		t.Errorf("Expected cache size 1 after limit exceeded, got %d", len(m.formatCache))
	}

	// Verify the new message is in cache
	if _, ok := m.formatCache[msg.ID]; !ok {
		t.Error("New message should be cached after cache clear")
	}
}
