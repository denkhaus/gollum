// Package channel provides the channel abstraction layer for Gollum.
package channel

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
)

// channelFacadeImpl implements ChannelFacade
type channelFacadeImpl struct {
	mu             sync.RWMutex
	channels       map[uuid.UUID]Channel
	commandManager CommandManager
	registry       registry.AgentRegistry
	agentFactory   shared.AgentFactory
	sessionManager shared.SessionManager
	logs           []LogEntry
	maxLogs        int
	logger         logger.LoggerService

	// Track active inputs per channel for cancellation
	activeInputs map[uuid.UUID]context.CancelFunc
}

// Ensure channelFacadeImpl implements ChannelFacade at compile time
var _ ChannelFacade = (*channelFacadeImpl)(nil)

// NewChannelFacade creates a new channel facade service
func NewChannelFacade(injector do.Injector) (ChannelFacade, error) {
	cm := do.MustInvoke[CommandManagerService](injector)
	reg := do.MustInvoke[registry.AgentRegistry](injector)
	af := do.MustInvoke[shared.AgentFactory](injector)
	sm := do.MustInvoke[shared.SessionManager](injector)
	cfg := do.MustInvoke[config.ConfigService](injector)
	log := do.MustInvoke[logger.LoggerService](injector)

	// Use existing LoggingConfig.SessionLogBufferSize for display log buffer
	maxLogs := cfg.GetLoggingConfig().SessionLogBufferSize

	return &channelFacadeImpl{
		commandManager: cm,
		registry:       reg,
		agentFactory:   af,
		sessionManager: sm,
		channels:       make(map[uuid.UUID]Channel),
		logs:           make([]LogEntry, 0, maxLogs),
		maxLogs:        maxLogs,
		logger:         log,
		activeInputs:   make(map[uuid.UUID]context.CancelFunc),
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
// channelID is used to assign each agent message to a channel
// sessionID is used to get or create a session for this interaction
func (p *channelFacadeImpl) SubmitInput(ctx context.Context, channelID uuid.UUID, sessionID string, input string) (InputResult, error) {
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

	// Get or create session for this interaction
	_, err = p.sessionManager.GetOrCreateSession(sessionID, channelID)
	if err != nil {
		return InputResult{}, fmt.Errorf("failed to get/create session: %w", err)
	}

	// Create cancellable context for this input
	inputCtx, cancelFunc := context.WithCancel(ctx)

	// Track this active input for cancellation
	p.mu.Lock()
	p.activeInputs[channelID] = cancelFunc
	p.mu.Unlock()

	// Clean up tracking when done
	defer func() {
		p.mu.Lock()
		delete(p.activeInputs, channelID)
		p.mu.Unlock()
		cancelFunc()
	}()

	// Get or create supervisor for this session
	supervisor, _, err := p.agentFactory.CreateSupervisorAgent(
		ctx,
		shared.WithSessionID(sessionID),
		shared.WithChannelID(channelID),
	)
	if err != nil {
		return InputResult{}, fmt.Errorf("failed to create supervisor: %w", err)
	}

	// Execute supervisor agent
	resp, err := supervisor.Execute(inputCtx, gollem.Text(input))
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

// CancelInput cancels an in-flight input for the given channel
func (p *channelFacadeImpl) CancelInput(channelID uuid.UUID) error {
	p.mu.Lock()
	cancelFunc, exists := p.activeInputs[channelID]
	p.mu.Unlock()

	if !exists {
		return fmt.Errorf("no active input for channel %s", channelID)
	}

	// Cancel the input context
	cancelFunc()

	return nil
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
