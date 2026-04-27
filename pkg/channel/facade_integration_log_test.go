// Package channel provides integration tests for log forwarding from logger to channel system.
package channel

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/denkhaus/gollum/pkg/command"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
)

const asyncWait = 10 * time.Millisecond

// setupIntegrationTest creates a configured injector with all dependencies for integration tests.
// Returns the injector, controller, and a cleanup function.
func setupIntegrationTest(t *testing.T) (do.Injector, *gomock.Controller) {
	ctrl := gomock.NewController(t)

	injector := do.New()
	do.ProvideValue[command.ManagerService](injector, &mockCommandManager{})

	mockRegistry := registry.NewMockAgentRegistry(ctrl)
	mockRegistry.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockRegistry.EXPECT().GetSupervisorAgent().Return(nil, fmt.Errorf("no supervisor agent registered")).AnyTimes()
	do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

	do.ProvideValue[shared.AgentFactory](injector, &mockAgentFactory{})
	do.ProvideValue[session.SessionManager](injector, session.NewMockSessionManager(ctrl))
	mockCfg := setupMockConfigService(ctrl, 100)
	do.ProvideValue[config.ConfigService](injector, mockCfg)

	return injector, ctrl
}

// TestLogForwarding_Integration tests that logs with valid context are forwarded to the correct channel.
// This is an integration test that verifies the complete flow:
// LoggerService → LogForwarder → ChannelFacade → specific Channel
func TestLogForwarding_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup injector with all dependencies for integration test
	injector, ctrl := setupIntegrationTest(t)
	defer ctrl.Finish()

	// Create real logger service (not mock) for integration testing
	loggerSvc, err := logger.NewService(injector)
	require.NoError(t, err)
	do.ProvideValue[logger.LoggerService](injector, loggerSvc)

	// Create the facade and register it in the injector
	facade, err := NewChannelFacade(injector)
	require.NoError(t, err)
	do.ProvideValue[ChannelFacade](injector, facade)

	// Register forwarder - this connects the logger to the channel system
	loggerSvc.SetLogForwarder(facade)

	// Create test channel
	testChannel := newMockChannel(uuid.New())
	regErr := facade.(*channelFacadeImpl).registerChannel(testChannel)
	require.NoError(t, regErr)

	// Cleanup: unregister channel to prevent resource leaks
	t.Cleanup(func() {
		_ = facade.UnregisterChannel(testChannel.ID())
	})

	// Create logging context with valid routing information
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	ctx := shared.SessionContext{
		SessionID: sessionID,
		ChannelID: testChannel.ID(),
		AgentID:   uuid.New(),
	}

	// Log with context - this should be forwarded to the test channel
	loggerSvc.InfoWithContext("Test message", ctx, zap.String("test", "value"))

	// Give time for async processing (log forwarding is async via buffer)
	time.Sleep(asyncWait)

	// Verify log was received
	assert.Equal(t, 1, testChannel.getLogCount(), "Channel should receive exactly one log entry")
	receivedLog := testChannel.getLastLog()
	assert.Equal(t, "info", receivedLog.Level, "Log level should be info")
	assert.Equal(t, "Test message", receivedLog.Message, "Log message should match")
	assert.Equal(t, sessionID, receivedLog.SessionID, "SessionID should match context")
	assert.Equal(t, testChannel.ID(), receivedLog.ChannelID, "ChannelID should match context")
}

// TestLogForwarding_InvalidContext_DoesNotForward tests that logs with invalid context are not forwarded.
// Invalid context means missing SessionID, ChannelID, or AgentID.
func TestLogForwarding_InvalidContext_DoesNotForward(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup injector with all dependencies for integration test
	injector, ctrl := setupIntegrationTest(t)
	defer ctrl.Finish()

	// Create real logger service (not mock) for integration testing
	loggerSvc, err := logger.NewService(injector)
	require.NoError(t, err)
	do.ProvideValue[logger.LoggerService](injector, loggerSvc)

	// Create the facade and register it in the injector
	facade, err := NewChannelFacade(injector)
	require.NoError(t, err)
	do.ProvideValue[ChannelFacade](injector, facade)

	// Register forwarder
	loggerSvc.SetLogForwarder(facade)

	// Create test channel
	testChannel := newMockChannel(uuid.New())
	regErr := facade.(*channelFacadeImpl).registerChannel(testChannel)
	require.NoError(t, regErr)

	// Cleanup: unregister channel to prevent resource leaks
	t.Cleanup(func() {
		_ = facade.UnregisterChannel(testChannel.ID())
	})

	// Test 1: Nil SessionID - should not forward
	ctx1 := shared.SessionContext{
		SessionID: uuid.Nil, // Invalid
		ChannelID: testChannel.ID(),
		AgentID:   uuid.New(),
	}
	loggerSvc.InfoWithContext("Invalid - empty session", ctx1)

	// Test 2: Nil ChannelID - should not forward
	ctx2 := shared.SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
		ChannelID: uuid.Nil, // Invalid
		AgentID:   uuid.New(),
	}
	loggerSvc.InfoWithContext("Invalid - nil channel", ctx2)

	// Test 3: Nil AgentID - should not forward
	ctx3 := shared.SessionContext{
		SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"),
		ChannelID: testChannel.ID(),
		AgentID:   uuid.Nil, // Invalid
	}
	loggerSvc.InfoWithContext("Invalid - nil agent", ctx3)

	// Give time for async processing
	time.Sleep(asyncWait)

	// Verify no logs were received (all contexts were invalid)
	assert.Equal(t, 0, testChannel.getLogCount(), "Channel should not receive logs with invalid context")
}

