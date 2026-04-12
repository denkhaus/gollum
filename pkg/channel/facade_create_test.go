// pkg/channel/facade_create_test.go
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

func TestCreateChannel_ValidIdentifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()

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

	ch, err := facade.CreateChannel(ChannelIdentifier("tui"))
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func TestCreateChannel_UnknownIdentifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any()).AnyTimes()

	facade := &channelFacadeImpl{
		channels:  make(map[uuid.UUID]Channel),
		logger:    mockLogger,
		providers: make(map[ChannelIdentifier]ChannelFactory),
	}

	_, err := facade.CreateChannel(ChannelIdentifier("unknown"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown channel identifier")
}

func TestCreateChannel_WithOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()

	// Track if option was applied
	optionApplied := false

	injector := do.New()
	do.ProvideNamedValue(injector, "channel_tui", ChannelFactory(func(opts ...ChannelOption) (Channel, error) {
		ch := &mockChannel{id: uuid.New()}
		for _, opt := range opts {
			if err := opt.Apply(ch); err != nil {
				return nil, err
			}
			optionApplied = true
		}
		return ch, nil
	}))

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)

	opt := &mockOption{applyFunc: func(c Channel) error { return nil }}
	ch, err := facade.CreateChannel(ChannelIdentifier("tui"), opt)

	assert.NoError(t, err)
	assert.NotNil(t, ch)
	assert.True(t, optionApplied, "Option should be applied")
}

func TestCreateChannel_FactoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()

	injector := do.New()
	do.ProvideNamedValue(injector, "channel_tui", ChannelFactory(func(opts ...ChannelOption) (Channel, error) {
		return nil, assert.AnError
	}))

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
	}

	err := facade.DiscoverProviders(injector)
	require.NoError(t, err)

	_, err = facade.CreateChannel(ChannelIdentifier("tui"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create channel")
}
