// Package middleware provides channel middleware for agent interactions.
package middleware

import (
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
)

type (
	// ChannelMiddlewareProvider creates channel middleware instances via DI
	ChannelMiddlewareProvider interface {
		CreateChannelMiddleware(agentID uuid.UUID, agentRole string) *channel.ChannelMiddleware
	}

	channelMiddlewareProvider struct {
		facade channel.ChannelFacade
	}
)

// NewChannelMiddlewareProvider creates a provider for channel middleware
func NewChannelMiddlewareProvider(injector do.Injector) (ChannelMiddlewareProvider, error) {
	facade := do.MustInvoke[channel.ChannelFacade](injector)

	return &channelMiddlewareProvider{
		facade: facade,
	}, nil
}

// CreateChannelMiddleware creates a new channel middleware for a specific agent
func (p *channelMiddlewareProvider) CreateChannelMiddleware(
	agentID uuid.UUID,
	agentRole string,
) *channel.ChannelMiddleware {
	return channel.NewChannelMiddleware(p.facade, agentID, agentRole)
}
