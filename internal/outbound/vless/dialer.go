// Package dialer provides network dialers for establishing secure outbound connections.
package vless

import (
	"context"
	"fmt"
	"net"

	"github.com/NoSuchNameException/sprout/internal/config"
)

// Dialer establishes the low-level network or TLS connection to the remote server.
type Dialer interface {
	// Dial connects to the server and returns a base net.Conn (e.g., raw TCP or authenticated uTLS connection).
	Dial(ctx context.Context) (net.Conn, error)
}

// Build constructs a Dialer based on the provided security type.
func Build(dialerType string, opts *config.OutboundOption) (Dialer, error) {
	dialer, exists := registry[dialerType]
	if !exists {
		return nil, fmt.Errorf("unsupported security type: %q", dialerType)
	}

	return dialer(opts), nil
}

// builder defines the constructor signature for all dialers.
type builder func(opts *config.OutboundOption) Dialer

var registry = map[string]builder{
	"none":    newRaw,
	"reality": newReality,
}
