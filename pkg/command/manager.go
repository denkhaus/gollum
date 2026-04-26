// Package command provides command handling functionality for Gollum.
package command

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/denkhaus/gollum/pkg/session"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
)

// ManagerService defines the command manager service interface for DI
type ManagerService interface {
	Manager
}

// managerImpl implements Manager
type managerImpl struct {
	mu             sync.RWMutex
	commands       map[string]Command
	sessionManager session.SessionManager
}

// Ensure managerImpl implements Manager at compile time
var _ Manager = (*managerImpl)(nil)

// NewManager creates a new command manager service
func NewManager(injector do.Injector) (ManagerService, error) {
	sessionManager := do.MustInvoke[session.SessionManager](injector)
	return &managerImpl{
		commands:       make(map[string]Command),
		sessionManager: sessionManager,
	}, nil
}

// Register adds a new slash command
func (p *managerImpl) Register(cmd Command) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.commands[cmd.Name]; exists {
		return fmt.Errorf("command %q already registered", cmd.Name)
	}

	p.commands[cmd.Name] = cmd
	return nil
}

// Unregister removes a command
func (p *managerImpl) Unregister(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.commands, name)
	return nil
}

// Execute parses input and executes command if it begins with "/"
func (p *managerImpl) Execute(ctx context.Context, sessionID uuid.UUID, input string) (bool, string, error) {
	if input == "" {
		return false, "", nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check if it starts with "/"
	if !strings.HasPrefix(input, "/") {
		return false, "", nil
	}

	// Extract command name and args
	parts := strings.SplitN(input, " ", 2)
	if len(parts) == 0 {
		return false, "", nil
	}

	cmdName := parts[0]
	cmd, exists := p.commands[cmdName]
	if !exists {
		return false, "", nil
	}

	// Get session
	session, ok := p.sessionManager.GetSession(sessionID)
	if !ok {
		return false, "", fmt.Errorf("session not found: %s", sessionID.String())
	}

	// Get args if present
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	// Execute command with session
	response, err := cmd.Handler(ctx, session, args)
	return true, response, err
}

// List returns all available commands
func (p *managerImpl) List() []Command {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]Command, 0, len(p.commands))
	for _, cmd := range p.commands {
		result = append(result, cmd)
	}
	return result
}

// IsCommand checks if input starts with "/"
func (p *managerImpl) IsCommand(input string) bool {
	if input == "" {
		return false
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !strings.HasPrefix(input, "/") {
		return false
	}

	parts := strings.SplitN(input, " ", 2)
	if len(parts) == 0 {
		return false
	}

	_, exists := p.commands[parts[0]]
	return exists
}
