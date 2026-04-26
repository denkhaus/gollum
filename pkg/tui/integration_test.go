package tui

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/markdown"
	"github.com/denkhaus/gollum/pkg/shared"
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
	agentMsg := shared.Message{
		ID:             uuid.New(),
		Type:           shared.MessageTypeAgentChat,
		Content:        "# Heading\n\nThis is **bold** and *italic*.",
		Timestamp:      time.Now(),
		SessionContext: shared.SessionContext{AgentID: uuid.New()},
		AgentRole:      "assistant",
	}

	// Step 6: Format the message (index 0 = no selection)
	result := m.formatMessage(0, agentMsg)

	t.Logf("Formatted message:\n%s", result)

	// Verify the result contains formatted elements
	// Glamour renders headings with colors/attributes, not plain "# Heading"
	if containsSubstring(result, "# Heading") {
		t.Error("FAILED: Raw markdown # Heading found - markdown was NOT rendered!")
		t.Logf("This means the markdown renderer was not called or failed")
	} else {
		t.Log("SUCCESS: Raw markdown not found - rendering likely worked")
	}

	// The result should contain visible content (may have ANSI codes)
	// Note: Glamour adds ANSI color codes which can interfere with simple string matching
	visibleContentFound := containsSubstring(result, "bold") ||
		containsSubstring(result, "This") ||
		containsSubstring(result, "is") ||
		containsSubstring(result, "Heading")

	if !visibleContentFound {
		t.Error("Expected some visible content in rendered output")
		t.Log("Note: This may be due to ANSI color codes interfering with string matching")
	} else {
		t.Log("SUCCESS: Visible content found in output")
	}
}
