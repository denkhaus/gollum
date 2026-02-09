# TUI Update Handler Refactoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor `pkg/tui/update.go` (670 lines) into 9 focused handler files organized by event type, reducing the main file to ~100 lines while preserving all functionality.

**Architecture:** Extract message handlers from the main `Update()` switch into separate files. Each file handles one message type or subsystem. The main `Update()` becomes a thin dispatcher routing to specialized handlers.

**Tech Stack:** Go 1.23+, Bubbletea TUI framework, existing test suite

**Design Document:** `docs/plans/2026-02-09-tui-update-refactor-design.md`

---

## Task 1: Phase 1 - Create Stub Files

**Files:**
- Create: `pkg/tui/update_export.go` (stub)
- Create: `pkg/tui/update_multiline.go` (stub)
- Create: `pkg/tui/update_history.go` (stub)
- Create: `pkg/tui/update_window.go` (stub)
- Create: `pkg/tui/update_tick.go` (stub)
- Create: `pkg/tui/update_mouse.go` (stub)
- Create: `pkg/tui/update_agent.go` (stub)
- Create: `pkg/tui/update_search.go` (stub)
- Create: `pkg/tui/update_key.go` (stub)

**Step 1: Create stub file for update_export.go**

```bash
cat > pkg/tui/update_export.go << 'EOF'
package tui

// Placeholder for export handling
// Will be implemented in Task 2
EOF
```

**Step 2: Create stub file for update_multiline.go**

```bash
cat > pkg/tui/update_multiline.go << 'EOF'
package tui

// Placeholder for multi-line input handling
// Will be implemented in Task 3
EOF
```

**Step 3: Create stub file for update_history.go**

```bash
cat > pkg/tui/update_history.go << 'EOF'
package tui

import tea "github.com/charmbracelet/bubbletea"

// Placeholder for input history handling
// Will be implemented in Task 4
EOF
```

**Step 4: Create stub file for update_window.go**

```bash
cat > pkg/tui/update_window.go << 'EOF'
package tui

import tea "github.com/charmbracelet/bubbletea"

// Placeholder for window resize handling
// Will be implemented in Task 5
EOF
```

**Step 5: Create stub file for update_tick.go**

```bash
cat > pkg/tui/update_tick.go << 'EOF'
package tui

// Placeholder for timer-based updates
// Will be implemented in Task 6
EOF
```

**Step 6: Create stub file for update_mouse.go**

```bash
cat > pkg/tui/update_mouse.go << 'EOF'
package tui

import tea "github.com/charmbracelet/bubbletea"

// Placeholder for mouse event handling
// Will be implemented in Task 7
EOF
```

**Step 7: Create stub file for update_agent.go**

```bash
cat > pkg/tui/update_agent.go << 'EOF'
package tui

// Placeholder for agent execution handling
// Will be implemented in Task 8
EOF
```

**Step 8: Create stub file for update_search.go**

```bash
cat > pkg/tui/update_search.go << 'EOF'
package tui

import tea "github.com/charmbracelet/bubbletea"

// Placeholder for search mode handling
// Will be implemented in Task 9
EOF
```

**Step 9: Create stub file for update_key.go**

```bash
cat > pkg/tui/update_key.go << 'EOF'
package tui

import tea "github.com/charmbracelet/bubbletea"

// Placeholder for keyboard input handling
// Will be implemented in Task 10
EOF
```

**Step 10: Verify compilation**

```bash
go build ./pkg/tui/...
```

Expected: SUCCESS (no errors, stubs compile)

**Step 11: Run tests to verify baseline**

```bash
go test ./pkg/tui/... -v
```

Expected: All tests pass

**Step 12: Commit stub files**

```bash
git add pkg/tui/update_*.go
git commit -m "refactor(tui): Add stub files for update handler refactoring

Creates 9 new stub files for event-type organized handlers.
Phase 1 of refactoring: setup complete.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Phase 2a - Extract Export Handler

**Files:**
- Modify: `pkg/tui/update.go:539-599` (remove handleExport)
- Modify: `pkg/tui/update.go:172` (update exportMsg case)
- Modify: `pkg/tui/update_export.go` (implement)

**Step 1: Read current handleExport implementation**

Reference: Lines 539-599 in `update.go`

**Step 2: Implement update_export.go**

```go
package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// handleExport handles conversation export to a file.
func (m Model) handleExport() (tea.Model, tea.Cmd) {
	filename := generateExportFilename()
	content := buildExportContent(m.messages)

	if err := writeExportFile(filename, content); err != nil {
		errorMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeError,
			Content:   fmt.Sprintf("Failed to export conversation: %v", err),
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, errorMsg)
	} else {
		successMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeSystem,
			Content:   fmt.Sprintf("Conversation exported to: %s", filename),
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, successMsg)
	}

	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoTop()
	return m, nil
}

// generateExportFilename creates a timestamped filename for export.
func generateExportFilename() string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("gollum_export_%s.txt", timestamp)
}

