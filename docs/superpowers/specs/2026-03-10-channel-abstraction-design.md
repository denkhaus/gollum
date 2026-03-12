# Channel Abstraction Layer Design

## Overview

Gollum currently has a TUI (Terminal User Interface) in `pkg/tui/` which needs to be abstracted away from agent logic and communication. This design creates a Channel facade that centralizes these concerns, enabling multiple channel implementations (web/telegram/TUI, etc.) through a plugin system.

## Goals

1. **Abstract the current TUI layer from the agent logic**
2. **Centralize slash commands** in one place (`/clear`, `/help`, `/quit`)
   - Each display handles its own prompt history (display-local history)
3. **Support logs via facade** - displays poll for logs
4. **Handle agent lifecycle events** (agent registration/removal)
5. **Clear separation of concerns**:
   - `pkg/channel/` = abstraction layer (types, interfaces, facade)
   - `pkg/tui/` = one implementation of the Channel interface
   - Each channel manages its own prompt history
   - Log storage (ring buffer with limits)
   - User input handling remains in channel

---

## Section 1: Core Types

**Location:** `pkg/channel/types.go`

```go
package channel

import (
	"time"

	"github.com/google/uuid"
)

// MessageType represents different types of messages
type MessageType int

const (
	MessageTypeUserChat MessageType = iota
	MessageTypeAgentChat    // Chat response from agent
	MessageTypeToolRequest  // Tool execution request
	MessageTypeToolResponse // Tool execution response
	MessageTypeThinking     // Agent thinking blocks
	MessageTypeSystemInfo   // System information messages
	MessageTypeError        // Error messages
)

// String returns a string representation of the MessageType
func (mt MessageType) String() string {
	switch mt {
	case MessageTypeUserChat:
		return "user_chat"
	case MessageTypeAgentChat:
		return "agent_chat"
	case MessageTypeToolRequest:
		return "tool_request"
	case MessageTypeToolResponse:
		return "tool_response"
	case MessageTypeThinking:
		return "thinking"
	case MessageTypeSystemInfo:
		return "system_info"
	case MessageTypeError:
		return "error"
	default:
		return "unknown"
	}
}

// Message represents a structured message data for displays
type Message struct {
	ID        uuid.UUID
	Type      MessageType
	AgentID   uuid.UUID
	AgentRole string
	Content   string
	Timestamp time.Time
	Metadata  map[string]any // tool_name, duration, collapsed, etc.
}

// LogEntry represents a log line for display polling
type LogEntry struct {
	Level     string
	Message   string
	Timestamp time.Time
	Fields    map[string]any
}

// InputResult represents the result of user input submission
type InputResult struct {
	Handled   bool   // true if command was executed
	IsCommand bool   // true if input was slash command
	Response  string // optional response (e.g., command help)
	Error     error  // optional error if command failed
}

// AgentLifecycleEvent represents agent registration/removal events
type AgentLifecycleEvent struct {
	AgentID uuid.UUID
	Role    string
	Added   bool
}
```

---

## Section 2: Interfaces

**Location:** `pkg/channel/interface.go`

```go
package channel

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Channel is the interface that all channel implementations must satisfy
type Channel interface {
	// ID returns a unique identifier for this channel
	ID() string

	// OnMessage is called when a new message should be displayed
	OnMessage(msg Message)

	// OnLog is called for log entries (channel can ignore if not applicable)
	OnLog(entry LogEntry)

	// OnAgentLifecycle is called when agent registration/removal events occur
	OnAgentLifecycle(event AgentLifecycleEvent)
}

// CommandHandler is a function that executes a slash command
type CommandHandler func(ctx context.Context, args string) (string, error)

// Command represents a registered slash command
type Command struct {
	Name        string
	Description string
	Handler     CommandHandler
}

// CommandManager handles slash command registration and execution
type CommandManager interface {
	// Register adds a new slash command
	Register(cmd Command) error

	// Unregister removes a command
	Unregister(name string) error

	// Execute parses input and executes command if it starts with "/"
	Execute(ctx context.Context, input string) (handled bool, response string, err error)

	// List returns all available commands
	List() []Command

	// IsCommand checks if input starts with "/"
	IsCommand(input string) bool
}

// ChannelFacade is the central coordinator for all channels
type ChannelFacade interface {
	// DisplayMessage sends a message to all registered channels
	DisplayMessage(msg Message)

	// DisplayLog sends a log entry to all registered channels
	DisplayLog(entry LogEntry)

	// SubmitInput handles user input from any channel
	SubmitInput(ctx context.Context, input string) (InputResult, error)

	// GetLogs returns recent log entries for channels to poll
	GetLogs(since time.Time, limit int) []LogEntry

	// RegisterChannel adds a channel to receive events
	RegisterChannel(channel Channel) error

	// UnregisterChannel removes a channel
	UnregisterChannel(channelID string) error

	// NotifyAgentLifecycle broadcasts agent lifecycle event
	NotifyAgentLifecycle(agentID uuid.UUID, role string, added bool)
}
```

