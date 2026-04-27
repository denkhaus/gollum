package tui

import (
	"github.com/m-mizutani/gollem"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestTUIChannel_OnMessage tests that TUIChannel sends messages to the message channel.
func TestTUIChannel_OnMessage(t *testing.T) {
	// Create a message channel
	msgChan := make(chan shared.Message, 10)

	// Create TUIChannel
	ch := NewTUIChannel(WithChannelMessageChan(msgChan))
	require.NotNil(t, ch)
	assert.NotEqual(t, uuid.Nil, ch.ID())

	// Create a test message
	testMsg := shared.Message{
		ID:             uuid.New(),
		Role: gollem.RoleAssistant,
		Content:        "Hello, world!",
		Timestamp:      time.Now(),
		SessionContext: shared.SessionContext{AgentID: uuid.New()},
		AgentRole:      "assistant",
	}

	// Send message
	ch.OnMessage(testMsg)

	// Verify message was received
	select {
	case msg := <-msgChan:
		assert.Equal(t, testMsg.ID, msg.ID)
		assert.Equal(t, testMsg.Content, msg.Content)
		assert.Equal(t, testMsg.Role, msg.Role)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received within timeout")
	}
}

// TestTUIChannel_OnMessage_ChannelFull tests that TUIChannel handles full channel gracefully.
func TestTUIChannel_OnMessage_ChannelFull(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a very small message channel
	msgChan := make(chan shared.Message, 1)

	// Create mock logger
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()

	// Create TUIChannel
	ch := NewTUIChannel(WithChannelMessageChan(msgChan), WithChannelLogger(mockLogger))

	// Fill the channel
	testMsg := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleAssistant,
		Content:   "First message",
		Timestamp: time.Now(),
	}
	ch.OnMessage(testMsg)

	// Try to send another message (should be dropped)
	secondMsg := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleAssistant,
		Content:   "Second message",
		Timestamp: time.Now(),
	}
	ch.OnMessage(secondMsg) // Should not block

	// Verify only first message is in channel
	select {
	case msg := <-msgChan:
		assert.Equal(t, testMsg.ID, msg.ID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("First message not received")
	}

	// Second message should not be in channel
	select {
	case <-msgChan:
		t.Fatal("Second message should have been dropped")
	case <-time.After(50 * time.Millisecond):
		// Expected - channel is empty
	}
}

// TestTUIChannel_OnMessage_NoChannel tests that TUIChannel handles nil channel gracefully.
func TestTUIChannel_OnMessage_NoChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock logger to avoid nil pointer
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()

	// Create TUIChannel with nil channel but with logger
	ch := NewTUIChannel(WithChannelLogger(mockLogger))
	require.NotNil(t, ch)

	// Send message (should not panic)
	testMsg := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleAssistant,
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	ch.OnMessage(testMsg) // Should not panic
}

// TestTUIChannel_OnLog tests that TUIChannel sends log entries as system messages.
func TestTUIChannel_OnLog(t *testing.T) {
	// Create a message channel
	msgChan := make(chan shared.Message, 10)

	// Create TUIChannel
	ch := NewTUIChannel(WithChannelMessageChan(msgChan))

	// Create a test log entry
	testLog := shared.LogEntry{
		Level:     "info",
		Message:   "Test log message",
		Timestamp: time.Now(),
		Fields: map[string]any{
			"component": "test-component",
		},
	}

	// Send log entry
	ch.OnLog(testLog)

	// Verify system message was received
	select {
	case msg := <-msgChan:
		assert.Equal(t, gollem.RoleSystem, msg.Role)
		assert.Contains(t, msg.Content, "[info]")
		assert.Contains(t, msg.Content, "test-component")
		assert.Contains(t, msg.Content, "Test log message")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received within timeout")
	}
}

// TestTUIChannel_OnAgentLifecycle tests that TUIChannel sends lifecycle events as system messages.
func TestTUIChannel_OnAgentLifecycle(t *testing.T) {
	// Create a message channel
	msgChan := make(chan shared.Message, 10)

	// Create TUIChannel
	ch := NewTUIChannel(WithChannelMessageChan(msgChan))

	// Create a test lifecycle event
	agentID := uuid.New()
	testEvent := channel.AgentLifecycleEvent{
		AgentID: agentID,
		Role:    "assistant",
		Added:   true,
	}

	// Send lifecycle event
	ch.OnAgentLifecycle(testEvent)

	// Verify system message was received
	select {
	case msg := <-msgChan:
		assert.Equal(t, gollem.RoleSystem, msg.Role)
		assert.Contains(t, msg.Content, "Agent added")
		assert.Contains(t, msg.Content, agentID.String())
		assert.Contains(t, msg.Content, "assistant")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received within timeout")
	}
}

// TestTUIChannel_SetAgentInfo tests that agent info can be set.
func TestTUIChannel_SetAgentInfo(t *testing.T) {
	// Create TUIChannel
	ch := NewTUIChannel()

	// Set agent info
	agentID := uuid.New()
	ch.SetAgentInfo(agentID, "test-role")

	// Verify agent info is set (we can't access private fields directly,
	// but we can verify it doesn't panic)
	assert.NotNil(t, ch)
}

// TestTUIChannel_ID_ReturnsUUID tests that TUIChannel returns a valid UUID.
func TestTUIChannel_ID_ReturnsUUID(t *testing.T) {
	channel := NewTUIChannel()

	id := channel.ID()

	assert.NotEqual(t, uuid.Nil, id)
	// Should be a valid UUID v4
	assert.Equal(t, uuid.Version(4), id.Version())
}

// TestTUIChannel_ImplementsChannelInterface tests that TUIChannel implements channel.Channel.
func TestTUIChannel_ImplementsChannelInterface(t *testing.T) {
	// This is a compile-time check, but we can verify at runtime too
	var _ channel.Channel = (*TUIChannel)(nil)
	ch := NewTUIChannel()
	assert.Implements(t, (*channel.Channel)(nil), ch)
}

// TestNewTUIChannel_WithOptions tests creating TUIChannel with options.
func TestNewTUIChannel_WithOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockRenderer := markdown.NewMockRenderer(ctrl)
	msgChan := make(chan shared.Message, 10)

	ch := NewTUIChannel(
		WithChannelMessageChan(msgChan),
		WithChannelLogger(mockLogger),
		WithChannelRenderer(mockRenderer),
	)

	assert.NotNil(t, ch)
	// Verify message channel is set
	got := ch.GetMessageChan()
	require.NotNil(t, got)
	got <- shared.Message{}
	<-msgChan
	// Verify logger and renderer
	assert.Equal(t, mockLogger, ch.GetLogger())
	assert.Equal(t, mockRenderer, ch.GetRenderer())
}

// TestNewTUIChannel_NoOptions tests creating TUIChannel without options.
func TestNewTUIChannel_NoOptions(t *testing.T) {
	ch := NewTUIChannel()
	assert.NotNil(t, ch)
	// Defaults should be nil/zero values
	assert.Nil(t, ch.GetMessageChan())
	assert.Nil(t, ch.GetLogger())
	assert.Nil(t, ch.GetRenderer())
}

// TestIdentifier tests that the Identifier constant is defined correctly.
func TestIdentifier(t *testing.T) {
	id := Identifier
	assert.Equal(t, channel.ChannelIdentifier("tui"), id)
	assert.NotEmpty(t, string(id))
}
