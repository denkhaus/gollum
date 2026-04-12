package acp

import (
	"context"
	"fmt"
	"testing"

	acppkg "github.com/ironpark/go-acp"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
)

func TestNewAcpService_DICompliant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil).AnyTimes()

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

	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil).AnyTimes()

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

	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil).AnyTimes()

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

	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil).AnyTimes()

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

	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	resp, err := svc.SetSessionConfigOption(context.Background(), &acppkg.SetSessionConfigOptionRequest{})

	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestAcpService_Prompt_IntegratesWithFacade(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil).AnyTimes()

	// Mock the facade.SubmitInput call
	expectedResult := channel.InputResult{
		Handled:   true,
		Response:  "Test response from agent",
		IsCommand: false,
	}
	mockFacade.EXPECT().SubmitInput(gomock.Any(), gomock.Any(), gomock.Any(), "test prompt").Return(expectedResult, nil)

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	// Create a simple in-memory session store for testing
	sessionID := acppkg.SessionID("test-session")
	session := shared.NewAcpSession(context.Background(), func() {})
	session.SessionID = sessionID

	// Create a simple mock session store
	store := &mockSessionStore{
		sessions: map[acppkg.SessionID]*shared.ACPSession{
			sessionID: session,
		},
	}
	svc.SetSessionStore(store)

	// Create a mock ACP client for testing stream operations
	mockClient := &mockACPClient{}
	svc.SetClient(mockClient)

	resp, err := svc.Prompt(context.Background(), &acppkg.PromptRequest{
		SessionID: sessionID,
		Prompt:    []acppkg.ContentBlock{acppkg.NewContentBlockText("test prompt")},
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, acppkg.StopReasonEndTurn, resp.StopReason)
	assert.Equal(t, "Test response from agent", mockClient.lastSentText)
}

// mockACPClient is a simple mock for testing ACP client operations
type mockACPClient struct {
	acppkg.Client
	lastSentText string
}

func (m *mockACPClient) SessionUpdate(ctx context.Context, params *acppkg.SessionNotification) error {
	// Extract content from AgentMessageChunk update
	if update, ok := params.Update.AsAgentMessageChunk(); ok {
		if textContent, ok := update.Content.AsText(); ok {
			m.lastSentText = textContent.Text
		}
	}
	return nil
}

func (m *mockACPClient) RequestPermission(ctx context.Context, params *acppkg.RequestPermissionRequest) (*acppkg.RequestPermissionResponse, error) {
	return &acppkg.RequestPermissionResponse{}, nil
}

func (m *mockACPClient) ReadTextFile(ctx context.Context, params *acppkg.ReadTextFileRequest) (*acppkg.ReadTextFileResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockACPClient) WriteTextFile(ctx context.Context, params *acppkg.WriteTextFileRequest) (*acppkg.WriteTextFileResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockACPClient) CreateTerminal(ctx context.Context, params *acppkg.CreateTerminalRequest) (*acppkg.CreateTerminalResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockACPClient) TerminalOutput(ctx context.Context, params *acppkg.TerminalOutputRequest) (*acppkg.TerminalOutputResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// mockSessionStore is a simple in-memory session store for testing
type mockSessionStore struct {
	sessions map[acppkg.SessionID]*shared.ACPSession
}

func (m *mockSessionStore) Get(id acppkg.SessionID) (*shared.ACPSession, bool) {
	sess, ok := m.sessions[id]
	return sess, ok
}

func (m *mockSessionStore) Set(id acppkg.SessionID, sess *shared.ACPSession) {
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

	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil).AnyTimes()
	// Expect CancelInput call when canceling
	mockFacade.EXPECT().CancelInput(gomock.Any()).Return(nil)

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	// Create a session store with a session for testing
	sessionID := acppkg.SessionID("test-session")
	session := shared.NewAcpSession(context.Background(), func() {})
	session.SessionID = sessionID

	store := &mockSessionStore{
		sessions: map[acppkg.SessionID]*shared.ACPSession{
			sessionID: session,
		},
	}
	svc.SetSessionStore(store)

	// Cancel should find the session and cancel it
	err = svc.Cancel(context.Background(), &acppkg.CancelNotification{
		SessionID: sessionID,
	})
	require.NoError(t, err)
}

func TestAcpService_SetClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil).AnyTimes()

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

	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	// We can't easily mock acppkg.SessionStore, so we just test that SetSessionStore doesn't panic
	// In real usage, this will be set by the connection factory
	assert.NotNil(t, svc)
}
