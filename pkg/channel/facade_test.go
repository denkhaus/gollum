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
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
)

// mockChannel is a test double for Channel interface
type mockChannel struct {
	id       uuid.UUID
	messages []Message
	logs     []LogEntry
	events   []AgentLifecycleEvent
	mu       sync.Mutex
}

func newMockChannel(id uuid.UUID) *mockChannel {
	return &mockChannel{
		id:       id,
		messages: make([]Message, 0),
		logs:     make([]LogEntry, 0),
		events:   make([]AgentLifecycleEvent, 0),
	}
}

func (m *mockChannel) ID() uuid.UUID {
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

// mockAgentFactory is a test double for shared.AgentFactory
type mockAgentFactory struct{}

func (m *mockAgentFactory) CreateAgent(ctx context.Context, config *shared.AgentConfig) (shared.Agent, error) {
	return nil, nil
}

func (m *mockAgentFactory) CreateSupervisorAgent(ctx context.Context, opts ...shared.SupervisorAgentOption) (shared.Agent, *shared.AgentConfig, error) {
	return nil, nil, nil
}

// mockAgentRegistryWithSupervisor is a configurable mock that can return a supervisor agent
type mockAgentRegistryWithSupervisor struct {
	supervisor shared.Agent
	err        error
}

func (m *mockAgentRegistryWithSupervisor) Register(agent shared.Agent, config *shared.AgentConfig, cancel ...context.CancelFunc) error {
	return nil
}

func (m *mockAgentRegistryWithSupervisor) Unregister(agentID uuid.UUID) error {
	return nil
}

func (m *mockAgentRegistryWithSupervisor) GetAgent(agentID uuid.UUID) (shared.Agent, bool) {
	return nil, false
}

func (m *mockAgentRegistryWithSupervisor) GetChildren(parentID uuid.UUID) []shared.Agent {
	return nil
}

func (m *mockAgentRegistryWithSupervisor) GetParent(agentID uuid.UUID) (shared.Agent, bool) {
	return nil, false
}

func (m *mockAgentRegistryWithSupervisor) IsDirectParent(callerID, targetID uuid.UUID) bool {
	return false
}

func (m *mockAgentRegistryWithSupervisor) ListAll() map[uuid.UUID]shared.Agent {
	return nil
}

func (m *mockAgentRegistryWithSupervisor) Cleanup(agentID uuid.UUID) error {
	return nil
}

func (m *mockAgentRegistryWithSupervisor) GetTotalAgentCount() int {
	return 0
}

func (m *mockAgentRegistryWithSupervisor) GetSubAgentCount(parentID uuid.UUID) int {
	return 0
}

func (m *mockAgentRegistryWithSupervisor) StoreAgentResult(result shared.AgentResult) error {
	return nil
}

func (m *mockAgentRegistryWithSupervisor) GetAgentResult(agentID uuid.UUID) (*shared.AgentResult, bool) {
	return nil, false
}

func (m *mockAgentRegistryWithSupervisor) WaitForAgent(ctx context.Context, agentID uuid.UUID, timeout time.Duration) (*shared.AgentResult, error) {
	return nil, nil
}

func (m *mockAgentRegistryWithSupervisor) SetCancelFunc(agentID uuid.UUID, cancel context.CancelFunc) error {
	return nil
}

func (m *mockAgentRegistryWithSupervisor) DeleteAgentResult(agentID uuid.UUID) error {
	return nil
}

func (m *mockAgentRegistryWithSupervisor) GetSupervisorAgent() (shared.Agent, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.supervisor, nil
}

// mockAgent is a test double for shared.Agent
type mockAgent struct {
	id uuid.UUID
}

func newMockAgent(id uuid.UUID) *mockAgent {
	return &mockAgent{id: id}
}

func (m *mockAgent) GetID() uuid.UUID {
	return m.id
}

func (m *mockAgent) GetConfig() *shared.AgentConfig {
	return &shared.AgentConfig{}
}

func (m *mockAgent) Session() gollem.Session {
	return nil
}

func (m *mockAgent) Execute(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
	return &gollem.ExecuteResponse{
		Texts: []string{"Supervisor response"},
	}, nil
}

func (m *mockAgent) GetMessageHistory(ctx context.Context) (*gollem.History, error) {
	return nil, nil
}

func (m *mockAgent) UpdateSystemPrompt(ctx context.Context, newPrompt string) error {
	return nil
}

func (m *mockAgent) UpdateHistory(ctx context.Context, modifier func(*gollem.History) (*gollem.History, error)) error {
	return nil
}

// setupTestInjector creates an injector with all mock dependencies for testing
func setupTestInjector() do.Injector {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})

	// Add a mock logger
	ctrl := gomock.NewController(&testing.T{})
	mockLogger := logger.NewMockLoggerService(ctrl)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	return injector
}

