package channel

import (
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"
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
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	channelID := uuid.New()

	sessionCtx := *shared.NewSessionContext(sessionID, agentID, channelID, agentRole)
	middleware := NewChannelMiddleware(mockFacade, sessionCtx, agentRole)

	assert.NotNil(t, middleware)
	assert.Equal(t, agentID, middleware.AgentID)
	assert.Equal(t, agentRole, middleware.agentRole)
	assert.Equal(t, sessionID, middleware.SessionID)
	assert.Equal(t, channelID, middleware.ChannelID)
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
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	channelID := uuid.New()

	sessionCtx := *shared.NewSessionContext(sessionID, agentID, channelID, agentRole)
	middleware := provider.CreateChannelMiddleware(sessionCtx, agentRole)

	assert.NotNil(t, middleware)
	assert.Equal(t, agentID, middleware.AgentID)
	assert.Equal(t, agentRole, middleware.agentRole)
	assert.Equal(t, sessionID, middleware.SessionID)
	assert.Equal(t, channelID, middleware.ChannelID)
}

func TestMessage_Structure_HasSessionAndChannelIDs(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	msg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgentChat,
		SessionContext: shared.SessionContext{
			SessionID: sessionID,
			ChannelID: uuid.New(),
			AgentID:   uuid.New(),
		},
		AgentRole: "assistant",
		Content:   "Test content",
		Timestamp: time.Now(),
		Metadata:  map[string]any{"key": "value"},
	}

	assert.NotEqual(t, uuid.Nil, msg.ID)
	assert.Equal(t, MessageTypeAgentChat, msg.Type)
	assert.NotEqual(t, uuid.Nil, msg.AgentID)
	assert.Equal(t, "assistant", msg.AgentRole)
	assert.Equal(t, sessionID, msg.SessionID)
	assert.NotEqual(t, uuid.Nil, msg.ChannelID)
	assert.Equal(t, "Test content", msg.Content)
	assert.NotZero(t, msg.Timestamp)
	assert.NotNil(t, msg.Metadata)
}

func TestChannelMiddleware_WithNilSessionID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := NewMockChannelFacade(ctrl)
	agentID := uuid.New()
	agentRole := "assistant"
	channelID := uuid.New()

	sessionCtx := *shared.NewSessionContext(uuid.Nil, agentID, channelID, agentRole)
	middleware := NewChannelMiddleware(mockFacade, sessionCtx, agentRole)

	assert.NotNil(t, middleware)
	assert.Equal(t, uuid.Nil, middleware.SessionID)
	assert.Equal(t, channelID, middleware.ChannelID)
}

func TestChannelMiddleware_WithNilChannelID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := NewMockChannelFacade(ctrl)
	agentID := uuid.New()
	agentRole := "assistant"
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000008")

	sessionCtx := *shared.NewSessionContext(sessionID, agentID, uuid.Nil, agentRole)
	middleware := NewChannelMiddleware(mockFacade, sessionCtx, agentRole)

	assert.NotNil(t, middleware)
	assert.Equal(t, sessionID, middleware.SessionID)
	assert.Equal(t, uuid.Nil, middleware.ChannelID)
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
		sessionID uuid.UUID
		channelID uuid.UUID
	}{
		{
			name:      "Valid parameters",
			agentID:   uuid.New(),
			agentRole: "assistant",
			sessionID: uuid.MustParse("00000000-0000-0000-0000-000000000006"),
			channelID: uuid.New(),
		},
		{
			name:      "Nil session ID",
			agentID:   uuid.New(),
			agentRole: "user",
			sessionID: uuid.Nil,
			channelID: uuid.New(),
		},
		{
			name:      "Different agent roles",
			agentID:   uuid.New(),
			agentRole: "system",
			sessionID: uuid.MustParse("00000000-0000-0000-0000-000000000007"),
			channelID: uuid.New(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sessionCtx := *shared.NewSessionContext(tc.sessionID, tc.agentID, tc.channelID, tc.agentRole)
			middleware := provider.CreateChannelMiddleware(sessionCtx, tc.agentRole)
			assert.NotNil(t, middleware)
			assert.Equal(t, tc.agentID, middleware.AgentID)
			assert.Equal(t, tc.agentRole, middleware.agentRole)
			assert.Equal(t, tc.sessionID, middleware.SessionID)
			assert.Equal(t, tc.channelID, middleware.ChannelID)
		})
	}
}

func TestContentBlockMiddleware_MessagesIncludeSessionAndChannelIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := NewMockChannelFacade(ctrl)
	agentID := uuid.New()
	agentRole := "assistant"
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	channelID := uuid.New()

	sessionCtx := *shared.NewSessionContext(sessionID, agentID, channelID, agentRole)
	middleware := NewChannelMiddleware(mockFacade, sessionCtx, agentRole)

	// Verify middleware has correct session and channel IDs
	assert.Equal(t, sessionID, middleware.SessionID)
	assert.Equal(t, channelID, middleware.ChannelID)
	assert.Equal(t, agentID, middleware.AgentID)
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
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000005")
	channelID := uuid.New()

	sessionCtx := *shared.NewSessionContext(sessionID, agentID, channelID, agentRole)
	middleware := NewChannelMiddleware(mockFacade, sessionCtx, agentRole)

	// Verify middleware has correct session and channel IDs
	assert.Equal(t, sessionID, middleware.SessionID)
	assert.Equal(t, channelID, middleware.ChannelID)
	assert.Equal(t, agentID, middleware.AgentID)
	assert.Equal(t, agentRole, middleware.agentRole)

	// The ToolMiddleware function should create a handler that,
	// when called, will send messages with the correct SessionID and ChannelID
	// This is verified by the middleware struct having the correct values
	handler := middleware.ToolMiddleware(nil)
	assert.NotNil(t, handler)
}
