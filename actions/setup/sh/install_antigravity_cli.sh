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
#   - Verifies SHA256 checksum against official checksums.txt before installation
#   - Fails fast if checksum verification fails
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
CHECKSUMS_URL="${GCS_BASE_URL}/${VERSION}/checksums.txt"

echo "Installing Antigravity CLI version ${VERSION} (os: ${OS}, arch: ${ARCH})..."

# Platform-portable SHA256 function
sha256_hash() {
  local file="$1"
  if command -v sha256sum &>/dev/null; then
    sha256sum "$file" | awk '{print $1}'
  elif command -v shasum &>/dev/null; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    echo "ERROR: No sha256sum or shasum found" >&2
    exit 1
  fi
}

# Create temp directory with cleanup on exit
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Download checksums file from GCS
echo "Downloading checksums from ${CHECKSUMS_URL}..."
curl -fsSL --retry 3 --retry-delay 5 -o "${TEMP_DIR}/checksums.txt" "${CHECKSUMS_URL}"

# Download binary tarball from GCS over HTTPS
echo "Downloading from ${TARBALL_URL}..."
curl -fsSL --retry 3 --retry-delay 5 -o "${TEMP_DIR}/${TARBALL_NAME}" "${TARBALL_URL}"

# Verify SHA256 checksum before extracting
echo "Verifying SHA256 checksum for ${TARBALL_NAME}..."
EXPECTED_CHECKSUM=$(awk -v fname="${TARBALL_NAME}" '$2 == fname {print $1; exit}' "${TEMP_DIR}/checksums.txt" | tr 'A-F' 'a-f')

if [ -z "$EXPECTED_CHECKSUM" ]; then
  echo "ERROR: Could not find checksum for ${TARBALL_NAME} in checksums.txt"
  exit 1
fi

ACTUAL_CHECKSUM=$(sha256_hash "${TEMP_DIR}/${TARBALL_NAME}" | tr 'A-F' 'a-f')

if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
  echo "ERROR: Checksum verification failed!"
  echo "  Expected: $EXPECTED_CHECKSUM"
  echo "  Got:      $ACTUAL_CHECKSUM"
  echo "  The downloaded file may be corrupted or tampered with"
  exit 1
fi

echo "✓ Checksum verification passed for ${TARBALL_NAME}"

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
