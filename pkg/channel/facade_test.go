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

	"github.com/denkhaus/gollum/pkg/command"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/testutil"
)

// mockChannel is a test double for Channel interface
type mockChannel struct {
	id       uuid.UUID
	messages []shared.Message
	logs     []shared.LogEntry
	events   []AgentLifecycleEvent
	mu       sync.Mutex
}

func newMockChannel(id uuid.UUID) *mockChannel {
	return &mockChannel{
		id:       id,
		messages: make([]shared.Message, 0),
		logs:     make([]shared.LogEntry, 0),
		events:   make([]AgentLifecycleEvent, 0),
	}
}

func (m *mockChannel) ID() uuid.UUID {
	return m.id
}

func (m *mockChannel) OnMessage(msg shared.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *mockChannel) OnLog(entry shared.LogEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, entry)
}

func (m *mockChannel) OnAgentLifecycle(event AgentLifecycleEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

// Start is a no-op for the mock channel (used in testing)
func (m *mockChannel) Start(ctx context.Context) error {
	return nil
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

func (m *mockChannel) getLastMessage() shared.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return shared.Message{}
	}
	return m.messages[len(m.messages)-1]
}

func (m *mockChannel) getLastLog() shared.LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.logs) == 0 {
		return shared.LogEntry{}
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
	executeFunc func(ctx context.Context, sessionID uuid.UUID, input string) (handled bool, response string, err error)
}

func (m *mockCommandManager) Register(cmd command.Command) error {
	return nil
}

func (m *mockCommandManager) Unregister(name string) error {
	return nil
}

func (m *mockCommandManager) Execute(ctx context.Context, sessionID uuid.UUID, input string) (bool, string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, sessionID, input)
	}
	return false, "", nil
}

func (m *mockCommandManager) List() []command.Command {
	return nil
}

func (m *mockCommandManager) IsCommand(input string) bool {
	return false
}

// mockAgentFactory is a test double for shared.AgentFactory
type mockAgentFactory struct {
	supervisor shared.Agent
	config     *shared.AgentConfig
	err        error
}

func (m *mockAgentFactory) CreateAgent(ctx context.Context, config *shared.AgentConfig) (shared.Agent, error) {
	return nil, nil
}

func (m *mockAgentFactory) CreateSupervisorAgent(ctx context.Context, opts ...shared.SupervisorAgentOption) (shared.Agent, *shared.AgentConfig, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	if m.supervisor != nil {
		return m.supervisor, m.config, nil
	}
	return nil, nil, fmt.Errorf("mock agent factory: no supervisor available")
}

// mockAgent is a test double for shared.Agent
type mockAgent struct {
	id          uuid.UUID
	executeFunc func(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error)
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
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input...)
	}
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

func (m *mockAgent) ToSessionContext() shared.SessionContext {
	return shared.SessionContext{
		AgentID: m.id,
	}
}

// setupTestInjector creates an injector with all mock dependencies for testing
func setupTestInjector(t testing.TB) do.Injector {
	injector := testutil.NewTestInjector(t)

	// Add channel-specific mocks
	ctrl := gomock.NewController(t)
	do.ProvideValue[command.ManagerService](injector, &mockCommandManager{})

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(nil, fmt.Errorf("no supervisor agent registered")).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})

	// Add a mock session manager
	mockSM := session.NewMockSessionManager(ctrl)
	do.ProvideValue[session.SessionManager](injector, mockSM)

	return injector
}

