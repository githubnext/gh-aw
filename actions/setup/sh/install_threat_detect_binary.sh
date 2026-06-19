#!/usr/bin/env bash
set +o histexpand

# Install the threat-detect binary from GitHub Releases with SHA256 checksum verification.
# Used when `features: gh-aw-detection: true` is set in the workflow frontmatter to enable
# the external threat-detect binary detection path instead of inline engine execution.
#
# Usage: install_threat_detect_binary.sh VERSION
#
# Arguments:
#   VERSION - threat-detect version to install (e.g., v0.2.2)
#
# Platform support:
#   - Linux (x64, arm64): Downloads pre-built binary
#
# Security features:
#   - Downloads directly from GitHub releases
#   - Verifies SHA256 checksum against official checksums.txt
#   - Fails fast if checksum verification fails

set -euo pipefail

# Configuration
THREAT_DETECT_VERSION="${1:-}"
THREAT_DETECT_REPO="github/gh-aw-threat-detection"
THREAT_DETECT_INSTALL_DIR="/usr/local/bin"
THREAT_DETECT_INSTALL_NAME="threat-detect"

if [ -z "$THREAT_DETECT_VERSION" ]; then
  echo "ERROR: threat-detect version is required"
  echo "Usage: $0 VERSION"
  exit 1
fi

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

echo "Installing threat-detect with checksum verification (version: ${THREAT_DETECT_VERSION}, os: ${OS}, arch: ${ARCH})"

# Download URLs
BASE_URL="https://github.com/${THREAT_DETECT_REPO}/releases/download/${THREAT_DETECT_VERSION}"
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
echo "Downloading checksums from \"${CHECKSUMS_URL}\"..."
curl -fsSL --retry 5 --retry-delay 10 --retry-max-time 180 -o "${TEMP_DIR}/checksums.txt" "${CHECKSUMS_URL}"

verify_checksum() {
  local file="$1"
  local fname="$2"

  echo "Verifying SHA256 checksum for ${fname}..."
  EXPECTED_CHECKSUM=$(awk -v fname="${fname}" '$2 == fname {print $1; exit}' "${TEMP_DIR}/checksums.txt" | tr 'A-F' 'a-f')

  if [ -z "$EXPECTED_CHECKSUM" ]; then
    echo "ERROR: Could not find checksum for ${fname} in checksums.txt"
    return 1
  fi

  ACTUAL_CHECKSUM=$(sha256_hash "$file" | tr 'A-F' 'a-f')

  if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
    echo "ERROR: Checksum verification failed!"
    echo "  Expected: $EXPECTED_CHECKSUM"
    echo "  Got:      $ACTUAL_CHECKSUM"
    echo "  The downloaded file may be corrupted or tampered with"
    return 1
  fi

  echo "✓ Checksum verification passed for ${fname}"
}

install_linux_binary() {
  # Determine binary name based on architecture
  local binary_name
  case "$ARCH" in
    x86_64|amd64) binary_name="threat-detect-linux-amd64" ;;
    aarch64|arm64) binary_name="threat-detect-linux-arm64" ;;
    *) echo "ERROR: Unsupported Linux architecture: ${ARCH}"; exit 1 ;;
  esac

  local binary_url="${BASE_URL}/${binary_name}"
  echo "Downloading binary from \"${binary_url}\"..."
  curl -fsSL --retry 5 --retry-delay 10 --retry-max-time 180 -o "${TEMP_DIR}/${binary_name}" "${binary_url}"

  # Verify checksum
  verify_checksum "${TEMP_DIR}/${binary_name}" "${binary_name}"

  # Make binary executable and install
  chmod +x "${TEMP_DIR}/${binary_name}"
  sudo mv "${TEMP_DIR}/${binary_name}" "${THREAT_DETECT_INSTALL_DIR}/${THREAT_DETECT_INSTALL_NAME}"
}

install_darwin_binary() {
  # Determine binary name based on architecture
  local binary_name
  case "$ARCH" in
    x86_64) binary_name="threat-detect-darwin-x64" ;;
    arm64) binary_name="threat-detect-darwin-arm64" ;;
    *) echo "ERROR: Unsupported macOS architecture: ${ARCH}"; exit 1 ;;
  esac

  local binary_url="${BASE_URL}/${binary_name}"
  echo "Downloading binary from \"${binary_url}\"..."
  curl -fsSL --retry 5 --retry-delay 10 --retry-max-time 180 -o "${TEMP_DIR}/${binary_name}" "${binary_url}"

  # Verify checksum
  verify_checksum "${TEMP_DIR}/${binary_name}" "${binary_name}"

  # Make binary executable and install
  chmod +x "${TEMP_DIR}/${binary_name}"
  sudo mv "${TEMP_DIR}/${binary_name}" "${THREAT_DETECT_INSTALL_DIR}/${THREAT_DETECT_INSTALL_NAME}"
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
"${THREAT_DETECT_INSTALL_DIR}/${THREAT_DETECT_INSTALL_NAME}" --version

echo "✓ threat-detect installation complete"
