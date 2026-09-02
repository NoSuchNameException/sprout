package transport

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/NoSuchNameException/sprout/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ Transport = (*grpcConn)(nil)

// grpcConn handles the setup of gRPC-tunneled connections for VLESS outbound traffic.
type grpcConn struct {
	serviceName string
	uuidBytes   [16]byte
	address     string
}

// newGRPCConn creates a new [grpcConn] transport instance configured with the provided outbound options.
func newGRPCConn(opts *config.OutboundOption) Transport {
	return &grpcConn{
		serviceName: opts.ServiceName,
		uuidBytes:   opts.UUIDBytes,
		address:     opts.Address,
	}
}

// HandshakeAndWrap establishes a bidirectional gRPC stream over the provided TLS connection,
// wraps it in an adapter implementing [io.ReadWriteCloser], and transmits the initial VLESS header.
func (g *grpcConn) HandshakeAndWrap(ctx context.Context, tlsUConn net.Conn, target string) (io.ReadWriteCloser, error) {
	grpcStream, cleanup, err := packetStream(ctx, tlsUConn, g.address, g.serviceName)
	if err != nil {
		tlsUConn.Close()
		return nil, fmt.Errorf("packet stream: %w", err)
	}

	grpcConn := NewGRPCAdapter(grpcStream, cleanup)

	if err := writeVLESSHeader(grpcConn, g.uuidBytes, target); err != nil {
		grpcConn.Close()
		return nil, fmt.Errorf("vless header: %w", err)
	}

	slog.Debug("[client] connection established", "target", target)
	return grpcConn, nil
}

// packetStream establishes a bidirectional gRPC stream over a pre-existing net.Conn.
// It returns the active client stream and a cleanup function to close the underlying gRPC ClientConn.
func packetStream(ctx context.Context, tlsConn net.Conn, target string, serviceName string) (grpc.ClientStream, func(), error) {
	conn, err := grpc.NewClient(
		target,
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return tlsConn, nil
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("new grpc client: %w", err)
	}

	stream, err := conn.NewStream(
		ctx,
		&grpc.StreamDesc{
			StreamName:    "Tun",
			ServerStreams: true,
			ClientStreams: true,
		},
		"/"+serviceName+"/Tun",
		grpc.ForceCodec(codec{}),
	)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("new stream: %w", err)
	}

	cleanup := func() {
		conn.Close()
	}

	return stream, cleanup, nil
}
