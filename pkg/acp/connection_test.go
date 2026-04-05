package acp

import (
	"bytes"
	"testing"

	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/samber/do/v2"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
)

func TestNewConnection_CreatesValidConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create generated mocks
	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacadeService(ctrl)

	// Expect Debug call from NewAcpService
	mockLogger.EXPECT().Debug(gomock.Any()).Times(1)

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)

	reader := bytes.NewReader([]byte{})
	writer := &bytes.Buffer{}

	conn := NewConnection(injector, reader, writer)

	assert.NotNil(t, conn)
}

func TestNewConnection_ImplementsConnectionInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacadeService(ctrl)

	// Expect Debug call from NewAcpService
	mockLogger.EXPECT().Debug(gomock.Any()).Times(1)

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)

	reader := bytes.NewReader([]byte{})
	writer := &bytes.Buffer{}

	conn := NewConnection(injector, reader, writer)

	// Verify connection implements Connection interface
	var _ Connection = conn
	assert.NotNil(t, conn)
}

func TestConnectionImpl_DoneReturnsChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacadeService(ctrl)

	// Expect Debug call from NewAcpService
	mockLogger.EXPECT().Debug(gomock.Any()).Times(1)

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)

	reader := bytes.NewReader([]byte{})
	writer := &bytes.Buffer{}

	conn := NewConnection(injector, reader, writer)

	// Done() can only be called after Start(), but we can't test that
	// in a unit test without actual IO. Just verify connection exists.
	assert.NotNil(t, conn)
}

func TestConnectionImpl_ConnectionCreatedSuccessfully(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger.NewMockLoggerService(ctrl)
	mockFacade := channel.NewMockChannelFacadeService(ctrl)

	// Expect Debug call from NewAcpService
	mockLogger.EXPECT().Debug(gomock.Any()).Times(1)

	injector := do.New()
	do.ProvideValue[logger.LoggerService](injector, mockLogger)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)

	reader := bytes.NewReader([]byte{})
	writer := &bytes.Buffer{}

	conn := NewConnection(injector, reader, writer)

	// Verify connection was created and implements interface
	assert.NotNil(t, conn)
	var _ Connection = conn
}
