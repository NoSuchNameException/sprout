// Package dialer provides network dialers for establishing secure outbound connections.
package dialer

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/NoSuchNameException/sprout/internal/config"
	"github.com/NoSuchNameException/sprout/internal/outbound"
	utls "github.com/refraction-networking/utls"
)

const (
	// TLS ClientHello offsets
	sessionIDOffset   = 39
	sessionIDLen      = 32
	minClientHelloLen = sessionIDOffset + sessionIDLen // 71 bytes

	// SessionID payload offsets
	timestampOffset   = 4
	shortIDOffset     = 8
	authSaltOffset    = 20
	authSaltLen       = 20
	sessionIDPlainLen = 16 // Unencrypted prefix length before GCM tag

	// Cryptographic parameters
	authKeyLen      = 32
	gcmNonceLen     = 12
	hkdfInfoReality = "REALITY"
)

var _ outbound.Dialer = (*Reality)(nil)

// Reality implements the [outbound.Dialer] interface to perform VLESS+REALITY handshakes over uTLS.
type Reality struct {
	serverName string
	serverPub  []byte
	shortID    []byte
	uuidBytes  [16]byte
	address    string
}

// NewReality creates and initializes a new VLESS+REALITY outbound dialer with the given configuration options.
func NewReality(opts *config.OutboundOption) *Reality {
	return &Reality{
		serverName: opts.ServerName,
		serverPub:  opts.ServerPub,
		shortID:    opts.ShortID,
		uuidBytes:  opts.UUIDBytes,
		address:    opts.Address,
	}
}

// Dial establishes a TCP connection to the target server, executes a spoofed TLS 1.3 ClientHello
// using uTLS (Firefox fingerprint), performs authentication via REALITY ECDHE key exchange, and verifies the response.
func (r *Reality) Dial(ctx context.Context) (net.Conn, error) {
	slog.Debug("[reality] dialing server", "address", r.address, "sni", r.serverName)

	var dialer net.Dialer
	tcpConn, err := dialer.DialContext(ctx, "tcp", r.address)
	if err != nil {
		return nil, fmt.Errorf("dial tcp: %w", err)
	}

	tlsUConn := utls.UClient(tcpConn, &utls.Config{
		ServerName:         r.serverName,
		InsecureSkipVerify: true,
	}, utls.HelloFirefox_Auto)

	if err := tlsUConn.BuildHandshakeState(); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("build handshake state: %w", err)
	}

	authKey, err := r.prepareSessionID(&tlsUConn.HandshakeState)
	if err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("prepare reality session: %w", err)
	}

	if err := tlsUConn.HandshakeContext(ctx); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("tls handshake context: %w", err)
	}

	if err := verifyRealityCert(tlsUConn.ConnectionState(), authKey); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("verify reality cert: %w", err)
	}

	slog.Debug("[reality] TLS handshake completed and certificate verified", "sni", r.serverName)

	return tlsUConn, nil
}

// prepareSessionID extracts ECDHE keys, constructs the masked REALITY SessionID payload,
// and returns the derived authKey for certificate verification.
func (r *Reality) prepareSessionID(state *utls.PubClientHandshakeState) ([]byte, error) {
	hello := state.Hello

	if len(hello.Raw) < minClientHelloLen {
		return nil, fmt.Errorf("invalid client hello length: %d (expected at least %d bytes)", len(hello.Raw), minClientHelloLen)
	}

	ecdhe := state.State13.KeyShareKeys.Ecdhe
	if ecdhe == nil {
		ecdhe = state.State13.KeyShareKeys.MlkemEcdhe
	}
	if ecdhe == nil {
		return nil, fmt.Errorf("no ecdhe key available in key share")
	}

	// Construct custom SessionID payload for REALITY authentication
	hello.SessionId = make([]byte, sessionIDLen)
	copy(hello.Raw[sessionIDOffset:], hello.SessionId)
	hello.SessionId[0] = 1 // x
	hello.SessionId[1] = 8 // y
	hello.SessionId[2] = 4 // z
	hello.SessionId[3] = 0 // reserved
	binary.BigEndian.PutUint32(hello.SessionId[timestampOffset:], uint32(time.Now().Unix()))
	copy(hello.SessionId[shortIDOffset:], r.shortID)

	authKey, err := deriveAuthKey(ecdhe, r.serverPub, hello.Random)
	if err != nil {
		return nil, fmt.Errorf("derive auth key: %w", err)
	}

	hello.SessionId, err = sealSessionID(hello.SessionId, authKey, hello.Random[authSaltOffset:], hello.Raw)
	if err != nil {
		return nil, fmt.Errorf("seal session id: %w", err)
	}

	copy(hello.Raw[sessionIDOffset:], hello.SessionId)
	return authKey, nil
}