// buildExportContent formats messages for export.
func buildExportContent(messages []Message) string {
	var content strings.Builder
	content.WriteString("# Gollum Conversation Export\n")
	content.WriteString(fmt.Sprintf("# Exported: %s\n", time.Now().Format(time.RFC3339)))
	content.WriteString(fmt.Sprintf("# Total Messages: %d\n", len(messages)))
	content.WriteString(strings.Repeat("=", 60) + "\n\n")

	for _, msg := range messages {
		timestamp := msg.Timestamp.Format("2006-01-02 15:04:05")
		var prefix string

		switch msg.Type {
		case MessageTypeUser:
			prefix = fmt.Sprintf("[%s] 👤 You:", timestamp)
		case MessageTypeAgent:
			prefix = fmt.Sprintf("[%s] 🤖 Agent:", timestamp)
		case MessageTypeTool:
			prefix = fmt.Sprintf("[%s] ⚡ Tool:", timestamp)
		case MessageTypeSystem:
			prefix = fmt.Sprintf("[%s] 🚀 System:", timestamp)
		case MessageTypeError:
			prefix = fmt.Sprintf("[%s] ❌ Error:", timestamp)
		default:
			prefix = fmt.Sprintf("[%s] ❓ Unknown:", timestamp)
		}

		content.WriteString(prefix + "\n")
		content.WriteString(msg.Content)
		content.WriteString("\n\n")
	}

	return content.String()
}

// writeExportFile writes content to the export file.
func writeExportFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}
```

**Step 3: Update update.go exportMsg case to call handler**

Change line 172 in `update.go`:
```go
// Before:
case exportMsg:
	return m.handleExport()

// After (no change needed, already calls handler):
// This stays the same, just confirming it routes correctly
```

**Step 4: Remove handleExport from update.go**

Delete lines 539-599 from `update.go`

**Step 5: Verify tests pass**

```bash
go test ./pkg/tui/... -v
```

Expected: All tests pass

**Step 6: Verify build**

```bash
go build ./pkg/tui/...
```

Expected: SUCCESS

**Step 7: Commit**

```bash
git add pkg/tui/update_export.go pkg/tui/update.go
git commit -m "refactor(tui): Extract export handling to update_export.go

Moves conversation export functionality to dedicated file.
- handleExport() -> update_export.go
- Adds helper functions: generateExportFilename, buildExportContent, writeExportFile
- Reduces update.go from 670 to ~600 lines

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Phase 2b - Extract Multi-line Handler

**Files:**
- Modify: `pkg/tui/update.go:507-537` (remove multiline functions)
- Modify: `pkg/tui/update_key.go:252-268` (update multiline toggle calls)
- Modify: `pkg/tui/update_multiline.go` (implement)

**Step 1: Read current multi-line implementation**

Reference: Lines 507-537 in `update.go`

**Step 2: Implement update_multiline.go**

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

// submitMultiLineInput submits the multi-line buffer as a single message.
func (m Model) submitMultiLineInput() (tea.Model, tea.Cmd) {
	currentInput := m.textInput.Value()
	if currentInput != "" {
		m.multiLineBuffer = append(m.multiLineBuffer, currentInput)
	}

	// Join all lines with newlines
	input := strings.Join(m.multiLineBuffer, "\n")
	input = strings.TrimSpace(input)

	if input == "" {
		return m.exitMultiLineMode()
	}

	// Exit multi-line mode first
	m.multiLineInput = false
	m.multiLineBuffer = []string{}
	m.textInput.SetValue(input)

	// Now submit as regular input
	return m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
}

// exitMultiLineMode exits multi-line input mode, discarding the buffer.
func (m Model) exitMultiLineMode() (tea.Model, tea.Cmd) {
	m.multiLineInput = false
	m.multiLineBuffer = []string{}
	m.textInput.SetValue("")
	return m, nil
}

// appendToMultiLineBuffer adds current input to the multi-line buffer.
func (m Model) appendToMultiLineBuffer(input string) Model {
	m.multiLineBuffer = append(m.multiLineBuffer, input)
	return m
}
```

**Step 3: Remove functions from update.go**

Delete lines 507-537 from `update.go`

**Step 4: Verify tests pass**

```bash
go test ./pkg/tui/... -v
```

Expected: All tests pass

**Step 5: Commit**

```bash
git add pkg/tui/update_multiline.go pkg/tui/update.go
git commit -m "refactor(tui): Extract multi-line input to update_multiline.go

Moves multi-line input mode functionality to dedicated file.
- submitMultiLineInput() -> update_multiline.go
- exitMultiLineMode() -> update_multiline.go
- Adds helper: appendToMultiLineBuffer
- Reduces update.go from ~600 to ~570 lines

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Phase 2c - Extract History Handler

**Files:**
- Modify: `pkg/tui/update.go:601-631` (remove handleHistoryNavigation)
- Modify: `pkg/tui/model.go` (check if addToHistory should move)
- Modify: `pkg/tui/update_history.go` (implement)

**Step 1: Read current history implementation**

Reference: Lines 601-631 in `update.go`, and find `addToHistory` in `model.go`

**Step 2: Implement update_history.go**

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

// handleHistoryNavigation handles up/down arrow for input history.
func (m Model) handleHistoryNavigation(keyType tea.KeyType) (tea.Model, tea.Cmd) {
	if len(m.inputHistory) == 0 {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(tea.KeyMsg{Type: keyType})
		return m, cmd
	}

	return m.navigateHistory(keyType), nil
}

