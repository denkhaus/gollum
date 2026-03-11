# TUI Update Handler Refactoring Design

**Date:** 2026-02-09
**Status:** Design
**Target:** `pkg/tui/update.go` (~670 lines → ~100 lines)

## Overview

Refactor the monolithic `update.go` file into multiple focused files organized by event type. The main `Update()` function will become a thin dispatcher that routes messages to specialized handler functions across multiple files.

## Goals

1. **Reduce file size** - Break down 670-line `update.go` into manageable pieces
2. **Improve maintainability** - Each file has a single, clear responsibility
3. **Enable testing** - Smaller, focused functions are easier to unit test
4. **Follow Bubbletea patterns** - Organize by message type (event type)
5. **Preserve functionality** - No behavioral changes; pure refactoring

## Organization Principle: By Event Type

Files are organized by the type of Bubbletea message they handle:

| Message Type | File | Purpose |
|--------------|------|---------|
| `tea.KeyMsg` | `update_key.go` | All keyboard input |
| `tea.MouseMsg` | `update_mouse.go` | Mouse events + debouncing |
| `tea.WindowSizeMsg` | `update_window.go` | Terminal resize |
| `tickMsg`, `logTickMsg` | `update_tick.go` | Timer-based updates |
| `agentCompleteMsg`, `newMessageMsg` | `update_agent.go` | Agent execution & messages |
| `exportMsg` | `update_export.go` | Conversation export |
| Mouse debounce | `update_mouse.go` | `mouseDebounceMsg` handling |
| Search subsystem | `update_search.go` | History search mode |
| Multi-line subsystem | `update_multiline.go` | Multi-line input mode |
| History navigation | `update_history.go` | Input history |

## File Structure

### Core Files (Existing)

**`update.go`** - Main dispatcher (simplified)
- `Update(msg tea.Msg)` - Routes to appropriate handler
- `handleDefaultMsg(msg)` - Fallback for textInput updates

**`model.go`** - Model struct (unchanged)
- Contains all state including `searchState`, `mouseDebounceTag`, etc.

### New Handler Files

#### `update_key.go` - Keyboard Input (~300-400 lines)

```go
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd)

// Individual key handlers (per key type)
func (m Model) handleCtrlC() (Model, tea.Cmd)
func (m Model) handleEnter(msg tea.KeyMsg) (Model, tea.Cmd)
func (m Model) handleEscape() (Model, tea.Cmd)
func (m Model) handleMultiLineToggle(msg tea.KeyMsg) (Model, tea.Cmd)
func (m Model) handleArrowKeys(keyType tea.KeyType) (Model, tea.Cmd)
func (m Model) handlePageKeys(msg tea.KeyMsg) (Model, tea.Cmd)
func (m Model) handleShiftArrows(keyType tea.KeyType) (Model, tea.Cmd)
func (m Model) handleCtrlL() (Model, tea.Cmd)
func (m Model) handleCtrlR() (Model, tea.Cmd)
func (m Model) handleDefaultKey(msg tea.KeyMsg) (Model, tea.Cmd)

// Input submission helpers
func (m Model) handleSubmitInput(input string) (Model, tea.Cmd)
func (m Model) handleSlashCommand(input string) (Model, tea.Cmd)
func (m Model) handleQuitCommand(input string) (Model, tea.Cmd)
```

#### `update_search.go` - History Search Subsystem (~150 lines)

```go
func (m Model) handleSearchKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd)

// Search-specific handlers
func (m Model) handleSearchEscape() (Model, tea.Cmd)
func (m Model) handleSearchEnter() (Model, tea.Cmd)
func (m Model) handleSearchNavigation(keyType tea.KeyType) (Model, tea.Cmd)
func (m Model) handleSearchQueryUpdate(query string) (Model, tea.Cmd)
func (m Model) handleSearchBackspace() (Model, tea.Cmd)

// Search helpers
func (m Model) nextSearchResult() Model
func (m Model) prevSearchResult() Model
func (m Model) exitSearch() Model
func (m Model) updateSearchResults(query string) Model
func (m Model) searchHistory(query string) []int
```

#### `update_agent.go` - Agent Execution (~150 lines)

```go
func (m Model) handleAgentCompleteMsg(msg agentCompleteMsg) (Model, tea.Cmd)
func (m Model) handleNewMessageMsg(msg newMessageMsg) (Model, tea.Cmd)

// Agent state helpers
func (m Model) handleAgentError(err error) Model
func (m Model) handleAgentResponse(texts []string) Model
func (m Model) restorePreservedInput() Model

// Agent execution preparation
func (m Model) prepareAgentExecution(input string) (Model, context.CancelFunc)
func (m Model) createAgentCommand(ctx context.Context, input string, cancel context.CancelFunc) tea.Cmd

// Validation helpers
func (m Model) shouldAddResponseTexts() bool
```

#### `update_mouse.go` - Mouse Events (~100 lines)

```go
func (m Model) handleMouseMsg(msg tea.MouseMsg) (Model, tea.Cmd)
func (m Model) handleMouseDebounceMsg(msg mouseDebounceMsg) (Model, tea.Cmd)

// Scrolling helpers
func (m Model) scrollViewport(viewport string, direction int) Model
func (m Model) scrollMainViewport(direction int) Model
func (m Model) scrollLogViewport(direction int) Model

// Debounce helpers
func (m Model) incrementDebounceTag() Model
func (m Model) createDebounceCommand(direction int, viewport string) tea.Cmd
```

