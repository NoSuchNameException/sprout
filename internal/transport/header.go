// Package transport provides network transport primitives and protocol header builders.
package transport

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
)

// WriteVLESSHeader builds and writes a standard VLESS request header to the connection.
// It must be called immediately after establishing the underlying transport connection
// and before proxying payload data.
//
// The header binary format follows the VLESS spec:
//   - [1]  Version (0x00)
//   - [16] User ID (UUID)
//   - [1]  Addon Length (0x00)
//   - [1]  Command (0x01 - TCP)
//   - [2]  Port (BigEndian uint16)
//   - [1]  Address Type (0x01 IPv4 / 0x02 Domain / 0x03 IPv6)
//   - [N]  Address Payload (4 bytes for IPv4, 1 + N bytes for Domain, 16 bytes for IPv6)
func WriteVLESSHeader(conn io.ReadWriteCloser, uuid [16]byte, target string) error {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return fmt.Errorf("split host / port: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port range: %s", portStr)
	}

	slog.Debug("[vless] dialing target via VLESS", "host", host, "port", port)

	// Pre-allocate slice with capacity covering max possible header size to avoid re-allocations
	buf := make([]byte, 0, 32+len(host))
	buf = append(buf, 0x00)       //Version 0
	buf = append(buf, uuid[:]...) //Client UUID
	buf = append(buf, 0x00)       //Addon length (0)
	buf = append(buf, 0x01)       //Command (TCP)
	buf = binary.BigEndian.AppendUint16(buf, uint16(port))

	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			buf = append(buf, 0x01) // 0x01 - IPv4
			buf = append(buf, ip4...)
		} else {
			buf = append(buf, 0x03) // 0x03 - IPv6
			buf = append(buf, ip.To16()...)
		}
	} else {
		if len(host) > 255 {
			return fmt.Errorf("domain name too long: %d bytes (max 255)", len(host))
		}
		buf = append(buf, 0x02) // 0x02 - Domain
		buf = append(buf, byte(len(host)))
		buf = append(buf, host...)
	}

	if _, err := conn.Write(buf); err != nil {
		return fmt.Errorf("write vless header: %w", err)
	}

	return nil
}

// ParseUUID parses a standard 36-character UUID string (with or without hyphens)
// into a [16]byte array. Returns an error if the string is not a valid 128-bit hex representation.
func ParseUUID(uuid string) ([16]byte, error) {
	var uuidBytes [16]byte
	cleaned := strings.ReplaceAll(uuid, "-", "")

	if len(cleaned) != 32 {
		return uuidBytes, fmt.Errorf("parse uuid: invalid length %d (expected 32 hex characters)", len(cleaned))
	}

	if _, err := hex.Decode(uuidBytes[:], []byte(cleaned)); err != nil {
		return uuidBytes, fmt.Errorf("parse uuid: %w", err)
	}

	return uuidBytes, nil
}
