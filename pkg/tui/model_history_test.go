package tui

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestAddToHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Set a small history max size for testing
	m.config.HistoryMaxSize = 3

	// Add some entries
	m.addToHistory("first")
	m.addToHistory("second")
	m.addToHistory("third")
	m.addToHistory("fourth")

	// Should only have 3 entries due to size limit
	if len(m.inputHistory) != 3 {
		t.Errorf("History should be limited to max size, got %d", len(m.inputHistory))
	}

	// Oldest entry should be dropped
	if m.inputHistory[0] != "second" {
		t.Error("Oldest entry should be dropped when exceeding max size")
	}

	// Newest entry should be present
	if m.inputHistory[2] != "fourth" {
		t.Error("Newest entry should be present")
	}

	// Test duplicate prevention
	m.addToHistory("fourth") // Same as last entry
	if len(m.inputHistory) != 3 {
		t.Error("Duplicate entry should not be added")
	}
}

func TestSearchHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	m.inputHistory = []string{"git status", "git commit", "ls -la", "git log"}

	// Search for "git"
	results := m.searchHistory("git")
	if len(results) != 3 {
		t.Errorf("Search for 'git' should find 3 results, got %d", len(results))
	}

	// Verify indices are correct
	if results[0] != 0 || results[1] != 1 || results[2] != 3 {
		t.Error("Search results should have correct indices")
	}

	// Search for "ls"
	results = m.searchHistory("ls")
	if len(results) != 1 {
		t.Errorf("Search for 'ls' should find 1 result, got %d", len(results))
	}

	// Search for non-existent
	results = m.searchHistory("xyz")
	if len(results) != 0 {
		t.Error("Search for non-existent should find 0 results")
	}

	// Case insensitive search
	results = m.searchHistory("GIT")
	if len(results) != 3 {
		t.Error("Search should be case insensitive")
	}
}

func TestGetMessageCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	if m.getMessageCount() != 0 {
		t.Error("getMessageCount should return 0 for empty model")
	}

	m.messages = []shared.Message{
		{ID: uuid.New(), Type: shared.MessageTypeSystemInfo, Content: "msg1", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeUserChat, Content: "msg2", Timestamp: time.Now()},
	}

	if m.getMessageCount() != 2 {
		t.Error("getMessageCount should return correct count")
	}
}
