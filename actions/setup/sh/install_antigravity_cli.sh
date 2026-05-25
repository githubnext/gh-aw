#!/usr/bin/env bash
set +o histexpand

# Install Antigravity CLI (agy) from Google Cloud Storage
# Usage: install_antigravity_cli.sh VERSION
#
# This script downloads and installs the Antigravity CLI binary directly from
# Google Cloud Storage (https://storage.googleapis.com/antigravity-public/).
#
# Arguments:
#   VERSION - Antigravity CLI version to install (required)
#
# Security features:
#   - Downloads binary directly from Google Cloud Storage over HTTPS
#   - Fails fast on any curl errors

set -euo pipefail

# Configuration
VERSION="${1:-}"
GCS_BASE_URL="https://storage.googleapis.com/antigravity-public/antigravity-cli"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="agy"

if [ -z "$VERSION" ]; then
  echo "ERROR: Version argument is required"
  exit 1
fi

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

# Map OS and architecture to Antigravity CLI GCS path components
case "$OS" in
  Linux)
    case "$ARCH" in
      x86_64|amd64) ARCH_DIR="linux-x64"; TARBALL_NAME="cli_linux_x64.tar.gz" ;;
      aarch64|arm64) ARCH_DIR="linux-arm"; TARBALL_NAME="cli_linux_arm64.tar.gz" ;;
      *) echo "ERROR: Unsupported architecture: ${ARCH}"; exit 1 ;;
    esac
    ;;
  Darwin)
    case "$ARCH" in
      x86_64|amd64) ARCH_DIR="darwin-x64"; TARBALL_NAME="cli_mac_x64.tar.gz" ;;
      aarch64|arm64) ARCH_DIR="darwin-arm"; TARBALL_NAME="cli_mac_arm64.tar.gz" ;;
      *) echo "ERROR: Unsupported architecture: ${ARCH}"; exit 1 ;;
    esac
    ;;
  *) echo "ERROR: Unsupported operating system: ${OS}"; exit 1 ;;
esac

TARBALL_URL="${GCS_BASE_URL}/${VERSION}/${ARCH_DIR}/${TARBALL_NAME}"

echo "Installing Antigravity CLI version ${VERSION} (os: ${OS}, arch: ${ARCH})..."
echo "Downloading from ${TARBALL_URL}..."

# Create temp directory with cleanup on exit
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Download binary tarball from GCS over HTTPS
curl -fsSL --retry 3 --retry-delay 5 -o "${TEMP_DIR}/${TARBALL_NAME}" "${TARBALL_URL}"

# Extract and install binary
echo "Installing binary to ${INSTALL_DIR}/${BINARY_NAME}..."
tar -xz -C "${TEMP_DIR}" -f "${TEMP_DIR}/${TARBALL_NAME}"

# The archive contains a binary named "antigravity" (per GCS tarball structure);
# install it as "agy" in the expected location.
if [ ! -f "${TEMP_DIR}/antigravity" ]; then
  echo "ERROR: Expected binary 'antigravity' not found in the extracted archive"
  exit 1
fi
sudo install -m 755 "${TEMP_DIR}/antigravity" "${INSTALL_DIR}/${BINARY_NAME}"

# Verify installation
echo "Verifying Antigravity CLI installation..."
if command -v "${BINARY_NAME}" >/dev/null 2>&1; then
  "${BINARY_NAME}" --version || true
  echo "✓ Antigravity CLI (${BINARY_NAME}) installation complete"
else
  echo "ERROR: Antigravity CLI installation failed - command not found"
  exit 1
fi
