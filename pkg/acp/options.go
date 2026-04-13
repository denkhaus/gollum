// Package acp provides ACP channel options for configuration.
package acp

import (
	"fmt"
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

// WithTransport sets the transport type for ACP communication.
func WithTransport(t TransportType) channel.ChannelOption {
	return &ACPOption{applyFunc: func(s *acpServiceImpl) error {
		s.transportType = t
		return nil
	}}
}

// WithHost sets the host address for HTTP transport.
func WithHost(host string) channel.ChannelOption {
	return &ACPOption{applyFunc: func(s *acpServiceImpl) error {
		if host == "" {
			return fmt.Errorf("host cannot be empty")
		}
		s.host = host
		return nil
	}}
}

// WithPort sets the port number for HTTP transport.
func WithPort(port int) channel.ChannelOption {
	return &ACPOption{applyFunc: func(s *acpServiceImpl) error {
		if port < 1 || port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535, got %d", port)
		}
		s.port = port
		return nil
	}}
}
