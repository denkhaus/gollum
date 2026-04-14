// Package acp provides ACP channel registration for the channel factory system.
package acp

import (
	"os"
	"github.com/google/uuid"
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

// Identifier is the unique channel identifier for the ACP channel.
const Identifier = channel.ChannelIdentifier("acp")

// RegisterChannels registers the ACP channel factory with the dependency injection system.
// This allows the ACP channel to be created via the channel facade.
func RegisterChannels(injector do.Injector) {
	do.ProvideNamedValue(injector, "channel_acp", channel.ChannelFactory(func(opts ...channel.ChannelOption) (channel.Channel, error) {
		svc, err := NewAcpServiceWithOptions(injector, opts...)
		if err != nil {
			return nil, err
		}
		// Debug: write to file to verify factory was called
		os.WriteFile("/tmp/acp_factory_debug", []byte("Factory called, ID: "+svc.ID().String()+"\n"), 0644)
		return svc, nil
	}))
}


// NewAcpServiceWithOptions creates a new ACP service with channel options.
// This is used by the channel factory to apply options like stdin/stdout/transport.
func NewAcpServiceWithOptions(injector do.Injector, opts ...channel.ChannelOption) (channel.Channel, error) {
	logger := do.MustInvoke[logger.LoggerService](injector)
	facade := do.MustInvoke[channel.ChannelFacade](injector)
	cfg := do.MustInvoke[config.ConfigService](injector)

	// Generate unique channel ID for this ACP service instance
	id := uuid.New()

	// Create service with generated ID
	svc := &acpServiceImpl{
		logger:   logger,
		facade:   facade,
		config:   cfg,
		id:       id,
		injector: injector,
	}

	// Apply any channel options (stdin/stdout/transport for ACP connection)
	for _, opt := range opts {
		if err := opt.Apply(svc); err != nil {
			return nil, err
		}
	}

	logger.Debug("ACP service created with options", 
		zap.String("channel_id", id.String()),
		zap.Int("options_count", len(opts)))

	return svc, nil  // Type assert to channel.Channel interface
}
