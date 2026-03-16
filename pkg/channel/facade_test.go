// Package channel provides unit tests for the channel facade service.
package channel

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
)

// mockChannel is a test double for Channel interface
type mockChannel struct {
	id       string
	messages []Message
	logs     []LogEntry
	events   []AgentLifecycleEvent
	mu       sync.Mutex
}

func newMockChannel(id string) *mockChannel {
	return &mockChannel{
		id:       id,
		messages: make([]Message, 0),
		logs:     make([]LogEntry, 0),
		events:   make([]AgentLifecycleEvent, 0),
	}
}

func (m *mockChannel) ID() string {
	return m.id
}

func (m *mockChannel) OnMessage(msg Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *mockChannel) OnLog(entry LogEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, entry)
}

func (m *mockChannel) OnAgentLifecycle(event AgentLifecycleEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockChannel) getMessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func (m *mockChannel) getLogCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.logs)
}

func (m *mockChannel) getEventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

func (m *mockChannel) getLastMessage() Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return Message{}
	}
	return m.messages[len(m.messages)-1]
}

func (m *mockChannel) getLastLog() LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.logs) == 0 {
		return LogEntry{}
	}
	return m.logs[len(m.logs)-1]
}

func (m *mockChannel) getLastEvent() AgentLifecycleEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.events) == 0 {
		return AgentLifecycleEvent{}
	}
	return m.events[len(m.events)-1]
}

// mockCommandManager is a test double for CommandManager
type mockCommandManager struct {
	executeFunc func(ctx context.Context, input string) (handled bool, response string, err error)
}

func (m *mockCommandManager) Register(cmd Command) error {
	return nil
}

func (m *mockCommandManager) Unregister(name string) error {
	return nil
}

func (m *mockCommandManager) Execute(ctx context.Context, input string) (bool, string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return false, "", nil
}

func (m *mockCommandManager) List() []Command {
	return nil
}

func (m *mockCommandManager) IsCommand(input string) bool {
	return false
}

// mockAgentRegistry is a test double for registry.AgentRegistry
type mockAgentRegistry struct{}

func (m *mockAgentRegistry) Register(agent shared.Agent, config *shared.AgentConfig, cancel ...context.CancelFunc) error {
	return nil
}

func (m *mockAgentRegistry) Unregister(agentID uuid.UUID) error {
	return nil
}

func (m *mockAgentRegistry) GetAgent(agentID uuid.UUID) (shared.Agent, bool) {
	return nil, false
}

func (m *mockAgentRegistry) GetChildren(parentID uuid.UUID) []shared.Agent {
	return nil
}

func (m *mockAgentRegistry) GetParent(agentID uuid.UUID) (shared.Agent, bool) {
	return nil, false
}

func (m *mockAgentRegistry) IsDirectParent(callerID, targetID uuid.UUID) bool {
	return false
}

func (m *mockAgentRegistry) ListAll() map[uuid.UUID]shared.Agent {
	return nil
}

func (m *mockAgentRegistry) Cleanup(agentID uuid.UUID) error {
	return nil
}

func (m *mockAgentRegistry) GetTotalAgentCount() int {
	return 0
}

func (m *mockAgentRegistry) GetSubAgentCount(parentID uuid.UUID) int {
	return 0
}

func (m *mockAgentRegistry) StoreAgentResult(result shared.AgentResult) error {
	return nil
}

func (m *mockAgentRegistry) GetAgentResult(agentID uuid.UUID) (*shared.AgentResult, bool) {
	return nil, false
}

func (m *mockAgentRegistry) WaitForAgent(ctx context.Context, agentID uuid.UUID, timeout time.Duration) (*shared.AgentResult, error) {
	return nil, nil
}

func (m *mockAgentRegistry) SetCancelFunc(agentID uuid.UUID, cancel context.CancelFunc) error {
	return nil
}

func (m *mockAgentRegistry) DeleteAgentResult(agentID uuid.UUID) error {
	return nil
}

// mockConfigService is a test double for config.ConfigService
type mockConfigService struct {
	logBufferSize int
}

func (m *mockConfigService) GetLogLevel() string {
	return "info"
}

func (m *mockConfigService) IsDevMode() bool {
	return false
}

func (m *mockConfigService) GetAnthropicConfig() *config.AnthropicConfig {
	return nil
}

