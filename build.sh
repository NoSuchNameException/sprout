#!/bin/bash
set -e

VERSION=${1:-"dev"}
LDFLAGS="-s -w -X main.Version=$VERSION"

echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o proxy-client-linux-amd64 ./cmd/sprout

echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o proxy-client-windows-amd64.exe ./cmd/sprout

echo "Building for macOS (Intel)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -o proxy-client-darwin-amd64 ./cmd/sprout

echo "Building for macOS (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" -o proxy-client-darwin-arm64 ./cmd/sprout

echo "Build complete!"