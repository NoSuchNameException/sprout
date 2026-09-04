# Sprout

A lightweight, modular console proxy client written in Go utilizing **VLESS + REALITY**.

## About the Project

This project started as an exploratory deep-dive to understand how standard proxy clients work under the hood with Xray-core. As development progressed, I chose to minimize external dependencies and generated code by implementing key protocol components from scratch—including custom Protobuf parsing, manual gRPC stream handling, and transport integration.

This intentional constraint keeps the client lightweight and transparent, focusing the implementation entirely on core mechanics and protocol internals.

## Features

- **Flexible Architecture:** Modular dialer and transport design (supports TCP and gRPC layers).
- **Core Protocol:** VLESS + REALITY integration.
- **Low-Level Implementation:** 
  - Custom Protobuf parser without heavy code-generation bloat.
  - Manual stream management using standard grpc primitives instead of generated stubs.
- **Local Proxy:** Built-in SOCKS5 inbound proxy server.
- **Hardware-Protected Configs:** Encrypted configuration bound to the local machine via Machine ID.
- **Developer Experience:** Interactive CLI configuration prompts.
- **Automation:** Cross-platform builds (Linux, Windows, macOS) powered by GitHub Actions CI/CD.

## Configuration

```yaml
inbound:
  listen: 127.0.0.1
  port: 1080

outbound:
  address: 203.0.113.42
  port: 443
  uuid: Sugxy6pEbqzgzLO8ha3KwQk9zufA6SssrkysQ3KcOraM034WoWAajUqMdgZ2AitB2ic+8jtAyR7wCKcWFmDTdg==
  public_key: edgM73XpMM42M7SIze+Mdgpe9Drmltw5JbMIAse+AEhATyPuZBrryOP/s6AAIxc2g5I881168mz01Kr0Ow39TRCn5q3bPHc=
  security: reality
  short_id: WKMmM44rH9axBz0WaWHe87064ZXE6zEu8gMj4cLdlUQz+JbAse5Rxw==
  server_name: www.amd.com
  service_name: grpc-tunnel
  type: grpc
  fingerprint: firefox
  remark: gRPC
```

## Design Goals

The project focuses on keeping the implementation small and explicit:

- minimal runtime dependencies;
- no generated Protobuf code;
- protocol logic separated from traffic handling;
- configuration encrypted at rest;
- reproducible cross-platform builds.

## Status

The project is currently in active development. The current version provides 
a working VLESS + REALITY client supporting both **TCP** and **gRPC** transport layers, 
paired with a local SOCKS5 inbound interface.

Planned features include a TUN-based interface for system-wide proxying.

## Installation & Usage

1. Download Binary

    Go to the [Releases](../../releases) page and download the compiled binary for your operating system:
    * proxy-client-linux-amd64 — Linux (x64)
    * proxy-client-windows-amd64.exe — Windows (x64)
    * proxy-client-darwin-amd64 — macOS (Intel)
    * proxy-client-darwin-arm64 — macOS (Apple Silicon / M1/M2/M3)

2. Set Execution Permissions (macOS / Linux)

    Before running the downloaded binary, grant it execution permissions:
    ```
    chmod +x proxy-client-<os>-<arch>
    ```

3. Execution

    Run the client via the terminal by passing the required configuration parameters:

    * **Standard run**
    ```
    ./proxy-client-<os>-<arch>
    ```
    
    * **Debug run**
    ```
    ./proxy-client-<os>-<arch> -v
    ```

## Project Architecture
The project is structured around a modular design, ensuring a clear separation of concerns between handling incoming traffic, proxying, and core protocol implementations:

```
.
├── cmd/
│   └── vpn/            # Application entry point (main.go)
├── internal/
│   ├── config/         # Configuration management, options, CLI parsing, and crypto
│   ├── inbound/        # Incoming traffic handling
│   │   └── socks5/     # SOCKS5 server implementation (socks.go, inbound.go)
│   ├── outbound/       # Outbound connection management and VLESS/REALITY protocols
│   │   └── vless/      # VLESS core logic, dialers (raw, REALITY), and builders
│   ├── relay/          # Relay logic connecting inbound streams to outbound transport
│   └── transport/      # Unified transport layer (TCP, gRPC adapters, codecs, and headers)
├── .github/workflows/  # CI/CD pipelines
└── build.sh            # Cross-platform build script
```

---