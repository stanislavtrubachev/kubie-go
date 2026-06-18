#!/bin/sh
set -e

REPO="stanislavtrubachev/kubie-go"
BINARY="kubie-go"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Get latest release version
VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"v\([^"]*\)".*/\1/')"

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version"
  exit 1
fi

ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"

echo "Installing ${BINARY} v${VERSION} (${OS}/${ARCH})..."

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "${TMP}/${ARCHIVE}"
tar -xzf "${TMP}/${ARCHIVE}" -C "$TMP"

# Install binary
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  ln -sf "${INSTALL_DIR}/${BINARY}" "${INSTALL_DIR}/kubie"
else
  sudo mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  sudo ln -sf "${INSTALL_DIR}/${BINARY}" "${INSTALL_DIR}/kubie"
fi

echo "Done! Installed to ${INSTALL_DIR}/${BINARY}"
echo "Available as: kubie-go and kubie"
