// Package acp provides ACP channel options for configuration.
package acp

import (
	"io"

	"github.com/denkhaus/gollum/pkg/channel"
)

// ACPOption implements channel.ChannelOption for ACP-specific configuration.
type ACPOption struct {
	applyFunc func(*acpServiceImpl) error
}

// Apply implements channel.ChannelOption.
func (o *ACPOption) Apply(ch channel.Channel) error {
	// Type assert to acpServiceImpl (concrete type)
	acpCh, ok := ch.(*acpServiceImpl)
	if !ok {
		return nil // Not an ACP channel, ignore
	}
	return o.applyFunc(acpCh)
}

// WithStdin sets the stdin reader for the ACP connection.
func WithStdin(r io.Reader) channel.ChannelOption {
	return &ACPOption{applyFunc: func(s *acpServiceImpl) error {
		s.stdin = r
		return nil
	}}
}

// WithStdout sets the stdout writer for the ACP connection.
func WithStdout(w io.Writer) channel.ChannelOption {
	return &ACPOption{applyFunc: func(s *acpServiceImpl) error {
		s.stdout = w
		return nil
	}}
}
