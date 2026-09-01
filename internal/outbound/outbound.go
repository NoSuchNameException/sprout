// Package outbound defines core interfaces for establishing and managing outgoing connections.
package outbound

import (
	"context"
	"io"
	"net"
)

// Outbound orchestrates the dialing and transport wrapping to establish a connection to a target via VPS.
type Outbound interface {
	// Connect dials the target address and returns an active connection.
	Connect(ctx context.Context, target string) (io.ReadWriteCloser, error)
}

// Dialer establishes the low-level network or TLS connection to the remote server.
type Dialer interface {
	// Dial connects to the server and returns a base net.Conn (e.g., after REALITY/uTLS verification).
	Dial(ctx context.Context) (net.Conn, error)
}

// Transport performs the high-level protocol handshake (e.g., VLESS over gRPC) over an established connection.
type Transport interface {
	// HandshakeAndWrap performs the protocol handshake over rawConn and wraps it into a multiplexed or proxied stream.
	HandshakeAndWrap(ctx context.Context, conn net.Conn, target string) (io.ReadWriteCloser, error)
}
