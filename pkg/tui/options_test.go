// pkg/tui/options_test.go
package tui

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTUIOption_WithChannelMessageChan(t *testing.T) {
	msgChan := make(chan channel.Message, 10)
	ch := NewTUIChannel()

	opt := WithChannelMessageChan(msgChan)
	err := opt.Apply(ch)

	require.NoError(t, err)
	// Verify channel is set by sending a test message
	got := ch.GetMessageChan()
	require.NotNil(t, got)
	got <- channel.Message{} // Test that we can send
	<-msgChan               // Drain the test message
}

func TestTUIOption_WithChannelLogger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	ch := NewTUIChannel()

	opt := WithChannelLogger(mockLogger)
	err := opt.Apply(ch)

	require.NoError(t, err)
	assert.Equal(t, mockLogger, ch.GetLogger())
}

func TestTUIOption_WithChannelRenderer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRenderer := markdown.NewMockRenderer(ctrl)
	ch := NewTUIChannel()

	opt := WithChannelRenderer(mockRenderer)
	err := opt.Apply(ch)

	require.NoError(t, err)
	assert.Equal(t, mockRenderer, ch.GetRenderer())
}

func TestTUIOption_WrongChannelType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	opt := WithChannelMessageChan(make(chan channel.Message))

	// Apply to wrong channel type
	wrongCh := &mockOtherChannel{id: uuid.New()}
	err := opt.Apply(wrongCh)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TUI option applied to wrong channel type")
}

func TestTUIOptions_Multiple(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockRenderer := markdown.NewMockRenderer(ctrl)
	msgChan := make(chan channel.Message, 10)

	ch := NewTUIChannel()

	// Apply multiple options
	opts := []TUIOption{
		WithChannelMessageChan(msgChan),
		WithChannelLogger(mockLogger),
		WithChannelRenderer(mockRenderer),
	}

	for _, opt := range opts {
		err := opt.Apply(ch)
		require.NoError(t, err)
	}

	// Verify message channel is set
	got := ch.GetMessageChan()
	require.NotNil(t, got)
	got <- channel.Message{} // Test that we can send
	<-msgChan               // Drain the test message

	assert.Equal(t, mockLogger, ch.GetLogger())
	assert.Equal(t, mockRenderer, ch.GetRenderer())
}

// mockOtherChannel for testing type mismatch
type mockOtherChannel struct {
	id uuid.UUID
}

func (m *mockOtherChannel) ID() uuid.UUID                              { return m.id }
func (m *mockOtherChannel) OnMessage(msg channel.Message)               {}
func (m *mockOtherChannel) OnLog(entry shared.LogEntry)                 {}
func (m *mockOtherChannel) OnAgentLifecycle(event channel.AgentLifecycleEvent) {}
func (m *mockOtherChannel) Start(ctx context.Context) error             { return nil }
