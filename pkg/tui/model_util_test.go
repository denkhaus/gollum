package tui

import (
	"context"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/mock/gomock"
)

func TestIsLogViewportAtBottom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20

	// Empty content - should be at bottom
	if !m.isLogViewportAtBottom() {
		t.Error("Empty log viewport should be at bottom")
	}
}

// TestHandleWindowSizeMsg tests handleWindowSizeMsg.
func TestHandleWindowSizeMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	result, _ := m.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 120, Height: 50})

	if result.width != 120 {
		t.Errorf("Expected width 120, got %d", result.width)
	}
	if result.height != 50 {
		t.Errorf("Expected height 50, got %d", result.height)
	}
}

// TestRestorePreservedInput tests restorePreservedInput.
func TestRestorePreservedInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.preservedInput = testPreservedInput
	m.cancelRequested = true // Required for restore to happen

	result := m.restorePreservedInput()

	if result.textInput.Value() != testPreservedInput {
		t.Errorf("Expected %q, got %q", testPreservedInput, result.textInput.Value())
	}
	if result.preservedInput != "" {
		t.Error("preservedInput should be cleared after restore")
	}
}

// TestRestorePreservedInput_NoCancel tests restorePreservedInput without cancelRequested.
func TestRestorePreservedInput_NoCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.preservedInput = testPreservedInput
	m.cancelRequested = false // No cancel requested

	result := m.restorePreservedInput()

	if result.textInput.Value() != "" {
		t.Errorf("Expected empty input, got '%s'", result.textInput.Value())
	}
	if result.preservedInput != testPreservedInput {
		t.Error("preservedInput should NOT be cleared without cancelRequested")
	}
}

// TestCreateAgentCommand tests createAgentCommand.
func TestCreateAgentCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)

	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20
	m = m.prepareAgentExecution()

	cmd := m.createAgentCommand("test input")
	if cmd == nil {
		t.Error("createAgentCommand should return a command")
	}
}

// TestWrapText tests wrapText helper.
func TestWrapText(t *testing.T) {
	tests := []struct {
		input    string
		maxWidth int
		want     []string
	}{
		{"short", 10, []string{"short"}},
		{"very long text", 5, []string{"very", "long", "text"}},
		{"", 10, []string{""}},
	}

	for _, tt := range tests {
		got := wrapText(tt.input, tt.maxWidth)
		if len(got) != len(tt.want) {
			t.Errorf("wrapText(%q, %d) returned %d lines, want %d", tt.input, tt.maxWidth, len(got), len(tt.want))
		}
	}
}

// TestTruncateVisual tests truncateVisual helper.
func TestTruncateVisual(t *testing.T) {
	tests := []struct {
		input    string
		maxWidth int
		want     string
	}{
		{"short", 10, "short"},
		{"very long text", 8, "very lon"},
		{"exact", 5, "exact"},
		{"", 10, ""},
		{"test", 0, ""},
	}

	for _, tt := range tests {
		got := truncateVisual(tt.input, tt.maxWidth)
		if got != tt.want {
			t.Errorf("truncateVisual(%q, %d) = %q, want %q", tt.input, tt.maxWidth, got, tt.want)
		}
	}
}

// TestHandleSearchEnter tests handleSearchEnter.
func TestHandleSearchEnter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.searchState.active = true
	m.searchState.query = "test"

	resultModel, cmd := m.handleSearchEnter()
	result := resultModel.(Model)
	if cmd != nil {
		t.Error("handleSearchEnter should return nil command")
	}
	if result.searchState.active {
		t.Error("Search should be deactivated")
	}
}

// TestHandleSearchNavigation tests handleSearchNavigation.
func TestHandleSearchNavigation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.inputHistory = []string{"first", "second", "third"}
	m.searchState.results = []int{0, 1, 2}
	m.searchState.matchedIdx = 0

	// Test Ctrl+R (prev) - should not panic and return a valid model
	resultModel, cmd := m.handleSearchNavigation(tea.KeyCtrlR)
	if cmd != nil {
		t.Error("handleSearchNavigation should return nil command")
	}
	_ = resultModel.(Model)

	// Test Ctrl+S (next) - should not panic and return a valid model
	resultModel, cmd = m.handleSearchNavigation(tea.KeyCtrlS)
	if cmd != nil {
		t.Error("handleSearchNavigation should return nil command")
	}
	_ = resultModel.(Model)
}

