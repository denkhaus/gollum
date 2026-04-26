// Package acp provides tests for ACP service session management methods.
package acp

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	acppkg "github.com/ironpark/go-acp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
)

// TestACPService_ListSessions tests the ListSessions method.
func TestACPService_ListSessions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionMgr := session.NewMockSessionManager(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockConfig := config.NewMockConfigService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	// Setup config expectations
	mockConfig.EXPECT().GetACPConfig().AnyTimes().Return(&config.ACPConfig{})

	svc := &acpServiceImpl{
		logger:         mockLogger,
		config:         mockConfig,
		facade:         mockFacade,
		sessionManager: mockSessionMgr,
		id:             uuid.New(),
	}

	t.Run("successful list", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		expectedSessions := []*shared.Session{
			{ID: uuid.New(), ChannelID: uuid.New(), Cwd: "/tmp"},
			{ID: uuid.New(), ChannelID: uuid.New(), Cwd: "/home"},
		}

		mockSessionMgr.EXPECT().ListSessions(ctx).Return(expectedSessions, nil)

		sessions, err := svc.ListSessions(ctx)
		require.NoError(t, err)
		assert.Len(t, sessions, 2)
		assert.Equal(t, expectedSessions, sessions)
	})

	t.Run("list with error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		expectedErr := errors.New("repository error")

		mockSessionMgr.EXPECT().ListSessions(ctx).Return(nil, expectedErr)

		sessions, err := svc.ListSessions(ctx)
		assert.Error(t, err)
		assert.Nil(t, sessions)
		assert.Equal(t, expectedErr, err)
	})
}

// TestACPService_LoadSession tests the LoadSession method.
func TestACPService_LoadSession(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionMgr := session.NewMockSessionManager(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockConfig := config.NewMockConfigService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	// Setup config expectations
	mockConfig.EXPECT().GetACPConfig().AnyTimes().Return(&config.ACPConfig{})

	svc := &acpServiceImpl{
		logger:         mockLogger,
		config:         mockConfig,
		facade:         mockFacade,
		sessionManager: mockSessionMgr,
		id:             uuid.New(),
	}

	t.Run("successful load", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		sessionID := uuid.New()
		acpSessionID := acppkg.SessionID(sessionID.String())

		sessionCtx, cancel := context.WithCancel(context.Background())
		expectedSession := &shared.Session{
			ID:         sessionID,
			ChannelID:  uuid.New(),
			Context:    sessionCtx,
			CancelFunc: cancel,
			Cwd:        "/tmp/test",
		}

		mockSessionMgr.EXPECT().LoadSession(ctx, sessionID).Return(expectedSession, nil)

		result, err := svc.LoadSession(ctx, acpSessionID)
		require.NoError(t, err)
		assert.Equal(t, expectedSession.ID, result.SessionID)
		assert.Equal(t, expectedSession.Cwd, result.Cwd)
		assert.Equal(t, expectedSession.Context, result.Context)
		assert.NotNil(t, result.CancelFunc)
	})

	t.Run("invalid session ID format", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		invalidSessionID := acppkg.SessionID("not-a-uuid")

		result, err := svc.LoadSession(ctx, invalidSessionID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid session ID")
	})

	t.Run("session not found", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		sessionID := uuid.New()
		acpSessionID := acppkg.SessionID(sessionID.String())

		mockSessionMgr.EXPECT().LoadSession(ctx, sessionID).Return(nil, errors.New("session not found"))

		result, err := svc.LoadSession(ctx, acpSessionID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "session not found")
	})
}

