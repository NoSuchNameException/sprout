// Package outbound defines core interfaces for establishing and managing outgoing connections.
package outbound

import (
	"context"
	"io"
)

// Outbound orchestrates the dialing and transport wrapping to establish a connection to a target via VPS.
type Outbound interface {
	// Connect dials the target address and returns an active connection.
	Connect(ctx context.Context, target string) (io.ReadWriteCloser, error)
}