// navigateHistory moves through input history based on key type.
func (m Model) navigateHistory(keyType tea.KeyType) Model {
	switch keyType {
	case tea.KeyUp:
		// Navigate to older history
		if m.inputHistoryIndex > 0 {
			m.inputHistoryIndex--
			m.textInput.SetValue(m.inputHistory[m.inputHistoryIndex])
			m.textInput.CursorEnd()
		}
	case tea.KeyDown:
		// Navigate to newer history
		if m.inputHistoryIndex < len(m.inputHistory)-1 {
			m.inputHistoryIndex++
			m.textInput.SetValue(m.inputHistory[m.inputHistoryIndex])
			m.textInput.CursorEnd()
		} else if m.inputHistoryIndex == len(m.inputHistory)-1 {
			// Clear input when going past the newest history item
			m.inputHistoryIndex = len(m.inputHistory)
			m.textInput.SetValue("")
		}
	}
	return m
}
```

Note: `addToHistory` has a pointer receiver (`*Model`), so it should stay in `model.go` or be refactored. For now, leave it in `model.go`.

**Step 3: Remove handleHistoryNavigation from update.go**

Delete lines 601-631 from `update.go`

**Step 4: Verify tests pass**

```bash
go test ./pkg/tui/... -v
```

Expected: All tests pass

**Step 5: Commit**

```bash
git add pkg/tui/update_history.go pkg/tui/update.go
git commit -m "refactor(tui): Extract history navigation to update_history.go

Moves input history navigation functionality to dedicated file.
- handleHistoryNavigation() -> update_history.go
- Adds helper: navigateHistory
- addToHistory remains in model.go (pointer receiver)
- Reduces update.go from ~570 to ~540 lines

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Phase 3a - Extract Window Resize Handler

**Files:**
- Modify: `pkg/tui/update.go:47-71` (remove WindowSizeMsg case)
- Modify: `pkg/tui/update_window.go` (implement)

**Step 1: Read current WindowSizeMsg implementation**

Reference: Lines 47-71 in `update.go`

**Step 2: Implement update_window.go**

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

// handleWindowSizeMsg handles terminal resize events.
func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Update viewport size (reserve space for input, footer, status bar, and log panel)
	mainHeight, logHeight := m.calculateViewportDimensions()

	m.viewport.Width = msg.Width
	m.viewport.Height = mainHeight
	m.logViewport.Width = msg.Width
	m.logViewport.Height = logHeight

	return m, nil
}

// calculateViewportDimensions returns the height for main viewport, log panel,
// and the number of reserved lines.
func (m Model) calculateViewportDimensions() (mainHeight int, logHeight int) {
	reservedLines := 4 // status bar + prompt + footer
	if m.config.StatusEnabled {
		reservedLines++ // Extra line for status bar
	}

	logHeight = 6 // Default log panel height
	reservedLines += logHeight + 1 // +1 for log separator

	mainHeight = m.height - reservedLines
	if mainHeight < 1 {
		mainHeight = 1
		// If we're very short, reduce log panel height
		logHeight = m.height - reservedLines + logHeight
		if logHeight < 3 {
			logHeight = 3 // Minimum log panel height
		}
	}

	return mainHeight, logHeight
}
```

**Step 3: Update update.go WindowSizeMsg case**

Change lines 47-71 in `update.go`:
```go
// Before (full case body):
case tea.WindowSizeMsg:
	m.width = msg.Width
	m.height = msg.Height
	// ... all the calculation code ...

// After:
case tea.WindowSizeMsg:
	return m.handleWindowSizeMsg(msg)
```

**Step 4: Verify tests pass**

```bash
go test ./pkg/tui/... -v
```

Expected: All tests pass

**Step 5: Commit**

```bash
git add pkg/tui/update_window.go pkg/tui/update.go
git commit -m "refactor(tui): Extract window resize handling to update_window.go

Moves terminal resize functionality to dedicated file.
- WindowSizeMsg case -> handleWindowSizeMsg()
- Adds helper: calculateViewportDimensions
- Reduces update.go from ~540 to ~510 lines

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Phase 3b - Extract Tick Handler

**Files:**
- Modify: `pkg/tui/update.go:73-106` (remove tickMsg and logTickMsg cases)
- Modify: `pkg/tui/update_tick.go` (implement)

**Step 1: Read current tick implementation**

Reference: Lines 73-106 in `update.go`

**Step 2: Implement update_tick.go**

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/logger"
)

// handleTickMsg handles periodic context cancellation checks.
func (m Model) handleTickMsg(msg tickMsg) (Model, tea.Cmd) {
	// Check if context was canceled
	select {
	case <-m.ctx.Done():
		m.quit = true
		return m, tea.Quit
	default:
	}
	return m, tea.Batch(m.tickCmd(), m.logTickCmd())
}

// handleLogTickMsg handles periodic log entry fetching.
func (m Model) handleLogTickMsg(msg logTickMsg) (Model, tea.Cmd) {
	m = m.fetchNewLogEntries()
	return m, m.logTickCmd()
}

// fetchNewLogEntries fetches and appends new log entries from the logger service.
func (m Model) fetchNewLogEntries() Model {
	if m.logService == nil {
		return m
	}

	newLogs := m.logService.GetLogs(logger.LogFilter{
		SinceSeq: m.lastLogFetchSeq,
		Reverse:  false,
	})

	if len(newLogs) == 0 {
		return m
	}

	// Update last fetch sequence to the most recent log's sequence
	m.lastLogFetchSeq = newLogs[len(newLogs)-1].Sequence

	// Format and append new log entries
	for _, log := range newLogs {
		formatted := fmt.Sprintf("%s [%s] %s",
			log.Timestamp.Format("15:04:05"),
			strings.ToUpper(log.Level),
			log.Message)
		m.logEntries = append(m.logEntries, formatted)
	}

	// Update log viewport content
	m.logViewport.SetContent(strings.Join(m.logEntries, "\n"))
	m.logViewport.GotoBottom()

	return m
}
```

**Step 3: Update update.go tick cases**

Change lines 73-106 in `update.go`:
```go
// Before:
case tickMsg:
	// ... full implementation ...