// setupTestInjectorWithLogger creates an injector with a specific logger mock
func setupTestInjectorWithLogger(logService logger.LoggerService) do.Injector {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})
	do.ProvideValue[logger.LoggerService](injector, logService)

	return injector
}

// setupTestInjectorWithConfig creates an injector with a specific config
func setupTestInjectorWithConfig(cfg config.ConfigService) do.Injector {
	injector := do.New()
	do.ProvideValue[CommandManagerService](injector, &mockCommandManager{})
	do.ProvideValue[registry.AgentRegistry](injector, &mockAgentRegistry{})
	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	do.ProvideValue[config.ConfigService](injector, cfg)

	// Add a mock logger
	ctrl := gomock.NewController(&testing.T{})
	mockLogger := logger.NewMockLoggerService(ctrl)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	return injector
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

func (m *mockAgentRegistry) GetSupervisorAgent() (shared.Agent, error) {
	return nil, fmt.Errorf("no supervisor agent registered")
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
	injector := setupTestInjector()

	service, err := NewChannelFacade(injector)

	require.NoError(t, err)
	assert.NotNil(t, service)

	// Verify it implements ChannelFacade interface
	_, ok := service.(ChannelFacade)
	assert.True(t, ok, "NewChannelFacade should return a ChannelFacade implementation")
}

// TestChannelFacade_RegisterChannel_Success tests successful channel registration
func TestChannelFacade_RegisterChannel_Success(t *testing.T) {
	injector := setupTestInjector()

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	channel := newMockChannel(uuid.New())
	err = service.RegisterChannel(channel)

	assert.NoError(t, err)
}

// TestChannelFacade_RegisterChannel_Duplicate tests that registering a duplicate channel returns an error
func TestChannelFacade_RegisterChannel_Duplicate(t *testing.T) {
	injector := setupTestInjector()

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	channel := newMockChannel(uuid.New())

	// Register first time - should succeed
	err = service.RegisterChannel(channel)
	require.NoError(t, err)

	// Register second time - should fail
	err = service.RegisterChannel(channel)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
	assert.Contains(t, err.Error(), channel.ID().String())
}

// TestChannelFacade_UnregisterChannel_Success tests successful channel unregistration
func TestChannelFacade_UnregisterChannel_Success(t *testing.T) {
	injector := setupTestInjector()

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	channel := newMockChannel(uuid.New())
	err = service.RegisterChannel(channel)
	require.NoError(t, err)

	// Unregister channel
	err = service.UnregisterChannel(channel.ID())
	assert.NoError(t, err)
}

// TestChannelFacade_UnregisterChannel_NonExistent tests that unregistering a non-existent channel doesn't error
func TestChannelFacade_UnregisterChannel_NonExistent(t *testing.T) {
	injector := setupTestInjector()

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Unregister non-existent channel - should not error
	err = service.UnregisterChannel(uuid.New())
	assert.NoError(t, err)
}

// TestChannelFacade_DisplayMessage_RoutesToTargetChannel tests that DisplayMessage routes to the specific channel by ID
func TestChannelFacade_DisplayMessage_RoutesToTargetChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)

	injector := setupTestInjectorWithLogger(mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register multiple channels
	channels := []*mockChannel{
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
	}

	for _, ch := range channels {
		err = service.RegisterChannel(ch)
		require.NoError(t, err)
	}

	// Send a message to the second channel only
	targetChannel := channels[1]
	msg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgentChat,
		ChannelID: targetChannel.id,
		Content:   "Test message",
		Timestamp: time.Now(),
	}

	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).Times(0) // No warning expected
	service.DisplayMessage(msg)

	// Verify only the target channel received the message
	for i, ch := range channels {
		if i == 1 {
			assert.Equal(t, 1, ch.getMessageCount(), "Target channel should receive exactly one message")
			received := ch.getLastMessage()
			assert.Equal(t, msg.ID, received.ID)
			assert.Equal(t, msg.Content, received.Content)
		} else {
			assert.Equal(t, 0, ch.getMessageCount(), "Non-target channels should not receive the message")
		}
	}
}