func (m *mockConfigService) GetGeminiConfig() *config.GeminiConfig {
	return nil
}

func (m *mockConfigService) GetOpenAIConfig() *config.OpenAIConfig {
	return nil
}

func (m *mockConfigService) GetAgentLimits() *config.AgentLimitsConfig {
	return nil
}

func (m *mockConfigService) GetFilesConfig() *config.FilesConfig {
	return nil
}

func (m *mockConfigService) GetLoggingConfig() *config.LoggingConfig {
	return &config.LoggingConfig{
		SessionLogBufferSize: m.logBufferSize,
	}
}

func (m *mockConfigService) GetBashConfig() *config.BashConfig {
	return nil
}

func (m *mockConfigService) GetHooksConfig() *config.HooksConfig {
	return nil
}

func (m *mockConfigService) GetPromptStoreConfig() *config.PromptStoreConfig {
	return nil
}

func (m *mockConfigService) GetPromptOptimizerConfig() *config.PromptOptimizerConfig {
	return nil
}

func (m *mockConfigService) GetLangfuseConfig() *config.LangfuseConfig {
	return nil
}

func (m *mockConfigService) GetEventsConfig() *config.EventsConfig {
	return nil
}

func (m *mockConfigService) GetMCPConfig() *config.MCPConfig {
	return nil
}

// TestNewChannelFacade tests that NewChannelFacade creates a valid instance
func TestNewChannelFacade(t *testing.T) {
	injector := do.New()

	// Register mock dependencies
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)

	require.NoError(t, err)
	assert.NotNil(t, service)

	// Verify it implements ChannelFacade interface
	_, ok := service.(ChannelFacade)
	assert.True(t, ok, "NewChannelFacade should return a ChannelFacade implementation")
}

// TestChannelFacade_RegisterChannel_Success tests successful channel registration
func TestChannelFacade_RegisterChannel_Success(t *testing.T) {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	channel := newMockChannel("test-channel")
	err = service.RegisterChannel(channel)

	assert.NoError(t, err)
}

// TestChannelFacade_RegisterChannel_Duplicate tests that registering a duplicate channel returns an error
func TestChannelFacade_RegisterChannel_Duplicate(t *testing.T) {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	channel := newMockChannel("test-channel")

	// Register first time - should succeed
	err = service.RegisterChannel(channel)
	require.NoError(t, err)

	// Register second time - should fail
	err = service.RegisterChannel(channel)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
	assert.Contains(t, err.Error(), "test-channel")
}

// TestChannelFacade_UnregisterChannel_Success tests successful channel unregistration
func TestChannelFacade_UnregisterChannel_Success(t *testing.T) {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	channel := newMockChannel("test-channel")
	err = service.RegisterChannel(channel)
	require.NoError(t, err)

	// Unregister channel
	err = service.UnregisterChannel("test-channel")
	assert.NoError(t, err)
}

// TestChannelFacade_UnregisterChannel_NonExistent tests that unregistering a non-existent channel doesn't error
func TestChannelFacade_UnregisterChannel_NonExistent(t *testing.T) {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Unregister non-existent channel - should not error
	err = service.UnregisterChannel("non-existent")
	assert.NoError(t, err)
}

// TestChannelFacade_DisplayMessage_BroadcastsToAllChannels tests that DisplayMessage broadcasts to all registered channels
func TestChannelFacade_DisplayMessage_BroadcastsToAllChannels(t *testing.T) {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register multiple channels
	channels := []*mockChannel{
		newMockChannel("channel-1"),
		newMockChannel("channel-2"),
		newMockChannel("channel-3"),
	}

	for _, ch := range channels {
		err = service.RegisterChannel(ch)
		require.NoError(t, err)
	}

	// Send a message
	msg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgentChat,
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	service.DisplayMessage(msg)

	// Verify all channels received the message
	for _, ch := range channels {
		assert.Equal(t, 1, ch.getMessageCount(), "Channel should receive exactly one message")
		received := ch.getLastMessage()
		assert.Equal(t, msg.ID, received.ID)
		assert.Equal(t, msg.Content, received.Content)
	}
}