case logTickMsg:
	// ... full implementation ...

// After:
case tickMsg:
	return m.handleTickMsg(msg)

case logTickMsg:
	return m.handleLogTickMsg(msg)
```

**Step 4: Verify tests pass**

```bash
go test ./pkg/tui/... -v
```

Expected: All tests pass

**Step 5: Commit**

```bash
git add pkg/tui/update_tick.go pkg/tui/update.go
git commit -m "refactor(tui): Extract tick handling to update_tick.go

Moves timer-based update functionality to dedicated file.
- tickMsg case -> handleTickMsg()
- logTickMsg case -> handleLogTickMsg()
- Adds helper: fetchNewLogEntries
- Reduces update.go from ~510 to ~480 lines

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: Phase 3c - Extract Mouse Handler

**Files:**
- Modify: `pkg/tui/update.go:174-198` (remove MouseMsg and mouseDebounceMsg cases)
- Modify: `pkg/tui/update.go:633-670` (remove handleMouseMsg)
- Modify: `pkg/tui/update_mouse.go` (implement)

**Step 1: Read current mouse implementation**

Reference: Lines 174-198, 633-670 in `update.go`

**Step 2: Implement update_mouse.go**

```go
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// handleMouseMsg handles mouse events with debouncing for smooth scrolling.
func (m Model) handleMouseMsg(msg tea.MouseMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseWheelUp, tea.MouseWheelDown:
		// Increment tag to invalidate any pending debounce commands
		m = m.incrementDebounceTag()

		// Determine scroll direction (-1 for up, 1 for down)
		direction := -1
		if msg.Type == tea.MouseWheelDown {
			direction = 1
		}

		// Return a debounce command with the current tag
		return m, m.createDebounceCommand(direction, m.activeViewport)
	}

	// For other mouse events, pass to text input (for clicks, etc.)
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// handleMouseDebounceMsg processes debounced mouse scroll events.
func (m Model) handleMouseDebounceMsg(msg mouseDebounceMsg) (Model, tea.Cmd) {
	// Only apply if the tag matches (this is the latest scroll event)
	if msg.tag != m.mouseDebounceTag {
		return m, nil
	}

	scrollLines := 3 // Scroll 3 lines per debounced event for smooth scrolling
	m = m.scrollViewport(msg.viewport, msg.direction*scrollLines)

	return m, nil
}

// scrollViewport scrolls the specified viewport by the given number of lines.
func (m Model) scrollViewport(viewport string, lines int) Model {
	if viewport == "logs" {
		return m.scrollLogViewport(lines)
	}
	return m.scrollMainViewport(lines)
}

// scrollMainViewport scrolls the main conversation viewport.
func (m Model) scrollMainViewport(lines int) Model {
	if lines < 0 {
		m.viewport.LineUp(-lines)
	} else {
		m.viewport.LineDown(lines)
	}
	return m
}

// scrollLogViewport scrolls the log panel viewport.
func (m Model) scrollLogViewport(lines int) Model {
	if lines < 0 {
		m.logViewport.LineUp(-lines)
	} else {
		m.logViewport.LineDown(lines)
	}
	return m
}

// incrementDebounceTag increments the debounce tag to invalidate pending events.
func (m Model) incrementDebounceTag() Model {
	m.mouseDebounceTag++
	return m
}

// createDebounceCommand creates a debounce command for mouse scrolling.
func (m Model) createDebounceCommand(direction int, viewport string) tea.Cmd {
	return tea.Tick(m.mouseDebounceDuration, func(_ time.Time) tea.Msg {
		return mouseDebounceMsg{
			tag:       m.mouseDebounceTag,
			direction: direction,
			viewport:  viewport,
		}
	})
}
```

**Step 3: Update update.go mouse cases**

Change lines 174-198 in `update.go`:
```go
// Before:
case tea.MouseMsg:
	// ... full implementation ...

case mouseDebounceMsg:
	// ... full implementation ...

// After:
case tea.MouseMsg:
	return m.handleMouseMsg(msg)

case mouseDebounceMsg:
	return m.handleMouseDebounceMsg(msg)
```

**Step 4: Remove handleMouseMsg from update.go**

Delete lines 633-670 from `update.go`

**Step 5: Verify tests pass**

```bash
go test ./pkg/tui/... -v
```

Expected: All tests pass

**Step 6: Commit**

```bash
git add pkg/tui/update_mouse.go pkg/tui/update.go
git commit -m "refactor(tui): Extract mouse handling to update_mouse.go

Moves mouse event functionality to dedicated file.
- MouseMsg case -> handleMouseMsg()
- mouseDebounceMsg case -> handleMouseDebounceMsg()
- Adds helpers: scrollViewport, scrollMainViewport, scrollLogViewport
- Adds helpers: incrementDebounceTag, createDebounceCommand
- Reduces update.go from ~480 to ~420 lines

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 8: Phase 3d - Extract Agent Handler

**Files:**
- Modify: `pkg/tui/update.go:108-168` (remove agentCompleteMsg and newMessageMsg cases)
- Modify: `pkg/tui/update_agent.go` (implement)

**Step 1: Read current agent implementation**

Reference: Lines 108-168 in `update.go`

**Step 2: Implement update_agent.go**

```go
package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