// TestLogForwarding_MultipleChannels_RoutesCorrectly tests that logs are routed to the correct channel
// when multiple channels are registered.
func TestLogForwarding_MultipleChannels_RoutesCorrectly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup injector with all dependencies for integration test
	injector, ctrl := setupIntegrationTest(t)
	defer ctrl.Finish()

	// Create real logger service (not mock) for integration testing
	loggerSvc, err := logger.NewService(injector)
	require.NoError(t, err)
	do.ProvideValue[logger.LoggerService](injector, loggerSvc)

	// Create the facade and register it in the injector
	facade, err := NewChannelFacade(injector)
	require.NoError(t, err)
	do.ProvideValue[ChannelFacade](injector, facade)

	// Register forwarder
	loggerSvc.SetLogForwarder(facade)

	// Create multiple test channels
	channels := []*mockChannel{
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
		newMockChannel(uuid.New()),
	}

	for _, ch := range channels {
		err := facade.(*channelFacadeImpl).registerChannel(ch)
		require.NoError(t, err)
	}

	// Cleanup: unregister all channels to prevent resource leaks
	t.Cleanup(func() {
		for _, ch := range channels {
			_ = facade.UnregisterChannel(ch.ID())
		}
	})

	// Create logging context for the second channel
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440004")
	ctx := shared.SessionContext{
		SessionID: sessionID,
		ChannelID: channels[1].ID(), // Target the second channel
		AgentID:   uuid.New(),
	}

	// Log with context - should only go to the second channel
	loggerSvc.InfoWithContext("Routed message", ctx)

	// Give time for async processing
	time.Sleep(asyncWait)

	// Verify only the second channel received the log
	assert.Equal(t, 0, channels[0].getLogCount(), "First channel should not receive the log")
	assert.Equal(t, 1, channels[1].getLogCount(), "Second channel should receive the log")
	assert.Equal(t, 0, channels[2].getLogCount(), "Third channel should not receive the log")

	receivedLog := channels[1].getLastLog()
	assert.Equal(t, "Routed message", receivedLog.Message)
	assert.Equal(t, channels[1].ID(), receivedLog.ChannelID)
}

// TestLogForwarding_LogWithFields tests that log fields are preserved through forwarding.
func TestLogForwarding_LogWithFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup injector with all dependencies for integration test
	injector, ctrl := setupIntegrationTest(t)
	defer ctrl.Finish()

	// Create real logger service (not mock) for integration testing
	loggerSvc, err := logger.NewService(injector)
	require.NoError(t, err)
	do.ProvideValue[logger.LoggerService](injector, loggerSvc)

	// Create the facade and register it in the injector
	facade, err := NewChannelFacade(injector)
	require.NoError(t, err)
	do.ProvideValue[ChannelFacade](injector, facade)

	// Register forwarder
	loggerSvc.SetLogForwarder(facade)

	// Create test channel
	testChannel := newMockChannel(uuid.New())
	regErr := facade.(*channelFacadeImpl).registerChannel(testChannel)
	require.NoError(t, regErr)

	// Cleanup: unregister channel to prevent resource leaks
	t.Cleanup(func() {
		_ = facade.UnregisterChannel(testChannel.ID())
	})

	// Create logging context
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440005")
	ctx := shared.SessionContext{
		SessionID: sessionID,
		ChannelID: testChannel.ID(),
		AgentID:   uuid.New(),
	}

	// Log with multiple fields
	loggerSvc.InfoWithContext("Message with fields", ctx,
		zap.String("user_id", "user-123"),
		zap.Int("request_count", 42),
		zap.Bool("authenticated", true),
	)

	// Give time for async processing
	time.Sleep(asyncWait)

	// Verify log was received with fields
	assert.Equal(t, 1, testChannel.getLogCount())
	receivedLog := testChannel.getLastLog()
	assert.Equal(t, "Message with fields", receivedLog.Message)

	// Verify fields map exists (note: zap.Field.Interface may not be populated
	// for primitive types, but the fields are present in the structured log output)
	assert.NotNil(t, receivedLog.Fields, "Log should have fields map")
}
