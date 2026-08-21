package socks5

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/NoSuchNameException/sprout/internal/inbound"
)

var _ inbound.Inbound = (*Server)(nil)

type Server struct {
	addr    string
	channel chan *inbound.Request
}

func NewServer(addr string) *Server {
	return &Server{
		addr:    addr,
		channel: make(chan *inbound.Request, 256),
	}
}

// ListenAndServe starts the local inbound SOCKS5 proxy, listens for connections,
// and dispatches incoming requests to the internal processing channel.
func (s *Server) ListenAndServe(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	defer func() {
		wg.Wait()
		close(s.channel)
	}()

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("accept: %w", err)
		}

		wg.Go(func() {
			_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
			target, err := handshakeSocks5(conn)
			if err != nil {
				conn.Close()
				return
			}
			_ = conn.SetDeadline(time.Time{})

			select {
			case s.channel <- &inbound.Request{Conn: conn, Target: target}:
			case <-ctx.Done():
				conn.Close()
			}
		})
	}
}

func (s *Server) Accept() (*inbound.Request, error) {
	req, ok := <-s.channel
	if !ok {
		return nil, inbound.ErrClosed
	}
	return req, nil
}

func handshakeSocks5(conn net.Conn) (string, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", fmt.Errorf("greeting read: %w", err)
	}
	if header[0] != 0x05 {
		return "", fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}

	methods := make([]byte, header[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", fmt.Errorf("methods read: %w", err)
	}

	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return "", fmt.Errorf("greeting write: %w", err)
	}

	reqHeader := make([]byte, 4)
	if _, err := io.ReadFull(conn, reqHeader); err != nil {
		return "", fmt.Errorf("request header read: %w", err)
	}
	if reqHeader[1] != 0x01 {
		return "", fmt.Errorf("unsupported command: %d (only TCP CONNECT)", reqHeader[1])
	}

	var host string
	atyp := reqHeader[3]

	switch atyp {
	case 0x01:
		ip := make([]byte, 4)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", fmt.Errorf("ipv4 read: %w", err)
		}
		host = net.IP(ip).String()
	case 0x03:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return "", fmt.Errorf("domain len read: %w", err)
		}
		domain := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", fmt.Errorf("domain read: %w", err)
		}
		host = string(domain)
	case 0x04:
		ip := make([]byte, 16)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", fmt.Errorf("ipv6 read: %w", err)
		}
		host = net.IP(ip).String()
	default:
		return "", fmt.Errorf("unknown address type: %d", atyp)
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", fmt.Errorf("port read: %w", err)
	}
	port := binary.BigEndian.Uint16(portBuf)

	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return "", fmt.Errorf("success response write: %w", err)
	}
	return fmt.Sprintf("%s:%d", host, port), nil
}