// setupMockConfigService creates a mock config service with standard expectations
func setupMockConfigService(ctrl *gomock.Controller, logBufferSize int) config.ConfigService {
	mockCfg := config.NewMockConfigService(ctrl)
	mockCfg.EXPECT().GetLogLevel().Return("info").AnyTimes()
	mockCfg.EXPECT().IsDevMode().Return(false).AnyTimes()
	mockCfg.EXPECT().GetLoggingConfig().Return(&config.LoggingConfig{
		SessionLogBufferSize: logBufferSize,
		SessionLogEnabled:    true,
	}).AnyTimes()
	mockCfg.EXPECT().GetAgentLimits().Return(&config.AgentLimitsConfig{}).AnyTimes()
	mockCfg.EXPECT().GetEventsConfig().Return(&config.EventsConfig{}).AnyTimes()
	mockCfg.EXPECT().GetSubAgentConfig().Return(&config.SubAgentConfig{}).AnyTimes()
	mockCfg.EXPECT().GetSupervisorConfig().Return(&config.SupervisorConfig{}).AnyTimes()
	mockCfg.EXPECT().GetDatabaseConfig().Return(config.DatabaseConfig{}).AnyTimes()
	return mockCfg
}

// Note: This doesn't use testutil.NewTestInjector because it needs to override the logger service
func setupTestInjectorWithLogger(t testing.TB, logService logger.LoggerService) do.Injector {
	ctrl := gomock.NewController(t)
	injector := do.New()
	do.ProvideValue[command.ManagerService](injector, &mockCommandManager{})

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(nil, fmt.Errorf("no supervisor agent registered")).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)
	do.ProvideValue(injector, logService)

	mockSM := session.NewMockSessionManager(ctrl)
	do.ProvideValue[session.SessionManager](injector, mockSM)

	return injector
}

// TestNewChannelFacade tests that NewChannelFacade creates a valid instance
func TestNewChannelFacade(t *testing.T) {
	injector := setupTestInjector(t)

	service, err := NewChannelFacade(injector)

	require.NoError(t, err)
	assert.NotNil(t, service)

	// Verify it implements ChannelFacade interface
	_, ok := service.(ChannelFacade)
	assert.True(t, ok, "NewChannelFacade should return a ChannelFacade implementation")
}

// TestChannelFacade_RegisterChannel_Success tests successful channel registration
func TestChannelFacade_RegisterChannel_Success(t *testing.T) {
	injector := setupTestInjector(t)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	channel := newMockChannel(uuid.New())
	err = service.(*channelFacadeImpl).registerChannel(channel)

	assert.NoError(t, err)
}

// TestChannelFacade_RegisterChannel_Duplicate tests that registering a duplicate channel returns an error
func TestChannelFacade_RegisterChannel_Duplicate(t *testing.T) {
	injector := setupTestInjector(t)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	channel := newMockChannel(uuid.New())

	// Register first time - should succeed
	err = service.(*channelFacadeImpl).registerChannel(channel)
	require.NoError(t, err)

	// Register second time - should fail
	err = service.(*channelFacadeImpl).registerChannel(channel)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
	assert.Contains(t, err.Error(), channel.ID().String())
}

// TestChannelFacade_UnregisterChannel_Success tests successful channel unregistration
func TestChannelFacade_UnregisterChannel_Success(t *testing.T) {
	injector := setupTestInjector(t)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register a mock channel
	ch := newMockChannel(uuid.New())
	err = service.(*channelFacadeImpl).registerChannel(ch)
	require.NoError(t, err)

	// Unregister channel
	err = service.UnregisterChannel(ch.ID())
	assert.NoError(t, err)
}

