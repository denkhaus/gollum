// pkg/tui/registration.go
package tui

import (
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/samber/do/v2"
)

// RegisterChannels registers the TUI channel with the DI container.
// Call this during application initialization before creating services.
func RegisterChannels(injector do.Injector) {
	do.ProvideNamedValue(injector, channel.ProviderPrefix+"tui", channel.ChannelFactory(func(opts ...channel.ChannelOption) (channel.Channel, error) {
		return NewTUIChannelWithOptions(injector, opts...)
	}))
}