// TestChannelFacade_DisplayMessage_ChannelNotFound tests that DisplayMessage logs warning when channel not found
func TestChannelFacade_DisplayMessage_ChannelNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)

	injector := setupTestInjectorWithLogger(mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register one channel
	channel := newMockChannel(uuid.New())
	err = service.RegisterChannel(channel)
	require.NoError(t, err)

	// Send a message to a non-existent channel
	nonExistentChannelID := uuid.New()
	msg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgentChat,
		ChannelID: nonExistentChannelID,
		Content:   "Test message",
		Timestamp: time.Now(),
	}

	// Expect a warning log with the channel ID
	mockLogger.EXPECT().Warn("channel not found", gomock.Any()).
		Do(func(msg string, fields ...zap.Field) {
			// Verify the channel ID field matches
			found := false
			for _, field := range fields {
				if field.Key == "channel_id" && field.String == nonExistentChannelID.String() {
					found = true
				}
			}
			assert.True(t, found, "Should log warning with correct channel_id")
		})

	service.DisplayMessage(msg)

	// Verify no channel received the message
	assert.Equal(t, 0, channel.getMessageCount(), "Registered channel should not receive message for different channel ID")
}

// TestChannelFacade_DisplayMessage_NoBroadcast tests that DisplayMessage does not broadcast to all channels
func TestChannelFacade_DisplayMessage_NoBroadcast(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)

	injector := setupTestInjectorWithLogger(mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register multiple channels
	channels := []*mockChannel{
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
	}

	for _, ch := range channels {
		err = service.RegisterChannel(ch)
		require.NoError(t, err)
	}

	// Send messages to different channels
	for i, ch := range channels {
		msg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeAgentChat,
			ChannelID: ch.id,
			Content:   fmt.Sprintf("Message %d", i),
			Timestamp: time.Now(),
		}
		service.DisplayMessage(msg)
	}

	// Verify each channel received exactly one message (the one addressed to it)
	for i, ch := range channels {
		assert.Equal(t, 1, ch.getMessageCount(), "Channel %d should receive exactly one message", i)
		received := ch.getLastMessage()
		assert.Equal(t, fmt.Sprintf("Message %d", i), received.Content)
	}
}

// TestChannelFacade_DisplayLog_StoresAndBroadcasts tests that DisplayLog stores entry and broadcasts to all channels
func TestChannelFacade_DisplayLog_StoresAndBroadcasts(t *testing.T) {
	injector := setupTestInjector()

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register multiple channels
	channels := []*mockChannel{
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
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
	cfg := &mockConfigService{logBufferSize: 5}
	injector := setupTestInjectorWithConfig(cfg)

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
	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})
	
	// Add mock logger
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	ctx := context.Background()
	channelID := uuid.New()
	result, err := service.SubmitInput(ctx, channelID, "/test args")

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
	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})
	
	// Add mock logger
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	ctx := context.Background()
	channelID := uuid.New()
	result, err := service.SubmitInput(ctx, channelID, "hello world")

	// Since no supervisor agent is available, this should return an error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supervisor agent not available")
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
	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})
	
	// Add mock logger
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	ctx := context.Background()
	channelID := uuid.New()
	result, err := service.SubmitInput(ctx, channelID, "/test")

	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.True(t, result.IsCommand)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "command failed")
}

