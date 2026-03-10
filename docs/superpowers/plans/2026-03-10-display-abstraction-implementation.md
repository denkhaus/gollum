# Display Abstraction Layer Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a display abstraction layer that decouples the TUI from agent logic, enabling multiple display implementations.

**Architecture:** Facade pattern with Display interface for implementations (TUI, web, telegram), CommandManager for centralized slash command handling, and DisplayFacade as the coordinator. All services follow DI patterns from guide.golang.di.md.

**Tech Stack:** Go, samber/do/v2 for DI, google/uuid for identifiers

---

## File Structure

```
pkg/
├── display/                    # NEW PACKAGE
│   ├── types.go               # Core types
│   ├── interface.go           # Interfaces
│   ├── command_manager.go     # Command manager service
│   ├── facade.go              # Display facade service
│   ├── middleware.go          # Agent pipeline middleware
│   ├── command_manager_test.go
│   └── facade_test.go
├── di/
│   └── container.go           # MODIFY: Add display registrations
├── tui/
│   ├── model.go               # MODIFY: Use display.Message
│   └── display.go             # NEW: TUIDisplay implementation
└── middleware/
    └── display.go             # MODIFY: Use DisplayFacade
```

---

## Chunk 1: Core Types and Interfaces

### Task 1: Create Core Types

**Files:**
- Create: `pkg/display/types.go`
- Create: `pkg/display/types_test.go`

- [ ] **Step 1: Create the types file**

```go
// pkg/display/types.go
package display

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

- [ ] **Step 2: Write tests for MessageType String()**

```go
// pkg/display/types_test.go
package display

import (
	"testing"
)

