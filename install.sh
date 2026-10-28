#!/usr/bin/env sh
# Auto-install tnow.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/andev0x/tnow/main/install.sh | sh
#
# The script prefers `go install` when a Go toolchain is available, and
# otherwise downloads a prebuilt binary from the latest GitHub release.

set -eu

REPO="andev0x/tnow"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

mkdir -p "$INSTALL_DIR"

if command -v go >/dev/null 2>&1; then
    echo "Installing tnow with go install..."
    go install "github.com/${REPO}/cmd/tnow@latest"
    GOBIN="$(go env GOBIN 2>/dev/null || true)"
    if [ -z "$GOBIN" ]; then
        GOBIN="$(go env GOPATH)/bin"
    fi
    if [ -x "$GOBIN/tnow" ]; then
        ln -sf "$GOBIN/tnow" "$INSTALL_DIR/tnow" 2>/dev/null || true
    fi
    echo "tnow installed to $INSTALL_DIR/tnow"
    exit 0
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac
case "$OS" in
    linux|darwin) ;;
    *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

if ! command -v curl >/dev/null 2>&1; then
    echo "curl is required to download tnow" >&2
    exit 1
fi

VERSION="${TNOW_VERSION:-latest}"
if [ "$VERSION" = "latest" ]; then
    URL="https://github.com/${REPO}/releases/latest/download/tnow_${OS}_${ARCH}.tar.gz"
else
    URL="https://github.com/${REPO}/releases/download/${VERSION}/tnow_${OS}_${ARCH}.tar.gz"
fi

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading tnow from $URL..."
curl -fsSL "$URL" -o "$TMPDIR/tnow.tar.gz"
tar -xzf "$TMPDIR/tnow.tar.gz" -C "$TMPDIR"
install -m 0755 "$TMPDIR/tnow" "$INSTALL_DIR/tnow"

echo "tnow installed to $INSTALL_DIR/tnow"
echo "Make sure $INSTALL_DIR is on your PATH."
