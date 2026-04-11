// Package command provides command handling functionality for Gollum.
package command

import (
	"context"

	"github.com/denkhaus/gollum/pkg/shared"
)

// CommandHandler is a function that executes a slash command in the context of a session
type CommandHandler func(ctx context.Context, session *shared.Session, args string) (string, error)

// Command represents a registered slash command
type Command struct {
	Name        string
	Description string
	Handler     CommandHandler
}

// Manager handles slash command registration and execution
type Manager interface {
	// Register adds a new slash command
	Register(cmd Command) error

	// Unregister removes a command
	Unregister(name string) error

	// Execute parses input and executes command if it starts with "/"
	Execute(ctx context.Context, sessionID string, input string) (handled bool, response string, err error)

	// List returns all available commands
	List() []Command

	// IsCommand checks if input starts with "/"
	IsCommand(input string) bool
}