// TestACPService_ResumeSession tests the ResumeSession method.
func TestACPService_ResumeSession(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionMgr := session.NewMockSessionManager(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockConfig := config.NewMockConfigService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	// Setup config expectations
	mockConfig.EXPECT().GetACPConfig().AnyTimes().Return(&config.ACPConfig{})

	svc := &acpServiceImpl{
		logger:         mockLogger,
		config:         mockConfig,
		facade:         mockFacade,
		sessionManager: mockSessionMgr,
		id:             uuid.New(),
	}

	t.Run("successful resume", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		sessionID := uuid.New()
		acpSessionID := acppkg.SessionID(sessionID.String())

		mockSessionMgr.EXPECT().ResumeSession(ctx, sessionID).Return(nil)

		err := svc.ResumeSession(ctx, acpSessionID)
		require.NoError(t, err)
	})

	t.Run("invalid session ID format", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		invalidSessionID := acppkg.SessionID("invalid-uuid")

		err := svc.ResumeSession(ctx, invalidSessionID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid session ID")
	})

	t.Run("resume with error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		sessionID := uuid.New()
		acpSessionID := acppkg.SessionID(sessionID.String())

		expectedErr := errors.New("session already active")
		mockSessionMgr.EXPECT().ResumeSession(ctx, sessionID).Return(expectedErr)

		err := svc.ResumeSession(ctx, acpSessionID)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

// TestACPService_CloseSession tests the CloseSession method.
func TestACPService_CloseSession(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionMgr := session.NewMockSessionManager(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockConfig := config.NewMockConfigService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	// Setup config expectations
	mockConfig.EXPECT().GetACPConfig().AnyTimes().Return(&config.ACPConfig{})

	svc := &acpServiceImpl{
		logger:         mockLogger,
		config:         mockConfig,
		facade:         mockFacade,
		sessionManager: mockSessionMgr,
		id:             uuid.New(),
	}

	t.Run("successful close", func(t *testing.T) {
		t.Parallel()

		sessionID := uuid.New()
		acpSessionID := acppkg.SessionID(sessionID.String())

		mockSessionMgr.EXPECT().CloseSession(sessionID).Return(nil)

		err := svc.CloseSession(context.Background(), acpSessionID)
		require.NoError(t, err)
	})

	t.Run("invalid session ID format", func(t *testing.T) {
		t.Parallel()

		invalidSessionID := acppkg.SessionID("bad-format")

		err := svc.CloseSession(context.Background(), invalidSessionID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid session ID")
	})

	t.Run("close with error", func(t *testing.T) {
		t.Parallel()

		sessionID := uuid.New()
		acpSessionID := acppkg.SessionID(sessionID.String())

		expectedErr := errors.New("session not found")
		mockSessionMgr.EXPECT().CloseSession(sessionID).Return(expectedErr)

		err := svc.CloseSession(context.Background(), acpSessionID)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

// TestACPService_ForkSession tests the ForkSession method.
func TestACPService_ForkSession(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionMgr := session.NewMockSessionManager(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockConfig := config.NewMockConfigService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	// Setup config expectations
	mockConfig.EXPECT().GetACPConfig().AnyTimes().Return(&config.ACPConfig{})

	svc := &acpServiceImpl{
		logger:         mockLogger,
		config:         mockConfig,
		facade:         mockFacade,
		sessionManager: mockSessionMgr,
		id:             uuid.New(),
	}

	t.Run("successful fork", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		originalSessionID := uuid.New()
		newSessionID := uuid.New()
		acpSessionID := acppkg.SessionID(originalSessionID.String())

		sessionCtx, cancel := context.WithCancel(context.Background())
		newSession := &shared.Session{
			ID:         newSessionID,
			ChannelID:  uuid.New(),
			Context:    sessionCtx,
			CancelFunc: cancel,
			Cwd:        "/tmp/forked",
		}

		mockSessionMgr.EXPECT().ForkSession(ctx, originalSessionID).Return(newSession, nil)

		resultID, err := svc.ForkSession(ctx, acpSessionID)
		require.NoError(t, err)
		assert.Equal(t, acppkg.SessionID(newSessionID.String()), resultID)
		assert.NotEqual(t, acpSessionID, resultID)
	})

	t.Run("invalid session ID format", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		invalidSessionID := acppkg.SessionID("not-uuid")

		resultID, err := svc.ForkSession(ctx, invalidSessionID)
		assert.Error(t, err)
		assert.Empty(t, resultID)
		assert.Contains(t, err.Error(), "invalid session ID")
	})

	t.Run("fork with error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		sessionID := uuid.New()
		acpSessionID := acppkg.SessionID(sessionID.String())

		expectedErr := errors.New("session not found")
		mockSessionMgr.EXPECT().ForkSession(ctx, sessionID).Return(nil, expectedErr)

		resultID, err := svc.ForkSession(ctx, acpSessionID)
		assert.Error(t, err)
		assert.Empty(t, resultID)
		assert.Equal(t, expectedErr, err)
	})
}
