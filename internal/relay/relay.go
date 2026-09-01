// Package relay connects inbound listeners and outbound dialers,
// proxying bidirectional data between them.
package relay

import (
	"context"
	"log/slog"
	"sync"

	"github.com/NoSuchNameException/sprout/internal/inbound"
	"github.com/NoSuchNameException/sprout/internal/outbound"
)

// Relay bridges an inbound listener and an outbound dialer.
// Each accepted [inbound.Request] is proxied to its target destination in a separate goroutine.
type Relay struct {
	Inbound  inbound.Inbound
	Outbound outbound.Outbound
}

// NewRelay creates and initializes a new [Relay] with the provided inbound and outbound instances.
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
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Error("[relay] failed to accept inbound connection", "err", err)
			continue
		}

		slog.Debug("[relay] accepted request", "target", req.Target)
		go r.handle(ctx, req)
	}
}

// bufferPool reuses 32 KB byte buffers across connection goroutines to minimize heap allocations and GC overhead.
var bufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

// handle establishes an outbound connection for a given request and manages bidirectional data copy.
func (r *Relay) handle(ctx context.Context, req *inbound.Request) {
	conn, err := r.Outbound.Connect(ctx, req.Target)
	if err != nil {
		req.Conn.Close()
		slog.Debug("[relay] outbound connection failed", "target", req.Target, "err", err)
		return
	}

	var closeOnce sync.Once
	closeAll := func() {
		closeOnce.Do(func() {
			req.Conn.Close()
			conn.Close()
		})
	}
	defer closeAll()

	var wg sync.WaitGroup
	wg.Add(2)

	// Inbound -> Outbound
	go func() {
		defer wg.Done()
		defer closeAll()

		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)
		buf := *bufPtr

		for {
			n, err := req.Conn.Read(buf)
			if n > 0 {
				if _, sendErr := conn.Write(buf[:n]); sendErr != nil {
					slog.Debug("[relay] write to outbound failed", "target", req.Target, "err", sendErr)
					break
				}
			}
			if err != nil {
				break
			}
		}
	}()

	// Outbound -> Inbound
	go func() {
		defer wg.Done()
		defer closeAll()

		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)
		buf := *bufPtr

		isFirstRecv := true
		var headerBuf []byte

		for {
			n, err := conn.Read(buf)
			if n > 0 {
				data := buf[:n]

				if isFirstRecv {
					headerBuf = append(headerBuf, data...)

					if len(headerBuf) < 2 {
						continue
					}

					addonLen := int(headerBuf[1])
					headerLen := 2 + addonLen

					if len(headerBuf) < headerLen {
						continue
					}

					isFirstRecv = false
					data = headerBuf[headerLen:]
					headerBuf = nil
				}

				if len(data) > 0 {
					if _, writeErr := req.Conn.Write(data); writeErr != nil {
						slog.Debug("[relay] write to inbound failed", "target", req.Target, "err", writeErr)
						break
					}
				}
			}
			if err != nil {
				break
			}
		}
	}()

	wg.Wait()
	slog.Debug("[relay] connection closed", "tartet", req.Target)
}
