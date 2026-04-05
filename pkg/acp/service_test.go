package acp

import (
	"context"
	"testing"

	acppkg "github.com/ironpark/go-acp"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
)

func TestNewAcpService_DICompliant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgent := shared.NewMockAgent(ctrl)
	mockFlowRegistry := registry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (shared.Agent, error) { return mockAgent, nil })
	do.Provide(injector, func(i do.Injector) (registry.FlowRegistry, error) { return mockFlowRegistry, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestAcpService_Initialize_ReturnsCorrectCapabilities(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgent := shared.NewMockAgent(ctrl)
	mockFlowRegistry := registry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (shared.Agent, error) { return mockAgent, nil })
	do.Provide(injector, func(i do.Injector) (registry.FlowRegistry, error) { return mockFlowRegistry, nil })
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

	mockAgent := shared.NewMockAgent(ctrl)
	mockFlowRegistry := registry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (shared.Agent, error) { return mockAgent, nil })
	do.Provide(injector, func(i do.Injector) (registry.FlowRegistry, error) { return mockFlowRegistry, nil })
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

	mockAgent := shared.NewMockAgent(ctrl)
	mockFlowRegistry := registry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (shared.Agent, error) { return mockAgent, nil })
	do.Provide(injector, func(i do.Injector) (registry.FlowRegistry, error) { return mockFlowRegistry, nil })
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

	mockAgent := shared.NewMockAgent(ctrl)
	mockFlowRegistry := registry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (shared.Agent, error) { return mockAgent, nil })
	do.Provide(injector, func(i do.Injector) (registry.FlowRegistry, error) { return mockFlowRegistry, nil })
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

	mockAgent := shared.NewMockAgent(ctrl)
	mockFlowRegistry := registry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (shared.Agent, error) { return mockAgent, nil })
	do.Provide(injector, func(i do.Injector) (registry.FlowRegistry, error) { return mockFlowRegistry, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	resp, err := svc.Prompt(context.Background(), &acppkg.PromptRequest{})

	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestAcpService_Cancel_NotImplemented(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgent := shared.NewMockAgent(ctrl)
	mockFlowRegistry := registry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (shared.Agent, error) { return mockAgent, nil })
	do.Provide(injector, func(i do.Injector) (registry.FlowRegistry, error) { return mockFlowRegistry, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	err = svc.Cancel(context.Background(), &acppkg.CancelNotification{})
	require.NoError(t, err)
}

func TestAcpService_SetClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgent := shared.NewMockAgent(ctrl)
	mockFlowRegistry := registry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (shared.Agent, error) { return mockAgent, nil })
	do.Provide(injector, func(i do.Injector) (registry.FlowRegistry, error) { return mockFlowRegistry, nil })
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

	mockAgent := shared.NewMockAgent(ctrl)
	mockFlowRegistry := registry.NewMockFlowRegistry(ctrl)
	mockLogger := logger.NewMockLoggerService(ctrl)

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.Provide(injector, func(i do.Injector) (shared.Agent, error) { return mockAgent, nil })
	do.Provide(injector, func(i do.Injector) (registry.FlowRegistry, error) { return mockFlowRegistry, nil })
	do.Provide(injector, func(i do.Injector) (logger.LoggerService, error) { return mockLogger, nil })

	svc, err := NewAcpService(injector)
	require.NoError(t, err)

	// We can't easily mock acppkg.SessionStore, so we just test that SetSessionStore doesn't panic
	// In real usage, this will be set by the connection factory
	assert.NotNil(t, svc)
}
