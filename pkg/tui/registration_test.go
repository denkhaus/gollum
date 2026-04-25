// pkg/tui/registration_test.go
package tui

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRegisterChannels(t *testing.T) {
	injector := do.New()

	// Register a mock ChannelFacade first (required by NewTUIChannelWithOptions)
	ctrl := gomock.NewController(t)
	mockFacade := channel.NewMockChannelFacade(ctrl)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)

	RegisterChannels(injector)

	// Verify the factory was registered
	factory, err := do.InvokeNamed[channel.ChannelFactory](injector, "channel_tui")
	assert.NoError(t, err)
	assert.NotNil(t, factory)

	// Verify factory creates TUIChannel
	ch, err := factory()
	assert.NoError(t, err)
	assert.NotNil(t, ch)

	// Type check
	_, ok := ch.(*TUIChannel)
	assert.True(t, ok, "Factory should create *TUIChannel")

	// Verify TUIChannel implements Channel interface
	_, ok = ch.(channel.Channel)
	assert.True(t, ok, "TUIChannel should implement channel.Channel")
}

func TestRegisterChannels_Duplicate(t *testing.T) {
	injector := do.New()

	// Register a mock ChannelFacade first
	ctrl := gomock.NewController(t)
	mockFacade := channel.NewMockChannelFacade(ctrl)
	do.ProvideValue[channel.ChannelFacade](injector, mockFacade)

	// First registration should succeed
	RegisterChannels(injector)

	// Second registration should panic (DI constraint)
	assert.Panics(t, func() {
		RegisterChannels(injector)
	})
}
