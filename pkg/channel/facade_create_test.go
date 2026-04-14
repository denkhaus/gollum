// pkg/channel/facade_create_test.go
package channel

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCreateChannel_ValidIdentifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()

	facade := &channelFacadeImpl{
		channels:  make(map[uuid.UUID]Channel),
		logger:    mockLogger,
		providers: map[ChannelIdentifier]ChannelFactory{
			"tui": func(opts ...ChannelOption) (Channel, error) {
				return &mockChannel{id: uuid.New()}, nil
			},
		},
	}

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

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
		providers: map[ChannelIdentifier]ChannelFactory{
			"tui": func(opts ...ChannelOption) (Channel, error) {
				ch := &mockChannel{id: uuid.New()}
				for _, opt := range opts {
					if err := opt.Apply(ch); err != nil {
						return nil, err
					}
					optionApplied = true
				}
				return ch, nil
			},
		},
	}

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

	facade := &channelFacadeImpl{
		channels: make(map[uuid.UUID]Channel),
		logger:   mockLogger,
		providers: map[ChannelIdentifier]ChannelFactory{
			"tui": func(opts ...ChannelOption) (Channel, error) {
				return nil, assert.AnError
			},
		},
	}

	_, err := facade.CreateChannel(ChannelIdentifier("tui"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create channel")
}
