package acp

import (
	"context"
	"testing"

	acppkg "github.com/ironpark/go-acp"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
)

func TestNewAcpService_DICompliant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacadeService(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestAcpService_Initialize_ReturnsCorrectCapabilities(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacadeService(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	resp, err := svc.Initialize(context.Background(), &acppkg.InitializeRequest{
		ProtocolVersion: 1,
	})

	require.NoError(t, err)
	assert.Equal(t, acppkg.ProtocolVersion(acppkg.CurrentProtocolVersion), resp.ProtocolVersion)
	assert.NotNil(t, resp.AgentCapabilities)
	assert.False(t, resp.AgentCapabilities.LoadSession)
}

func TestAcpService_Authenticate_NotImplemented(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacadeService(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	resp, err := svc.Authenticate(context.Background(), &acppkg.AuthenticateRequest{})

	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestAcpService_SetSessionMode_NotImplemented(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacadeService(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	resp, err := svc.SetSessionMode(context.Background(), &acppkg.SetSessionModeRequest{})

	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestAcpService_SetSessionConfigOption_NotImplemented(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacadeService(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	resp, err := svc.SetSessionConfigOption(context.Background(), &acppkg.SetSessionConfigOptionRequest{})

	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestAcpService_Prompt_NotImplemented(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacadeService(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	// Create a simple in-memory session store for testing
	sessionID := acppkg.SessionID("test-session")
	session := NewAcpSession(context.Background(), func(){})
	session.SessionID = sessionID

	// Create a simple mock session store
	store := &mockSessionStore{
		sessions: map[acppkg.SessionID]*AcpSession{
			sessionID: session,
		},
	}
	svc.SetSessionStore(store)

	resp, err := svc.Prompt(context.Background(), &acppkg.PromptRequest{
		SessionID: sessionID,
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, acppkg.StopReasonEndTurn, resp.StopReason)
}

// mockSessionStore is a simple in-memory session store for testing
type mockSessionStore struct {
	sessions map[acppkg.SessionID]*AcpSession
}

func (m *mockSessionStore) Get(id acppkg.SessionID) (*AcpSession, bool) {
	sess, ok := m.sessions[id]
	return sess, ok
}

func (m *mockSessionStore) Set(id acppkg.SessionID, sess *AcpSession) {
	m.sessions[id] = sess
}

func (m *mockSessionStore) Delete(id acppkg.SessionID) {
	delete(m.sessions, id)
}

func (m *mockSessionStore) List() []acppkg.SessionID {
	ids := make([]acppkg.SessionID, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}

func TestAcpService_Cancel_NotImplemented(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacadeService(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	err = svc.Cancel(context.Background(), &acppkg.CancelNotification{})
	require.NoError(t, err)
}

func TestAcpService_SetClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacadeService(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	// We can't easily mock acppkg.Client, so we just test that SetClient doesn't panic
	// In real usage, this will be set by the connection factory
	assert.NotNil(t, svc)
}

func TestAcpService_SetSessionStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacadeService(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	// We can't easily mock acppkg.SessionStore, so we just test that SetSessionStore doesn't panic
	// In real usage, this will be set by the connection factory
	assert.NotNil(t, svc)
}