// handleAgentCompleteMsg handles agent execution completion.
func (m Model) handleAgentCompleteMsg(msg agentCompleteMsg) (Model, tea.Cmd) {
	m.agentExecuting = false
	m.agentStartTime = time.Time{} // Clear start time
	m.textInput.Focus()

	if msg.err != nil {
		m = m.handleAgentError(msg.err)
	} else if msg.response != nil {
		m = m.handleAgentResponse(msg.response.Texts)
	}

	// Restore preserved input if user canceled during execution
	m = m.restorePreservedInput()

	return m, nil
}

// handleNewMessageMsg handles streaming messages from AgentMessenger.
func (m Model) handleNewMessageMsg(msg newMessageMsg) (Model, tea.Cmd) {
	m.messages = append(m.messages, msg.message)
	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoTop()
	return m, m.waitForMessages()
}

// handleAgentError adds an error message to the conversation.
func (m Model) handleAgentError(err error) Model {
	errorMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeError,
		Content:   err.Error(),
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, errorMsg)
	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoTop()
	return m
}

// handleAgentResponse adds agent response texts to the conversation.
func (m Model) handleAgentResponse(texts []string) Model {
	// Only add response texts if NOT using AgentMessenger
	if !m.shouldAddResponseTexts() {
		return m
	}

	for _, text := range texts {
		agentMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeAgent,
			Content:   text,
			Timestamp: time.Now(),
			AgentID:   uuid.Nil, // Will be set by agent messenger
			AgentRole: "",
		}
		m.messages = append(m.messages, agentMsg)
	}

	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoTop()
	return m
}

// restorePreservedInput restores the user's input after a cancel.
func (m Model) restorePreservedInput() Model {
	if m.cancelRequested && m.preservedInput != "" {
		m.textInput.SetValue(m.preservedInput)
		m.textInput.CursorEnd()
		m.preservedInput = ""
		m.cancelRequested = false
	}
	return m
}

// shouldAddResponseTexts returns true if legacy mode (messageChan is nil).
func (m Model) shouldAddResponseTexts() bool {
	return m.messageChan == nil
}

// prepareAgentExecution sets up the model for agent execution.
func (m Model) prepareAgentExecution(input string) (Model, context.CancelFunc) {
	// Add user message to messages
	userMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeUser,
		Content:   input,
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, userMsg)
	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoTop()

	// Add to history
	m.addToHistory(input)

	// Clear input and prepare for agent execution
	m.textInput.SetValue("")
	m.agentExecuting = true
	m.agentStartTime = time.Now()
	m.textInput.Blur()

	// Create a per-request context that can be canceled
	var reqCtx context.Context
	var cancel context.CancelFunc
	if m.ctx != nil {
		reqCtx, cancel = context.WithCancel(m.ctx)
	} else {
		reqCtx, cancel = context.WithCancel(context.Background())
	}
	m.currentCancel = cancel

	return m, cancel
}

// createAgentCommand creates the command that will execute the agent.
func (m Model) createAgentCommand(ctx context.Context, input string, cancel context.CancelFunc) tea.Cmd {
	return func() tea.Msg {
		defer cancel()
		m.currentCancel = nil

		response, err := m.agent.Execute(ctx, input)
		return agentCompleteMsg{response: response, err: err}
	}
}
```

**Step 3: Update update.go agent cases**

Change lines 108-168 in `update.go`:
```go
// Before:
case agentCompleteMsg:
	// ... full implementation ...

case newMessageMsg:
	// ... full implementation ...

// After:
case agentCompleteMsg:
	return m.handleAgentCompleteMsg(msg)

case newMessageMsg:
	return m.handleNewMessageMsg(msg)
```

**Step 4: Update update_key.go to use new helpers**

The `handleEnter` function (to be implemented in Task 10) will need to use:
- `prepareAgentExecution()`
- `createAgentCommand()`

This will be done in Task 10.

**Step 5: Verify tests pass**

```bash
go test ./pkg/tui/... -v
```

Expected: All tests pass

**Step 6: Commit**

```bash
git add pkg/tui/update_agent.go pkg/tui/update.go
git commit -m "refactor(tui): Extract agent handling to update_agent.go

Moves agent execution functionality to dedicated file.
- agentCompleteMsg case -> handleAgentCompleteMsg()
- newMessageMsg case -> handleNewMessageMsg()
- Adds helpers: handleAgentError, handleAgentResponse, restorePreservedInput
- Adds helpers: prepareAgentExecution, createAgentCommand, shouldAddResponseTexts
- Reduces update.go from ~420 to ~350 lines

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 9: Phase 4a - Extract Search Handler

**Files:**
- Modify: `pkg/tui/update.go:430-505` (remove handleSearchKeyMsg)
- Modify: `pkg/tui/model.go` (find and move searchHistory, nextSearchResult, prevSearchResult, exitSearch)
- Modify: `pkg/tui/update_search.go` (implement)

**Step 1: Read current search implementation**

Reference: Lines 430-505 in `update.go`, and find search-related methods in `model.go`

