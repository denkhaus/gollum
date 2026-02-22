package tui

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
)

const testPreservedInput = "saved input"

func TestNewModel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Verify initial state
	if m.quit {
		t.Error("NewModel() should have quit=false")
	}

	if len(m.messages) != 0 {
		t.Error("NewModel() should have empty messages")
	}

	// Verify text input is initialized
	if !m.textInput.Focused() {
		t.Error("NewModel() textInput should be focused")
	}
}

func TestInit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return a command")
	}
}
