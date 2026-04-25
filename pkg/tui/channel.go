package tui

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// Identifier is the unique channel identifier for registration.
const Identifier = channel.ChannelIdentifier("tui")

// TUIChannel implements the channel.Channel interface for the TUI.
// It forwards messages to the TUI model via the message channel for display.
type TUIChannel struct {
	id          uuid.UUID
	messageChan chan<- channel.Message
	agentID     uuid.UUID
	agentRole   string
	logger      logger.LoggerService
	renderer    markdown.Renderer
	facade      channel.ChannelFacade
}

// NewTUIChannel creates a new TUIChannel instance with optional configuration.
// Options can be provided to configure the channel's dependencies.
func NewTUIChannel(opts ...TUIOption) *TUIChannel {
	ch := &TUIChannel{
		id:        uuid.New(),
		agentID:   uuid.Nil, // Will be set by SetAgentInfo
		agentRole: "assistant",
		// messageChan, logger, renderer default to nil
	}

	for _, opt := range opts {
		_ = opt.Apply(ch) // TUIOption.Apply handles the type assertion
	}

	return ch
}

// SetAgentInfo sets the agent ID and role for messages sent via this channel.
func (c *TUIChannel) SetAgentInfo(id uuid.UUID, role string) {
	c.agentID = id
	c.agentRole = role
}

// ID returns the unique identifier for this channel.
func (c *TUIChannel) ID() uuid.UUID {
	return c.id
}

// OnMessage handles incoming messages from the channel.
// Messages are sent to the TUI model's message channel for display.
func (c *TUIChannel) OnMessage(msg channel.Message) {
	if c.messageChan == nil {
		log.Printf("[TUIChannel] No message channel configured, message not displayed: %+v", msg)
		return
	}

	// Send message to TUI model
	// Use non-blocking send to avoid blocking if channel is full
	select {
	case c.messageChan <- msg:
		// Message sent successfully
	default:
		log.Printf("[TUIChannel] Message channel full, message dropped: %+v", msg)
	}
}

// OnLog handles log entries from the channel.
// Log entries are converted to system messages and sent to the TUI.
func (c *TUIChannel) OnLog(entry shared.LogEntry) {
	if c.messageChan == nil {
		log.Printf("[TUIChannel] No message channel configured, log entry not displayed: %+v", entry)
		return
	}

	// Convert log entry to a system message
	msg := channel.Message{
		Type:      channel.MessageTypeSystemInfo,
		Content:   formatLogEntry(entry),
		Timestamp: entry.Timestamp,
		AgentID:   c.agentID,
		AgentRole: c.agentRole,
	}

	// Send to TUI model
	select {
	case c.messageChan <- msg:
	default:
		log.Printf("[TUIChannel] Message channel full, log entry dropped: %+v", entry)
	}
}

// OnAgentLifecycle handles agent lifecycle events.
// Lifecycle events are converted to system messages and sent to the TUI.
func (c *TUIChannel) OnAgentLifecycle(event channel.AgentLifecycleEvent) {
	if c.messageChan == nil {
		log.Printf("[TUIChannel] No message channel configured, lifecycle event not displayed: %+v", event)
		return
	}

	// Convert lifecycle event to a system message
	msg := channel.Message{
		Type:      channel.MessageTypeSystemInfo,
		Content:   formatLifecycleEvent(event),
		Timestamp: time.Now(),
		AgentID:   event.AgentID,
		AgentRole: event.Role,
	}

	// Send to TUI model
	select {
	case c.messageChan <- msg:
	default:
		log.Printf("[TUIChannel] Message channel full, lifecycle event dropped: %+v", event)
	}
}

// formatLogEntry formats a log entry for display as a message.
func formatLogEntry(entry shared.LogEntry) string {
	// Format: [LEVEL] message
	prefix := "[" + entry.Level + "] "

	// Extract component from fields if present
	component := ""
	if comp, ok := entry.Fields["component"].(string); ok && comp != "" {
		component = comp + ": "
	}

	return prefix + component + entry.Message
}

// formatLifecycleEvent formats an agent lifecycle event for display as a message.
func formatLifecycleEvent(event channel.AgentLifecycleEvent) string {
	// Format: Agent lifecycle: action - agent_id
	action := "added"
	if !event.Added {
		action = "removed"
	}

	return "Agent " + action + ": " + event.AgentID.String() + " (" + event.Role + ")"
}

// GetMessageChan returns the message channel (for testing).
func (c *TUIChannel) GetMessageChan() chan<- channel.Message {
	return c.messageChan
}

// GetLogger returns the logger service (for testing).
func (c *TUIChannel) GetLogger() logger.LoggerService {
	return c.logger
}

// GetRenderer returns the markdown renderer (for testing).
func (c *TUIChannel) GetRenderer() markdown.Renderer {
	return c.renderer
}

// Start begins the TUI channel's lifecycle by creating and running the Bubbletea program.
func (c *TUIChannel) Start(ctx context.Context) error {
	if c.facade == nil {
		log.Print("TUIChannel: no facade configured, cannot start TUI")
		return errors.New("TUIChannel: no facade configured")
	}

	// Create agent executor adapter
	executor := &agentExecutorAdapter{
		facade:    c.facade,
		channelID: c.id,
	}

	// Create options for the TUI program
	opts := []func(*Model){
		WithMessageChannel(),
	}

	// Add logger and renderer if configured
	if c.logger != nil {
		opts = append(opts, WithLoggerService(c.logger))
	}
	if c.renderer != nil {
		opts = append(opts, WithMarkdownRenderer(c.renderer))
	}

	// Create the TUI program
	program := NewProgramWithContext(ctx, executor, opts...)

	// Run the program (blocks until TUI exits)
	_, err := program.Run()
	return err
}

// agentExecutorAdapter adapts channel.ChannelFacade to tui.AgentExecutor
type agentExecutorAdapter struct {
	facade    channel.ChannelFacade
	channelID uuid.UUID
}

// Execute implements tui.AgentExecutor by delegating to the channel facade
func (a *agentExecutorAdapter) Execute(ctx context.Context, input string) (*gollem.ExecuteResponse, error) {
	// Use empty session ID for TUI (single session)
	result, err := a.facade.SubmitInput(ctx, a.channelID, "", input)
	if err != nil {
		return nil, err
	}

	// Return the response text from InputResult
	// If there's an error, return it wrapped in the response
	if result.Error != nil {
		return nil, result.Error
	}

	// Create ExecuteResponse with the response text
	return &gollem.ExecuteResponse{
		Texts: []string{result.Response},
	}, nil
}

// Compile-time check to ensure TUIChannel implements channel.Channel
var _ channel.Channel = (*TUIChannel)(nil)
