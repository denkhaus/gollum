// Package channel provides the channel abstraction layer for Gollum.
package channel

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/denkhaus/gollum/pkg/command"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
)

// channelFacadeImpl implements ChannelFacade
type channelFacadeImpl struct {
	mu             sync.RWMutex
	channels       map[uuid.UUID]Channel
	commandManager command.Manager
	registry       registry.AgentRegistry
	agentFactory   shared.AgentFactory
	sessionManager session.SessionManager
	logger         logger.LoggerService
	providers      map[ChannelIdentifier]ChannelFactory
}

// Ensure channelFacadeImpl implements ChannelFacade at compile time
var _ ChannelFacade = (*channelFacadeImpl)(nil)

// Ensure channelFacadeImpl implements shared.LogForwarder at compile time
var _ shared.LogForwarder = (*channelFacadeImpl)(nil)

// NewChannelFacade creates a new channel facade service
func NewChannelFacade(injector do.Injector) (ChannelFacade, error) {
	cm := do.MustInvoke[command.ManagerService](injector)
	reg := do.MustInvoke[registry.AgentRegistry](injector)
	af := do.MustInvoke[shared.AgentFactory](injector)
	sm := do.MustInvoke[session.SessionManager](injector)
	log := do.MustInvoke[logger.LoggerService](injector)

	return &channelFacadeImpl{
		commandManager: cm,
		registry:       reg,
		agentFactory:   af,
		sessionManager: sm,
		channels:       make(map[uuid.UUID]Channel),
		logger:         log,
	}, nil
}

// DiscoverProviders scans the DI container for channel providers.
// Looks for named providers matching the "channel_*" pattern.
func (f *channelFacadeImpl) DiscoverProviders(injector do.Injector) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.providers = make(map[ChannelIdentifier]ChannelFactory)

	// NOTE: samber/do/v2 doesn't expose ListServices directly
	// We need to iterate known channel names or use reflection
	// For now, implement a simple approach with known channels
	// This can be extended with reflection if needed

	knownChannels := []string{"tui", "acp"} // Can be extended
	foundAny := false

	for _, name := range knownChannels {
		providerName := "channel_" + name
		factory, err := do.InvokeNamed[ChannelFactory](injector, providerName)
		if err != nil {
			// Provider not registered, skip
			continue
		}

		identifier := ChannelIdentifier(name)
		f.providers[identifier] = factory
		f.logger.Infof("Discovered channel: %s", identifier)
		foundAny = true
	}

	if !foundAny {
		f.logger.Warn("No channel providers discovered - channels may not be available")
	}

	return nil
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

// UnregisterChannel removes a channel from the registry
func (p *channelFacadeImpl) UnregisterChannel(channelID uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.channels, channelID)
	return nil
}

// DisplayMessage sends a message to the specific channel by ID
func (p *channelFacadeImpl) DisplayMessage(msg Message) {
	p.mu.RLock()
	channel, exists := p.channels[msg.ChannelID]
	p.mu.RUnlock()

	if !exists {
		p.logger.Warn("channel not found", zap.String("channel_id", msg.ChannelID.String()))
		return
	}

	channel.OnMessage(msg)
}

// SubmitInput handles user input from any channel
// channelID is used to assign each agent message to a channel
// sessionID is used to get or create a session for this interaction
func (p *channelFacadeImpl) SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (InputResult, error) {
	// First check if it's a slash command
	handled, response, err := p.commandManager.Execute(ctx, sessionID, input)
	if handled {
		return InputResult{
			Handled:   true,
			IsCommand: true,
			Response:  response,
			Error:     err,
		}, nil
	}

	// Get or create session for this interaction
	session, err := p.sessionManager.GetOrCreateSession(sessionID, channelID)
	if err != nil {
		return InputResult{}, fmt.Errorf("failed to get/create session: %w", err)
	}

	// Get or create supervisor for this session (lazy, thread-safe)
	supervisor, err := session.GetOrCreateSupervisor(p.agentFactory)
	if err != nil {
		return InputResult{}, fmt.Errorf("failed to get/create supervisor: %w", err)
	}

	// Execute supervisor agent with session context
	resp, err := supervisor.Execute(session.Context, gollem.Text(input))
	if err != nil {
		return InputResult{
			Handled: true,
			Error:   err,
		}, nil
	}

	// Extract response
	var content string
	if resp != nil && len(resp.Texts) > 0 {
		content = strings.Join(resp.Texts, "\n")
	}

	return InputResult{
		Handled:  true,
		Response: content,
	}, nil
}

// CancelInput cancels an in-flight input for the given session
func (p *channelFacadeImpl) CancelInput(sessionID string) error {
	session, ok := p.sessionManager.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Cancel the session context
	session.CancelFunc()

	return nil
}

// NotifyAgentLifecycle broadcasts agent lifecycle event
// TODO: this method must be session and channel aware and should not broadcast to all channels
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

// ForwardLog implements shared.LogForwarder for channel-based log routing.
func (p *channelFacadeImpl) ForwardLog(entry shared.LogEntry) {
	p.mu.RLock()
	targetChannel, exists := p.channels[entry.ChannelID]
	p.mu.RUnlock()

	if !exists {
		p.logger.Warn("channel not found for log entry",
			zap.String("channel_id", entry.ChannelID.String()),
			zap.String("session_id", entry.SessionID),
		)
		return
	}

	// Forward to specific channel only
	targetChannel.OnLog(entry)
}
