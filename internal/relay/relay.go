// Package relay connects inbound and outbound, proxying data between them.
package relay

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/NoSuchNameException/sprout/internal/inbound"
	"github.com/NoSuchNameException/sprout/internal/outbound"
)

// Relay bridges an inbound listener and an outbound dialer.
// Each accepted Request is proxied to its target in a separate goroutine.
type Relay struct {
	Inbound  inbound.Inbound
	Outbound outbound.Outbound
}

// NewRelay creates a new Relay with the given inbound and outbound.
func NewRelay(in inbound.Inbound, out outbound.Outbound) *Relay {
	return &Relay{
		Inbound:  in,
		Outbound: out,
	}
}

// Run accepts requests from inbound in a loop and proxies each one
// through outbound until ctx is cancelled.
func (r *Relay) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := r.Inbound.Accept()
		if err != nil {
			return fmt.Errorf("inbound accept: %w", err)
		}

		slog.Debug("[relay] accepted request", "target", req.Target)
		go r.handle(ctx, req)
	}
}

func (r *Relay) handle(ctx context.Context, req *inbound.Request) {
	defer req.Conn.Close()

	stream, cleanup, err := r.Outbound.Connect(ctx, req.Target)
	if err != nil {
		slog.Debug("[relay] outbound error", "err", err)
		return
	}
	defer cleanup()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer stream.CloseSend()

		buf := make([]byte, 32*1024)

		for {
			n, err := req.Conn.Read(buf)
			if n > 0 {
				if sendErr := stream.SendMsg(buf[:n]); sendErr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		defer wg.Done()
		defer req.Conn.Close()

		isFirstRecv := true
		for {
			var resp []byte
			if err := stream.RecvMsg(&resp); err != nil {
				break
			}

			if isFirstRecv {
				isFirstRecv = false

				if len(resp) < 2 {
					slog.Debug("vless response header too short")
					break
				}

				addonLen := int(resp[1])
				headerLen := 2 + addonLen

				if len(resp) < headerLen {
					slog.Debug("vless response header truncated")
					break
				}

				resp = resp[headerLen:]
			}

			if _, err := req.Conn.Write(resp); err != nil {
				break
			}
		}
	}()

	wg.Wait()
}
