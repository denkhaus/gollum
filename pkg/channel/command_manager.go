// Package channel provides the channel abstraction layer for Gollum.
package channel

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/denkhaus/gollum/pkg/session"
	"github.com/samber/do/v2"
)

// CommandManagerService defines the command manager service interface for DI
type CommandManagerService interface {
	CommandManager
}

// commandManagerImpl implements CommandManager
type commandManagerImpl struct {
	mu             sync.RWMutex
	commands       map[string]Command
	sessionManager session.SessionManager
}

// Ensure commandManagerImpl implements CommandManager at compile time
var _ CommandManager = (*commandManagerImpl)(nil)

// NewCommandManager creates a new command manager service
func NewCommandManager(injector do.Injector) (CommandManagerService, error) {
	sessionManager := do.MustInvoke[session.SessionManager](injector)
	return &commandManagerImpl{
		commands:       make(map[string]Command),
		sessionManager: sessionManager,
	}, nil
}

// Register adds a new slash command
func (p *commandManagerImpl) Register(cmd Command) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.commands[cmd.Name]; exists {
		return fmt.Errorf("command %q already registered", cmd.Name)
	}

	p.commands[cmd.Name] = cmd
	return nil
}

// Unregister removes a command
func (p *commandManagerImpl) Unregister(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.commands, name)
	return nil
}

// Execute parses input and executes command if it begins with "/"
func (p *commandManagerImpl) Execute(ctx context.Context, sessionID string, input string) (bool, string, error) {
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
		return false, "", fmt.Errorf("session not found: %s", sessionID)
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
func (p *commandManagerImpl) List() []Command {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]Command, 0, len(p.commands))
	for _, cmd := range p.commands {
		result = append(result, cmd)
	}
	return result
}

// IsCommand checks if input starts with "/"
func (p *commandManagerImpl) IsCommand(input string) bool {
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
