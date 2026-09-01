// Package grpc implements gRPC-based transport for outbound proxy connections.
package grpc

import (
	"fmt"
	"io"

	"google.golang.org/grpc"
)

var _ io.ReadWriteCloser = (*GRPCAdapter)(nil)

// GRPCAdapter wraps a bidirectional [grpc.ClientStream] to implement the standard [io.ReadWriteCloser] interface.
// It handles frame-to-stream byte buffering and connection lifecycle management.
type GRPCAdapter struct {
	stream  grpc.ClientStream
	cleanup func()
	buf     []byte
}

// NewGRPCAdapter constructs a new [GRPCAdapter] using the provided gRPC stream and cleanup function.
func NewGRPCAdapter(stream grpc.ClientStream, cleanup func()) io.ReadWriteCloser {
	return &GRPCAdapter{
		stream:  stream,
		cleanup: cleanup,
	}
}

// Close closes the sending side of the gRPC stream and executes the underlying cleanup function.
func (g *GRPCAdapter) Close() error {
	if g.stream != nil {
		g.stream.CloseSend()
	}

	if g.cleanup != nil {
		g.cleanup()
	}

	return nil
}

// Read reads data from the gRPC stream into p.
// If leftover bytes from a previous gRPC message remain in the internal buffer, they are drained first.
func (g *GRPCAdapter) Read(p []byte) (n int, err error) {
	if len(g.buf) > 0 {
		n = copy(p, g.buf)
		g.buf = g.buf[n:]
		return n, nil
	}

	var resp []byte
	if err := g.stream.RecvMsg(&resp); err != nil {
		return 0, fmt.Errorf("stream get message: %w", err)
	}

	n = copy(p, resp)
	if n < len(resp) {
		g.buf = resp[n:]
	}

	return n, nil
}

// Write sends raw bytes from p as a single frame through the gRPC stream using the custom codec.
func (g *GRPCAdapter) Write(p []byte) (n int, err error) {
	if err := g.stream.SendMsg(p); err != nil {
		return 0, fmt.Errorf("stream send message: %w", err)
	}

	return len(p), nil
}
