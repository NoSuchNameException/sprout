package vless

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ net.Conn = (*GRPCConn)(nil)

type GRPCConn struct {
	writer *io.PipeWriter
	reader io.ReadCloser
	respCh chan *http.Response
	errCh  chan error
}

// packetStream establishes and manages a bidirectional gRPC stream for network traffic.
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
		grpc.ForceCodec(vlessCodec{}),
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

func (g *GRPCConn) Close() error {
	err1 := g.writer.Close()
	if g.reader == nil {
		return err1
	}
	err2 := g.reader.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (g *GRPCConn) Read(b []byte) (n int, err error) {
	panic("unimplemented")
}

func (g *GRPCConn) Write(b []byte) (n int, err error) {
	panic("unimplemented")
}

func (g *GRPCConn) LocalAddr() net.Addr                { return nil }
func (g *GRPCConn) RemoteAddr() net.Addr               { return nil }
func (g *GRPCConn) SetDeadline(t time.Time) error      { return nil }
func (g *GRPCConn) SetReadDeadline(t time.Time) error  { return nil }
func (g *GRPCConn) SetWriteDeadline(t time.Time) error { return nil }
