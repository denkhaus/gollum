// pkg/channel/facade_discovery_test.go
package channel

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/command"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDiscoverProviders_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof("Discovered channel: %s (from service: %s)", gomock.Any(), gomock.Any()).Times(1)

	// Create injector with registered channel
	injector := do.New()
	do.ProvideNamedValue(injector, "channel_tui", ChannelFactory(func(opts ...ChannelOption) (Channel, error) {
		return &mockChannel{id: uuid.New()}, nil
	}))

	// Use the existing mocks from facade_test.go
	mockCmdMgr := &mockCommandManager{}
	mockReg := &mockAgentRegistryWithSupervisor{}
	mockFactory := &mockAgentFactory{}
	mockSessMgr := session.NewMockSessionManager(ctrl)

	do.ProvideValue[command.ManagerService](injector, mockCmdMgr)
	do.ProvideValue[registry.AgentRegistry](injector, mockReg)
	do.ProvideValue[shared.AgentFactory](injector, mockFactory)
	do.ProvideValue[session.SessionManager](injector, mockSessMgr)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	facade, err := NewChannelFacade(injector)
	require.NoError(t, err)
	assert.NotNil(t, facade)

	// Verify providers were discovered
	facadeImpl := facade.(*channelFacadeImpl)
	assert.Contains(t, facadeImpl.providers, ChannelIdentifier("tui"))
}

func TestDiscoverProviders_NoChannels(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Warn("No channel providers discovered - channels may not be available").Times(1)

	// Create injector without channels
	injector := do.New()

	// Use the existing mocks
	mockCmdMgr := &mockCommandManager{}
	mockReg := &mockAgentRegistryWithSupervisor{}
	mockFactory := &mockAgentFactory{}
	mockSessMgr := session.NewMockSessionManager(ctrl)

	do.ProvideValue[command.ManagerService](injector, mockCmdMgr)
	do.ProvideValue[registry.AgentRegistry](injector, mockReg)
	do.ProvideValue[shared.AgentFactory](injector, mockFactory)
	do.ProvideValue[session.SessionManager](injector, mockSessMgr)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	facade, err := NewChannelFacade(injector)
	require.NoError(t, err)
	assert.NotNil(t, facade)

	// Verify no providers were discovered
	facadeImpl := facade.(*channelFacadeImpl)
	assert.Empty(t, facadeImpl.providers)
}

func TestDiscoverProviders_MultipleChannels(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).Times(2)

	// Create injector with multiple channels
	injector := do.New()
	do.ProvideNamedValue(injector, "channel_tui", ChannelFactory(func(opts ...ChannelOption) (Channel, error) {
		return &mockChannel{id: uuid.New()}, nil
	}))
	do.ProvideNamedValue(injector, "channel_acp", ChannelFactory(func(opts ...ChannelOption) (Channel, error) {
		return &mockChannel{id: uuid.New()}, nil
	}))

	// Use the existing mocks
	mockCmdMgr := &mockCommandManager{}
	mockReg := &mockAgentRegistryWithSupervisor{}
	mockFactory := &mockAgentFactory{}
	mockSessMgr := session.NewMockSessionManager(ctrl)

	do.ProvideValue[command.ManagerService](injector, mockCmdMgr)
	do.ProvideValue[registry.AgentRegistry](injector, mockReg)
	do.ProvideValue[shared.AgentFactory](injector, mockFactory)
	do.ProvideValue[session.SessionManager](injector, mockSessMgr)
	do.ProvideValue[logger.LoggerService](injector, mockLogger)

	facade, err := NewChannelFacade(injector)
	require.NoError(t, err)
	assert.NotNil(t, facade)

	// Verify both providers were discovered
	facadeImpl := facade.(*channelFacadeImpl)
	assert.Len(t, facadeImpl.providers, 2)
	assert.Contains(t, facadeImpl.providers, ChannelIdentifier("tui"))
	assert.Contains(t, facadeImpl.providers, ChannelIdentifier("acp"))
}
