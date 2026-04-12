// Package acp provides ACP channel registration for the channel factory system.
package acp

import (
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/samber/do/v2"
)

// Identifier is the unique channel identifier for the ACP channel.
const Identifier = channel.ChannelIdentifier("acp")

// RegisterChannels registers the ACP channel factory with the dependency injection system.
// This allows the ACP channel to be created via the channel facade.
func RegisterChannels(injector do.Injector) {
	do.ProvideNamedValue(injector, "channel_acp", channel.ChannelFactory(func(opts ...channel.ChannelOption) (channel.Channel, error) {
		// Create ACP service directly to avoid circular dependency
		// (ACP service depends on ChannelFacade, so we can't inject it here)
		logService := do.MustInvoke[logger.LoggerService](injector)
		facade := do.MustInvoke[channel.ChannelFacade](injector)

		// Create minimal ACP service instance
		svc := &acpServiceImpl{
			logger:   logService,
			facade:   facade,
			injector: injector,
		}

		// Apply any channel options (stdin/stdout for ACP connection)
		for _, opt := range opts {
			if err := opt.Apply(svc); err != nil {
				return nil, err
			}
		}

		return svc, nil
	}))
}
