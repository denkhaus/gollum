package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	m := NewModel()

	// Verify initial state
	if m.quit {
		t.Error("NewModel() should have quit=false")
	}

	if m.output != "" {
		t.Error("NewModel() should have empty output")
	}

	// Verify text input is initialized
	if !m.textInput.Focused() {
		t.Error("NewModel() textInput should be focused")
	}
}

func TestInit(t *testing.T) {
	m := NewModel()

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return a command to focus text input")
	}
}

func TestUpdate_QuitKeys(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		wantQuit bool
	}{
		{"ctrl+c quits", "ctrl+c", true},
		{"esc quits", "esc", true},
		{"other keys don't quit", "a", false},
		{"enter doesn't quit", "enter", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			msg := tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(tt.key)})

			newModel, _ := m.Update(msg)

			newM, ok := newModel.(Model)
			if !ok {
				t.Fatal("Update() should return a Model")
			}

			if newM.quit != tt.wantQuit {
				t.Errorf("Update() quit = %v, want %v", newM.quit, tt.wantQuit)
			}
		})
	}
}

func TestUpdate_EnterKey(t *testing.T) {
	m := NewModel()

	// Set some text input
	m.textInput.SetValue("test message")
	msg := tea.KeyMsg(tea.Key{Type: tea.KeyEnter, Runes: []rune("\r")})

	newModel, _ := m.Update(msg)

	newM, ok := newModel.(Model)
	if !ok {
		t.Fatal("Update() should return a Model")
	}

	// Check output was updated
	expectedOutput := "You: test message"
	if newM.output != expectedOutput {
		t.Errorf("Update() output = %q, want %q", newM.output, expectedOutput)
	}

	// Check text input was reset
	if newM.textInput.Value() != "" {
		t.Error("Update() should reset text input after enter")
	}
}

func TestUpdate_EnterKeyMultiple(t *testing.T) {
	m := NewModel()

	// First message
	m.textInput.SetValue("first message")
	newModel, _ := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter, Runes: []rune("\r")}))
	m = newModel.(Model)

	// Second message
	m.textInput.SetValue("second message")
	newModel, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter, Runes: []rune("\r")}))
	m = newModel.(Model)

	// Check both messages are in output with newlines
	if !strings.Contains(m.output, "You: first message") {
		t.Error("Output should contain first message")
	}
	if !strings.Contains(m.output, "You: second message") {
		t.Error("Output should contain second message")
	}
}

func TestUpdate_EnterKeyEmptyInput(t *testing.T) {
	m := NewModel()

	// Press enter with no text
	msg := tea.KeyMsg(tea.Key{Type: tea.KeyEnter, Runes: []rune("\r")})

	newModel, _ := m.Update(msg)

	newM, ok := newModel.(Model)
	if !ok {
		t.Fatal("Update() should return a Model")
	}

	// Check output is still empty
	if newM.output != "" {
		t.Error("Update() should not add to output for empty input")
	}
}

func TestUpdate_TextInputDelegation(t *testing.T) {
	m := NewModel()

	// Type some characters
	msg := tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("h")})
	newModel, _ := m.Update(msg)
	newM := newModel.(Model)

	if newM.textInput.Value() != "h" {
		t.Errorf("Update() textInput value = %q, want %q", newM.textInput.Value(), "h")
	}

	// Type another character
	msg = tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("i")})
	newModel, _ = newM.Update(msg)
	newM = newModel.(Model)

	if newM.textInput.Value() != "hi" {
		t.Errorf("Update() textInput value = %q, want %q", newM.textInput.Value(), "hi")
	}
}

func TestView(t *testing.T) {
	tests := []struct {
		name  string
		model Model
		check func(string) bool
		desc  string
	}{
		{
			name:  "view contains title",
			model: NewModel(),
			check: func(v string) bool {
				return strings.Contains(v, "Gollum TUI")
			},
			desc: "view should contain title",
		},
		{
			name:  "view contains instructions",
			model: NewModel(),
			check: func(v string) bool {
				return strings.Contains(v, "ctrl+c") || strings.Contains(v, "quit")
			},
			desc: "view should contain quit instructions",
		},
		{
			name:  "view contains prompt",
			model: NewModel(),
			check: func(v string) bool {
				return strings.Contains(v, "▶")
			},
			desc: "view should contain prompt character",
		},
		{
			name: "view shows output when present",
			model: Model{
				textInput: textinput.New(),
				quit:      false,
				output:    "You: test",
			},
			check: func(v string) bool {
				return strings.Contains(v, "You: test")
			},
			desc: "view should contain output when present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := tt.model.View()
			if !tt.check(view) {
				t.Error(tt.desc)
			}
			// Also verify view is not empty
			if view == "" {
				t.Error("View() should not return empty string")
			}
		})
	}
}

func TestNewProgram(t *testing.T) {
	p := NewProgram()

	if p == nil {
		t.Fatal("NewProgram() should not return nil")
	}
}
