package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/samber/do/v2"
)

// This file serves as the initialization and DI provider for the TUI package.
// The main model logic is in model.go, update logic in update.go, and view
// rendering in view.go following Bubbletea's clear separation of concerns.
//
// Note: As of Phase 2, the TUI is created directly in service.go with
// the agent and context, so the DI provider is not used. The provider
// is kept for future DI integration if needed.

// NewTUIProvider provides a new TUI program factory for dependency injection.
//
// This provider function follows the samber/do pattern used throughout
// the Gollum codebase. Note that as of Phase 2, this is not actively used
// since the TUI needs to be created with a specific agent and context.
//
// Usage in DI (future use):
//
//	programFactory := do.MustInvoke[func(context.Context, AgentExecutor) *tea.Program](injector)
//	p := programFactory(ctx, agent)
//	if _, err := p.Run(); err != nil {
//	    log.Fatal(err)
//	}
func NewTUIProvider(_ do.Injector) func(context.Context, AgentExecutor) *tea.Program {
	return NewProgramWithContext
}