func TestMessageTypeString(t *testing.T) {
	tests := []struct {
		mt       MessageType
		expected string
	}{
		{MessageTypeUserChat, "user_chat"},
		{MessageTypeAgentChat, "agent_chat"},
		{MessageTypeToolRequest, "tool_request"},
		{MessageTypeToolResponse, "tool_response"},
		{MessageTypeThinking, "thinking"},
		{MessageTypeSystemInfo, "system_info"},
		{MessageTypeError, "error"},
		{MessageType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.mt.String(); got != tt.expected {
				t.Errorf("MessageType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./pkg/display/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add pkg/display/types.go pkg/display/types_test.go
git commit -m "feat(display): add core types for display abstraction layer

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 2: Create Interfaces

**Files:**
- Create: `pkg/display/interface.go`

- [ ] **Step 1: Create the interface file**

```go
// pkg/display/interface.go
package display

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Display is the interface that all display implementations must satisfy
type Display interface {
	// ID returns a unique identifier for this display
	ID() string

	// OnMessage is called when a new message should be displayed
	OnMessage(msg Message)

	// OnLog is called for log entries (display can ignore if not applicable)
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

// DisplayFacade is the central coordinator for all displays
type DisplayFacade interface {
	// DisplayMessage sends a message to all registered displays
	DisplayMessage(msg Message)

	// DisplayLog sends a log entry to all registered displays
	DisplayLog(entry LogEntry)

	// SubmitInput handles user input from any display
	SubmitInput(ctx context.Context, input string) (InputResult, error)

	// GetLogs returns recent log entries for displays to poll
	GetLogs(since time.Time, limit int) []LogEntry

	// RegisterDisplay adds a display to receive events
	RegisterDisplay(display Display) error

	// UnregisterDisplay removes a display
	UnregisterDisplay(displayID string) error

	// NotifyAgentLifecycle broadcasts agent lifecycle event
	NotifyAgentLifecycle(agentID uuid.UUID, role string, added bool)
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/display/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/display/interface.go
git commit -m "feat(display): add Display, CommandManager, and DisplayFacade interfaces

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 2: Command Manager Service

### Task 3: Implement Command Manager with TDD

**Files:**
- Create: `pkg/display/command_manager.go`
- Create: `pkg/display/command_manager_test.go`

- [ ] **Step 1: Write failing test for Register**

```go
// pkg/display/command_manager_test.go
package display

import (
	"context"
	"testing"
)

func TestCommandManager_Register(t *testing.T) {
	cm := NewCommandManager()

	cmd := Command{
		Name:        "/test",
		Description: "Test command",
		Handler: func(_ context.Context, _ string) (string, error) {
			return "test response", nil
		},
	}

	err := cm.Register(cmd)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// Verify duplicate registration fails
	err = cm.Register(cmd)
	if err == nil {
		t.Error("Register() should fail for duplicate command")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/display/... -run TestCommandManager_Register -v`
Expected: FAIL (undefined: NewCommandManager)

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/display/command_manager.go
package display

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

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/display/... -run TestCommandManager_Register -v`
Expected: PASS

- [ ] **Step 5: Write failing test for Execute**

```go
// Add to pkg/display/command_manager_test.go

func TestCommandManager_Execute(t *testing.T) {
	cm := NewCommandManager()

	// Register a test command
	_ = cm.Register(Command{
		Name: "/echo",
		Handler: func(_ context.Context, args string) (string, error) {
			return args, nil
		},
	})

	tests := []struct {
		name          string
		input         string
		wantHandled   bool
		wantResponse  string
	}{
		{"valid command", "/echo hello", true, "hello"},
		{"command no args", "/echo", true, ""},
		{"not a command", "hello", false, ""},
		{"empty input", "", false, ""},
		{"unknown command", "/unknown", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handled, response, err := cm.Execute(context.Background(), tt.input)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}
			if handled != tt.wantHandled {
				t.Errorf("Execute() handled = %v, want %v", handled, tt.wantHandled)
			}
			if response != tt.wantResponse {
				t.Errorf("Execute() response = %v, want %v", response, tt.wantResponse)
			}
		})
	}
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./pkg/display/... -run TestCommandManager_Execute -v`
Expected: PASS

- [ ] **Step 7: Write test for IsCommand**

```go
// Add to pkg/display/command_manager_test.go

func TestCommandManager_IsCommand(t *testing.T) {
	cm := NewCommandManager()
	_ = cm.Register(Command{Name: "/test", Handler: func(_ context.Context, _ string) (string, error) { return "", nil }})

	tests := []struct {
		input string
		want  bool
	}{
		{"/test", true},
		{"/test args", true},
		{"test", false},
		{"", false},
		{"/unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := cm.IsCommand(tt.input); got != tt.want {
				t.Errorf("IsCommand(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 8: Run all tests**

Run: `go test ./pkg/display/... -v`
Expected: All PASS

- [ ] **Step 9: Commit**

```bash
git add pkg/display/command_manager.go pkg/display/command_manager_test.go
git commit -m "feat(display): add CommandManager service with TDD

- Implements Register, Unregister, Execute, List, IsCommand
- Follows DI guide patterns with private implementation
- Thread-safe with sync.RWMutex

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 3: Display Facade Service

### Task 4: Implement Display Facade with TDD

**Files:**
- Create: `pkg/display/facade.go`
- Create: `pkg/display/facade_test.go`

- [ ] **Step 1: Write failing test for RegisterDisplay**

```go
// pkg/display/facade_test.go
package display

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/samber/do/v2"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// mockDisplay implements Display for testing
type mockDisplay struct {
	id           string
	messages     []Message
	logs         []LogEntry
	lifecycle    []AgentLifecycleEvent
}

func (m *mockDisplay) ID() string                          { return m.id }
func (m *mockDisplay) OnMessage(msg Message)               { m.messages = append(m.messages, msg) }
func (m *mockDisplay) OnLog(entry LogEntry)                { m.logs = append(m.logs, entry) }
func (m *mockDisplay) OnAgentLifecycle(event AgentLifecycleEvent) {
	m.lifecycle = append(m.lifecycle, event)
}

func TestDisplayFacade_RegisterDisplay(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create facade with mocked dependencies
	facade := createTestFacade(ctrl)

	display1 := &mockDisplay{id: "display-1"}
	display2 := &mockDisplay{id: "display-2"}

	// Register first display
	err := facade.RegisterDisplay(display1)
	if err != nil {
		t.Errorf("RegisterDisplay() error = %v", err)
	}

	// Register second display
	err = facade.RegisterDisplay(display2)
	if err != nil {
		t.Errorf("RegisterDisplay() error = %v", err)
	}

	// Duplicate registration should fail
	err = facade.RegisterDisplay(display1)
	if err == nil {
		t.Error("RegisterDisplay() should fail for duplicate")
	}
}

func createTestFacade(ctrl *gomock.Controller) DisplayFacade {
	injector := do.New()

	// Register mocks
	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		mockCfg := mocks.NewMockConfigService(ctrl)
		mockCfg.EXPECT().GetLoggingConfig().Return(&config.LoggingConfig{
			SessionLogBufferSize: 100,
		}).AnyTimes()
		return mockCfg, nil
	})

	do.Provide(injector, func(_ do.Injector) (registry.AgentRegistry, error) {
		return mocks.NewMockAgentRegistry(ctrl), nil
	})

	do.Provide(injector, NewCommandManager)

	facade, _ := NewDisplayFacade(injector)
	return facade
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/display/... -run TestDisplayFacade_RegisterDisplay -v`
Expected: FAIL (undefined: NewDisplayFacade)

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/display/facade.go
package display

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

// DisplayFacadeService defines the display facade service interface for DI
type DisplayFacadeService interface {
	DisplayFacade
}

// displayFacadeImpl implements DisplayFacade
type displayFacadeImpl struct {
	mu             sync.RWMutex
	displays       map[string]Display
	commandManager CommandManager
	registry       registry.AgentRegistry
	logs           []LogEntry
	maxLogs        int
}

// Ensure displayFacadeImpl implements DisplayFacade at compile time
var _ DisplayFacade = (*displayFacadeImpl)(nil)

// NewDisplayFacade creates a new display facade service
func NewDisplayFacade(injector do.Injector) (DisplayFacadeService, error) {
	cm := do.MustInvoke[CommandManagerService](injector)
	reg := do.MustInvoke[registry.AgentRegistry](injector)
	cfg := do.MustInvoke[config.ConfigService](injector)

	// Use existing LoggingConfig.SessionLogBufferSize for display log buffer
	maxLogs := cfg.GetLoggingConfig().SessionLogBufferSize

	return &displayFacadeImpl{
		commandManager: cm,
		registry:       reg,
		displays:       make(map[string]Display),
		logs:           make([]LogEntry, 0, maxLogs),
		maxLogs:        maxLogs,
	}, nil
}

// RegisterDisplay adds a display to receive events
func (p *displayFacadeImpl) RegisterDisplay(display Display) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.displays[display.ID()]; exists {
		return fmt.Errorf("display %s already registered", display.ID())
	}

	p.displays[display.ID()] = display
	return nil
}

// UnregisterDisplay removes a display
func (p *displayFacadeImpl) UnregisterDisplay(displayID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.displays, displayID)
	return nil
}

// DisplayMessage sends a message to all registered displays
func (p *displayFacadeImpl) DisplayMessage(msg Message) {
	p.mu.RLock()
	displays := make([]Display, 0, len(p.displays))
	for _, d := range p.displays {
		displays = append(displays, d)
	}
	p.mu.RUnlock()

	for _, display := range displays {
		display.OnMessage(msg)
	}
}

// DisplayLog sends a log entry to all registered displays
func (p *displayFacadeImpl) DisplayLog(entry LogEntry) {
	p.mu.Lock()

	// Store log entry with ring buffer behavior
	p.logs = append(p.logs, entry)
	if len(p.logs) > p.maxLogs {
		p.logs = p.logs[1:]
	}

	displays := make([]Display, 0, len(p.displays))
	for _, d := range p.displays {
		displays = append(displays, d)
	}
	p.mu.Unlock()

	// Send to all displays
	for _, display := range displays {
		display.OnLog(entry)
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
func (p *displayFacadeImpl) NotifyAgentLifecycle(agentID uuid.UUID, role string, added bool) {
	event := AgentLifecycleEvent{
		AgentID: agentID,
		Role:    role,
		Added:   added,
	}

	p.mu.RLock()
	displays := make([]Display, 0, len(p.displays))
	for _, d := range p.displays {
		displays = append(displays, d)
	}
	p.mu.RUnlock()

	for _, display := range displays {
		display.OnAgentLifecycle(event)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/display/... -run TestDisplayFacade_RegisterDisplay -v`
Expected: PASS

- [ ] **Step 5: Write test for DisplayMessage**

```go
// Add to pkg/display/facade_test.go

func TestDisplayFacade_DisplayMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	facade := createTestFacade(ctrl)

	display1 := &mockDisplay{id: "display-1"}
	display2 := &mockDisplay{id: "display-2"}

	_ = facade.RegisterDisplay(display1)
	_ = facade.RegisterDisplay(display2)

	msg := Message{
		ID:      uuid.New(),
		Type:    MessageTypeAgentChat,
		Content: "Hello world",
	}

	facade.DisplayMessage(msg)

	if len(display1.messages) != 1 {
		t.Errorf("display1 should have 1 message, got %d", len(display1.messages))
	}
	if len(display2.messages) != 1 {
		t.Errorf("display2 should have 1 message, got %d", len(display2.messages))
	}
	if display1.messages[0].Content != "Hello world" {
		t.Errorf("display1 message content = %v, want 'Hello world'", display1.messages[0].Content)
	}
}
```

- [ ] **Step 6: Run test**

Run: `go test ./pkg/display/... -run TestDisplayFacade_DisplayMessage -v`
Expected: PASS

- [ ] **Step 7: Write test for DisplayLog ring buffer**

```go
// Add to pkg/display/facade_test.go

func TestDisplayFacade_DisplayLog_RingBuffer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create facade with small buffer for testing
	injector := do.New()
	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		mockCfg := mocks.NewMockConfigService(ctrl)
		mockCfg.EXPECT().GetLoggingConfig().Return(&config.LoggingConfig{
			SessionLogBufferSize: 3, // Small buffer for testing
		}).AnyTimes()
		return mockCfg, nil
	})
	do.Provide(injector, func(_ do.Injector) (registry.AgentRegistry, error) {
		return mocks.NewMockAgentRegistry(ctrl), nil
	})
	do.Provide(injector, NewCommandManager)

	facade, _ := NewDisplayFacade(injector)

	// Add 5 logs to buffer of size 3
	for i := 0; i < 5; i++ {
		facade.DisplayLog(LogEntry{Message: string(rune('A' + i))})
	}

	logs := facade.GetLogs(time.Time{}, 10)
	if len(logs) != 3 {
		t.Errorf("GetLogs() should return 3 entries, got %d", len(logs))
	}
	// Should have last 3 entries (C, D, E)
	if logs[0].Message != "C" || logs[1].Message != "D" || logs[2].Message != "E" {
		t.Errorf("GetLogs() should have [C, D, E], got %v", logs)
	}
}
```

- [ ] **Step 8: Run all facade tests**

Run: `go test ./pkg/display/... -run TestDisplayFacade -v`
Expected: All PASS

- [ ] **Step 9: Commit**

```bash
git add pkg/display/facade.go pkg/display/facade_test.go
git commit -m "feat(display): add DisplayFacade service with TDD

- Implements RegisterDisplay, UnregisterDisplay
- DisplayMessage broadcasts to all displays
- DisplayLog with ring buffer using SessionLogBufferSize
- SubmitInput routes to CommandManager or agent
- GetLogs for polling, NotifyAgentLifecycle for events

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 4: Display Middleware

### Task 5: Implement Display Middleware

**Files:**
- Create: `pkg/display/middleware.go`

- [ ] **Step 1: Create the middleware file**

```go
// pkg/display/middleware.go
package display

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/denkhaus/gollum/pkg/shared"
)

// DisplayMiddleware sends agent outputs to display facade.
// This middleware bridges the agent execution pipeline with the display system.
type DisplayMiddleware struct {
	facade    DisplayFacade
	agentID   uuid.UUID
	agentRole string
}

// NewDisplayMiddleware creates a new display middleware
func NewDisplayMiddleware(facade DisplayFacade, agentID uuid.UUID, agentRole string) *DisplayMiddleware {
	return &DisplayMiddleware{
		facade:    facade,
		agentID:   agentID,
		agentRole: agentRole,
	}
}

// Process handles ProcessPart output and sends to display
func (p *DisplayMiddleware) Process(_ context.Context, part shared.ProcessPart) error {
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

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/display/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/display/middleware.go
git commit -m "feat(display): add DisplayMiddleware for agent pipeline integration

- Converts ProcessPart types to Display messages
- Handles Text, ToolUse, ToolResult, Thinking, Error types

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 5: DI Registration

### Task 6: Register Display Services in DI Container

**Files:**
- Modify: `pkg/di/container.go`

- [ ] **Step 1: Add import and registrations**

Add import:
```go
import (
	// ... existing imports ...
	"github.com/denkhaus/gollum/pkg/display"
)
```

Add to `RegisterServices` function (after Registry registration):
```go
	// Display Abstraction Layer
	do.Provide(p.injector, display.NewCommandManager)
	do.Provide(p.injector, display.NewDisplayFacade)
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 3: Run all tests**

Run: `go test ./... -short`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add pkg/di/container.go
git commit -m "feat(di): register display services in DI container

- Register CommandManager service
- Register DisplayFacade service

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 6: TUI Integration

### Task 7: Create TUIDisplay Implementation

**Files:**
- Create: `pkg/tui/display.go`

- [ ] **Step 1: Create TUIDisplay file**

```go
// pkg/tui/display.go
package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/denkhaus/gollum/pkg/display"
)

// TUIDisplay implements display.Display interface for Bubbletea
type TUIDisplay struct {
	id              string
	mu              sync.RWMutex
	history         []string
	historyIndex    int
	multiLine       bool
	multiLineBuffer strings.Builder
	commandManager  display.CommandManager
	messageChan     chan display.Message
}

// Ensure TUIDisplay implements display.Display at compile time
var _ display.Display = (*TUIDisplay)(nil)

// NewTUIDisplay creates a new TUI display
func NewTUIDisplay(cm display.CommandManager) *TUIDisplay {
	return &TUIDisplay{
		id:              fmt.Sprintf("tui-%s", uuid.New().String()),
		commandManager:  cm,
		history:         []string{},
		historyIndex:    -1,
		multiLine:       false,
		multiLineBuffer: strings.Builder{},
		messageChan:     make(chan display.Message, 100),
	}
}

// ID returns unique identifier
func (p *TUIDisplay) ID() string {
	return p.id
}

// OnMessage handles new messages from the facade
func (p *TUIDisplay) OnMessage(msg display.Message) {
	select {
	case p.messageChan <- msg:
	default:
		// Channel full, drop message
	}
}

// OnLog handles log entries
func (p *TUIDisplay) OnLog(_ display.LogEntry) {
	// TUI handles logs via polling, not push
}

// OnAgentLifecycle handles agent events
func (p *TUIDisplay) OnAgentLifecycle(_ display.AgentLifecycleEvent) {
	// Could update status bar or agent list
}

// GetMessageChannel returns the channel for receiving messages
func (p *TUIDisplay) GetMessageChannel() <-chan display.Message {
	return p.messageChan
}

// GetHistory returns the input history
func (p *TUIDisplay) GetHistory() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.history
}

// GetHistoryIndex returns current position in history
func (p *TUIDisplay) GetHistoryIndex() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.historyIndex
}

// SetHistoryIndex sets the current position in history
func (p *TUIDisplay) SetHistoryIndex(idx int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.historyIndex = idx
}

// NavigateHistory navigates through input history
func (p *TUIDisplay) NavigateHistory(direction int) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	newIdx := p.historyIndex + direction
	if newIdx < 0 {
		newIdx = 0
	} else if newIdx >= len(p.history) {
		newIdx = len(p.history) - 1
	}

	if newIdx >= 0 && newIdx < len(p.history) {
		p.historyIndex = newIdx
		return p.history[newIdx]
	}
	return ""
}

// ToggleMultiLine toggles multi-line input mode
func (p *TUIDisplay) ToggleMultiLine() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.multiLine = !p.multiLine
}

// IsMultiLine returns whether multi-line mode is active
func (p *TUIDisplay) IsMultiLine() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.multiLine
}

// HandleInput processes user input
func (p *TUIDisplay) HandleInput(ctx context.Context, input string) (display.InputResult, error) {
	// Handle multi-line input
	if p.IsMultiLine() {
		if input == "" {
			// Submit multi-line buffer
			content := p.multiLineBuffer.String()
			p.multiLineBuffer.Reset()
			p.ToggleMultiLine()

			// Add to history
			p.addToHistory(content)

			return display.InputResult{Handled: false}, nil
		}

		p.multiLineBuffer.WriteString(input + "\n")
		return display.InputResult{Handled: true, Response: "(multi-line mode, empty line to submit)"}, nil
	}

	// Check for commands
	if p.commandManager.IsCommand(input) {
		return p.commandManager.Execute(ctx, input)
	}

	// Regular input - add to history
	p.addToHistory(input)

	return display.InputResult{Handled: false}, nil
}

// addToHistory adds input to local history
func (p *TUIDisplay) addToHistory(input string) {
	if input == "" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Don't add duplicates
	if len(p.history) > 0 && p.history[len(p.history)-1] == input {
		return
	}

	p.history = append(p.history, input)

	// Keep only last 100 entries
	if len(p.history) > 100 {
		p.history = p.history[len(p.history)-100:]
	}

	p.historyIndex = len(p.history)
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/tui/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/tui/display.go
git commit -m "feat(tui): add TUIDisplay implementing display.Display

- Implements OnMessage, OnLog, OnAgentLifecycle
- Manages input history with navigation
- Supports multi-line input mode
- Uses buffered channel for message delivery

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 8: Update TUI Model to Use Display Types

**Files:**
- Modify: `pkg/tui/model.go`

- [ ] **Step 1: Update imports**

Add import:
```go
import (
	// ... existing imports ...
	"github.com/denkhaus/gollum/pkg/display"
)
```

- [ ] **Step 2: Update Message type alias**

Replace the local `MessageType` and `Message` definitions with aliases to display package types:

```go
// MessageType represents the type of message being displayed.
// Alias to display.MessageType for compatibility.
type MessageType = display.MessageType

// Message type constants - aliases to display package
const (
	MessageTypeUser   = display.MessageTypeUserChat
	MessageTypeAgent  = display.MessageTypeAgentChat
	MessageTypeTool   = display.MessageTypeToolRequest
	MessageTypeSystem = display.MessageTypeSystemInfo
	MessageTypeError  = display.MessageTypeError
)

// Message represents a single message in the conversation history.
// Alias to display.Message for compatibility.
type Message = display.Message
```

- [ ] **Step 3: Update String() method if needed**

The String() method on MessageType now delegates to display.MessageType.String(). Remove the local String() method if it exists.

- [ ] **Step 4: Verify compilation**

Run: `go build ./pkg/tui/...`
Expected: No errors

- [ ] **Step 5: Run TUI tests**

Run: `go test ./pkg/tui/... -v`
Expected: All PASS (may need minor adjustments for type aliases)

- [ ] **Step 6: Commit**

```bash
git add pkg/tui/model.go
git commit -m "refactor(tui): use display.Message types instead of local definitions

- Alias MessageType and Message to display package types
- Map legacy TUI message types to display types
- Maintain backward compatibility

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 7: Final Verification

### Task 9: Run Full Test Suite

- [ ] **Step 1: Run all tests**

Run: `go test ./... -short`
Expected: All PASS

- [ ] **Step 2: Run linting**

Run: `golangci-lint run ./pkg/display/... ./pkg/tui/...`
Expected: No errors

- [ ] **Step 3: Final commit (if any fixes needed)**

```bash
git add -A
git commit -m "fix(display): address linting issues and test failures

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Summary

### Files Created
- `pkg/display/types.go` - Core types
- `pkg/display/interface.go` - Interfaces
- `pkg/display/command_manager.go` - Command manager service
- `pkg/display/facade.go` - Display facade service
- `pkg/display/middleware.go` - Agent pipeline middleware
- `pkg/display/types_test.go` - Type tests
- `pkg/display/command_manager_test.go` - Command manager tests
- `pkg/display/facade_test.go` - Facade tests
- `pkg/tui/display.go` - TUIDisplay implementation

### Files Modified
- `pkg/di/container.go` - Add display service registrations
- `pkg/tui/model.go` - Use display.Message types

### Key Patterns
- All services follow `guide.golang.di.md` patterns
- Private implementations (`commandManagerImpl`, `displayFacadeImpl`)
- Method receiver `p` as per coding standards
- Compile-time interface verification with `var _ Interface = (*impl)(nil)`
