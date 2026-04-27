package tui

import (
	"github.com/m-mizutani/gollem"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestView(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	view := m.View()
	if view == "" {
		t.Error("View() should not return empty string")
	}

	if !strings.Contains(view, ">") {
		t.Error("View() should contain prompt")
	}
}

func TestViewWithAgentExecuting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	m.agentExecuting = true

	view := m.View()
	if !strings.Contains(view, "Executing") {
		t.Error("View() with executing agent should show execution status")
	}
}

func TestViewWithMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Set dimensions to allow proper rendering
	m.width = 80
	m.height = 20

	// Configure viewport size to match dimensions
	m.viewport.Width = 80
	m.viewport.Height = 20

	m.messages = []shared.Message{
		{
			ID:        uuid.New(),
			Role: gollem.RoleSystem,
			Content:   "message 1",
			Timestamp: time.Now(),
		},
		{
			ID:        uuid.New(),
			Role: gollem.RoleSystem,
			Content:   "message 2",
			Timestamp: time.Now(),
		},
	}
	m.viewport.SetContent(m.updateViewportContent())

	view := m.View()

	// The view adds newlines after each message
	hasMessage1 := strings.Contains(view, "message 1")
	hasMessage2 := strings.Contains(view, "message 2")

	if !hasMessage1 || !hasMessage2 {
		t.Errorf("View() should contain all messages, got: %q", view)
	}
}
