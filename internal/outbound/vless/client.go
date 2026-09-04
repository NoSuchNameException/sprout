// Package vless provides a high-level VLESS client orchestrating outbound dialers and transports.
package vless

import (
	"context"
	"fmt"
	"io"

	"github.com/NoSuchNameException/sprout/internal/config"
	"github.com/NoSuchNameException/sprout/internal/outbound"
	"github.com/NoSuchNameException/sprout/internal/transport"
)

var _ outbound.Outbound = (*Client)(nil)

// Client coordinates the underlying dialer (REALITY) and transport layer (gRPC) to manage VLESS connections.
type Client struct {
	dialer    Dialer
	transport transport.Transport
}

// NewClient parses configuration options and constructs a fully initialized VLESS+REALITY [Client].
func NewClient(cfg config.OutboundConfig) (*Client, error) {
	opts, err := config.ParseOutboundOption(cfg)
	if err != nil {
		return nil, fmt.Errorf("parse outbound options: %w", err)
	}

	d, err := Build(cfg.Security, opts)
	if err != nil {
		return nil, fmt.Errorf("dialer build: %w", err)
	}

	t, err := transport.Build(cfg.Type, opts)
	if err != nil {
		return nil, fmt.Errorf("trannsport build: %w", err)
	}

	return &Client{
		dialer:    d,
		transport: t,
	}, nil
}

// Connect dials the outbound server via the configured dialer and wraps the connection using the transport layer.
func (c *Client) Connect(ctx context.Context, target string) (io.ReadWriteCloser, error) {
	tlsUConn, err := c.dialer.Dial(ctx)
	if err != nil {
		return nil, fmt.Errorf("dial outbound: %w", err)
	}

	conn, err := c.transport.HandshakeAndWrap(ctx, tlsUConn, target)
	if err != nil {
		tlsUConn.Close()
		return nil, fmt.Errorf("transport handshake: %w", err)
	}

	return conn, nil
}