---

## Section 3: Command Manager

**Location:** `pkg/channel/command_manager.go`

```go
package channel

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/samber/do/v2"
)

// CommandManagerService defines the command manager service interface for DI
type CommandManagerService interface {
	CommandManager
}

// commandManagerImpl implements CommandManager
type commandManagerImpl struct {
	mu       sync.RWMutex
	commands map[string]Command
}

// Ensure commandManagerImpl implements CommandManager at compile time
var _ CommandManager = (*commandManagerImpl)(nil)

// NewCommandManager creates a new command manager service
func NewCommandManager(_ do.Injector) (CommandManagerService, error) {
	return &commandManagerImpl{
		commands: make(map[string]Command),
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
func (p *commandManagerImpl) Execute(ctx context.Context, input string) (bool, string, error) {
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

	// Get args if present
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	// Execute command
	response, err := cmd.Handler(ctx, args)
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
```

---

## Section 4: Channel Facade

**Location:** `pkg/channel/facade.go`

```go
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
	mu               sync.RWMutex
	channels         map[string]Channel
	commandManager   CommandManager
	registry         registry.AgentRegistry
	logs             []LogEntry
	maxLogs          int
}

// Ensure channelFacadeImpl implements ChannelFacade at compile time
var _ ChannelFacade = (*channelFacadeImpl)(nil)

// NewChannelFacade creates a new channel facade service
func NewChannelFacade(injector do.Injector) (ChannelFacadeService, error) {
	cm := do.MustInvoke[CommandManagerService](injector)
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

// SubmitInput handles user input from any display
func (p *displayFacadeImpl) SubmitInput(ctx context.Context, input string) (InputResult, error) {
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
	agent, err := p.registry.GetAgentBySenderID(uuid.Nil)
	if err != nil {
		return InputResult{}, fmt.Errorf("agent not found: %w", err)
	}

	// Execute agent
	_, err = agent.Execute(ctx, input)
	if err != nil {
		return InputResult{}, err
	}

	return InputResult{
		Handled:   false,
		IsCommand: false,
	}, nil
}

// GetLogs returns recent log entries for displays to poll
func (p *displayFacadeImpl) GetLogs(since time.Time, limit int) []LogEntry {
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
```

---

## Section 5: Channel Middleware

**Location:** `pkg/channel/middleware.go`

```go
package channel

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/denkhaus/gollum/pkg/shared"
)

// ChannelMiddleware sends agent outputs to channel facade.
// This middleware bridges the agent execution pipeline with the channel system.
type ChannelMiddleware struct {
	facade    ChannelFacade
	agentID   uuid.UUID
	agentRole string
}

// NewChannelMiddleware creates a new channel middleware
func NewChannelMiddleware(facade ChannelFacade, agentID uuid.UUID, agentRole string) *ChannelMiddleware {
	return &ChannelMiddleware{
		facade:    facade,
		agentID:   agentID,
		agentRole: agentRole,
	}
}

// Process handles ProcessPart output and sends to channel
func (p *ChannelMiddleware) Process(_ context.Context, part shared.ProcessPart) error {
	if part == nil {
		return nil
	}

	switch part.Type {
	case shared.PartTypeText:
		if part.Text != nil {
			p.facade.DisplayMessage(Message{
				ID:        uuid.New(),
				Type:      MessageTypeAgentChat,
				AgentID:   p.agentID,
				AgentRole: p.agentRole,
				Content:   *part.Text,
				Timestamp: time.Now(),
			})
		}

	case shared.PartTypeToolUse:
		p.facade.DisplayMessage(Message{
			ID:        uuid.New(),
			Type:      MessageTypeToolRequest,
			AgentID:   p.agentID,
			AgentRole: p.agentRole,
			Content:   fmt.Sprintf("Tool use: %s", part.ToolName),
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"tool_name": part.ToolName,
				"tool_id":   part.ToolID,
			},
		})

	case shared.PartTypeToolResult:
		p.facade.DisplayMessage(Message{
			ID:        uuid.New(),
			Type:      MessageTypeToolResponse,
			AgentID:   p.agentID,
			AgentRole: p.agentRole,
			Content:   fmt.Sprintf("Tool result: %s", part.ToolName),
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"tool_name": part.ToolName,
				"tool_id":   part.ToolID,
			},
		})

	case shared.PartTypeThinking:
		if part.Text != nil {
			p.facade.DisplayMessage(Message{
				ID:        uuid.New(),
				Type:      MessageTypeThinking,
				AgentID:   p.agentID,
				AgentRole: p.agentRole,
				Content:   *part.Text,
				Timestamp: time.Now(),
			})
		}

	case shared.PartTypeError:
		p.facade.DisplayMessage(Message{
			ID:        uuid.New(),
			Type:      MessageTypeError,
			AgentID:   p.agentID,
			AgentRole: p.agentRole,
			Content:   part.ErrorMessage,
			Timestamp: time.Now(),
		})
	}

	return nil
}
```