// TestChannelFacade_UnregisterChannel_NonExistent tests that unregistering a non-existent channel doesn't error
func TestChannelFacade_UnregisterChannel_NonExistent(t *testing.T) {
	injector := setupTestInjector(t)

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
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)

	injector := setupTestInjectorWithLogger(t, mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register multiple mock channels using private registerChannel
	channels := []*mockChannel{
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
	}

	for _, ch := range channels {
		err = service.(*channelFacadeImpl).registerChannel(ch)
		require.NoError(t, err)
	}

	// Send a message to the second channel only
	msg := shared.Message{
		Role:           gollem.RoleAssistant,
		SessionContext: shared.SessionContext{
			ChannelID: channels[1].id,
		},
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
			assert.Equal(t, msg.SessionContext.SessionID, received.SessionContext.SessionID)
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
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)

	injector := setupTestInjectorWithLogger(t, mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register one channel
	channel := newMockChannel(uuid.New())
	err = service.(*channelFacadeImpl).registerChannel(channel)
	require.NoError(t, err)

	// Send a message to a non-existent channel
	msg := shared.Message{
		Role:           gollem.RoleAssistant,
		SessionContext: shared.SessionContext{},
		Content:        "Test message",
		Timestamp:      time.Now(),
	}

	// Expect a warning log
	mockLogger.EXPECT().Warn("channel not found", gomock.Any())

	service.DisplayMessage(msg)

	// Verify no channel received the message
	assert.Equal(t, 0, channel.getMessageCount(), "Registered channel should not receive message for different channel ID")
}

// TestChannelFacade_DisplayMessage_NoBroadcast tests that DisplayMessage does not broadcast to all channels
func TestChannelFacade_DisplayMessage_NoBroadcast(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)

	injector := setupTestInjectorWithLogger(t, mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register multiple channels
	channels := []*mockChannel{
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
	}

	for _, ch := range channels {
		err = service.(*channelFacadeImpl).registerChannel(ch)
		require.NoError(t, err)
	}

	// Send messages to different channels
	for i, ch := range channels {
		msg := shared.Message{
			Role:           gollem.RoleAssistant,
			SessionContext: shared.SessionContext{
				ChannelID: ch.id,
			},
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

// TestChannelFacade_SubmitInput_SlashCommand tests that SubmitInput routes slash commands to CommandManager
func TestChannelFacade_SubmitInput_SlashCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Mock command manager that handles the command
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, sessionID uuid.UUID, input string) (bool, string, error) {
			if input == "/test args" {
				return true, "command executed", nil
			}
			return false, "", nil
		},
	}
	do.ProvideValue[command.ManagerService](injector, cmdMgr)

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(nil, fmt.Errorf("no supervisor agent registered")).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)

	// Add mock logger
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	sessionCtx := &shared.SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		AgentID:   uuid.New(),
	}
	session := &shared.Session{
		SessionContext: *sessionCtx,
		Context:        context.Background(),
		CancelFunc:     func() {},
	}
	result, err := service.SubmitInput(session, "/test args")

	require.NoError(t, err)
	assert.True(t, result.Handled, "Slash command should be handled")
	assert.True(t, result.IsCommand, "Input should be marked as command")
	assert.Equal(t, "command executed", result.Response)
	assert.NoError(t, result.Error)
}

// TestChannelFacade_SubmitInput_NonCommand_NoAgentRegistry tests that SubmitInput returns error for non-command when no agent routing
func TestChannelFacade_SubmitInput_NonCommand_NoAgentRouting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Mock command manager that doesn't handle the input
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, sessionID uuid.UUID, input string) (bool, string, error) {
			return false, "", nil
		},
	}

	do.ProvideValue[command.ManagerService](injector, cmdMgr)

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(nil, fmt.Errorf("no supervisor agent registered")).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})

	mockSessionManager := session.NewMockSessionManager(ctrl)
	testSession := &shared.Session{
		Context: context.Background(),
	}
	mockSessionManager.EXPECT().GetOrCreateSession(gomock.Any(), gomock.Any()).Return(testSession, nil).AnyTimes()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[session.SessionManager](injector, mockSessionManager)
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	sessionCtx := &shared.SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		AgentID:   uuid.New(),
	}
	session := &shared.Session{
		SessionContext: *sessionCtx,
		Context:        context.Background(),
		CancelFunc:     func() {},
	}
	result, err := service.SubmitInput(session, "hello world")

	// Since no supervisor agent is available, this should return an error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get/create supervisor")
	assert.Empty(t, result)
}

