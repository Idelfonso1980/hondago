#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

mkdir -p dist/macos

echo "Building macOS arm64 binaries..."
GOOS=darwin GOARCH=arm64 go build -o dist/macos/honda-go-gui-arm64 ./cmd/honda-gui
GOOS=darwin GOARCH=arm64 go build -o dist/macos/honda-go-engine-arm64 ./cmd/honda-engine

echo "Building macOS amd64 binaries..."
GOOS=darwin GOARCH=amd64 go build -o dist/macos/honda-go-gui-amd64 ./cmd/honda-gui
GOOS=darwin GOARCH=amd64 go build -o dist/macos/honda-go-engine-amd64 ./cmd/honda-engine

echo "Done. Files:"
ls -la dist/macos
