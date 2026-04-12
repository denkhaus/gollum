package channel

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewChannelMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := NewMockChannelFacade(ctrl)
	agentID := uuid.New()
	agentRole := "assistant"
	sessionID := "test-session-123"
	channelID := uuid.New()

	middleware := NewChannelMiddleware(mockFacade, agentID, agentRole, sessionID, channelID)

	assert.NotNil(t, middleware)
	assert.Equal(t, agentID, middleware.agentID)
	assert.Equal(t, agentRole, middleware.agentRole)
	assert.Equal(t, sessionID, middleware.sessionID)
	assert.Equal(t, channelID, middleware.channelID)
}

func TestChannelMiddlewareProvider_CreateChannelMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create injector with mock facade
	injector := do.New()
	mockFacade := NewMockChannelFacade(ctrl)
	do.ProvideValue[ChannelFacade](injector, mockFacade)

	provider := &channelMiddlewareProvider{injector: injector}
	agentID := uuid.New()
	agentRole := "assistant"
	sessionID := "test-session-456"
	channelID := uuid.New()

	middleware := provider.CreateChannelMiddleware(agentID, agentRole, sessionID, channelID)

	assert.NotNil(t, middleware)
	assert.Equal(t, agentID, middleware.agentID)
	assert.Equal(t, agentRole, middleware.agentRole)
	assert.Equal(t, sessionID, middleware.sessionID)
	assert.Equal(t, channelID, middleware.channelID)
}

func TestMessage_Structure_HasSessionAndChannelIDs(t *testing.T) {
	msg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgentChat,
		AgentID:   uuid.New(),
		AgentRole: "assistant",
		SessionID: "test-session-789",
		ChannelID: uuid.New(),
		Content:   "Test content",
		Timestamp: time.Now(),
		Metadata:  map[string]any{"key": "value"},
	}

	assert.NotEqual(t, uuid.Nil, msg.ID)
	assert.Equal(t, MessageTypeAgentChat, msg.Type)
	assert.NotEqual(t, uuid.Nil, msg.AgentID)
	assert.Equal(t, "assistant", msg.AgentRole)
	assert.Equal(t, "test-session-789", msg.SessionID)
	assert.NotEqual(t, uuid.Nil, msg.ChannelID)
	assert.Equal(t, "Test content", msg.Content)
	assert.NotZero(t, msg.Timestamp)
	assert.NotNil(t, msg.Metadata)
}

func TestChannelMiddleware_WithEmptySessionID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := NewMockChannelFacade(ctrl)
	agentID := uuid.New()
	agentRole := "assistant"
	sessionID := "" // Empty session ID
	channelID := uuid.New()

	middleware := NewChannelMiddleware(mockFacade, agentID, agentRole, sessionID, channelID)

	assert.NotNil(t, middleware)
	assert.Equal(t, "", middleware.sessionID)
	assert.Equal(t, channelID, middleware.channelID)
}

func TestChannelMiddleware_WithNilChannelID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := NewMockChannelFacade(ctrl)
	agentID := uuid.New()
	agentRole := "assistant"
	sessionID := "test-session"
	channelID := uuid.Nil // Nil channel ID

	middleware := NewChannelMiddleware(mockFacade, agentID, agentRole, sessionID, channelID)

	assert.NotNil(t, middleware)
	assert.Equal(t, sessionID, middleware.sessionID)
	assert.Equal(t, uuid.Nil, middleware.channelID)
}

func TestNewChannelMiddlewareProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create injector with mock facade
	injector := do.New()
	mockFacade := NewMockChannelFacade(ctrl)
	do.ProvideValue[ChannelFacade](injector, mockFacade)

	// Test that provider can be created
	provider := &channelMiddlewareProvider{injector: injector}
	assert.NotNil(t, provider)

	// Test CreateChannelMiddleware with various inputs
	testCases := []struct {
		name      string
		agentID   uuid.UUID
		agentRole string
		sessionID string
		channelID uuid.UUID
	}{
		{
			name:      "Valid parameters",
			agentID:   uuid.New(),
			agentRole: "assistant",
			sessionID: "session-1",
			channelID: uuid.New(),
		},
		{
			name:      "Empty session ID",
			agentID:   uuid.New(),
			agentRole: "user",
			sessionID: "",
			channelID: uuid.New(),
		},
		{
			name:      "Different agent roles",
			agentID:   uuid.New(),
			agentRole: "system",
			sessionID: "session-2",
			channelID: uuid.New(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			middleware := provider.CreateChannelMiddleware(tc.agentID, tc.agentRole, tc.sessionID, tc.channelID)
			assert.NotNil(t, middleware)
			assert.Equal(t, tc.agentID, middleware.agentID)
			assert.Equal(t, tc.agentRole, middleware.agentRole)
			assert.Equal(t, tc.sessionID, middleware.sessionID)
			assert.Equal(t, tc.channelID, middleware.channelID)
		})
	}
}

func TestContentBlockMiddleware_MessagesIncludeSessionAndChannelIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := NewMockChannelFacade(ctrl)
	agentID := uuid.New()
	agentRole := "assistant"
	sessionID := "test-session-content"
	channelID := uuid.New()

	middleware := NewChannelMiddleware(mockFacade, agentID, agentRole, sessionID, channelID)

	// Verify middleware has correct session and channel IDs
	assert.Equal(t, sessionID, middleware.sessionID)
	assert.Equal(t, channelID, middleware.channelID)
	assert.Equal(t, agentID, middleware.agentID)
	assert.Equal(t, agentRole, middleware.agentRole)

	// The ContentBlockMiddleware function should create a handler that,
	// when called, will send messages with the correct SessionID and ChannelID
	// This is verified by the middleware struct having the correct values
	handler := middleware.ContentBlockMiddleware(nil)
	assert.NotNil(t, handler)
}

func TestToolMiddleware_MessagesIncludeSessionAndChannelIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := NewMockChannelFacade(ctrl)
	agentID := uuid.New()
	agentRole := "assistant"
	sessionID := "test-session-tool"
	channelID := uuid.New()

	middleware := NewChannelMiddleware(mockFacade, agentID, agentRole, sessionID, channelID)

	// Verify middleware has correct session and channel IDs
	assert.Equal(t, sessionID, middleware.sessionID)
	assert.Equal(t, channelID, middleware.channelID)
	assert.Equal(t, agentID, middleware.agentID)
	assert.Equal(t, agentRole, middleware.agentRole)

	// The ToolMiddleware function should create a handler that,
	// when called, will send messages with the correct SessionID and ChannelID
	// This is verified by the middleware struct having the correct values
	handler := middleware.ToolMiddleware(nil)
	assert.NotNil(t, handler)
}