// TestChannelFacade_DisplayLog_StoresAndBroadcasts tests that DisplayLog stores entry and broadcasts to all channels
func TestChannelFacade_DisplayLog_StoresAndBroadcasts(t *testing.T) {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register multiple channels
	channels := []*mockChannel{
		newMockChannel("channel-1"),
		newMockChannel("channel-2"),
	}

	for _, ch := range channels {
		err = service.RegisterChannel(ch)
		require.NoError(t, err)
	}

	// Send a log entry
	entry := LogEntry{
		Level:     "info",
		Message:   "Test log message",
		Timestamp: time.Now(),
		Fields:    map[string]any{"key": "value"},
	}
	service.DisplayLog(entry)

	// Verify all channels received the log
	for _, ch := range channels {
		assert.Equal(t, 1, ch.getLogCount(), "Channel should receive exactly one log entry")
		received := ch.getLastLog()
		assert.Equal(t, entry.Level, received.Level)
		assert.Equal(t, entry.Message, received.Message)
	}

	// Verify log is stored
	logs := service.GetLogs(time.Time{}, 100)
	assert.Len(t, logs, 1)
	assert.Equal(t, entry.Message, logs[0].Message)
}

// TestChannelFacade_DisplayLog_RingBufferBehavior tests that DisplayLog implements ring buffer behavior
func TestChannelFacade_DisplayLog_RingBufferBehavior(t *testing.T) {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 5})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Add 10 log entries (buffer size is 5)
	for i := 0; i < 10; i++ {
		entry := LogEntry{
			Level:     "info",
			Message:   fmt.Sprintf("Log entry %d", i),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}
		service.DisplayLog(entry)
	}

	// Verify only the last 5 entries are stored
	logs := service.GetLogs(time.Time{}, 100)
	assert.Len(t, logs, 5, "Ring buffer should only keep maxLogs entries")

	// Verify the oldest entry is entry 5 (0-4 were evicted)
	assert.Equal(t, "Log entry 5", logs[0].Message)
	assert.Equal(t, "Log entry 9", logs[4].Message)
}

// TestChannelFacade_SubmitInput_SlashCommand tests that SubmitInput routes slash commands to CommandManager
func TestChannelFacade_SubmitInput_SlashCommand(t *testing.T) {
	injector := do.New()

	// Mock command manager that handles the command
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, input string) (bool, string, error) {
			if input == "/test args" {
				return true, "command executed", nil
			}
			return false, "", nil
		},
	}
	do.ProvideValue[CommandManagerService](injector, cmdMgr)
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := service.SubmitInput(ctx, "/test args")

	require.NoError(t, err)
	assert.True(t, result.Handled, "Slash command should be handled")
	assert.True(t, result.IsCommand, "Input should be marked as command")
	assert.Equal(t, "command executed", result.Response)
	assert.NoError(t, result.Error)
}

// TestChannelFacade_SubmitInput_NonCommand_NoAgentRegistry tests that SubmitInput returns error for non-command when no agent routing
func TestChannelFacade_SubmitInput_NonCommand_NoAgentRouting(t *testing.T) {
	injector := do.New()

	// Mock command manager that doesn't handle the input
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, input string) (bool, string, error) {
			return false, "", nil
		},
	}

	do.ProvideValue[CommandManagerService](injector, cmdMgr)
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := service.SubmitInput(ctx, "hello world")

	// Since GetAgentBySenderID doesn't exist yet, this should return an error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
	assert.Empty(t, result)
}

// TestChannelFacade_SubmitInput_CommandError tests that SubmitInput returns command errors
func TestChannelFacade_SubmitInput_CommandError(t *testing.T) {
	injector := do.New()

	// Mock command manager that returns an error
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, input string) (bool, string, error) {
			return true, "", errors.New("command failed")
		},
	}

	do.ProvideValue[CommandManagerService](injector, cmdMgr)
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := service.SubmitInput(ctx, "/test")

	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.True(t, result.IsCommand)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "command failed")
}