// TestChannelFacade_SubmitInput_CommandError tests that SubmitInput returns command errors
func TestChannelFacade_SubmitInput_CommandError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Mock command manager that returns an error
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, sessionID uuid.UUID, input string) (bool, string, error) {
			return true, "", errors.New("command failed")
		},
	}

	do.ProvideValue[command.ManagerService](injector, cmdMgr)

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(nil, fmt.Errorf("no supervisor agent registered")).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)

	// Add mock logger
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	sessionCtx := &shared.SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		AgentID:   uuid.New(),
	}
	session := &shared.Session{
		SessionContext: *sessionCtx,
		Context:        context.Background(),
		CancelFunc:     func() {},
	}
	result, err := service.SubmitInput(session, "/test args")

	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.True(t, result.IsCommand)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "command failed")
}
func TestChannelFacade_NotifyAgentLifecycle_TargetsSpecificChannel(t *testing.T) {
	injector := setupTestInjector(t)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Register multiple channels
	channel1 := newMockChannel(uuid.New())
	channel2 := newMockChannel(uuid.New())
	channel3 := newMockChannel(uuid.New())

	err = service.(*channelFacadeImpl).registerChannel(channel1)
	require.NoError(t, err)
	err = service.(*channelFacadeImpl).registerChannel(channel2)
	require.NoError(t, err)
	err = service.(*channelFacadeImpl).registerChannel(channel3)
	require.NoError(t, err)

	// Notify agent lifecycle event for channel2 only
	agentID := uuid.New()
	role := "tester"
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")
	service.NotifyAgentLifecycle(agentID, channel2.id, sessionID, role, true)

	// Verify only channel2 received the event
	assert.Equal(t, 1, channel2.getEventCount(), "Target channel should receive exactly one lifecycle event")
	received := channel2.getLastEvent()
	assert.Equal(t, agentID, received.AgentID)
	assert.Equal(t, role, received.Role)
	assert.Equal(t, sessionID.String(), received.SessionID)
	assert.Equal(t, channel2.id, received.ChannelID)
	assert.True(t, received.Added)

	// Verify other channels did NOT receive the event
	assert.Equal(t, 0, channel1.getEventCount(), "Other channels should not receive lifecycle event")
	assert.Equal(t, 0, channel3.getEventCount(), "Other channels should not receive lifecycle event")
}

// TestChannelFacade_Concurrency tests that concurrent access is safe
func TestChannelFacade_Concurrency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)

	injector := setupTestInjectorWithLogger(t, mockLogger)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Create channels first and get their IDs
	channels := make([]*mockChannel, 5)
	for i := range 5 {
		channels[i] = newMockChannel(uuid.New())
		_ = service.(*channelFacadeImpl).registerChannel(channels[i])
	}

	wg := sync.WaitGroup{}

	// Register channels concurrently (these will fail as duplicates, but that's OK for testing)
	for i := range 5 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ch := newMockChannel(uuid.New())
			_ = service.(*channelFacadeImpl).registerChannel(ch)
		}(i)
	}

	// Send messages concurrently to specific channels
	for i := range 10 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Send to one of the registered channels (round-robin)
			targetChannel := channels[i%len(channels)]
			msg := shared.Message{
				SessionContext: shared.SessionContext{
					ChannelID: targetChannel.id,
				},
				Content:   fmt.Sprintf("message %d", i),
				Timestamp: time.Now(),
			}
			service.DisplayMessage(msg)
		}(i)
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

	// Verify channels received their messages
	totalMessages := 0
	for _, ch := range channels {
		totalMessages += ch.getMessageCount()
	}
	assert.Greater(t, totalMessages, 0, "Channels should have received messages")
}

