#!/usr/bin/env bash
set +o histexpand

# Install Antigravity CLI (agy) from Google Cloud Storage
# Usage: install_antigravity_cli.sh VERSION [--rootless]
#
# This script downloads and installs the Antigravity CLI binary directly from
# Google Cloud Storage (https://storage.googleapis.com/antigravity-public/).
#
# Arguments:
#   VERSION    - Antigravity CLI version to install (required)
#   --rootless - Install to ~/.local/bin without sudo; appends that directory to
#                $GITHUB_PATH so subsequent steps find the binary.  Use this on
#                ARC/DinD runners that enforce allowPrivilegeEscalation: false.
#
# Security features:
#   - Downloads binary directly from Google Cloud Storage over HTTPS
#   - Verifies SHA256 checksum against official checksums.txt before installation
#   - Warns and skips checksum verification if checksums.txt is unavailable (HTTP 404)
#   - Fails fast if checksum verification fails
#   - Fails fast on any curl errors

set -euo pipefail

# Configuration
GCS_BASE_URL="https://storage.googleapis.com/antigravity-public/antigravity-cli"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="agy"

# Parse arguments: treat the first non-flag argument as VERSION, all --<flag> arguments as flags.
VERSION=""
ROOTLESS=false
for arg in "$@"; do
  case "$arg" in
    --rootless) ROOTLESS=true ;;
    --*) echo "WARNING: Unknown flag: $arg" >&2 ;;
    *)
      if [ -z "$VERSION" ]; then
        VERSION="$arg"
      fi
      ;;
  esac
done

if [ -z "$VERSION" ]; then
  echo "ERROR: Version argument is required"
  echo "Usage: $0 VERSION [--rootless]"
  exit 1
fi

# In rootless mode, install into the user's home directory instead of /usr/local/bin
# so that ARC/DinD runners with allowPrivilegeEscalation: false can run without sudo.
if [ "$ROOTLESS" = "true" ]; then
  INSTALL_DIR="${HOME}/.local/bin"
fi

# maybe_sudo runs a command with sudo unless --rootless was specified.
# In rootless mode, sudo is not available or needed.
maybe_sudo() {
  if [ "$ROOTLESS" = "true" ]; then
    "$@"
  else
    sudo "$@"
  fi
}

# Rootless mode preflight: create and verify write access to the install directory.
if [ "$ROOTLESS" = "true" ]; then
  if ! { mkdir -p "${INSTALL_DIR}" && [ -w "${INSTALL_DIR}" ]; }; then
    echo "ERROR: --rootless could not create a writable install directory at ${INSTALL_DIR}" >&2
    exit 1
  fi
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

# Download checksums file from GCS (if available for this version)
echo "Downloading checksums from ${CHECKSUMS_URL}..."
if ! CHECKSUMS_DOWNLOAD_STATUS=$(curl -sSL --retry 3 --retry-delay 5 -w "%{http_code}" -o "${TEMP_DIR}/checksums.txt" "${CHECKSUMS_URL}"); then
  echo "ERROR: Failed to download checksums.txt due to a network or TLS error"
  exit 1
fi

VERIFY_CHECKSUM=true
if [ "${CHECKSUMS_DOWNLOAD_STATUS}" = "404" ]; then
  echo "WARNING: checksums.txt not found for version ${VERSION}; skipping checksum verification."
  rm -f "${TEMP_DIR}/checksums.txt"
  VERIFY_CHECKSUM=false
elif [ "${CHECKSUMS_DOWNLOAD_STATUS}" != "200" ]; then
  echo "ERROR: Failed to download checksums.txt (HTTP ${CHECKSUMS_DOWNLOAD_STATUS})"
  exit 1
fi

# Download binary tarball from GCS over HTTPS
echo "Downloading from ${TARBALL_URL}..."
curl -fsSL --retry 3 --retry-delay 5 -o "${TEMP_DIR}/${TARBALL_NAME}" "${TARBALL_URL}"

# Verify SHA256 checksum before extracting (when checksums.txt is available)
if [ "${VERIFY_CHECKSUM}" = "true" ]; then
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
else
  echo "WARNING: Proceeding without checksum verification for ${TARBALL_NAME}"
fi

# Extract and install binary
echo "Installing binary to ${INSTALL_DIR}/${BINARY_NAME}..."
tar -xz -C "${TEMP_DIR}" -f "${TEMP_DIR}/${TARBALL_NAME}"

# The archive contains a binary named "antigravity" (per GCS tarball structure);
# install it as "agy" in the expected location.
if [ ! -f "${TEMP_DIR}/antigravity" ]; then
  echo "ERROR: Expected binary 'antigravity' not found in the extracted archive"
  exit 1
fi
maybe_sudo install -m 755 "${TEMP_DIR}/antigravity" "${INSTALL_DIR}/${BINARY_NAME}"

# In rootless mode, add the install dir to PATH for subsequent steps.
if [ "$ROOTLESS" = "true" ]; then
  if [ -n "${GITHUB_PATH:-}" ]; then
    echo "${INSTALL_DIR}" >> "${GITHUB_PATH}"
    echo "  Exported ${INSTALL_DIR} to GITHUB_PATH"
  else
    echo "  GITHUB_PATH not set — binary installed at ${INSTALL_DIR}/${BINARY_NAME}"
  fi
fi

# Verify installation
echo "Verifying Antigravity CLI installation..."
if command -v "${BINARY_NAME}" >/dev/null 2>&1; then
  "${BINARY_NAME}" --version || true
  echo "✓ Antigravity CLI (${BINARY_NAME}) installation complete"
else
  echo "ERROR: Antigravity CLI installation failed - command not found"
  exit 1
fi