**Step 2: Implement update_search.go**

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

// handleSearchKeyMsg handles keyboard input during history search.
func (m Model) handleSearchKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		return m.handleSearchEscape()

	case tea.KeyEnter:
		return m.handleSearchEnter()

	case tea.KeyCtrlS, tea.KeyCtrlR:
		return m.handleSearchNavigation(msg.Type)

	case tea.KeyRunes:
		return m.handleSearchQueryUpdate(string(msg.Runes))

	case tea.KeyBackspace:
		return m.handleSearchBackspace()

	default:
		// Ignore other keys in search mode
		return m, nil
	}
}

// handleSearchEscape exits search mode.
func (m Model) handleSearchEscape() (tea.Model, tea.Cmd) {
	m = m.exitSearch()
	return m, nil
}

// handleSearchEnter accepts the selected history entry and exits search mode.
func (m Model) handleSearchEnter() (tea.Model, tea.Cmd) {
	m = m.exitSearch()
	return m, nil
}

// handleSearchNavigation navigates to next/previous search result.
func (m Model) handleSearchNavigation(keyType tea.KeyType) (tea.Model, tea.Cmd) {
	if keyType == tea.KeyCtrlR {
		m = m.prevSearchResult()
	} else {
		m = m.nextSearchResult()
	}
	return m, nil
}

// handleSearchQueryUpdate updates the search query as user types.
func (m Model) handleSearchQueryUpdate(input string) (tea.Model, tea.Cmd) {
	m.textInput.SetValue(input)
	m.searchState.query = input
	m = m.updateSearchResults(input)

	// Update display to show first match
	if len(m.searchState.results) > 0 {
		m.searchState.matchedIdx = 0
		m.textInput.SetValue(m.inputHistory[m.searchState.results[0]])
	}

	return m, nil
}

// handleSearchBackspace handles backspace in search mode.
func (m Model) handleSearchBackspace() (tea.Model, tea.Cmd) {
	current := m.textInput.Value()
	if len(current) == 0 {
		return m, nil
	}

	m.textInput.SetValue(current[:len(current)-1])
	m.searchState.query = m.textInput.Value()
	m = m.updateSearchResults(m.searchState.query)

	// Update display
	if len(m.searchState.results) > 0 {
		m.searchState.matchedIdx = 0
		m.textInput.SetValue(m.inputHistory[m.searchState.results[0]])
	}

	return m, nil
}

// nextSearchResult moves to the next search result.
func (m Model) nextSearchResult() Model {
	if len(m.searchState.results) == 0 {
		return m
	}

	m.searchState.matchedIdx++
	if m.searchState.matchedIdx >= len(m.searchState.results) {
		m.searchState.matchedIdx = 0
	}

	m.textInput.SetValue(m.inputHistory[m.searchState.results[m.searchState.matchedIdx]])
	m.textInput.CursorEnd()
	return m
}

// prevSearchResult moves to the previous search result.
func (m Model) prevSearchResult() Model {
	if len(m.searchState.results) == 0 {
		return m
	}

	m.searchState.matchedIdx--
	if m.searchState.matchedIdx < 0 {
		m.searchState.matchedIdx = len(m.searchState.results) - 1
	}

	m.textInput.SetValue(m.inputHistory[m.searchState.results[m.searchState.matchedIdx]])
	m.textInput.CursorEnd()
	return m
}

// exitSearch resets search state and exits search mode.
func (m Model) exitSearch() Model {
	m.searchState.active = false
	m.searchState.query = ""
	m.searchState.results = nil
	m.searchState.matchedIdx = 0
	m.textInput.SetValue("")
	return m
}

// updateSearchResults updates the search results based on query.
func (m Model) updateSearchResults(query string) Model {
	if query == "" {
		// Show all history when query is empty
		m.searchState.results = make([]int, 0, len(m.inputHistory))
		for i := range m.inputHistory {
			m.searchState.results = append(m.searchState.results, i)
		}
	} else {
		m.searchState.results = m.searchHistory(query)
	}
	return m
}

// searchHistory returns indices of history entries matching the query.
func (m Model) searchHistory(query string) []int {
	var results []int
	query = strings.ToLower(query)

	for i, entry := range m.inputHistory {
		if strings.Contains(strings.ToLower(entry), query) {
			results = append(results, i)
		}
	}

	return results
}
```

**Step 3: Remove handleSearchKeyMsg from update.go**

Delete lines 430-505 from `update.go`

**Step 4: Remove search methods from model.go**

Find and remove: `searchHistory`, `nextSearchResult`, `prevSearchResult`, `exitSearch` from `model.go`

**Step 5: Verify tests pass**

```bash
go test ./pkg/tui/... -v
```

Expected: All tests pass

**Step 6: Commit**

```bash
git add pkg/tui/update_search.go pkg/tui/update.go pkg/tui/model.go
git commit -m "refactor(tui): Extract search handling to update_search.go

Moves history search mode functionality to dedicated file.
- handleSearchKeyMsg() -> update_search.go
- Moves from model.go: searchHistory, nextSearchResult, prevSearchResult, exitSearch
- Adds helpers: handleSearchEscape, handleSearchEnter, handleSearchNavigation
- Adds helpers: handleSearchQueryUpdate, handleSearchBackspace, updateSearchResults
- Reduces update.go from ~350 to ~280 lines

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 10: Phase 4b - Extract Keyboard Handler (Final Complex Task)

