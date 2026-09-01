// Package config provides configuration loading, parsing, and saving from a YAML file.
package config

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"

	"github.com/NoSuchNameException/sprout/internal/transport"
)

const (
	x25519PubKeyLen = 32
)

// OutboundOption holds parsed and validated parameters required to establish outbound connections.
type OutboundOption struct {
	ServiceName string
	ServerName  string
	ServerPub   []byte
	ShortID     []byte
	UUIDBytes   [16]byte
	Address     string
}

// ParseOutboundOption decodes and validates raw configuration fields into a strongly-typed [OutboundOption].
func ParseOutboundOption(cfg OutboundConfig) (*OutboundOption, error) {
	serverPub, err := base64.RawURLEncoding.DecodeString(cfg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	if len(serverPub) != x25519PubKeyLen {
		return nil, fmt.Errorf("invalid public key length: expected %d bytes, got %d", x25519PubKeyLen, len(serverPub))
	}

	shortID, err := hex.DecodeString(cfg.ShortID)
	if err != nil {
		return nil, fmt.Errorf("invalid short id: %w", err)
	}

	uuidBytes, err := transport.ParseUUID(cfg.UUID)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}

	address := net.JoinHostPort(cfg.Address, strconv.Itoa(int(cfg.Port)))

	return &OutboundOption{
		ServiceName: cfg.ServiceName,
		ServerName:  cfg.ServerName,
		ServerPub:   serverPub,
		ShortID:     shortID,
		UUIDBytes:   uuidBytes,
		Address:     address,
	}, nil
}
