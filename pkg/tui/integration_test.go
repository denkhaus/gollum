package tui

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/markdown"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// TestIntegrationMarkdownRendering tests the full integration of markdown rendering
// This test simulates the actual flow from app/service.go through the TUI
func TestIntegrationMarkdownRendering(t *testing.T) {
	ctx := context.Background()

	// Step 1: Create markdown renderer (same as DI container)
	renderer, err := markdown.NewRenderer()
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}

	// Step 2: Create model using existing mock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Step 3: Set markdown renderer (same as WithMarkdownRenderer)
	m.SetMarkdownRenderer(renderer)

	// Step 4: Simulate WindowSizeMsg (TUI initialization)
	m.width = 80
	m.height = 24

	// Step 5: Create agent message with markdown
	agentMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgent,
		Content:   "# Heading\n\nThis is **bold** and *italic*.",
		Timestamp: time.Now(),
		AgentID:   uuid.New(),
		AgentRole: "assistant",
	}

	// Step 6: Format the message
	result := m.formatMessage(agentMsg)

	t.Logf("Formatted message:\n%s", result)

	// Verify the result contains formatted elements
	// Glamour renders headings with colors/attributes, not plain "# Heading"
	if containsSubstring(result, "# Heading") {
		t.Error("FAILED: Raw markdown # Heading found - markdown was NOT rendered!")
		t.Logf("This means the markdown renderer was not called or failed")
	} else {
		t.Log("SUCCESS: Raw markdown not found - rendering likely worked")
	}

	// The result should contain the word "Heading" (rendered form)
	if !containsSubstring(result, "Heading") {
		t.Error("Expected 'Heading' to be in rendered output")
	}

	// The result should contain "bold" (though possibly with formatting)
	if !containsSubstring(result, "bold") {
		t.Error("Expected 'bold' to be in rendered output")
	}
}