// TestChannelFacade_SubmitInput_RoutesToSupervisorAgent tests that SubmitInput routes to the singleton Supervisor-Agent
func TestChannelFacade_SubmitInput_RoutesToSupervisorAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Mock command manager that doesn't handle the input
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, sessionID uuid.UUID, input string) (bool, string, error) {
			return false, "", nil
		},
	}
	do.ProvideValue[command.ManagerService](injector, cmdMgr)

	// Create a supervisor agent
	supervisorID := uuid.New()
	supervisor := newMockAgent(supervisorID)

	// Mock registry that returns the supervisor
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(supervisor, nil).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	// Mock agent factory that returns the supervisor
	mockAgentFactory := &mockAgentFactory{
		supervisor: supervisor,
		config:     &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: supervisorID}},
		err:        nil,
	}
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

	// Set up mocks
	mockSessionManager := session.NewMockSessionManager(ctrl)
	testChannelID := uuid.New()
	testSession := &shared.Session{
		Context: context.Background(),
	}
	mockSessionManager.EXPECT().GetOrCreateSession(gomock.Any(), gomock.Any()).Return(testSession, nil).AnyTimes()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[session.SessionManager](injector, mockSessionManager)
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Submit input
	session := &shared.Session{
		SessionContext: shared.SessionContext{
			SessionID: testSession.SessionContext.SessionID,
			ChannelID: testChannelID,
			AgentID:   uuid.New(),
		},
		Context:    context.Background(),
		CancelFunc: func() {},
	}
	result, err := service.SubmitInput(session, "test input")
	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, "Supervisor response", result.Response)
	assert.NoError(t, result.Error)
}

// TestChannelFacade_SubmitInput_NoSupervisorError tests that SubmitInput returns error when supervisor not available
func TestChannelFacade_SubmitInput_NoSupervisorError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Mock command manager that doesn't handle the input
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, sessionID uuid.UUID, input string) (bool, string, error) {
			return false, "", nil
		},
	}
	do.ProvideValue[command.ManagerService](injector, cmdMgr)

	// Mock registry that returns error for supervisor
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(nil, fmt.Errorf("no supervisor agent registered")).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})

	mockSessionManager := session.NewMockSessionManager(ctrl)
	testSession := &shared.Session{
		Context: context.Background(),
	}
	mockSessionManager.EXPECT().GetOrCreateSession(gomock.Any(), gomock.Any()).Return(testSession, nil).AnyTimes()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[session.SessionManager](injector, mockSessionManager)
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Submit input should return error

	sessionCtx := &shared.SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		AgentID:   uuid.New(),
	}
	session := &shared.Session{
		SessionContext: *sessionCtx,
		Context:        context.Background(),
		CancelFunc:     func() {},
	}
	result, err := service.SubmitInput(session, "/test args")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get/create supervisor")
	assert.Empty(t, result)
}

// TestChannelFacade_CancelInput_SessionNotFound tests that CancelInput returns error for non-existent session
func TestChannelFacade_CancelInput_SessionNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()
	do.ProvideValue[command.ManagerService](injector, &mockCommandManager{})

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(nil, fmt.Errorf("no supervisor agent registered")).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)

	mockSessionManager := session.NewMockSessionManager(ctrl)
	// Expect GetSession to return not found
	mockSessionManager.EXPECT().GetSession(uuid.Nil).Return(nil, false)

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[session.SessionManager](injector, mockSessionManager)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Try to cancel a non-existent session
	err = service.CancelInput(uuid.Nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestChannelFacade_SubmitInput_ExecuteError tests that SubmitInput handles supervisor execution errors
func TestChannelFacade_SubmitInput_ExecuteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Mock command manager that doesn't handle the input
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, sessionID uuid.UUID, input string) (bool, string, error) {
			return false, "", nil
		},
	}
	do.ProvideValue[command.ManagerService](injector, cmdMgr)

	// Create a supervisor agent that returns an error
	supervisorID := uuid.New()
	supervisor := &mockAgent{
		id: supervisorID,
		executeFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
			return nil, fmt.Errorf("supervisor execution failed")
		},
	}

	// Mock registry that returns the supervisor
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(supervisor, nil).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	// Mock agent factory that returns the supervisor
	mockAgentFactory := &mockAgentFactory{
		supervisor: supervisor,
		config:     &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: supervisorID}},
		err:        nil,
	}
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)
	defer ctrl.Finish()

	mockSessionManager := session.NewMockSessionManager(ctrl)
	testSession := &shared.Session{
		Context: context.Background(),
	}
	mockSessionManager.EXPECT().GetOrCreateSession(gomock.Any(), gomock.Any()).Return(testSession, nil).AnyTimes()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[session.SessionManager](injector, mockSessionManager)
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Submit input should handle the error gracefully

	sessionCtx := &shared.SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		AgentID:   uuid.New(),
	}
	session := &shared.Session{
		SessionContext: *sessionCtx,
		Context:        context.Background(),
		CancelFunc:     func() {},
	}
	result, err := service.SubmitInput(session, "/test args")

	require.NoError(t, err) // No error returned, error is in result
	assert.True(t, result.Handled)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "supervisor execution failed")
}

