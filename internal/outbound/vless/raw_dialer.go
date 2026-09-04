package vless

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/NoSuchNameException/sprout/internal/config"
)

var _ Dialer = (*rawDialer)(nil)

// rawDialer implements a direct TCP connection without any encryption (security="none").
type rawDialer struct {
	address string
}

// newRaw creates a new rawDialer. It requires no specific configuration beyond the destination address.
func newRaw(opts *config.OutboundOption) Dialer {
	return &rawDialer{
		address: opts.Address,
	}
}

// Dial establishes a TCP connection to the target server.
func (t *rawDialer) Dial(ctx context.Context) (net.Conn, error) {
	slog.Debug("[raw] dialing server", "address", t.address)

	var dialer net.Dialer
	tcpConn, err := dialer.DialContext(ctx, "tcp", t.address)
	if err != nil {
		return nil, fmt.Errorf("dial tcp: %w", err)
	}

	return tcpConn, nil
}
