#!/usr/bin/env bash
# Install GitHub Copilot CLI with SHA256 checksum verification
# Usage: install_copilot_cli.sh [VERSION]
#
# This script downloads and installs the GitHub Copilot CLI directly from GitHub
# releases with SHA256 checksum verification, following the secure pattern from
# install_awf_binary.sh to avoid executing unverified downloaded scripts.
#
# Arguments:
#   VERSION - Optional Copilot CLI version to install (default: latest release)
#
# Security features:
#   - Downloads binary directly from GitHub releases (no installer script execution)
#   - Verifies SHA256 checksum against official SHA256SUMS.txt
#   - Fails fast if checksum verification fails

set -euo pipefail

# Configuration
VERSION="${1:-}"
COPILOT_REPO="github/copilot-cli"
INSTALL_DIR="/usr/local/bin"
COPILOT_DIR="/home/runner/.copilot"

# Fix directory ownership before installation
# This is needed because a previous AWF run on the same runner may have used
# `sudo -E awf --enable-chroot ...`, which creates the .copilot directory with
# root ownership. The Copilot CLI (running as the runner user) then fails when
# trying to create subdirectories. See: https://github.com/github/gh-aw/issues/12066
echo "Ensuring correct ownership of $COPILOT_DIR..."
mkdir -p "$COPILOT_DIR"
sudo chown -R runner:runner "$COPILOT_DIR"

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

# Map architecture to Copilot CLI naming
case "$ARCH" in
  x86_64|amd64) ARCH_NAME="x64" ;;
  aarch64|arm64) ARCH_NAME="arm64" ;;
  *) echo "ERROR: Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

# Map OS to Copilot CLI naming
case "$OS" in
  Linux) PLATFORM="linux" ;;
  Darwin) PLATFORM="darwin" ;;
  *) echo "ERROR: Unsupported operating system: ${OS}"; exit 1 ;;
esac

TARBALL_NAME="copilot-${PLATFORM}-${ARCH_NAME}.tar.gz"

# Build download URLs
if [ -z "$VERSION" ]; then
  BASE_URL="https://github.com/${COPILOT_REPO}/releases/latest/download"
else
  # Prefix version with 'v' if not already present
  case "$VERSION" in
    v*) ;;
    *) VERSION="v$VERSION" ;;
  esac
  BASE_URL="https://github.com/${COPILOT_REPO}/releases/download/${VERSION}"
fi

TARBALL_URL="${BASE_URL}/${TARBALL_NAME}"
CHECKSUMS_URL="${BASE_URL}/SHA256SUMS.txt"

echo "Installing GitHub Copilot CLI${VERSION:+ version $VERSION} (os: ${OS}, arch: ${ARCH})..."

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

# Download checksums
echo "Downloading checksums from ${CHECKSUMS_URL}..."
curl -fsSL --retry 3 --retry-delay 5 -o "${TEMP_DIR}/SHA256SUMS.txt" "${CHECKSUMS_URL}"

# Download binary tarball
echo "Downloading binary from ${TARBALL_URL}..."
curl -fsSL --retry 3 --retry-delay 5 -o "${TEMP_DIR}/${TARBALL_NAME}" "${TARBALL_URL}"

# Verify checksum
echo "Verifying SHA256 checksum for ${TARBALL_NAME}..."
EXPECTED_CHECKSUM=$(awk -v fname="${TARBALL_NAME}" '$2 == fname {print $1; exit}' "${TEMP_DIR}/SHA256SUMS.txt" | tr 'A-F' 'a-f')

if [ -z "$EXPECTED_CHECKSUM" ]; then
  echo "ERROR: Could not find checksum for ${TARBALL_NAME} in SHA256SUMS.txt"
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
echo "Installing binary to ${INSTALL_DIR}..."
sudo tar -xz -C "${INSTALL_DIR}" -f "${TEMP_DIR}/${TARBALL_NAME}"
sudo chmod +x "${INSTALL_DIR}/copilot"

# Verify installation
echo "Verifying Copilot CLI installation..."
if command -v copilot >/dev/null 2>&1; then
  copilot --version
  echo "✓ Copilot CLI installation complete"
else
  echo "ERROR: Copilot CLI installation failed - command not found"
  exit 1
fi
