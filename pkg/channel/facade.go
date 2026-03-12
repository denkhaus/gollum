// Package channel provides the channel abstraction layer for Gollum.
package channel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/registry"
)

// ChannelFacadeService defines the channel facade service interface for DI
type ChannelFacadeService interface {
	ChannelFacade
}

// channelFacadeImpl implements ChannelFacade
type channelFacadeImpl struct {
	mu             sync.RWMutex
	channels       map[string]Channel
	commandManager CommandManager
	registry       registry.AgentRegistry
	logs           []LogEntry
	maxLogs        int
}

// Ensure channelFacadeImpl implements ChannelFacade at compile time
var _ ChannelFacade = (*channelFacadeImpl)(nil)

// NewChannelFacade creates a new channel facade service
func NewChannelFacade(injector do.Injector) (ChannelFacadeService, error) {
	cm := do.MustInvoke[CommandManager](injector)
	reg := do.MustInvoke[registry.AgentRegistry](injector)
	cfg := do.MustInvoke[config.ConfigService](injector)

	// Use existing LoggingConfig.SessionLogBufferSize for display log buffer
	maxLogs := cfg.GetLoggingConfig().SessionLogBufferSize

	return &channelFacadeImpl{
		commandManager: cm,
		registry:       reg,
		channels:       make(map[string]Channel),
		logs:           make([]LogEntry, 0, maxLogs),
		maxLogs:        maxLogs,
	}, nil
}

// RegisterChannel adds a channel to receive events
func (p *channelFacadeImpl) RegisterChannel(channel Channel) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.channels[channel.ID()]; exists {
		return fmt.Errorf("channel %s already registered", channel.ID())
	}

	p.channels[channel.ID()] = channel
	return nil
}

// UnregisterChannel removes a channel
func (p *channelFacadeImpl) UnregisterChannel(channelID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.channels, channelID)
	return nil
}

// DisplayMessage sends a message to all registered channels
func (p *channelFacadeImpl) DisplayMessage(msg Message) {
	p.mu.RLock()
	channels := make([]Channel, 0, len(p.channels))
	for _, c := range p.channels {
		channels = append(channels, c)
	}
	p.mu.RUnlock()

	for _, channel := range channels {
		channel.OnMessage(msg)
	}
}

// DisplayLog sends a log entry to all registered channels
func (p *channelFacadeImpl) DisplayLog(entry LogEntry) {
	p.mu.Lock()

	// Store log entry with ring buffer behavior
	p.logs = append(p.logs, entry)
	if len(p.logs) > p.maxLogs {
		p.logs = p.logs[1:]
	}

	channels := make([]Channel, 0, len(p.channels))
	for _, c := range p.channels {
		channels = append(channels, c)
	}
	p.mu.Unlock()

	// Send to all channels
	for _, channel := range channels {
		channel.OnLog(entry)
	}
}

// SubmitInput handles user input from any channel
func (p *channelFacadeImpl) SubmitInput(ctx context.Context, input string) (InputResult, error) {
	// First check if it's a slash command
	handled, response, err := p.commandManager.Execute(ctx, input)
	if handled {
		return InputResult{
			Handled:   true,
			IsCommand: true,
			Response:  response,
			Error:     err,
		}, nil
	}

	// Not a command - route through agent executor
	// Note: GetAgentBySenderID doesn't exist yet in the registry interface
	// For now, return an error. This will be implemented when the registry is updated.
	return InputResult{}, fmt.Errorf("agent not found: GetAgentBySenderID not implemented in registry yet")
}

// GetLogs returns recent log entries for channels to poll
func (p *channelFacadeImpl) GetLogs(since time.Time, limit int) []LogEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []LogEntry
	for _, entry := range p.logs {
		if entry.Timestamp.After(since) {
			result = append(result, entry)
			if len(result) >= limit {
				break
			}
		}
	}

	return result
}

// NotifyAgentLifecycle broadcasts agent lifecycle event
func (p *channelFacadeImpl) NotifyAgentLifecycle(agentID uuid.UUID, role string, added bool) {
	event := AgentLifecycleEvent{
		AgentID: agentID,
		Role:    role,
		Added:   added,
	}

	p.mu.RLock()
	channels := make([]Channel, 0, len(p.channels))
	for _, c := range p.channels {
		channels = append(channels, c)
	}
	p.mu.RUnlock()

	for _, channel := range channels {
		channel.OnAgentLifecycle(event)
	}
}
