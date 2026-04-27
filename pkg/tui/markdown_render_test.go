package tui

import (
	"github.com/m-mizutani/gollem"
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/markdown"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// TestMarkdownRendering verifies that markdown rendering works when renderer is set
func TestMarkdownRendering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockAgent := NewMockAgentExecutor(ctrl)
	m := NewModel(ctx, mockAgent)

	// Set up mock markdown renderer
	mockRenderer := markdown.NewMockRenderer(ctrl)
	m.SetMarkdownRenderer(mockRenderer)

	// Set viewport dimensions
	m.width = 80

	// Test message with markdown content
	agentMsg := shared.Message{
		ID:             uuid.New(),
		Role: gollem.RoleAssistant,
		Content:        "# Test Heading\n\nThis is **bold** text.",
		Timestamp:      time.Now(),
		SessionContext: shared.SessionContext{AgentID: uuid.New()},
		AgentRole:      "assistant",
	}

	// Expect the renderer to be called
	// New layout: totalWidth = 80-2 = 78, col1Width=14, col2Width=10, col3Width=52
	// contentWidth = col1Width + col2Width + col3Width + 2 = 14 + 10 + 52 + 2 = 78
	mockRenderer.EXPECT().
		Render(gomock.Any(), "# Test Heading\n\nThis is **bold** text.", 78).
		Return("RENDERED_MARKDOWN_OUTPUT", nil)

	// Format the message (index 0 = no selection)
	result := m.formatMessage(0, agentMsg)

	// Verify the rendered markdown is in the output
	if len(result) == 0 {
		t.Error("formatMessage() returned empty string")
	}

	// The result should contain the rendered markdown (not the raw content)
	if containsSubstring(result, "RENDERED_MARKDOWN_OUTPUT") {
		t.Log("SUCCESS: Markdown rendering is working - rendered output found in result")
	} else {
		t.Error("FAILED: Markdown rendering output not found in result")
		t.Logf("Result: %q", result)
	}
}

// TestPlainTextFallback verifies fallback to plain text when renderer is nil
func TestPlainTextFallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockAgent := NewMockAgentExecutor(ctrl)
	m := NewModel(ctx, mockAgent)

	// DO NOT set markdown renderer - test fallback behavior
	// m.SetMarkdownRenderer(mockRenderer) // <-- deliberately not called

	// Set viewport dimensions
	m.width = 80

	// Test message with markdown content
	agentMsg := shared.Message{
		ID:             uuid.New(),
		Role: gollem.RoleAssistant,
		Content:        "# Test Heading\n\nThis is **bold** text.",
		Timestamp:      time.Now(),
		SessionContext: shared.SessionContext{AgentID: uuid.New()},
		AgentRole:      "assistant",
	}

	// Format the message (index 0 = no selection)
	result := m.formatMessage(0, agentMsg)

	// Verify the raw markdown content is in the output (no rendering)
	if len(result) == 0 {
		t.Error("formatMessage() returned empty string")
	}

	// The result should contain the raw content (fallback behavior)
	if containsSubstring(result, "# Test Heading") {
		t.Log("SUCCESS: Plain text fallback is working - raw content found in result")
	} else {
		t.Error("FAILED: Raw content not found in result (expected fallback)")
		t.Logf("Result: %q", result)
	}
}