// TestChannelFacade_SubmitInput_ExecuteEmptyResponse tests that SubmitInput handles empty supervisor response
func TestChannelFacade_SubmitInput_ExecuteEmptyResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Mock command manager that doesn't handle the input
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, sessionID uuid.UUID, input string) (bool, string, error) {
			return false, "", nil
		},
	}
	do.ProvideValue[command.ManagerService](injector, cmdMgr)

	// Create a supervisor agent that returns empty response
	supervisorID := uuid.New()
	supervisor := &mockAgent{
		id: supervisorID,
		executeFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
			return &gollem.ExecuteResponse{
				Texts: []string{}, // Empty response
			}, nil
		},
	}

	// Mock registry that returns the supervisor
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(supervisor, nil).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	// Mock agent factory that returns the supervisor
	mockAgentFactory := &mockAgentFactory{
		supervisor: supervisor,
		config:     &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: supervisorID}},
		err:        nil,
	}
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

	mockSessionManager := session.NewMockSessionManager(ctrl)
	testSession := &shared.Session{
		Context: context.Background(),
	}
	mockSessionManager.EXPECT().GetOrCreateSession(gomock.Any(), gomock.Any()).Return(testSession, nil).AnyTimes()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[session.SessionManager](injector, mockSessionManager)
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Submit input should handle empty response

	sessionCtx := &shared.SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		AgentID:   uuid.New(),
	}
	session := &shared.Session{
		SessionContext: *sessionCtx,
		Context:        context.Background(),
		CancelFunc:     func() {},
	}
	result, err := service.SubmitInput(session, "/test args")

	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Empty(t, result.Response, "Empty response should be handled")
	assert.NoError(t, result.Error)
}

// TestChannelFacade_SubmitInput_ExecuteNilResponse tests that SubmitInput handles nil supervisor response
func TestChannelFacade_SubmitInput_ExecuteNilResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Mock command manager that doesn't handle the input
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, sessionID uuid.UUID, input string) (bool, string, error) {
			return false, "", nil
		},
	}
	do.ProvideValue[command.ManagerService](injector, cmdMgr)

	// Create a supervisor agent that returns nil response
	supervisorID := uuid.New()
	supervisor := &mockAgent{
		id: supervisorID,
		executeFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
			return nil, nil // Nil response
		},
	}

	// Mock registry that returns the supervisor
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(supervisor, nil).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	// Mock agent factory that returns the supervisor
	mockAgentFactory := &mockAgentFactory{
		supervisor: supervisor,
		config:     &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: supervisorID}},
		err:        nil,
	}
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

	mockSessionManager := session.NewMockSessionManager(ctrl)
	testSession := &shared.Session{
		Context: context.Background(),
	}
	mockSessionManager.EXPECT().GetOrCreateSession(gomock.Any(), gomock.Any()).Return(testSession, nil).AnyTimes()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[session.SessionManager](injector, mockSessionManager)
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Submit input should handle nil response

	sessionCtx := &shared.SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		AgentID:   uuid.New(),
	}
	session := &shared.Session{
		SessionContext: *sessionCtx,
		Context:        context.Background(),
		CancelFunc:     func() {},
	}
	result, err := service.SubmitInput(session, "/test args")

	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Empty(t, result.Response, "Nil response should be handled")
	assert.NoError(t, result.Error)
}

