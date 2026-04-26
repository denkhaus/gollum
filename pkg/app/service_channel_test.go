// pkg/app/service_channel_test.go
package app

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/channel"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/logger"
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
	mockFacade.EXPECT().CreateAndRegister(tui.Identifier, gomock.Any()).Return(mockCh, nil)
	mockCh.EXPECT().Start(ctx).Return(nil)
	mockFacade.EXPECT().UnregisterChannel(mockChID).Return(nil)

	svc := &applicationServiceImpl{
		logService:    mockLogger,
		channelFacade: mockFacade,
		flowRegistry:  mockFlowRegistry,
		fsm:           mockFSM,
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

	mockFacade.EXPECT().CreateAndRegister(tui.Identifier, gomock.Any()).
		Return(nil, assert.AnError)

	svc := &applicationServiceImpl{
		logService:    mockLogger,
		channelFacade: mockFacade,
		flowRegistry:  mockFlowRegistry,
		fsm:           mockFSM,
	}

	err := svc.runChannel(ctx, tui.Identifier)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create and register channel")
}

func TestRunChannel_CreateAndRegisterError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)
	mockFlowRegistry := flowregistry.NewMockFlowRegistry(ctrl)
	mockFSM := state.NewMockFileStateManager(ctrl)

	// CreateAndRegister fails - no channel created, no cleanup needed
	mockFacade.EXPECT().CreateAndRegister(tui.Identifier, gomock.Any()).Return(nil, assert.AnError)

	svc := &applicationServiceImpl{
		logService:    mockLogger,
		channelFacade: mockFacade,
		flowRegistry:  mockFlowRegistry,
		fsm:           mockFSM,
	}

	err := svc.runChannel(ctx, tui.Identifier)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create and register channel")
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
	mockFacade.EXPECT().CreateAndRegister(tui.Identifier, gomock.Any()).Return(mockCh, nil)
	mockCh.EXPECT().Start(ctx).Return(assert.AnError)
	mockFacade.EXPECT().UnregisterChannel(mockChID).Return(nil)

	svc := &applicationServiceImpl{
		logService:    mockLogger,
		channelFacade: mockFacade,
		flowRegistry:  mockFlowRegistry,
		fsm:           mockFSM,
	}

	err := svc.runChannel(ctx, tui.Identifier)
	assert.Error(t, err)
}
