package tui

// This file handles history search mode for the TUI.

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
		return m.handleSearchQueryUpdate(msg)

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
func (m Model) handleSearchQueryUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	input := string(msg.Runes)
	m.textInput.SetValue(input)
	m.searchState.query = input

	// Update search results
	if input == "" {
		// Show all history when query is empty
		m.searchState.results = make([]int, 0, len(m.inputHistory))
		for i := range m.inputHistory {
			m.searchState.results = append(m.searchState.results, i)
		}
	} else {
		m.searchState.results = m.searchHistory(input)
	}

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
	if len(current) > 0 {
		m.textInput.SetValue(current[:len(current)-1])
		m.searchState.query = m.textInput.Value()

		// Update search results
		if m.searchState.query == "" {
			m.searchState.results = make([]int, 0, len(m.inputHistory))
			for i := range m.inputHistory {
				m.searchState.results = append(m.searchState.results, i)
			}
		} else {
			m.searchState.results = m.searchHistory(m.searchState.query)
		}

		// Update display
		if len(m.searchState.results) > 0 {
			m.searchState.matchedIdx = 0
			m.textInput.SetValue(m.inputHistory[m.searchState.results[0]])
		}
	}
	return m, nil
}

// nextSearchResult navigates to the next search result.
func (m Model) nextSearchResult() Model {
	if len(m.searchState.results) == 0 {
		return m
	}

	m.searchState.matchedIdx = (m.searchState.matchedIdx + 1) % len(m.searchState.results)
	m.textInput.SetValue(m.inputHistory[m.searchState.results[m.searchState.matchedIdx]])
	m.textInput.CursorEnd()
	return m
}

// prevSearchResult navigates to the previous search result.
func (m Model) prevSearchResult() Model {
	if len(m.searchState.results) == 0 {
		return m
	}

	m.searchState.matchedIdx = (m.searchState.matchedIdx - 1 + len(m.searchState.results)) % len(m.searchState.results)
	m.textInput.SetValue(m.inputHistory[m.searchState.results[m.searchState.matchedIdx]])
	m.textInput.CursorEnd()
	return m
}

// exitSearch exits history search mode.
func (m Model) exitSearch() Model {
	m.searchState = searchState{}
	return m
}

// updateSearchResults updates the search results based on the query.
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

// searchHistory performs a case-insensitive search through history.
// Returns indices of matching entries.
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
