package transport

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/NoSuchNameException/sprout/internal/config"
)

// Compile-time check to ensure tcpConn implements the Transport interface.
var _ Transport = (*tcpConn)(nil)

// tcpConn manages plain TCP-tunneled connections for VLESS outbound traffic.
type tcpConn struct {
	uuidBytes [16]byte
}

// newTCPConn creates a new [tcpConn] transport instance configured with the provided outbound options.
func newTCPConn(opts *config.OutboundOption) Transport {
	return &tcpConn{
		uuidBytes: opts.UUIDBytes,
	}
}

// HandshakeAndWrap implements [Transport].
// It transmits the initial VLESS header over the raw TCP stream to establish the proxy connection.
func (t *tcpConn) HandshakeAndWrap(ctx context.Context, conn net.Conn, target string) (io.ReadWriteCloser, error) {
	if err := writeVLESSHeader(conn, t.uuidBytes, target); err != nil {
		conn.Close()
		return nil, fmt.Errorf("vless header: %w", err)
	}

	slog.Debug("[client] connection established", "target", target)

	return conn, nil
}
