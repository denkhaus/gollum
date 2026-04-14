// pkg/channel/facade_discovery_test.go
package channel

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
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

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)
	assert.NotNil(t, facade.providers)
	assert.Contains(t, facade.providers, ChannelIdentifier("tui"))
}

func TestDiscoverProviders_NoChannels(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Warn("No channel providers discovered - channels may not be available").Times(1)

	// Create empty injector
	injector := do.New()

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)
	assert.NotNil(t, facade.providers)
	assert.Empty(t, facade.providers)
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

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)
	assert.Len(t, facade.providers, 2)
	assert.Contains(t, facade.providers, ChannelIdentifier("tui"))
	assert.Contains(t, facade.providers, ChannelIdentifier("acp"))
}
