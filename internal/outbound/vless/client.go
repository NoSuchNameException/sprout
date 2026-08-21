package vless

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/NoSuchNameException/sprout/internal/config"
	"github.com/NoSuchNameException/sprout/internal/outbound"
	utls "github.com/refraction-networking/utls"
	"google.golang.org/grpc"
)

var _ outbound.Outbound = (*Client)(nil)

type Client struct {
	cfg       config.OutboundConfig
	serverPub []byte
	shortID   []byte
	uuidBytes [16]byte
	address   string
}

// NewClient creates a new VLESS+REALITY outbound client.
func NewClient(cfg config.OutboundConfig) (*Client, error) {
	serverPub, err := base64.RawURLEncoding.DecodeString(cfg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public_key")
	}

	shortID, err := hex.DecodeString(cfg.ShortID)
	if err != nil {
		return nil, fmt.Errorf("invalid short_id")
	}

	uuidBytes, err := parseUUID(cfg.UUID)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid")
	}

	address := net.JoinHostPort(cfg.Address, fmt.Sprintf("%d", cfg.Port))

	return &Client{
		cfg:       cfg,
		serverPub: serverPub,
		shortID:   shortID,
		uuidBytes: uuidBytes,
		address:   address,
	}, nil
}

// Connect establishes a VLESS+REALITY connection to the VPS
// and returns a grpc.ClientStream ready for proxying data to target.
func (c *Client) Connect(ctx context.Context, target string) (grpc.ClientStream, func(), error) {
	var dialer net.Dialer
	tcpConn, err := dialer.DialContext(ctx, "tcp", c.address)
	if err != nil {
		return nil, nil, fmt.Errorf("dial: %w", err)
	}

	tlsUConn := utls.UClient(tcpConn, &utls.Config{
		ServerName:         c.cfg.ServerName,
		InsecureSkipVerify: true,
	}, utls.HelloFirefox_Auto)

	if err := tlsUConn.BuildHandshakeState(); err != nil {
		tcpConn.Close()
		return nil, nil, fmt.Errorf("build handshake state: %w", err)
	}

	hello := tlsUConn.HandshakeState.Hello

	ecdhe := tlsUConn.HandshakeState.State13.KeyShareKeys.Ecdhe
	if ecdhe == nil {
		ecdhe = tlsUConn.HandshakeState.State13.KeyShareKeys.MlkemEcdhe
	}
	if ecdhe == nil {
		tcpConn.Close()
		return nil, nil, fmt.Errorf("no ecdhe key")
	}

	hello.SessionId = make([]byte, 32)
	copy(hello.Raw[39:], hello.SessionId)
	hello.SessionId[0] = 1 // x
	hello.SessionId[1] = 8 // y
	hello.SessionId[2] = 4 // z
	hello.SessionId[3] = 0 // reserved
	binary.BigEndian.PutUint32(hello.SessionId[4:], uint32(time.Now().Unix()))
	copy(hello.SessionId[8:], c.shortID)

	authKey, err := deriveAuthKey(ecdhe, c.serverPub, hello.Random)
	if err != nil {
		tcpConn.Close()
		return nil, nil, fmt.Errorf("derive auth key: %w", err)
	}

	hello.SessionId, err = sealSessionID(hello.SessionId, authKey, hello.Random[20:], hello.Raw)
	if err != nil {
		tcpConn.Close()
		return nil, nil, fmt.Errorf("seal session id: %w", err)
	}

	copy(hello.Raw[39:], hello.SessionId)

	if err := tlsUConn.HandshakeContext(ctx); err != nil {
		tcpConn.Close()
		return nil, nil, fmt.Errorf("tls handshake: %w", err)
	}

	if err := verifyRealityCert(tlsUConn.ConnectionState(), authKey); err != nil {
		tcpConn.Close()
		return nil, nil, fmt.Errorf("reality certificate: %w", err)
	}
	slog.Debug("[xray] certificate cecked, connected")

	grpcStream, cleanup, err := packetStream(ctx, tlsUConn, c.address, c.cfg.GRPCServiceName)
	if err != nil {
		tlsUConn.Close()
		return nil, nil, fmt.Errorf("packet stream: %w", err)
	}

	if err := writeVLESSHeader(grpcStream, c.uuidBytes, target); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("vless header: %w", err)
	}

	slog.Debug("[xray] connection established", "target", target)
	return grpcStream, cleanup, nil
}

func verifyRealityCert(state utls.ConnectionState, authKey []byte) error {
	certs := state.PeerCertificates
	if len(certs) == 0 {
		return fmt.Errorf("certificates not found")
	}

	leafCert := certs[0]

	pubKey, ok := leafCert.PublicKey.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("certificate key not Ed25519, we are in fallback mod")
	}

	h := hmac.New(sha512.New, authKey)
	h.Write(pubKey)
	expectedSignature := h.Sum(nil)

	if !bytes.Equal(expectedSignature, leafCert.Signature) {
		return fmt.Errorf("certivicates signarute not coincide")
	}

	return nil
}