// TestChannelFacade_GetLogs_ReturnsEntriesAfterSpecifiedTime tests that GetLogs returns entries after specified time
func TestChannelFacade_GetLogs_ReturnsEntriesAfterSpecifiedTime(t *testing.T) {
	injector := setupTestInjector()

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
	injector := setupTestInjector()

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
	injector := setupTestInjector()

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register multiple channels
	channels := []*mockChannel{
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)

	injector := setupTestInjectorWithLogger(mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Create channels first and get their IDs
	channels := make([]*mockChannel, 5)
	for i := 0; i < 5; i++ {
		channels[i] = newMockChannel(uuid.New())
		_ = service.RegisterChannel(channels[i])
	}

	wg := sync.WaitGroup{}

	// Register channels concurrently (these will fail as duplicates, but that's OK for testing)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ch := newMockChannel(uuid.New())
			_ = service.RegisterChannel(ch)
		}(i)
	}

	// Send messages concurrently to specific channels
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Send to one of the registered channels
			targetChannel := channels[i%len(channels)]
			msg := Message{
				ID:        uuid.New(),
				ChannelID: targetChannel.id,
				Content:   fmt.Sprintf("message %d", i),
				Timestamp: time.Now(),
			}
			service.DisplayMessage(msg)
		}(i)
	}

	// Send logs concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			entry := LogEntry{
				Level:     "info",
				Message:   fmt.Sprintf("log %d", i),
				Timestamp: time.Now(),
			}
			service.DisplayLog(entry)
		}(i)
	}

	// Get logs concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = service.GetLogs(time.Time{}, 10)
		}()
	}

	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for concurrent operations")
	}

	// Verify final state is consistent
	logs := service.GetLogs(time.Time{}, 100)
	assert.Greater(t, len(logs), 0, "Should have logs stored")
}

// TestChannelFacade_SubmitInput_RoutesToSupervisorAgent tests that SubmitInput routes to the singleton Supervisor-Agent
func TestChannelFacade_SubmitInput_RoutesToSupervisorAgent(t *testing.T) {
	injector := do.New()

	// Mock command manager that doesn't handle the input
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, input string) (bool, string, error) {
			return false, "", nil
		},
	}
	do.ProvideValue[CommandManagerService](injector, cmdMgr)

	// Create a supervisor agent
	supervisorID := uuid.New()
	supervisor := newMockAgent(supervisorID)

	// Mock registry that returns the supervisor
	mockRegistry := &mockAgentRegistryWithSupervisor{
		supervisor: supervisor,
		err:        nil,
	}
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})
	
	// Add mock logger
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Submit input
	ctx := context.Background()
	channelID := uuid.New()
	result, err := service.SubmitInput(ctx, channelID, "test input")

	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, "Supervisor response", result.Response)
	assert.NoError(t, result.Error)
}

// TestChannelFacade_SubmitInput_NoSupervisorError tests that SubmitInput returns error when supervisor not available
func TestChannelFacade_SubmitInput_NoSupervisorError(t *testing.T) {
	injector := do.New()

	// Mock command manager that doesn't handle the input
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, input string) (bool, string, error) {
			return false, "", nil
		},
	}
	do.ProvideValue[CommandManagerService](injector, cmdMgr)

	// Mock registry that returns error for supervisor
	mockRegistry := &mockAgentRegistryWithSupervisor{
		supervisor: nil,
		err:        fmt.Errorf("no supervisor agent registered"),
	}
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	do.ProvideValue[config.ConfigService](injector, &mockConfigService{logBufferSize: 100})
	
	// Add mock logger
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Submit input should return error
	ctx := context.Background()
	channelID := uuid.New()
	result, err := service.SubmitInput(ctx, channelID, "test input")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supervisor agent not available")
	assert.Empty(t, result)
}
