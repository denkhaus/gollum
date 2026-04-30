// Package channel provides the channel abstraction layer for Gollum.
package channel

import (
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
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
	inputHandler   InputHandler
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

	// Create InputHandler for session/supervisor business logic
	inputHandler := NewInputHandler(cm, sm, af, log)

	facade := &channelFacadeImpl{
		commandManager: cm,
		registry:       reg,
		agentFactory:   af,
		sessionManager: sm,
		channels:       make(map[uuid.UUID]Channel),
		logger:         log,
		inputHandler:   inputHandler,
	}

	// Register facade as log forwarder with logger service
	log.SetLogForwarder(facade)

	return facade, nil
}

// DiscoverProviders scans the DI container for channel providers.
// Automatically discovers all services matching the ProviderPrefix naming pattern.
func (f *channelFacadeImpl) DiscoverProviders(injector do.Injector) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.providers = make(map[ChannelIdentifier]ChannelFactory)

	// List all services in the injector and find channel providers
	services := injector.ListProvidedServices()
	foundAny := false

	for _, service := range services {
		// Look for services with the ProviderPrefix pattern
		if !strings.HasPrefix(service.Service, ProviderPrefix) {
			continue
		}

		// Extract channel name from the service name (e.g., "channel_tui" -> "tui")
		channelName := strings.TrimPrefix(service.Service, ProviderPrefix)
		if channelName == "" {
			continue // Skip if name is empty after prefix removal
		}

		// Try to invoke the service as a ChannelFactory
		factory, err := do.InvokeNamed[ChannelFactory](injector, service.Service)
		if err != nil {
			// Service exists but doesn't implement ChannelFactory, skip
			f.logger.Debugf("Service %s found but is not a ChannelFactory: %v", service.Service, err)
			continue
		}

		identifier := ChannelIdentifier(channelName)
		f.providers[identifier] = factory
		f.logger.Infof("Discovered channel: %s (from service: %s)", identifier, service.Service)
		foundAny = true
	}

	if !foundAny {
		f.logger.Warn("No channel providers discovered - channels may not be available")
	}

	return nil
}

// CreateChannel creates a channel instance by identifier with options.
func (f *channelFacadeImpl) CreateChannel(identifier ChannelIdentifier, opts ...ChannelOption) (Channel, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	factory, ok := f.providers[identifier]
	if !ok {
		// Provide helpful error listing available channels
		available := make([]string, 0, len(f.providers))
		for id := range f.providers {
			available = append(available, string(id))
		}
		return nil, fmt.Errorf("unknown channel identifier: %s (available: %v)", identifier, available)
	}

	ch, err := factory(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel %s: %w", identifier, err)
	}

	// Apply options with error handling
	for _, opt := range opts {
		if err := opt.Apply(ch); err != nil {
			return nil, fmt.Errorf("failed to apply option to channel %s: %w", identifier, err)
		}
	}

	return ch, nil
}

// CreateAndRegister creates and registers a channel in one step.
func (f *channelFacadeImpl) CreateAndRegister(identifier ChannelIdentifier, opts ...ChannelOption) (Channel, error) {
	ch, err := f.CreateChannel(identifier, opts...)
	if err != nil {
		return nil, err
	}

	if err := f.registerChannel(ch); err != nil {
		return nil, fmt.Errorf("failed to register channel %s: %w", identifier, err)
	}

	return ch, nil
}

// registerChannel adds a channel to receive events (internal, for testing)
func (f *channelFacadeImpl) registerChannel(channel Channel) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.channels[channel.ID()]; exists {
		return fmt.Errorf("channel %s already registered", channel.ID())
	}
	f.channels[channel.ID()] = channel
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
func (p *channelFacadeImpl) DisplayMessage(msg shared.Message) {
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
// sessionCtx contains routing information (SessionID, ChannelID, Cwd) for this interaction
// Delegates to InputHandler for business logic (command execution, session/supervisor management)
func (p *channelFacadeImpl) SubmitInput(session *shared.Session, input string) (*InputResult, error) {
	return p.inputHandler.HandleInput(session, input)
}

// CancelInput cancels an in-flight input for the given session
// Delegates to InputHandler for session cancellation logic
func (p *channelFacadeImpl) CancelInput(sessionID uuid.UUID) error {
	return p.inputHandler.CancelInput(sessionID)
}

// NotifyAgentLifecycle sends agent lifecycle event to specific channel
func (p *channelFacadeImpl) NotifyAgentLifecycle(agentID uuid.UUID, channelID uuid.UUID, sessionID uuid.UUID, role string, added bool) {
	event := AgentLifecycleEvent{
		AgentID:   agentID,
		Role:      role,
		Added:     added,
		SessionID: sessionID.String(),
		ChannelID: channelID,
	}

	p.mu.RLock()
	targetChannel, exists := p.channels[channelID]
	p.mu.RUnlock()

	if !exists {
		p.logger.Warn("channel not found for agent lifecycle event",
			zap.String("channel_id", channelID.String()),
			zap.String("session_id", sessionID.String()),
			zap.String("agent_id", agentID.String()),
		)
		return
	}

	targetChannel.OnAgentLifecycle(event)
}

// ForwardLog implements shared.LogForwarder for channel-based log routing.
func (p *channelFacadeImpl) ForwardLog(entry shared.LogEntry) {
	p.mu.RLock()
	targetChannel, exists := p.channels[entry.ChannelID]
	p.mu.RUnlock()

	if !exists {
		p.logger.Warn("channel not found for log entry",
			zap.String("channel_id", entry.ChannelID.String()),
			zap.String("context", entry.String()),
		)
		return
	}

	// Forward to specific channel only
	targetChannel.OnLog(entry)
}
