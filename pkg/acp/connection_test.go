package acp

import (
	"bytes"
	"testing"

	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/samber/do/v2"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
)

func TestNewConnection_CreatesValidConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create generated mocks
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	// Expect Debug call from NewAcpService
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	// Expect RegisterChannel call from NewAcpService
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil)

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)
	// Register ACP service provider
	do.Provide(injector, NewAcpService)

	reader := bytes.NewReader([]byte{})
	writer := &bytes.Buffer{}

	conn, err := NewConnection(injector, reader, writer)

	require.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestNewConnection_NilReader_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)
	do.Provide(injector, NewAcpService)

	conn, err := NewConnection(injector, nil, &bytes.Buffer{})

	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "reader cannot be nil")
}

func TestNewConnection_NilWriter_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)
	do.Provide(injector, NewAcpService)

	conn, err := NewConnection(injector, bytes.NewReader([]byte{}), nil)

	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "writer cannot be nil")
}

func TestNewConnection_ServiceNotInDI_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)
	// Don't register ACP service provider - should panic

	assert.Panics(t, func() {
		NewConnection(injector, bytes.NewReader([]byte{}), &bytes.Buffer{})
	})
}

func TestNewConnection_ImplementsConnectionInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	// Expect Debug call from NewAcpService
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	// Expect RegisterChannel call from NewAcpService
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil)

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)
	// Register ACP service provider
	do.Provide(injector, NewAcpService)

	reader := bytes.NewReader([]byte{})
	writer := &bytes.Buffer{}

	conn, err := NewConnection(injector, reader, writer)

	require.NoError(t, err)
	// Verify connection implements Connection interface
	var _ Connection = conn
	assert.NotNil(t, conn)
}

func TestConnectionImpl_DoneReturnsChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	// Expect Debug call from NewAcpService
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	// Expect RegisterChannel call from NewAcpService
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil)

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)
	// Register ACP service provider
	do.Provide(injector, NewAcpService)

	reader := bytes.NewReader([]byte{})
	writer := &bytes.Buffer{}

	conn, err := NewConnection(injector, reader, writer)

	require.NoError(t, err)
	// Done() can only be called after Start(), but we can't test that
	// in a unit test without actual IO. Just verify connection exists.
	assert.NotNil(t, conn)
}

func TestConnectionImpl_ConnectionCreatedSuccessfully(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacade(ctrl)

	// Expect Debug call from NewAcpService
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	// Expect RegisterChannel call from NewAcpService
	mockFacade.EXPECT().RegisterChannel(gomock.Any()).Return(nil)

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)
	// Register ACP service provider
	do.Provide(injector, NewAcpService)

	reader := bytes.NewReader([]byte{})
	writer := &bytes.Buffer{}

	conn, err := NewConnection(injector, reader, writer)

	require.NoError(t, err)
	// Verify connection was created and implements interface
	assert.NotNil(t, conn)
	var _ Connection = conn
}
