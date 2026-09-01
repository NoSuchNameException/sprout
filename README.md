# Sprout

A lightweight, modular console proxy client written in Go utilizing **VLESS + REALITY** over a **gRPC** transport layer.

## About the Project

This project started as an exploratory attempt to understand how standard proxy clients work under the hood with Xray-core. As development progressed, I decided to minimize external dependencies and generated code by implementing the required protocol components myself, including custom Protobuf parsing, manual gRPC stream handling, and transport integration.

This intentional limitation keeps the client lightweight and transparent, while focusing the implementation on core mechanics and protocol internals.

## Features

- VLESS + REALITY over gRPC
- Custom Protobuf parser
- Manual gRPC stream handling using the standard gRPC API
- SOCKS5 inbound proxy
- Encrypted configuration
- CLI configuration
- Cross-platform builds for Linux, Windows and macOS
- GitHub Actions CI/CD

## Design Goals

The project focuses on keeping the implementation small and explicit:

- minimal runtime dependencies;
- no generated Protobuf code;
- protocol logic separated from traffic handling;
- configuration encrypted at rest;
- reproducible cross-platform builds.

## Status

The project is currently in early development. The current release provides
a working VLESS + REALITY client over gRPC with a SOCKS5 inbound interface.

Planned features include TCP-based connections and a TUN-based interface.

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
├── cmd/vpn/            # Application entry point (main.go)
├── internal/
│   ├── config/         # Configuration management, CLI parsing, and encryption (crypto.go)
│   ├── inbound/        # Incoming traffic handling (SOCKS5 server implementation)
│   ├── outbound/       # Outbound transport and protocols:
│   │   └── vless/      # VLESS, REALITY, gRPC streams, and custom Protobuf parser
│   └── relay/          # Relay logic connecting inbound streams to outbound transport
├── .github/workflows/  # CI/CD pipelines
└── build.sh            # Cross-platform build script
```

---