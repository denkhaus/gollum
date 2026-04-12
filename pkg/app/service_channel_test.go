// pkg/app/service_channel_test.go
package app

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/state"
	"github.com/denkhaus/gollum/pkg/tui"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRunChannel_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFSM := state.NewMockFileStateManager(ctrl)

	// Mock channel that implements Channel (now includes Start)
	mockCh := channel.NewMockChannel(ctrl)
	mockChID := uuid.New()
	mockCh.EXPECT().ID().Return(mockChID).AnyTimes()
	mockFacade.EXPECT().CreateChannel(tui.Identifier, gomock.Any()).Return(mockCh, nil)
	mockFacade.EXPECT().RegisterChannel(mockCh).Return(nil)
	mockCh.EXPECT().Start(ctx).Return(nil)

	svc := &applicationServiceImpl{
		logService:       mockLogger,
		channelFacade:    mockFacade,
		flowRegistry:     mockFlowRegistry,
		fsm:              mockFSM,
	}

	err := svc.runChannel(ctx, tui.Identifier)
	assert.NoError(t, err)
}

func TestRunChannel_CreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFSM := state.NewMockFileStateManager(ctrl)

	mockFacade.EXPECT().CreateChannel(tui.Identifier, gomock.Any()).
		Return(nil, assert.AnError)

	svc := &applicationServiceImpl{
		logService:    mockLogger,
		channelFacade: mockFacade,
		flowRegistry:  mockFlowRegistry,
		fsm:           mockFSM,
	}

	err := svc.runChannel(ctx, tui.Identifier)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create channel")
}

func TestRunChannel_RegisterError_Cleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFSM := state.NewMockFileStateManager(ctrl)

	mockCh := channel.NewMockChannel(ctrl)
	mockChID := uuid.New()
	mockCh.EXPECT().ID().Return(mockChID).Times(1)
	mockFacade.EXPECT().CreateChannel(tui.Identifier, gomock.Any()).Return(mockCh, nil)
	mockFacade.EXPECT().RegisterChannel(mockCh).Return(assert.AnError)
	mockFacade.EXPECT().UnregisterChannel(mockChID).Return(nil)

	svc := &applicationServiceImpl{
		logService:    mockLogger,
		channelFacade: mockFacade,
		flowRegistry:  mockFlowRegistry,
		fsm:           mockFSM,
	}

	err := svc.runChannel(ctx, tui.Identifier)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register channel")
}

func TestRunChannel_StartError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFSM := state.NewMockFileStateManager(ctrl)

	mockCh := channel.NewMockChannel(ctrl)
	mockChID := uuid.New()
	mockCh.EXPECT().ID().Return(mockChID).AnyTimes()
	mockFacade.EXPECT().CreateChannel(tui.Identifier, gomock.Any()).Return(mockCh, nil)
	mockFacade.EXPECT().RegisterChannel(mockCh).Return(nil)
	mockCh.EXPECT().Start(ctx).Return(assert.AnError)

	svc := &applicationServiceImpl{
		logService:    mockLogger,
		channelFacade: mockFacade,
		flowRegistry:  mockFlowRegistry,
		fsm:           mockFSM,
	}

	err := svc.runChannel(ctx, tui.Identifier)
	assert.Error(t, err)
}
