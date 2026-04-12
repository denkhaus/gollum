// Package acp provides ACP channel registration for the channel factory system.
package acp

import (
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
)

// Identifier is the unique channel identifier for the ACP channel.
const Identifier = channel.ChannelIdentifier("acp")

// RegisterChannels registers the ACP channel factory with the dependency injection system.
// This allows the ACP channel to be created via the channel facade.
func RegisterChannels(injector do.Injector) {
	do.ProvideNamedValue(injector, "channel_acp", channel.ChannelFactory(func(opts ...channel.ChannelOption) (channel.Channel, error) {
		// The ACP service is created by NewAcpService() in the DI system
		// We retrieve it here and return it as a channel
		service := do.MustInvoke[shared.ACPService](injector)

		// Type assert to channel.Channel (acpServiceImpl implements both)
		ch, ok := service.(channel.Channel)
		if !ok {
			return nil, nil // Should never happen - acpServiceImpl always implements Channel
		}

		// Apply any channel options (stdin/stdout for ACP connection)
		for _, opt := range opts {
			if err := opt.Apply(ch); err != nil {
				return nil, err
			}
		}

		return ch, nil
	}))
}
