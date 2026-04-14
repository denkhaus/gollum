// pkg/tui/registration_test.go
package tui

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
)

func TestRegisterChannels(t *testing.T) {
	injector := do.New()

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
}

func TestRegisterChannels_Duplicate(t *testing.T) {
	injector := do.New()

	// First registration should succeed
	RegisterChannels(injector)

	// Second registration should panic (DI constraint)
	assert.Panics(t, func() {
		RegisterChannels(injector)
	})
}