// TestChannelFacade_GetLogs_ReturnsEntriesAfterSpecifiedTime tests that GetLogs returns entries after specified time
func TestChannelFacade_GetLogs_ReturnsEntriesAfterSpecifiedTime(t *testing.T) {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	baseTime := time.Now()

	// Add log entries with different timestamps
	entries := []LogEntry{
		{Level: "info", Message: "entry 1", Timestamp: baseTime.Add(1 * time.Second)},
		{Level: "info", Message: "entry 2", Timestamp: baseTime.Add(2 * time.Second)},
		{Level: "info", Message: "entry 3", Timestamp: baseTime.Add(3 * time.Second)},
		{Level: "info", Message: "entry 4", Timestamp: baseTime.Add(4 * time.Second)},
		{Level: "info", Message: "entry 5", Timestamp: baseTime.Add(5 * time.Second)},
	}

	for _, entry := range entries {
		service.DisplayLog(entry)
	}

	// Get logs after entry 2's timestamp
	since := baseTime.Add(2 * time.Second)
	logs := service.GetLogs(since, 100)

	// Should return entries 3, 4, 5 (after entry 2, not including entry 2)
	assert.Len(t, logs, 3)
	assert.Equal(t, "entry 3", logs[0].Message)
	assert.Equal(t, "entry 4", logs[1].Message)
	assert.Equal(t, "entry 5", logs[2].Message)
}

// TestChannelFacade_GetLogs_RespectsLimitParameter tests that GetLogs respects limit parameter
func TestChannelFacade_GetLogs_RespectsLimitParameter(t *testing.T) {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	baseTime := time.Now()

	// Add 10 log entries
	for i := 1; i <= 10; i++ {
		entry := LogEntry{
			Level:     "info",
			Message:   fmt.Sprintf("entry %d", i),
			Timestamp: baseTime.Add(time.Duration(i) * time.Second),
		}
		service.DisplayLog(entry)
	}

	// Get logs with limit of 5
	logs := service.GetLogs(time.Time{}, 5)

	assert.Len(t, logs, 5, "GetLogs should respect limit parameter")
	assert.Equal(t, "entry 1", logs[0].Message)
	assert.Equal(t, "entry 5", logs[4].Message)
}

// TestChannelFacade_NotifyAgentLifecycle_BroadcastsToAllChannels tests that NotifyAgentLifecycle broadcasts event to all channels
func TestChannelFacade_NotifyAgentLifecycle_BroadcastsToAllChannels(t *testing.T) {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register multiple channels
	channels := []*mockChannel{
		newMockChannel("channel-1"),
		newMockChannel("channel-2"),
		newMockChannel("channel-3"),
	}

	for _, ch := range channels {
		err = service.RegisterChannel(ch)
		require.NoError(t, err)
	}

	// Notify agent lifecycle event
	agentID := uuid.New()
	role := "tester"
	service.NotifyAgentLifecycle(agentID, role, true)

	// Verify all channels received the event
	for _, ch := range channels {
		assert.Equal(t, 1, ch.getEventCount(), "Channel should receive exactly one lifecycle event")
		received := ch.getLastEvent()
		assert.Equal(t, agentID, received.AgentID)
		assert.Equal(t, role, received.Role)
		assert.True(t, received.Added)
	}
}

// TestChannelFacade_Concurrency tests that concurrent access is safe
func TestChannelFacade_Concurrency(t *testing.T) {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	done := make(chan bool)

	// Register channels concurrently
	for i := 0; i < 5; i++ {
		go func(idx int) {
			ch := newMockChannel(fmt.Sprintf("channel-%d", idx))
			_ = service.RegisterChannel(ch)
			done <- true
		}(i)
	}

	// Send messages concurrently
	for i := 0; i < 10; i++ {
		go func(idx int) {
			msg := Message{
				ID:        uuid.New(),
				Content:   fmt.Sprintf("message %d", idx),
				Timestamp: time.Now(),
			}
			service.DisplayMessage(msg)
			done <- true
		}(i)
	}

	// Send logs concurrently
	for i := 0; i < 10; i++ {
		go func(idx int) {
			entry := LogEntry{
				Level:     "info",
				Message:   fmt.Sprintf("log %d", idx),
				Timestamp: time.Now(),
			}
			service.DisplayLog(entry)
			done <- true
		}(i)
	}

	// Get logs concurrently
	for i := 0; i < 5; i++ {
		go func() {
			_ = service.GetLogs(time.Time{}, 10)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 30; i++ {
		<-done
	}

	// Verify final state is consistent
	logs := service.GetLogs(time.Time{}, 100)
	assert.Greater(t, len(logs), 0, "Should have logs stored")
}