---

## Section 6: DI Registration

Services are registered directly in `pkg/di/container.go` using the standard `do.Provide` pattern.

**Add to `pkg/di/container.go` in `RegisterServices`:**

```go
// Channel Abstraction Layer
do.Provide(p.injector, channel.NewCommandManager)
do.Provide(p.injector, channel.NewChannelFacade)
```

---

## Section 7: File Structure

```
pkg/
├── channel/
│   ├── types.go                # Core types (Message, LogEntry, InputResult, etc.)
│   ├── interface.go            # Channel, CommandManager, ChannelFacade interfaces
│   ├── command_manager.go      # commandManagerImpl + NewCommandManager
│   ├── facade.go               # channelFacadeImpl + NewChannelFacade
│   ├── middleware.go           # ChannelMiddleware for agent pipeline
│   ├── facade_test.go          # Tests for facade
│   └── command_manager_test.go # Tests for command manager
├── di/
│   └── container.go            # Updated with channel service registrations
├── tui/
│   ├── model.go                # REFACTORED to use channel.Message types
│   └── channel.go              # NEW: TUIChannel implementation (implements channel.Channel)
└── middleware/
    └── channel.go              # REFACTORED to use ChannelFacade (or deprecated)
```

**Separation of Concerns:**
- `pkg/channel/` = **Abstraction layer** (types, interfaces, facade, command manager)
- `pkg/tui/` = **One implementation** of the Channel interface
- Future implementations (web, telegram, etc.) would be in their own packages

---

## Section 8: Dependencies

**External dependencies:**
- `github.com/google/uuid` - UUID generation for messages and IDs
- `github.com/samber/do/v2` - Dependency injection

**Internal dependencies:**
- `github.com/denkhaus/gollum/pkg/channel` - Channel types and interfaces
- `github.com/denkhaus/gollum/pkg/config` - Configuration service (ConfigService)
- `github.com/denkhaus/gollum/pkg/registry` - Agent registry for input routing
- `github.com/denkhaus/gollum/pkg/shared` - ProcessPart types for middleware

---

## Section 9: Summary of Changes

### New Package
- `pkg/channel/` - Channel abstraction layer with:
  - Core types (`types.go`)
  - Interfaces (`interface.go`)
  - Command manager service (`command_manager.go`)
  - Channel facade service (`facade.go`)
  - Channel middleware (`middleware.go`)

### New Files
- `pkg/tui/channel.go` - TUIChannel implementation (implements `channel.Channel`)

### Refactored Packages
- `pkg/tui/model.go` - Use `channel.Message` types instead of local `Message` type
- `pkg/middleware/channel.go` - Refactored to use `ChannelFacade` instead of `ui.AgentMessenger`
- `pkg/di/container.go` - Add channel service registrations

### DI Registrations Added
```go
do.Provide(p.injector, channel.NewCommandManager)
do.Provide(p.injector, channel.NewChannelFacade)
```

### Benefits
1. **Testability**: Easy to mock channels for testing
2. **Extensibility**: New channels (web, telegram) can be added without modifying core logic
3. **Separation of concerns**:
   - `pkg/channel/` = abstraction layer
   - `pkg/tui/` = one implementation
4. **Centralized command handling**: All slash commands managed by `CommandManager`
5. **Log abstraction**: Channels poll for logs via `GetLogs()`
6. **DI compliance**: All services follow `guide.golang.di.md` patterns
