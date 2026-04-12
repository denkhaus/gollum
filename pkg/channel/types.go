// Package channel provides the channel abstraction layer for Gollum.
package channel

import (
	"context"
)

// ChannelOption is the interface for channel configuration options.
// Each channel package implements its own option types that satisfy this interface.
type ChannelOption interface {
	Apply(Channel) error
}

// ChannelFactory creates a channel instance with options.
// Registered as named providers in DI (e.g., "channel_tui", "channel_acp").
type ChannelFactory func(opts ...ChannelOption) (Channel, error)

// ChannelIdentifier is the const type each channel exports.
// Provides type safety - no magic strings.
type ChannelIdentifier string

// ChannelStarter is the interface for channels that manage their own lifecycle.
// Channels like TUI implement this to run their main loop.
type ChannelStarter interface {
	Channel
	Start(ctx context.Context) error
}
