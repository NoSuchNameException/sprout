// Package transport implements network transports (TCP, gRPC) and the VLESS protocol handshake for proxy connections.
package transport

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/NoSuchNameException/sprout/internal/config"
)

// Transport performs the high-level protocol handshake (e.g., VLESS over gRPC) over an established connection.
type Transport interface {
	// HandshakeAndWrap performs the protocol handshake over rawConn and wraps it into a multiplexed or proxied stream.
	HandshakeAndWrap(ctx context.Context, conn net.Conn, target string) (io.ReadWriteCloser, error)
}

// Build constructs and returns a [Transport] instance based on the requested transportType (e.g., "tcp", "grpc").
// It returns an error if the specified transport type is not supported.
func Build(transportType string, opts *config.OutboundOption) (Transport, error) {
	builder, exists := registry[transportType]
	if !exists {
		return nil, fmt.Errorf("unsupported transport: %q", transportType)
	}

	return builder(opts), nil
}

// builder defines the signature for transport constructor functions.
type builder func(opts *config.OutboundOption) Transport

// registry maps transport type strings to their respective builder functions.
var registry = map[string]builder{
	"tcp":  newTCPConn,
	"grpc": newGRPCConn,
}
