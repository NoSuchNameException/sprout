package vless

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"

	"google.golang.org/grpc"
)

// writeVLESSHeader writes a VLESS request header to conn.
// Must be called after TLS handshake, before proxying data.
func writeVLESSHeader(stream grpc.ClientStream, uuid [16]byte, target string) error {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return fmt.Errorf("split host / port: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("convert port to int: %w", err)
	}

	slog.Debug("[vless] dialing target via VLESS", "host", host, "port", port)
	// VLESS request header (see VLESS protocol spec):
	// [1]  version      = 0x00
	// [16] UUID         = client identity
	// [1]  addon length = 0x00 (no addons)
	// [1]  command      = 0x01 (TCP)
	// [2]  port         = big-endian uint16
	// [1]  addr type    = 0x01 IPv4 / 0x02 domain / 0x03 IPv6
	// [1]  addr length  = len(host)
	// [N]  addr         = host bytes
	var buf []byte
	buf = append(buf, 0x00)
	buf = append(buf, uuid[:]...)
	buf = append(buf, 0x00)
	buf = append(buf, 0x01)
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
		buf = append(buf, 0x02) // 0x02 - Domain
		buf = append(buf, byte(len(host)))
		buf = append(buf, []byte(host)...)
	}

	if err := stream.SendMsg(buf); err != nil {
		return fmt.Errorf("write vless header for grpc client send: %w", err)
	}

	return nil
}

// parseUUID parses a UUID string ("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx") into [16]byte.
func parseUUID(uuid string) ([16]byte, error) {
	var uuidBytes [16]byte
	cleaned := strings.ReplaceAll(uuid, "-", "")
	if _, err := hex.Decode(uuidBytes[:], []byte(cleaned)); err != nil {
		return uuidBytes, fmt.Errorf("parse uuid: %w", err)
	}
	return uuidBytes, nil
}