**Files:**
- Modify: `pkg/tui/update.go:212-428` (remove handleKeyMsg)
- Modify: `pkg/tui/update_key.go` (implement full keyboard handling)

**Step 1: Read current keyboard implementation**

Reference: Lines 212-428 in `update.go`

**Step 2: Implement update_key.go with per-key handlers**

```go
package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If in search mode, handle search-specific keys
	if m.searchState.active {
		return m.handleSearchKeyMsg(msg)
	}

	// Handle key combinations
	switch msg.Type {
	case tea.KeyCtrlC:
		return m.handleCtrlC()

	case tea.KeyEscape:
		return m.handleEscape()

	case tea.KeyEnter:
		return m.handleEnter(msg)

	case tea.KeyUp, tea.KeyDown:
		return m.handleArrowKeys(msg.Type)

	case tea.KeyCtrlR:
		return m.handleCtrlR()

	case tea.KeyCtrlL:
		return m.handleCtrlL()

	case tea.KeyPgUp, tea.KeyPgDown:
		return m.handlePageKeys(msg)

	case tea.KeyShiftUp, tea.KeyShiftDown:
		return m.handleShiftArrows(msg.Type)

	default:
		return m.handleDefaultKey(msg)
	}
}

// handleCtrlC handles graceful shutdown.
func (m Model) handleCtrlC() (tea.Model, tea.Cmd) {
	m.quit = true
	cancelMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeSystem,
		Content:   "^C",
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, cancelMsg)
	m.viewport.SetContent(m.updateViewportContent())
	return m, tea.Quit
}

// handleEscape handles ESC key (exit multi-line or cancel agent).
func (m Model) handleEscape() (tea.Model, tea.Cmd) {
	// Exit multi-line mode if active
	if m.multiLineInput && m.config.MultiLineEnabled {
		return m.exitMultiLineMode()
	}

	// Cancel current agent execution if active
	if m.agentExecuting && m.currentCancel != nil {
		m.cancelRequested = true
		m.preservedInput = m.textInput.Value()
		m.textInput.SetValue("")
		m.currentCancel()
		m.currentCancel = nil
	}

	return m, nil
}

// handleEnter handles ENTER key (multi-line toggle or submit input).
func (m Model) handleEnter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for Alt+Enter (multi-line input)
	if msg.Alt && m.config.MultiLineEnabled {
		return m.handleMultiLineToggle(msg)
	}

	// Handle multi-line submission
	if m.multiLineInput && m.config.MultiLineEnabled {
		return m.submitMultiLineInput()
	}

	// Submit input to agent
	input := strings.TrimSpace(m.textInput.Value())
	if input == "" {
		return m, nil
	}

	return m.handleSubmitInput(input)
}

// handleMultiLineToggle handles Alt+Enter to toggle multi-line mode.
func (m Model) handleMultiLineToggle(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !m.multiLineInput {
		// Start multi-line mode
		m.multiLineInput = true
		currentInput := m.textInput.Value()
		if currentInput != "" {
			m.multiLineBuffer = []string{currentInput}
			m.textInput.SetValue("")
		}
	} else {
		// Add newline to current buffer
		currentInput := m.textInput.Value()
		m.multiLineBuffer = append(m.multiLineBuffer, currentInput)
		m.textInput.SetValue("")
	}
	return m, nil
}

// handleSubmitInput submits user input to the agent.
func (m Model) handleSubmitInput(input string) (tea.Model, tea.Cmd) {
	// Check for slash commands
	if strings.HasPrefix(input, "/") {
		return m.executeCommand(input)
	}

	// Check for exit commands (legacy)
	if input == "quit" || input == "exit" {
		return m.handleQuitCommand(input)
	}

	// Prepare agent execution
	m, cancel := m.prepareAgentExecution(input)

	// Return the command that will execute the agent
	return m, m.createAgentCommand(m.ctx, input, cancel)
}

// handleQuitCommand handles legacy quit/exit commands.
func (m Model) handleQuitCommand(input string) (tea.Model, tea.Cmd) {
	m.quit = true
	goodbyeMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeSystem,
		Content:   "👋 Goodbye!",
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, goodbyeMsg)
	m.viewport.SetContent(m.updateViewportContent())
	return m, tea.Quit
}

// handleArrowKeys handles up/down arrow keys for viewport scrolling.
func (m Model) handleArrowKeys(keyType tea.KeyType) (tea.Model, tea.Cmd) {
	scrollLines := 5 // Scroll 5 lines per keypress

	if m.activeViewport == "logs" {
		if keyType == tea.KeyUp {
			m.logViewport.LineUp(scrollLines)
		} else {
			m.logViewport.LineDown(scrollLines)
		}
		return m, nil
	}

	// Main viewport
	if keyType == tea.KeyUp {
		m.viewport.LineUp(scrollLines)
	} else {
		m.viewport.LineDown(scrollLines)
	}
	return m, nil
}

// handleCtrlR starts history search mode.
func (m Model) handleCtrlR() (tea.Model, tea.Cmd) {
	if len(m.inputHistory) == 0 {
		return m, nil
	}

	m.searchState = searchState{
		active:     true,
		query:      "",
		matchedIdx: 0,
		results:    make([]int, 0, len(m.inputHistory)),
	}

	// Initialize with all history entries
	for i := range m.inputHistory {
		m.searchState.results = append(m.searchState.results, i)
	}

	if len(m.searchState.results) > 0 {
		m.searchState.matchedIdx = len(m.searchState.results) - 1
		m.textInput.SetValue(m.inputHistory[m.searchState.results[m.searchState.matchedIdx]])
		m.textInput.CursorEnd()
	}

	return m, nil
}

// handleCtrlL toggles between main and log viewport.
func (m Model) handleCtrlL() (tea.Model, tea.Cmd) {
	if m.activeViewport == "main" {
		m.activeViewport = "logs"
	} else {
		m.activeViewport = "main"
	}
	return m, nil
}

// handlePageKeys handles PageUp/PageDown keys.
func (m Model) handlePageKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.activeViewport == "logs" {
		var cmd tea.Cmd
		m.logViewport, cmd = m.logViewport.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleShiftArrows handles Shift+Up/Down for faster scrolling.
func (m Model) handleShiftArrows(keyType tea.KeyType) (tea.Model, tea.Cmd) {
	scrollLines := 10 // Scroll 10 lines for faster navigation

	if m.activeViewport == "logs" {
		if keyType == tea.KeyShiftUp {
			m.logViewport.LineUp(scrollLines)
		} else {
			m.logViewport.LineDown(scrollLines)
		}
		return m, nil
	}

	if keyType == tea.KeyShiftUp {
		m.viewport.LineUp(scrollLines)
	} else {
		m.viewport.LineDown(scrollLines)
	}
	return m, nil
}

// handleDefaultKey passes unhandled keys to textinput.
func (m Model) handleDefaultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}
```

