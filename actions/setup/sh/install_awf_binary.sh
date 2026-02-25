#!/usr/bin/env bash
# Install AWF (Agentic Workflow Firewall) binary with SHA256 checksum verification
# Usage: install_awf_binary.sh VERSION
#
# This script downloads the AWF binary directly from GitHub releases and verifies
# its SHA256 checksum before installation to protect against supply chain attacks.
#
# Arguments:
#   VERSION - AWF version to install (e.g., v0.10.0)
#
# Platform support:
#   - Linux (x64, arm64): Downloads pre-built binary
#   - macOS (x64, arm64): Downloads pre-built binary
#
# Security features:
#   - Downloads binary directly from GitHub releases
#   - Verifies SHA256 checksum against official checksums.txt
#   - Fails fast if checksum verification fails
#   - Eliminates trust dependency on installer scripts

set -euo pipefail

# Configuration
AWF_VERSION="${1:-}"
AWF_REPO="github/gh-aw-firewall"
AWF_INSTALL_DIR="/usr/local/bin"
AWF_INSTALL_NAME="awf"

if [ -z "$AWF_VERSION" ]; then
  echo "ERROR: AWF version is required"
  echo "Usage: $0 VERSION"
  exit 1
fi

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

echo "Installing awf with checksum verification (version: ${AWF_VERSION}, os: ${OS}, arch: ${ARCH})"

# Download URLs
BASE_URL="https://github.com/${AWF_REPO}/releases/download/${AWF_VERSION}"
CHECKSUMS_URL="${BASE_URL}/checksums.txt"

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

# Create temp directory
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Download checksums
echo "Downloading checksums from ${CHECKSUMS_URL@Q}..."
curl -fsSL --retry 3 --retry-delay 5 -o "${TEMP_DIR}/checksums.txt" "${CHECKSUMS_URL}"

verify_checksum() {
  local file="$1"
  local fname="$2"

  echo "Verifying SHA256 checksum for ${fname}..."
  EXPECTED_CHECKSUM=$(awk -v fname="${fname}" '$2 == fname {print $1; exit}' "${TEMP_DIR}/checksums.txt" | tr 'A-F' 'a-f')

  if [ -z "$EXPECTED_CHECKSUM" ]; then
    echo "ERROR: Could not find checksum for ${fname} in checksums.txt"
    exit 1
  fi

  ACTUAL_CHECKSUM=$(sha256_hash "$file" | tr 'A-F' 'a-f')

  if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
    echo "ERROR: Checksum verification failed!"
    echo "  Expected: $EXPECTED_CHECKSUM"
    echo "  Got:      $ACTUAL_CHECKSUM"
    echo "  The downloaded file may be corrupted or tampered with"
    exit 1
  fi

  echo "✓ Checksum verification passed for ${fname}"
}

install_linux_binary() {
  # Determine binary name based on architecture
  local awf_binary
  case "$ARCH" in
    x86_64|amd64) awf_binary="awf-linux-x64" ;;
    aarch64|arm64) awf_binary="awf-linux-arm64" ;;
    *) echo "ERROR: Unsupported Linux architecture: ${ARCH}"; exit 1 ;;
  esac

  local binary_url="${BASE_URL}/${awf_binary}"
  echo "Downloading binary from ${binary_url@Q}..."
  curl -fsSL --retry 3 --retry-delay 5 -o "${TEMP_DIR}/${awf_binary}" "${binary_url}"

  # Verify checksum
  verify_checksum "${TEMP_DIR}/${awf_binary}" "${awf_binary}"

  # Make binary executable and install
  chmod +x "${TEMP_DIR}/${awf_binary}"
  sudo mv "${TEMP_DIR}/${awf_binary}" "${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}"
}

install_darwin_binary() {
  # Determine binary name based on architecture
  local awf_binary
  case "$ARCH" in
    x86_64) awf_binary="awf-darwin-x64" ;;
    arm64) awf_binary="awf-darwin-arm64" ;;
    *) echo "ERROR: Unsupported macOS architecture: ${ARCH}"; exit 1 ;;
  esac

  echo "Note: AWF uses iptables for network firewalling, which is not available on macOS."
  echo "      The AWF CLI will be installed but container-based firewalling will not work natively."
  echo ""

  local binary_url="${BASE_URL}/${awf_binary}"
  echo "Downloading binary from ${binary_url@Q}..."
  curl -fsSL --retry 3 --retry-delay 5 -o "${TEMP_DIR}/${awf_binary}" "${binary_url}"

  # Verify checksum
  verify_checksum "${TEMP_DIR}/${awf_binary}" "${awf_binary}"

  # Make binary executable and install
  chmod +x "${TEMP_DIR}/${awf_binary}"
  sudo mv "${TEMP_DIR}/${awf_binary}" "${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}"
}

case "$OS" in
  Linux)
    install_linux_binary
    ;;
  Darwin)
    install_darwin_binary
    ;;
  *)
    echo "ERROR: Unsupported operating system: ${OS}"
    exit 1
    ;;
esac

# Verify installation
which awf
awf --version

echo "✓ AWF installation complete"
