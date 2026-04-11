package acp

import (
	"context"
	"sync"
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

// TestACP_LogRouting_SpecificSession verifies that logs route correctly to specific ACP sessions
func TestACP_LogRouting_SpecificSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil).AnyTimes()

	// Setup minimal DI with necessary services
	injector := do.New()
	do.Provide(injector, func(i do.Injector) (channel.ChannelFacade, error) { return mockFacade, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	// Create ACP service
	acpSvc, err := NewAcpService(injector)
	require.NoError(t, err)

	// Create mock ACP client to capture log messages
	mockClient := &logCaptureACPClient{
		receivedLogs: make(map[string][]string),
		mu:           sync.Mutex{},
	}
	acpSvc.SetClient(mockClient)

	// Create session with sessionID
	sessionID := acppkg.SessionID("test-log-session")
	session := shared.NewAcpSession(context.Background(), func() {})
	session.SessionID = sessionID

	// Create session store with the session
	store := &mockSessionStore{
		sessions: map[acppkg.SessionID]*shared.ACPSession{
			sessionID: session,
		},
	}
	acpSvc.SetSessionStore(store)

	// Type assert to channel.Channel to access OnLog and ID methods
	channelSvc, ok := acpSvc.(channel.Channel)
	require.True(t, ok, "ACP service should implement channel.Channel")

	// Get the channel ID from the service for logging
	channelID := channelSvc.ID()

	// Create a log entry with LoggingContext containing the SessionID
	logEntry := channel.LogEntry{
		Level:     "info",
		Message:   "Test log message for specific session",
		SessionID: string(sessionID),
		ChannelID: channelID,
	}

	// Call OnLog to route the log entry
	channelSvc.OnLog(logEntry)

	// Verify mock client received the log for the specific session
	mockClient.mu.Lock()
	defer mockClient.mu.Unlock()

	logs, exists := mockClient.receivedLogs[string(sessionID)]
	require.True(t, exists, "Expected logs to be received for session %s", sessionID)
	require.Len(t, logs, 1, "Expected exactly one log entry")
	assert.Equal(t, "[info] Test log message for specific session", logs[0])
}

// logCaptureACPClient captures log messages sent to specific sessions
type logCaptureACPClient struct {
	acppkg.Client
	receivedLogs map[string][]string // sessionID -> log messages
	mu           sync.Mutex
}

func (m *logCaptureACPClient) SessionUpdate(ctx context.Context, params *acppkg.SessionNotification) error {
	// Extract content from AgentMessageChunk update
	if update, ok := params.Update.AsAgentMessageChunk(); ok {
		if textContent, ok := update.Content.AsText(); ok {
			m.mu.Lock()
			defer m.mu.Unlock()

			sessionID := string(params.SessionID)
			m.receivedLogs[sessionID] = append(m.receivedLogs[sessionID], textContent.Text)
		}
	}
	return nil
}

func (m *logCaptureACPClient) RequestPermission(ctx context.Context, params *acppkg.RequestPermissionRequest) (*acppkg.RequestPermissionResponse, error) {
	return &acppkg.RequestPermissionResponse{}, nil
}

func (m *logCaptureACPClient) ReadTextFile(ctx context.Context, params *acppkg.ReadTextFileRequest) (*acppkg.ReadTextFileResponse, error) {
	return nil, nil
}

func (m *logCaptureACPClient) WriteTextFile(ctx context.Context, params *acppkg.WriteTextFileRequest) (*acppkg.WriteTextFileResponse, error) {
	return nil, nil
}

func (m *logCaptureACPClient) CreateTerminal(ctx context.Context, params *acppkg.CreateTerminalRequest) (*acppkg.CreateTerminalResponse, error) {
	return nil, nil
}

func (m *logCaptureACPClient) TerminalOutput(ctx context.Context, params *acppkg.TerminalOutputRequest) (*acppkg.TerminalOutputResponse, error) {
	return nil, nil
}
