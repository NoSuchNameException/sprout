package relay

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/NoSuchNameException/sprout/internal/inbound"
	"github.com/NoSuchNameException/sprout/internal/outbound"
)

const benchPayloadSize = 32 * 1024

var _ outbound.Outbound = (*MockOutbound)(nil)

type MockOutbound struct {
	conn io.ReadWriteCloser
}

func (m *MockOutbound) Connect(ctx context.Context, target string) (io.ReadWriteCloser, error) {
	if m.conn != nil {
		return m.conn, nil
	}
	return nil, nil
}

var bufPool = sync.Pool{
	New: func() any { return make([]byte, benchPayloadSize) },
}

var respPool = sync.Pool{
	New: func() any { return make([]byte, benchPayloadSize+2) },
}

// BenchmarkRelay_Lifecycle measures full connection lifecycle:
// dial → single packet → close.
// Most allocations originate from net.Pipe internals, not relay logic.
func BenchmarkRelay_Lifecycle(b *testing.B) {
	b.ReportAllocs()

	payload := make([]byte, benchPayloadSize)
	for i := range payload {
		payload[i] = 'x'
	}

	respBuf := make([]byte, benchPayloadSize)

	b.SetBytes(benchPayloadSize)
	b.ResetTimer()

	for b.Loop() {
		clientIn, serverIn := net.Pipe()
		clientOut, serverOut := net.Pipe()

		r := &Relay{Outbound: &MockOutbound{conn: serverOut}}
		req := &inbound.Request{Conn: serverIn, Target: "test.target:443"}

		ctx := context.Background()
		done := make(chan struct{})

		go func() {
			r.handle(ctx, req)
			close(done)
		}()

		go func() {
			buf := bufPool.Get().([]byte)
			defer bufPool.Put(buf)
			_, _ = clientOut.Read(buf)

			resp := respPool.Get().([]byte)
			defer respPool.Put(resp)
			resp[0], resp[1] = 0x00, 0x00
			copy(resp[2:], buf)
			_, _ = clientOut.Write(resp)
		}()

		_, _ = clientIn.Write(payload)
		_, _ = io.ReadFull(clientIn, respBuf)

		clientIn.Close()
		clientOut.Close()
		<-done
	}
}

// BenchmarkRelay_Stream measures sustained throughput on an established connection.
// Allocation-free on the hot path by design.
func BenchmarkRelay_Stream(b *testing.B) {
	b.ReportAllocs()

	payload := make([]byte, benchPayloadSize)
	for i := range payload {
		payload[i] = 'x'
	}

	clientIn, serverIn := net.Pipe()
	clientOut, serverOut := net.Pipe()

	r := &Relay{Outbound: &MockOutbound{conn: serverOut}}
	req := &inbound.Request{Conn: serverIn, Target: "test.target:443"}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.handle(ctx, req)
	}()

	firstResp := make([]byte, benchPayloadSize+2)
	firstResp[0], firstResp[1] = 0x00, 0x00

	go func() {
		buf := make([]byte, benchPayloadSize)
		isFirst := true

		for {
			n, err := clientOut.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				if isFirst {
					copy(firstResp[2:], buf[:n])
					_, _ = clientOut.Write(firstResp)
					isFirst = false
				} else {
					_, _ = clientOut.Write(buf[:n])
				}
			}
		}
	}()

	respBuf := make([]byte, benchPayloadSize)

	b.SetBytes(int64(benchPayloadSize))
	b.ResetTimer()

	for b.Loop() {
		_, _ = clientIn.Write(payload)
		if _, err := io.ReadFull(clientIn, respBuf); err != nil {
			b.Fatal(err)
		}
	}

	cancel()
	clientIn.Close()
	clientOut.Close()
	wg.Wait()
}