// TestChannelFacade_SubmitInput_MultipleTextsInResponse tests that SubmitInput joins multiple response texts
func TestChannelFacade_SubmitInput_MultipleTextsInResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	injector := do.New()

	// Mock command manager that doesn't handle the input
	cmdMgr := &mockCommandManager{
		executeFunc: func(ctx context.Context, sessionID uuid.UUID, input string) (bool, string, error) {
			return false, "", nil
		},
	}
	do.ProvideValue[command.ManagerService](injector, cmdMgr)

	// Create a supervisor agent that returns multiple texts
	supervisorID := uuid.New()
	supervisor := &mockAgent{
		id: supervisorID,
		executeFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
			return &gollem.ExecuteResponse{
				Texts: []string{"Line 1", "Line 2", "Line 3"},
			}, nil
		},
	}

	// Mock registry that returns the supervisor
	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(supervisor, nil).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	// Mock agent factory that returns the supervisor
	mockAgentFactory := &mockAgentFactory{
		supervisor: supervisor,
		config:     &shared.AgentConfig{SessionContext: shared.SessionContext{AgentID: supervisorID}},
		err:        nil,
	}
	do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

	mockSessionManager := session.NewMockSessionManager(ctrl)
	testSession := &shared.Session{
		Context: context.Background(),
	}
	mockSessionManager.EXPECT().GetOrCreateSession(gomock.Any(), gomock.Any()).Return(testSession, nil).AnyTimes().AnyTimes()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().SetLogForwarder(gomock.Any()).Times(1)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[session.SessionManager](injector, mockSessionManager)
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)

	service, err := NewChannelFacade(injector)
	require.NoError(t, err)

	// Submit input should join multiple texts

	sessionCtx := &shared.SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		AgentID:   uuid.New(),
	}
	session := &shared.Session{
		SessionContext: *sessionCtx,
		Context:        context.Background(),
		CancelFunc:     func() {},
	}
	result, err := service.SubmitInput(session, "/test args")

	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, "Line 1\nLine 2\nLine 3", result.Response, "Multiple texts should be joined with newlines")
	assert.NoError(t, result.Error)
}

// setupTestInjectorWithSessionManager creates an injector with a session manager mock with proper expectations
func setupTestInjectorWithSessionManager(t testing.TB, ctrl *gomock.Controller) do.Injector {
	injector := testutil.NewTestInjector(t)

	// Add channel-specific mocks
	do.ProvideValue[command.ManagerService](injector, &mockCommandManager{})

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(nil, fmt.Errorf("no supervisor agent registered")).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})

	// Add a mock session manager
	mockSM := session.NewMockSessionManager(ctrl)
	do.ProvideValue[session.SessionManager](injector, mockSM)

	return injector
}

// TestChannelFacade_SubmitInput_DelegatesToHandler tests that facade properly delegates to InputHandler
func TestChannelFacade_SubmitInput_DelegatesToHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHandler := NewMockInputHandler(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	// Create a minimal facade implementation for testing
	facade := &channelFacadeImpl{
		inputHandler: mockHandler,
		channels:     make(map[uuid.UUID]Channel),
		logger:       mockLogger,
	}

	sessionCtx := shared.NewSessionContext(uuid.Nil, uuid.Nil, uuid.Nil, "")
	session := &shared.Session{
		SessionContext: *sessionCtx,
		Context:        context.Background(),
		CancelFunc:     func() {},
	}
	input := "hello"

	expectedResult := &InputResult{Handled: true, Response: "Hi there"}
	mockHandler.EXPECT().HandleInput(session, input).Return(expectedResult, nil)

	result, err := facade.SubmitInput(session, input)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

// TestChannelFacade_CancelInput_DelegatesToHandler tests that facade properly delegates CancelInput to InputHandler
func TestChannelFacade_CancelInput_DelegatesToHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHandler := NewMockInputHandler(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	// Create a minimal facade implementation for testing
	mockFacade := &channelFacadeImpl{
		inputHandler: mockHandler,
		channels:     make(map[uuid.UUID]Channel),
		logger:       mockLogger,
	}

	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	mockHandler.EXPECT().CancelInput(sessionID).Return(nil)

	err := mockFacade.CancelInput(sessionID)

	require.NoError(t, err)
}
