package channel

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/command"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestInputHandler_HandleInput_SlashCommand(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	sessionID := "test-session"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cmdMgr := command.NewMockManager(ctrl)
	cmdMgr.EXPECT().Execute(ctx, sessionID, "/help").Return(true, "Help text", nil)

	sm := session.NewMockSessionManager(ctrl)
	reg := registry.NewMockAgentRegistry(ctrl)
	af := shared.NewMockAgentFactory(ctrl)
	log := logger.NewMockLoggerService(ctrl)

	handler := NewInputHandler(cmdMgr, sm, reg, af, log)

	result, err := handler.HandleInput(ctx, channelID, sessionID, "/help")

	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.True(t, result.IsCommand)
	assert.Equal(t, "Help text", result.Response)
}

func TestInputHandler_HandleInput_NonCommand(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	sessionID := "test-session"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cmdMgr := command.NewMockManager(ctrl)
	cmdMgr.EXPECT().Execute(ctx, sessionID, "hello").Return(false, "", nil)

	sm := session.NewMockSessionManager(ctrl)
	reg := registry.NewMockAgentRegistry(ctrl)
	mockCtx, mockCancel := context.WithCancel(ctx)
	mockSession := &shared.Session{
		ID:         sessionID,
		ChannelID:  channelID,
		Context:    mockCtx,
		CancelFunc: mockCancel,
	}
	sm.EXPECT().GetOrCreateSession(sessionID, channelID).Return(mockSession, nil)

	af := shared.NewMockAgentFactory(ctrl)
	mockSupervisor := shared.NewMockAgent(ctrl)
	mockConfig := &shared.AgentConfig{}
	af.EXPECT().CreateSupervisorAgent(gomock.Any(), gomock.Any()).Return(mockSupervisor, mockConfig, nil)

	mockSupervisor.EXPECT().GetID().Return(uuid.New())
	mockSupervisor.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{Texts: []string{"Hello back"}}, nil)

	log := logger.NewMockLoggerService(ctrl)

	handler := NewInputHandler(cmdMgr, sm, reg, af, log)

	result, err := handler.HandleInput(ctx, channelID, sessionID, "hello")

	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.False(t, result.IsCommand)
	assert.Equal(t, "Hello back", result.Response)
}

func TestInputHandler_CancelInput(t *testing.T) {
	sessionID := "test-session"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sm := session.NewMockSessionManager(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	mockSession := &shared.Session{
		ID:         sessionID,
		Context:    ctx,
		CancelFunc: cancel,
	}
	sm.EXPECT().GetSession(sessionID).Return(mockSession, true)

	handler := NewInputHandler(nil, sm, nil, nil, nil)

	err := handler.CancelInput(sessionID)
	require.NoError(t, err)

	// Verify that CancelFunc was called by checking context is cancelled
	select {
	case <-mockSession.Context.Done():
		// Context was cancelled as expected
	default:
		t.Error("Expected session context to be cancelled")
	}
}

func TestInputHandler_CancelInput_SessionNotFound(t *testing.T) {
	sessionID := "non-existent-session"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sm := session.NewMockSessionManager(ctrl)
	sm.EXPECT().GetSession(sessionID).Return(nil, false)

	handler := NewInputHandler(nil, sm, nil, nil, nil)

	err := handler.CancelInput(sessionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session "+sessionID+" not found")
}
