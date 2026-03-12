package tui

import (
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTUIChannel_OnMessage tests that TUIChannel sends messages to the message channel.
func TestTUIChannel_OnMessage(t *testing.T) {
	// Create a message channel
	msgChan := make(chan channel.Message, 10)

	// Create TUIChannel
	ch := NewTUIChannel(msgChan)
	require.NotNil(t, ch)
	assert.Equal(t, "tui", ch.ID())

	// Create a test message
	testMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeAgentChat,
		Content:   "Hello, world!",
		Timestamp: time.Now(),
		AgentID:   uuid.New(),
		AgentRole: "assistant",
	}

	// Send message
	ch.OnMessage(testMsg)

	// Verify message was received
	select {
	case msg := <-msgChan:
		assert.Equal(t, testMsg.ID, msg.ID)
		assert.Equal(t, testMsg.Content, msg.Content)
		assert.Equal(t, testMsg.Type, msg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received within timeout")
	}
}

// TestTUIChannel_OnMessage_ChannelFull tests that TUIChannel handles full channel gracefully.
func TestTUIChannel_OnMessage_ChannelFull(t *testing.T) {
	// Create a very small message channel
	msgChan := make(chan channel.Message, 1)

	// Create TUIChannel
	ch := NewTUIChannel(msgChan)

	// Fill the channel
	testMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeAgentChat,
		Content:   "First message",
		Timestamp: time.Now(),
	}
	ch.OnMessage(testMsg)

	// Try to send another message (should be dropped)
	secondMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeAgentChat,
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
	// Create TUIChannel with nil channel
	ch := NewTUIChannel(nil)
	require.NotNil(t, ch)

	// Send message (should not panic)
	testMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeAgentChat,
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	ch.OnMessage(testMsg) // Should not panic
}

// TestTUIChannel_OnLog tests that TUIChannel sends log entries as system messages.
func TestTUIChannel_OnLog(t *testing.T) {
	// Create a message channel
	msgChan := make(chan channel.Message, 10)

	// Create TUIChannel
	ch := NewTUIChannel(msgChan)

	// Create a test log entry
	testLog := channel.LogEntry{
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
		assert.Equal(t, channel.MessageTypeSystemInfo, msg.Type)
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
	msgChan := make(chan channel.Message, 10)

	// Create TUIChannel
	ch := NewTUIChannel(msgChan)

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
		assert.Equal(t, channel.MessageTypeSystemInfo, msg.Type)
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
	ch := NewTUIChannel(nil)

	// Set agent info
	agentID := uuid.New()
	ch.SetAgentInfo(agentID, "test-role")

	// Verify agent info is set (we can't access private fields directly,
	// but we can verify it doesn't panic)
	assert.NotNil(t, ch)
}

// TestTUIChannel_ImplementsChannelInterface tests that TUIChannel implements channel.Channel.
func TestTUIChannel_ImplementsChannelInterface(t *testing.T) {
	// This is a compile-time check, but we can verify at runtime too
	var _ channel.Channel = (*TUIChannel)(nil)
	ch := NewTUIChannel(nil)
	assert.Implements(t, (*channel.Channel)(nil), ch)
}
