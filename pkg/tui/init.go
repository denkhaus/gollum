package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/samber/do/v2"
)

// NewTUIProvider provides a new TUI program factory for dependency injection.
//
// This provider function follows the samber/do pattern used throughout
// the Gollum codebase. It returns a function that creates new TUI programs.
//
// Usage in DI:
//
//	programFactory := do.MustInvoke[func() *tea.Program](injector)
//	p := programFactory()
//	if _, err := p.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// Note: In Phase 1, this creates a basic program. Future phases will
// integrate with the AgentMessenger for rich output display.
func NewTUIProvider(_ do.Injector) func() *tea.Program {
	return NewProgram
}

// This file serves as the initialization and DI provider for the TUI package.
// The main model logic is in model.go, update logic in update.go, and view
// rendering in view.go following Bubbletea's clear separation of concerns.
