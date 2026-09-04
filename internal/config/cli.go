package config

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// CheckOrSetPort checks if the default port is available on the specified interface.
// If it's occupied, it interactively prompts the user to enter a new valid port.
func CheckOrSetPort(listen string, defaultPort uint16) uint16 {
	if isPortAvailable(listen, defaultPort) {
		return defaultPort
	}

	fmt.Printf("Default port %d is already in use.\n", defaultPort)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Enter new port: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		val, err := strconv.Atoi(input)
		if err != nil || val < 1 || val > 65535 {
			fmt.Println("Invalid port. Enter a number between 1 and 65535.")
			continue
		}

		port := uint16(val)
		if !isPortAvailable(listen, port) {
			fmt.Println("The port is alredy in use. Enter a different port.")
			continue
		}

		return port
	}
}

// isPortAvailable attempts to start a temporary TCP listener on the specified
// host and port to determine if it is currently free.
func isPortAvailable(listen string, port uint16) bool {
	addr := net.JoinHostPort(listen, strconv.Itoa(int(port)))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// CheckOrSetVLESS prompts the user for a VLESS URI link, parses it,
// and populates the OutboundConfig. It loops until a valid link is provided.
func CheckOrSetVLESS(out *OutboundConfig) bool {
	fmt.Println("Outbound not configured.")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Enter VLESS link (vless://...): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if !strings.HasPrefix(input, "vless://") {
			fmt.Println("Invalid link format. Must start with vless://")
			continue
		}

		if err := parseVLESSToOutbound(input, out); err != nil {
			fmt.Printf("Failed to parse link: %v\n", err)
			continue
		}

		return true
	}
}

// parseVLESSToOutbound extracts routing and security parameters from a raw
// VLESS URI string and maps them to the OutboundConfig structure.
func parseVLESSToOutbound(input string, out *OutboundConfig) error {
	u, err := url.Parse(input)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if u.Scheme != "vless" {
		return fmt.Errorf("unsupported protocol: %s", u.Scheme)
	}

	if u.User == nil || u.User.Username() == "" {
		return fmt.Errorf("uuid id missing in the link")
	}
	out.UUID = u.User.Username()

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("address (hostname) is missing in the link")
	}
	out.Address = host

	portStr := u.Port()
	if portStr == "" {
		return fmt.Errorf("port is missing in the link")
	}

	p, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid port in link: %w", err)
	}
	out.Port = uint16(p)

	q := u.Query()

	sec := q.Get("security")
	if sec == "" {
		sec = "none"
	}

	t := q.Get("type")
	if t == "" {
		t = "tcp"
	}

	out.PublicKey = q.Get("pbk")
	out.Security = sec
	out.ShortID = q.Get("sid")
	out.ServerName = q.Get("sni")
	out.Fingerprint = q.Get("fp")
	out.ServiceName = q.Get("serviceName")
	out.Type = t
	out.Remark = u.Fragment

	return nil
}