#### `update_window.go` - Window Resize (~50 lines)

```go
func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (Model, tea.Cmd)

func (m Model) calculateViewportDimensions() (mainHeight int, logHeight int, reserved int)
```

#### `update_tick.go` - Timer Updates (~80 lines)

```go
func (m Model) handleTickMsg(msg tickMsg) (Model, tea.Cmd)
func (m Model) handleLogTickMsg(msg logTickMsg) (Model, tea.Cmd)

func (m Model) fetchNewLogEntries() Model
```

#### `update_export.go` - Conversation Export (~100 lines)

```go
func (m Model) handleExport() (Model, tea.Cmd)

func generateExportFilename() string
func buildExportContent(messages []Message) string
func writeExportFile(filename, content string) error
```

#### `update_multiline.go` - Multi-line Input (~50 lines)

```go
func (m Model) submitMultiLineInput() (Model, tea.Cmd)
func (m Model) exitMultiLineMode() (Model, tea.Cmd)

func (m Model) appendToMultiLineBuffer(input string) Model
```

#### `update_history.go` - Input History (~60 lines)

```go
func (m Model) handleHistoryNavigation(keyType tea.KeyType) (Model, tea.Cmd)

func (m *Model) addToHistory(input string)
func (m Model) navigateHistory(direction int) Model
```

## Main Update Dispatcher (Simplified)

The `Update()` function in `update.go` becomes a clean router:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKeyMsg(msg)
    case tea.MouseMsg:
        return m.handleMouseMsg(msg)
    case tea.WindowSizeMsg:
        return m.handleWindowSizeMsg(msg)
    case tickMsg:
        return m.handleTickMsg(msg)
    case logTickMsg:
        return m.handleLogTickMsg(msg)
    case agentCompleteMsg:
        return m.handleAgentCompleteMsg(msg)
    case newMessageMsg:
        return m.handleNewMessageMsg(msg)
    case exportMsg:
        return m.handleExport(msg)
    case mouseDebounceMsg:
        return m.handleMouseDebounceMsg(msg)
    default:
        return m.handleDefaultMsg(msg)
    }
}
```

## Implementation Strategy

### Phase 1: Setup
1. Create all new `.go` files with stub functions
2. Ensure package compiles

### Phase 2: Simple Handlers (Low Risk)
Move isolated handlers first:
1. `update_export.go` - Single function, no dependencies
2. `update_multiline.go` - 2 functions, isolated
3. `update_history.go` - 2 functions, isolated

### Phase 3: Message Type Handlers
Extract from main `Update()` switch:
1. `update_window.go` - WindowSizeMsg case
2. `update_tick.go` - tickMsg, logTickMsg cases
3. `update_mouse.go` - MouseMsg + mouseDebounceMsg cases
4. `update_agent.go` - agentCompleteMsg, newMessageMsg cases

### Phase 4: Keyboard Handler (Most Complex)
1. Extract `update_search.go` - Well-contained subsystem
2. Split `handleKeyMsg` into per-key functions in `update_key.go`

### Phase 5: Final Cleanup
1. Simplify main `Update()` to use `return m.handleX(msg)` pattern
2. Move any remaining helpers to appropriate files
3. Update package documentation

### Testing Approach
- Run existing tests after each file move
- Manual TUI smoke test after each phase
- Consider adding unit tests for individual handlers

### Git Strategy
- Commit after each successful file extraction
- Descriptive commit messages: `refactor(tui): Extract export handling to update_export.go`

## Design Decisions

### Why "By Event Type" Organization?
- Aligns with Bubbletea's message-based architecture
- Makes it easy to find where a message is handled
- Each file corresponds to a message type switch case

### Why Separate Files for Subsystems?
- Search mode has complex state and navigation logic
- Multi-line input has distinct entry/exit flows
- Agent execution handles multiple message types
- Better separation of concerns than co-locating

### Why Per-Key Type Granularity?
- Makes each handler very focused and testable
- Easy to add/remove key bindings
- Clearer code navigation than large switch statements

### Why Keep Mouse Debounce Together?
- `handleMouseMsg` creates debounce command
- `handleMouseDebounceMsg` executes the scroll
- Keeping them together maintains the debounce pattern's coherence

## Success Criteria

1. ✅ All existing tests pass
2. ✅ TUI functions identically to before (manual smoke test)
3. ✅ Each file is under ~200 lines (except `update_key.go` at ~300-400)
4. ✅ `update.go` reduced to ~100 lines (dispatcher + helpers)
5. ✅ No circular dependencies between new files
6. ✅ No behavioral changes detected

## Estimated Impact

| Metric | Before | After |
|--------|--------|-------|
| `update.go` lines | ~670 | ~100 |
| Total files in tui/ | 8 | 17 |
| Longest file | 670 | ~400 |
| Handler functions per file | 15+ | 1-10 |

## Notes

- All functions remain unexported (internal to `tui` package)
- No changes to error handling patterns
- Pure function semantics preserved (Model by value, return updated Model)
- Constants may move to their respective handler files for better locality