// TestNextSearchResult tests nextSearchResult.
func TestNextSearchResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.inputHistory = []string{"first", "second", "third"}
	m.searchState.results = []int{0, 1, 2}
	m.searchState.matchedIdx = 0

	// Next should wrap around
	result := m.nextSearchResult()
	if result.searchState.matchedIdx != 1 {
		t.Errorf("Expected matchedIdx 1, got %d", result.searchState.matchedIdx)
	}

	result = result.nextSearchResult()
	if result.searchState.matchedIdx != 2 {
		t.Errorf("Expected matchedIdx 2, got %d", result.searchState.matchedIdx)
	}

	result = result.nextSearchResult()
	if result.searchState.matchedIdx != 0 {
		t.Errorf("Expected matchedIdx 0 (wrapped), got %d", result.searchState.matchedIdx)
	}
}

// TestPrevSearchResult tests prevSearchResult.
func TestPrevSearchResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.inputHistory = []string{"first", "second", "third"}
	m.searchState.results = []int{0, 1, 2}
	m.searchState.matchedIdx = 0

	// Prev should wrap around
	result := m.prevSearchResult()
	if result.searchState.matchedIdx != 2 {
		t.Errorf("Expected matchedIdx 2 (wrapped), got %d", result.searchState.matchedIdx)
	}
}

// TestNextPrevSearchResult_Empty tests with empty results.
func TestNextPrevSearchResult_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.searchState.results = []int{}

	result := m.nextSearchResult()
	if result.searchState.matchedIdx != 0 {
		t.Errorf("Expected matchedIdx 0 with empty results, got %d", result.searchState.matchedIdx)
	}

	result = m.prevSearchResult()
	if result.searchState.matchedIdx != 0 {
		t.Errorf("Expected matchedIdx 0 with empty results, got %d", result.searchState.matchedIdx)
	}
}

// TestUpdateSearchResults tests updateSearchResults.
func TestUpdateSearchResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.inputHistory = []string{"hello world", "foo bar", "hello test"}

	// Test with query
	result := m.updateSearchResults("hello")
	if len(result.searchState.results) != 2 {
		t.Errorf("Expected 2 results for 'hello', got %d", len(result.searchState.results))
	}

	// Test with empty query
	result = m.updateSearchResults("")
	if len(result.searchState.results) != 3 {
		t.Errorf("Expected 3 results for empty query, got %d", len(result.searchState.results))
	}
}

// TestHandleSearchBackspace tests handleSearchBackspace.
func TestHandleSearchBackspace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.inputHistory = []string{"test", "testing", "tester"}
	m.searchState.results = []int{0, 1, 2}

	// Set up textInput with a value
	ti := textinput.New()
	ti.SetValue("test")
	m.textInput = ti

	resultModel, cmd := m.handleSearchBackspace()
	result := resultModel.(Model)
	if cmd != nil {
		t.Error("handleSearchBackspace should return nil command")
	}
	// Verify the function was called and returned a valid model
	_ = result.textInput.Value()
}

// TestHandleSearchBackspace_Empty tests handleSearchBackspace with empty input.
func TestHandleSearchBackspace_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.searchState.query = ""
	m.textInput.SetValue("")

	resultModel, _ := m.handleSearchBackspace()
	result := resultModel.(Model)
	if result.textInput.Value() != "" {
		t.Errorf("Expected empty string, got '%s'", result.textInput.Value())
	}
}

// TestViewportString tests the Viewport String method.
func TestViewportString(t *testing.T) {
	if ViewportMain.String() != "main" {
		t.Errorf("Expected 'main', got '%s'", ViewportMain.String())
	}
	if ViewportLogs.String() != "logs" {
		t.Errorf("Expected 'logs', got '%s'", ViewportLogs.String())
	}
	if ViewportInput.String() != "input" {
		t.Errorf("Expected 'input', got '%s'", ViewportInput.String())
	}
}
