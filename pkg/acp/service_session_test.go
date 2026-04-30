// Package acp provides tests for ACP service session management methods.
package acp

import (
	"context"
	"encoding/json"
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
			{SessionContext: shared.SessionContext{SessionID: uuid.New(), ChannelID: uuid.New(), Cwd: "/tmp"}},
			{SessionContext: shared.SessionContext{SessionID: uuid.New(), ChannelID: uuid.New(), Cwd: "/home"}},
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
			SessionContext: shared.SessionContext{
				SessionID: sessionID,
				ChannelID: uuid.New(),
			},
			Context:    sessionCtx,
			CancelFunc: cancel,
		}

		mockSessionMgr.EXPECT().LoadSession(ctx, sessionID).Return(expectedSession, nil)

		result, err := svc.LoadSession(ctx, acpSessionID)
		require.NoError(t, err)
		assert.Equal(t, expectedSession.SessionContext.SessionID, result.SessionID)
		assert.Equal(t, expectedSession.SessionContext.Cwd, result.SessionContext.Cwd)
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

			expectedSession := &shared.Session{
				SessionContext: shared.SessionContext{
					SessionID: newSessionID,
					ChannelID: uuid.New(),
					Cwd:       "/tmp",
				},
			}
			mockSessionMgr.EXPECT().ForkSession(ctx, originalSessionID).Return(expectedSession, nil)

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

// =============================================================================
// Extension Method Handler Tests
// =============================================================================

// TestACPService_ExtMethod tests the ExtMethod implementation.
func TestACPService_ExtMethod(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionMgr := session.NewMockSessionManager(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockConfig := config.NewMockConfigService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	mockConfig.EXPECT().GetACPConfig().AnyTimes().Return(&config.ACPConfig{})

	svc := &acpServiceImpl{
		logger:         mockLogger,
		config:         mockConfig,
		facade:         mockFacade,
		sessionManager: mockSessionMgr,
		id:             uuid.New(),
	}

	t.Run("unknown method returns error", func(t *testing.T) {
		t.Parallel()

		result, err := svc.ExtMethod(context.Background(), "_unknown/method", json.RawMessage(`{}`))
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "method not found")
	})

	t.Run("_session/list successful", func(t *testing.T) {
		t.Parallel()

		sessionID1 := uuid.New()
		sessionID2 := uuid.New()
		expectedSessions := []*shared.Session{
			{SessionContext: shared.SessionContext{SessionID: sessionID1, ChannelID: uuid.New(), Cwd: "/tmp"}},
			{SessionContext: shared.SessionContext{SessionID: sessionID2, ChannelID: uuid.New(), Cwd: "/home"}},
		}

		mockSessionMgr.EXPECT().ListSessions(gomock.Any()).Return(expectedSessions, nil)

		result, err := svc.ExtMethod(context.Background(), "_session/list", json.RawMessage(`{}`))
		require.NoError(t, err)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok, "result should be a map")

		sessions, ok := resultMap["sessions"].([]any)
		require.True(t, ok, "sessions should be a slice")
		assert.Len(t, sessions, 2)
	})

	t.Run("_session/list with empty parameters", func(t *testing.T) {
		t.Parallel()

		mockSessionMgr.EXPECT().ListSessions(gomock.Any()).Return([]*shared.Session{}, nil)

		result, err := svc.ExtMethod(context.Background(), "_session/list", json.RawMessage(`{}`))
		require.NoError(t, err)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)

		sessions, ok := resultMap["sessions"].([]any)
		require.True(t, ok)
		assert.Len(t, sessions, 0)
	})

	t.Run("_session/list invalid JSON", func(t *testing.T) {
		t.Parallel()

		result, err := svc.ExtMethod(context.Background(), "_session/list", json.RawMessage(`invalid json`))
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid parameters")
	})

	t.Run("_session/load successful", func(t *testing.T) {
		t.Parallel()

		sessionID := uuid.New()
		expectedSession := &shared.Session{SessionContext: shared.SessionContext{SessionID: sessionID, ChannelID: uuid.New(), Cwd: "/tmp"}}

		mockSessionMgr.EXPECT().LoadSession(gomock.Any(), sessionID).Return(expectedSession, nil)

		params := json.RawMessage(`{"session_id":"` + sessionID.String() + `"}`)
		result, err := svc.ExtMethod(context.Background(), "_session/load", params)
		require.NoError(t, err)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)

		session, ok := resultMap["session"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, sessionID.String(), session["id"])
		assert.Equal(t, "/tmp", session["cwd"])
		assert.True(t, session["loaded"].(bool))
	})

	t.Run("_session/load missing session_id", func(t *testing.T) {
		t.Parallel()

		result, err := svc.ExtMethod(context.Background(), "_session/load", json.RawMessage(`{}`))
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "session_id is required")
	})

	t.Run("_session/load invalid session ID format", func(t *testing.T) {
		t.Parallel()

		result, err := svc.ExtMethod(context.Background(), "_session/load", json.RawMessage(`{"session_id":"invalid-uuid"}`))
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid session ID")
	})

	t.Run("_session/resume successful", func(t *testing.T) {
		t.Parallel()

		sessionID := uuid.New()
		mockSessionMgr.EXPECT().ResumeSession(gomock.Any(), sessionID).Return(nil)

		params := json.RawMessage(`{"session_id":"` + sessionID.String() + `"}`)
		result, err := svc.ExtMethod(context.Background(), "_session/resume", params)
		require.NoError(t, err)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, sessionID.String(), resultMap["session_id"])
		assert.True(t, resultMap["resumed"].(bool))
	})

	t.Run("_session/close successful", func(t *testing.T) {
		t.Parallel()

		sessionID := uuid.New()
		mockSessionMgr.EXPECT().CloseSession(sessionID).Return(nil)

		params := json.RawMessage(`{"session_id":"` + sessionID.String() + `"}`)
		result, err := svc.ExtMethod(context.Background(), "_session/close", params)
		require.NoError(t, err)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, sessionID.String(), resultMap["session_id"])
		assert.True(t, resultMap["closed"].(bool))
	})

	t.Run("_session/fork successful", func(t *testing.T) {
		t.Parallel()

		originalSessionID := uuid.New()
		newSessionID := uuid.New()
		expectedSession := &shared.Session{
			SessionContext: shared.SessionContext{
				SessionID: newSessionID,
				ChannelID: uuid.New(),
				Cwd:       "/tmp",
			},
		}

		mockSessionMgr.EXPECT().ForkSession(gomock.Any(), originalSessionID).Return(expectedSession, nil)

		params := json.RawMessage(`{"session_id":"` + originalSessionID.String() + `"}`)
		result, err := svc.ExtMethod(context.Background(), "_session/fork", params)
		require.NoError(t, err)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, originalSessionID.String(), resultMap["original_session_id"])
		assert.Equal(t, newSessionID.String(), resultMap["new_session_id"])
		assert.True(t, resultMap["forked"].(bool))
	})

	t.Run("_session/fork with error", func(t *testing.T) {
		t.Parallel()

		sessionID := uuid.New()
		expectedErr := errors.New("session not found")
		mockSessionMgr.EXPECT().ForkSession(gomock.Any(), sessionID).Return(nil, expectedErr)

		params := json.RawMessage(`{"session_id":"` + sessionID.String() + `"}`)
		result, err := svc.ExtMethod(context.Background(), "_session/fork", params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to fork session")
	})
}
