#!/usr/bin/env bash
set +o histexpand

# Install the threat-detect binary from GitHub Releases with SHA256 checksum verification.
# Used when `features: gh-aw-detection: true` is set in the workflow frontmatter to enable
# the external threat-detect binary detection path instead of inline engine execution.
#
# Usage: install_threat_detect_binary.sh VERSION [--rootless]
#
# Arguments:
#   VERSION    - threat-detect version to install (e.g., v0.2.2) or "latest" to
#                install the latest release via GitHub's latest-release download endpoint
#   --rootless - Install to ~/.local/bin without sudo; appends that directory to
#                $GITHUB_PATH so subsequent steps find the binary.  Use this on
#                ARC/DinD runners that enforce allowPrivilegeEscalation: false.
#
# Platform support:
#   - Linux (x64, arm64): Downloads pre-built binary
#   - macOS: NOT supported. Agentic workflows require Linux container jobs, and the
#     compiler rejects macOS runner labels (including
#     safe-outputs.threat-detection.runs-on) before a workflow is generated. If this
#     script is ever reached on Darwin it fails fast with an explicit message instead
#     of attempting a download.
#
# Security features:
#   - Downloads directly from GitHub releases
#   - Verifies SHA256 checksum against official checksums.txt
#   - Fails fast if checksum verification fails

set -euo pipefail

# Configuration
THREAT_DETECT_REPO="github/gh-aw-threat-detection"
THREAT_DETECT_INSTALL_DIR="/usr/local/bin"
THREAT_DETECT_INSTALL_NAME="threat-detect"
MACOS_FAQ_URL="https://github.github.com/gh-aw/reference/faq/#why-are-macos-runners-not-supported"

# Parse arguments: treat the first non-flag argument as VERSION, all --<flag> arguments as flags.
THREAT_DETECT_VERSION=""
ROOTLESS=false
for arg in "$@"; do
  case "$arg" in
    --rootless) ROOTLESS=true ;;
    --*) echo "WARNING: Unknown flag: $arg" >&2 ;;
    *)
      if [ -z "$THREAT_DETECT_VERSION" ]; then
        THREAT_DETECT_VERSION="$arg"
      fi
      ;;
  esac
done

if [ -z "$THREAT_DETECT_VERSION" ]; then
  echo "ERROR: threat-detect version is required"
  echo "Usage: $0 VERSION [--rootless]"
  exit 1
fi

# In rootless mode, install into the user's home directory instead of /usr/local/bin
# so that ARC/DinD runners with allowPrivilegeEscalation: false can run without sudo.
if [ "$ROOTLESS" = "true" ]; then
  THREAT_DETECT_INSTALL_DIR="${HOME}/.local/bin"
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
  if ! { mkdir -p "${THREAT_DETECT_INSTALL_DIR}" && [ -w "${THREAT_DETECT_INSTALL_DIR}" ]; }; then
    echo "ERROR: --rootless could not create a writable install directory at ${THREAT_DETECT_INSTALL_DIR}" >&2
    exit 1
  fi
fi

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

echo "Installing threat-detect with checksum verification (version: ${THREAT_DETECT_VERSION}, os: ${OS}, arch: ${ARCH})"

# Fail fast on unsupported platforms before any network access. Only Linux is supported:
# agentic workflows require Linux container jobs, and the compiler rejects macOS runner
# labels (including safe-outputs.threat-detection.runs-on) at compile time.
case "$OS" in
  Linux) ;;
  Darwin)
    echo "ERROR: macOS is not a supported platform for threat-detect."
    echo "  Agentic workflows require Linux container jobs; use a Linux runner instead."
    echo "  See ${MACOS_FAQ_URL} for details."
    exit 1
    ;;
  *)
    echo "ERROR: Unsupported operating system: ${OS}"
    echo "  threat-detect is only published for Linux (x64, arm64)."
    exit 1
    ;;
esac

# Download release assets directly rather than resolving a release through the GitHub API.
if [ "$THREAT_DETECT_VERSION" = "latest" ]; then
  BASE_URL="https://github.com/${THREAT_DETECT_REPO}/releases/latest/download"
else
  BASE_URL="https://github.com/${THREAT_DETECT_REPO}/releases/download/${THREAT_DETECT_VERSION}"
fi
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
curl -fsSL --retry 5 --retry-delay 10 --retry-max-time 180 --retry-all-errors -o "${TEMP_DIR}/checksums.txt" "${CHECKSUMS_URL}"

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
  curl -fsSL --retry 5 --retry-delay 10 --retry-max-time 180 --retry-all-errors -o "${TEMP_DIR}/${binary_name}" "${binary_url}"

  # Verify checksum
  verify_checksum "${TEMP_DIR}/${binary_name}" "${binary_name}"

  # Make binary executable and install
  chmod +x "${TEMP_DIR}/${binary_name}"
  maybe_sudo mv "${TEMP_DIR}/${binary_name}" "${THREAT_DETECT_INSTALL_DIR}/${THREAT_DETECT_INSTALL_NAME}"
}

install_linux_binary

# In rootless mode, add the install dir to PATH for subsequent steps.
if [ "$ROOTLESS" = "true" ]; then
  if [ -n "${GITHUB_PATH:-}" ]; then
    echo "${THREAT_DETECT_INSTALL_DIR}" >> "${GITHUB_PATH}"
    echo "  Exported ${THREAT_DETECT_INSTALL_DIR} to GITHUB_PATH"
  else
    echo "  GITHUB_PATH not set — binary installed at ${THREAT_DETECT_INSTALL_DIR}/${THREAT_DETECT_INSTALL_NAME}"
  fi
fi

# Verify installation
"${THREAT_DETECT_INSTALL_DIR}/${THREAT_DETECT_INSTALL_NAME}" --version

echo "✓ threat-detect installation complete"
