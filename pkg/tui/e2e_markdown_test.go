package tui

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/markdown"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// TestE2EMarkdownRendererSetup verifies that the markdown renderer is properly
// set up when using NewProgramWithContext with WithMarkdownRenderer option
func TestE2EMarkdownRendererSetup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Step 1: Create markdown renderer (simulating DI container)
	renderer, err := markdown.NewRenderer()
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}

	// Step 2: Create agent executor
	agent := setupMockAgent(ctrl)

	// Step 3: Create model using NewProgramWithContext pattern
	// This simulates what app/service.go does
	m := NewModel(ctx, agent)

	// Apply options in the same order as app/service.go
	opts := []func(*Model){
		WithMessageChannel(),
		func(m *Model) { m.SetLoggerService(nil) }, // WithLoggerService
		WithMarkdownRenderer(renderer),
	}

	for _, opt := range opts {
		opt(&m)
	}

	// Step 4: Verify markdown renderer is set
	if m.markdownRenderer == nil {
		t.Fatal("FAILED: markdownRenderer is nil after WithMarkdownRenderer option!")
	}

	// Step 5: Simulate WindowSizeMsg
	m.width = 80
	m.height = 24

	// Step 6: Create a real agent message
	agentMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgent,
		Content:   "# Test\n\nThis is **bold** text.",
		Timestamp: time.Now(),
		AgentID:   uuid.New(),
		AgentRole: "assistant",
	}

	// Step 7: Format the message
	result := m.formatMessage(agentMsg)

	t.Logf("E2E Test - Formatted message:\n%s", result)

	// Verify markdown was rendered
	if containsSubstring(result, "# Test") {
		t.Error("FAILED: Raw markdown # Test found - renderer was not called!")
	} else {
		t.Log("SUCCESS: Raw markdown not found - rendering worked")
	}

	// Note: ANSI color codes from Glamour may interfere with simple string matching
	// The key check is that raw markdown was NOT found (above), which indicates rendering worked

	// Check for at least one visible word (more robust check)
	visibleContentFound := containsSubstring(result, "bold") ||
		containsSubstring(result, "This") ||
		containsSubstring(result, "is")

	if !visibleContentFound {
		t.Error("FAILED: No visible content found in output")
		t.Log("Note: This may be due to ANSI color codes interfering with string matching")
	} else {
		t.Log("SUCCESS: Visible content found in output")
	}

	// Verify the model structure
	t.Logf("Model state: width=%d, height=%d, markdownRenderer=%v",
		m.width, m.height, m.markdownRenderer != nil)
}
