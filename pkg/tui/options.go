// pkg/tui/options.go
package tui

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
)

// TUIOption implements channel.ChannelOption for TUI-specific configuration.
type TUIOption struct {
	applyFunc func(*TUIChannel) error
}

// Apply implements channel.ChannelOption.
func (o TUIOption) Apply(ch channel.Channel) error {
	tuiCh, ok := ch.(*TUIChannel)
	if !ok {
		return fmt.Errorf("TUI option applied to wrong channel type (got %T, expected *TUIChannel)", ch)
	}
	return o.applyFunc(tuiCh)
}

// WithChannelMessageChan sets the message channel for TUI communication.
func WithChannelMessageChan(ch chan<- channel.Message) TUIOption {
	return TUIOption{applyFunc: func(c *TUIChannel) error {
		c.messageChan = ch
		return nil
	}}
}

// WithChannelLogger sets the logger service.
func WithChannelLogger(l logger.LoggerService) TUIOption {
	return TUIOption{applyFunc: func(c *TUIChannel) error {
		c.logger = l
		return nil
	}}
}

// WithChannelRenderer sets the markdown renderer.
func WithChannelRenderer(r markdown.Renderer) TUIOption {
	return TUIOption{applyFunc: func(c *TUIChannel) error {
		c.renderer = r
		return nil
	}}
}
