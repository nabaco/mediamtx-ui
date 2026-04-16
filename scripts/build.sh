#!/usr/bin/env bash
# build.sh — build the frontend then compile the single Go binary
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$ROOT/frontend"
EMBED_DIR="$ROOT/internal/web/dist"
BIN_DIR="$ROOT/bin"
BINARY="$BIN_DIR/mediamtx-ui"

VERSION="${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}"

echo "==> Building mediamtx-ui $VERSION"

# 1. Build frontend
echo "--- Building frontend..."
cd "$FRONTEND_DIR"
npm ci --silent
npm run build

# 2. Copy dist into the embed directory
echo "--- Copying frontend/dist → internal/web/dist..."
rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR"
cp -r "$FRONTEND_DIR/dist/." "$EMBED_DIR/"

# 3. Compile Go binary
echo "--- Compiling Go binary..."
mkdir -p "$BIN_DIR"
cd "$ROOT"
CGO_ENABLED=0 go build \
  -trimpath \
  -ldflags="-s -w -X main.Version=$VERSION" \
  -o "$BINARY" \
  ./cmd/mediamtx-ui

echo ""
echo "==> Done: $BINARY ($(du -sh "$BINARY" | cut -f1))"
