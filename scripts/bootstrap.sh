#!/usr/bin/env bash
# bootstrap.sh — install all build dependencies
set -euo pipefail

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RESET='\033[0m'

info()  { echo -e "${GREEN}[bootstrap]${RESET} $*"; }
warn()  { echo -e "${YELLOW}[bootstrap]${RESET} $*"; }

info "Checking dependencies..."

# Go
if ! command -v go &>/dev/null; then
  warn "Go not found. Please install Go 1.23+ from https://go.dev/dl/"
  exit 1
fi
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
info "Go: $GO_VERSION"

# Node.js
if ! command -v node &>/dev/null; then
  warn "Node.js not found. Please install Node.js 20+ from https://nodejs.org"
  exit 1
fi
NODE_VERSION=$(node --version)
info "Node.js: $NODE_VERSION"

# npm
if ! command -v npm &>/dev/null; then
  warn "npm not found."
  exit 1
fi
info "npm: $(npm --version)"

# Install frontend dependencies (generates package-lock.json on first run)
info "Installing frontend dependencies..."
cd "$(dirname "$0")/../frontend"
npm install
cd ..

# Download Go modules
info "Downloading Go modules..."
go mod download

info "${BOLD}Bootstrap complete.${RESET} Run 'make build' to compile the binary."