**Step 3: Remove handleKeyMsg from update.go**

Delete lines 212-428 from `update.go`

**Step 4: Add import for strings and context to update_key.go**

Already included in the code above.

**Step 5: Verify tests pass**

```bash
go test ./pkg/tui/... -v
```

Expected: All tests pass

**Step 6: Commit**

```bash
git add pkg/tui/update_key.go pkg/tui/update.go
git commit -m "refactor(tui): Extract keyboard handling to update_key.go

Moves all keyboard input functionality to dedicated file.
- handleKeyMsg() -> update_key.go
- Implements per-key handlers: handleCtrlC, handleEnter, handleEscape, etc.
- Adds helpers: handleMultiLineToggle, handleSubmitInput, handleQuitCommand
- Adds helpers: handleArrowKeys, handlePageKeys, handleShiftArrows
- Adds helpers: handleCtrlR, handleCtrlL, handleDefaultKey
- Reduces update.go from ~280 to ~50 lines (just dispatcher)

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 11: Phase 5 - Final Cleanup

**Files:**
- Modify: `pkg/tui/update.go` (simplify default case)
- Verify all files

**Step 1: Review and simplify handleDefaultMsg in update.go**

The default case should remain simple. Current implementation:

```go
default:
	// Update text input component with remaining messages (unless in search mode)
	if !m.searchState.active {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
	return m, nil
```

This can stay as-is or be extracted to a function if desired.

**Step 2: Add package documentation to each file**

Add to the top of each `update_*.go` file:

```go
// Package tui provides the terminal user interface for Gollum.
//
// This file contains [specific functionality] handlers.
```

**Step 3: Verify all files compile**

```bash
go build ./pkg/tui/...
```

Expected: SUCCESS

**Step 4: Run all tests**

```bash
go test ./pkg/tui/... -v
```

Expected: All tests pass

**Step 5: Check file sizes**

```bash
wc -l pkg/tui/update*.go
```

Expected:
- `update.go`: ~100 lines
- `update_key.go`: ~300-400 lines
- Others: ~50-150 lines each

**Step 6: Manual smoke test (optional but recommended)**

Run the TUI and verify:
- Keyboard input works
- Mouse scrolling works
- Search mode works (Ctrl+R)
- Multi-line input works (Alt+Enter)
- Agent execution works
- Export works

**Step 7: Final commit**

```bash
git add pkg/tui/
git commit -m "refactor(tui): Complete update handler refactoring

Final cleanup and documentation.
- Adds package documentation to all update_*.go files
- Verifies all files are under target line counts
- update.go reduced from 670 to ~100 lines
- All tests passing
- No behavioral changes

Summary of changes:
- Created 9 new handler files organized by event type
- Main Update() function now acts as clean dispatcher
- Each handler file has single, clear responsibility
- Improved maintainability and testability

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Success Criteria Verification

After completing all tasks:

1. ✅ All existing tests pass
2. ✅ TUI functions identically to before (manual smoke test)
3. ✅ Each file is under ~200 lines (except `update_key.go` at ~300-400)
4. ✅ `update.go` reduced to ~100 lines (dispatcher + helpers)
5. ✅ No circular dependencies between new files
6. ✅ No behavioral changes detected

---

## Notes for Implementation

- **Pure functions:** All handlers receive `Model` by value and return updated `Model`
- **No exports:** All handler functions remain unexported (lowercase)
- **Error handling:** No changes to error handling patterns
- **Constants:** Consider moving constants to their respective handler files for better locality
- **Git commits:** Commit after each successful file extraction to enable easy rollback

---

## Related Documentation

- Design: `docs/plans/2026-02-09-tui-update-refactor-design.md`
- Original file: `pkg/tui/update.go` (before refactoring)
- Bubbletea docs: https://github.com/charmbracelet/bubbletea
