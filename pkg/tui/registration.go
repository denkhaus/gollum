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
		// Inject the facade from DI (same pattern as ACP)
		facade := do.MustInvoke[channel.ChannelFacade](injector)

		ch := NewTUIChannel()
		// Apply facade first, then other options
		if err := WithChannelFacade(facade).Apply(ch); err != nil {
			return nil, err
		}
		for _, opt := range opts {
			if err := opt.Apply(ch); err != nil {
				return nil, err
			}
		}
		return ch, nil
	}))
}
