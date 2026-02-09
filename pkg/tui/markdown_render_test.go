package tui

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// TestMarkdownRendering verifies that markdown rendering works when renderer is set
func TestMarkdownRendering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockAgent := mocks.NewMockAgentExecutor(ctrl)
	m := NewModel(ctx, mockAgent)

	// Set up mock markdown renderer
	mockRenderer := mocks.NewMockRenderer(ctrl)
	m.SetMarkdownRenderer(mockRenderer)

	// Set viewport dimensions
	m.width = 80

	// Test message with markdown content
	agentMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgent,
		Content:   "# Test Heading\n\nThis is **bold** text.",
		Timestamp: time.Now(),
		AgentID:   uuid.New(),
		AgentRole: "assistant",
	}

	// Expect the renderer to be called
	mockRenderer.EXPECT().
		Render(gomock.Any(), "# Test Heading\n\nThis is **bold** text.", 74). // 80 - 6 for borders/gutter
		Return("RENDERED_MARKDOWN_OUTPUT", nil)

	// Format the message
	result := m.formatMessage(agentMsg)

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
	mockAgent := mocks.NewMockAgentExecutor(ctrl)
	m := NewModel(ctx, mockAgent)

	// DO NOT set markdown renderer - test fallback behavior
	// m.SetMarkdownRenderer(mockRenderer) // <-- deliberately not called

	// Set viewport dimensions
	m.width = 80

	// Test message with markdown content
	agentMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgent,
		Content:   "# Test Heading\n\nThis is **bold** text.",
		Timestamp: time.Now(),
		AgentID:   uuid.New(),
		AgentRole: "assistant",
	}

	// Format the message
	result := m.formatMessage(agentMsg)

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
