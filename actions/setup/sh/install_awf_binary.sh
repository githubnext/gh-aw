#!/usr/bin/env bash
set +o histexpand

# Install AWF (Agentic Workflow Firewall) with SHA256 checksum verification
# Usage: install_awf_binary.sh VERSION
#
# This script downloads the AWF bundle or binary from GitHub releases and verifies
# its SHA256 checksum before installation to protect against supply chain attacks.
#
# Arguments:
#   VERSION - AWF version to install (e.g., v0.25.10)
#
# Install strategy:
#   1. If Node.js >= 20 is available, download the lightweight awf-bundle.js (~357KB)
#   2. Otherwise, fall back to platform-specific pkg binary (~50MB)
#
# Platform support (fallback binary):
#   - Linux (x64, arm64): Downloads pre-built binary
#   - macOS (x64, arm64): Downloads pre-built binary
#
# Security features:
#   - Downloads directly from GitHub releases
#   - Verifies SHA256 checksum against official checksums.txt
#   - Fails fast if checksum verification fails
#   - Eliminates trust dependency on installer scripts

set -euo pipefail

# Configuration
AWF_VERSION="${1:-}"
AWF_REPO="github/gh-aw-firewall"
AWF_INSTALL_DIR="/usr/local/bin"
AWF_INSTALL_NAME="awf"
AWF_LIB_DIR="/usr/local/lib/awf"

# Optional cache directory for persisting the AWF binary across runs.
# Set AWF_CACHE_DIR to a directory path to enable cross-run caching.
# Set AWF_CACHE_HIT=true when the exact version was restored from cache.
AWF_CACHE_DIR="${AWF_CACHE_DIR:-}"
AWF_CACHE_HIT="${AWF_CACHE_HIT:-false}"

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

# ---------------------------------------------------------------------------
# Cache support
#
# When AWF_CACHE_DIR is set:
#   - AWF_CACHE_HIT=true means the exact requested version was restored from
#     cache; skip all downloads and install directly from the cache directory.
#   - Otherwise, attempt a fresh download. On download failure, fall back to
#     any binary present in AWF_CACHE_DIR (stale version from a prior run).
#   - After a successful download, save the binary and checksums to cache.
# ---------------------------------------------------------------------------

# Check for exact cache hit: exact version already restored into AWF_CACHE_DIR
if [ "$AWF_CACHE_HIT" = "true" ] && [ -n "$AWF_CACHE_DIR" ] && [ -f "${AWF_CACHE_DIR}/awf" ]; then
  echo "Cache hit! Restoring AWF ${AWF_VERSION} from cache directory..."
  sudo install -m 0755 "${AWF_CACHE_DIR}/awf" "${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}"

  # Restore bundle file if present (needed for the Node.js wrapper path)
  if [ -f "${AWF_CACHE_DIR}/awf-bundle.js" ]; then
    sudo mkdir -p "${AWF_LIB_DIR}"
    sudo cp "${AWF_CACHE_DIR}/awf-bundle.js" "${AWF_LIB_DIR}/awf-bundle.js"
  fi

  # Verify the restored binary works
  sudo env -u GITHUB_API_URL -u GITHUB_GRAPHQL_URL -u GH_HOST \
      "${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}" --version
  echo "✓ AWF installation complete (from cache)"
  exit 0
fi

# Download checksums (with increased retry budget to tolerate CDN blips)
echo "Downloading checksums from ${CHECKSUMS_URL@Q}..."
if ! curl -fsSL --retry 6 --retry-delay 15 --retry-max-time 300 \
    -o "${TEMP_DIR}/checksums.txt" "${CHECKSUMS_URL}"; then
  # Download failed — attempt stale-cache fallback before giving up
  if [ -n "$AWF_CACHE_DIR" ] && [ -f "${AWF_CACHE_DIR}/awf" ]; then
    cached_version="$(cat "${AWF_CACHE_DIR}/VERSION" 2>/dev/null || echo unknown)"
    echo "⚠ Checksums download failed. Falling back to cached AWF binary (${cached_version})."
    sudo install -m 0755 "${AWF_CACHE_DIR}/awf" "${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}"
    if [ -f "${AWF_CACHE_DIR}/awf-bundle.js" ]; then
      sudo mkdir -p "${AWF_LIB_DIR}"
      sudo cp "${AWF_CACHE_DIR}/awf-bundle.js" "${AWF_LIB_DIR}/awf-bundle.js"
    fi
    sudo env -u GITHUB_API_URL -u GITHUB_GRAPHQL_URL -u GH_HOST \
        "${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}" --version
    echo "✓ AWF installation complete (stale-cache fallback)"
    exit 0
  fi
  echo "ERROR: Failed to download checksums and no cached fallback available."
  exit 22
fi

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

# save_to_cache copies installed artifacts to AWF_CACHE_DIR for future runs.
save_to_cache() {
  if [ -z "$AWF_CACHE_DIR" ]; then
    return 0
  fi
  echo "Saving AWF ${AWF_VERSION} to cache directory (${AWF_CACHE_DIR})..."
  mkdir -p "$AWF_CACHE_DIR"
  # Copy the installed binary wrapper
  cp "${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}" "${AWF_CACHE_DIR}/awf"
  # Copy the bundle file if present
  if [ -f "${AWF_LIB_DIR}/awf-bundle.js" ]; then
    cp "${AWF_LIB_DIR}/awf-bundle.js" "${AWF_CACHE_DIR}/awf-bundle.js"
  fi
  # Record the version for stale-fallback identification
  echo "${AWF_VERSION}" > "${AWF_CACHE_DIR}/VERSION"
  echo "✓ Cached AWF ${AWF_VERSION}"
}

# Check if Node.js >= 20 is available
has_node_20() {
  if ! command -v node &>/dev/null; then
    return 1
  fi
  local node_major
  node_major=$(node --version | sed 's/^v//' | cut -d. -f1)
  if [ "$node_major" -ge 20 ] 2>/dev/null; then
    return 0
  fi
  return 1
}

install_bundle() {
  local bundle_name="awf-bundle.js"
  local bundle_url="${BASE_URL}/${bundle_name}"

  # Capture the absolute path to node so the wrapper works correctly when
  # invoked via sudo (where PATH may not include the setup-node install dir,
  # e.g. ~/.nvm/versions/node/v24.x.x/bin).
  local node_bin
  node_bin=$(command -v node)

  echo "Node.js >= 20 detected ($(node --version)), using lightweight bundle..."
  echo "Downloading bundle from ${bundle_url@Q}..."
  if ! curl -fsSL --retry 6 --retry-delay 15 --retry-max-time 300 -o "${TEMP_DIR}/${bundle_name}" "${bundle_url}"; then
    echo "⚠ Bundle download failed (asset may not exist for this version)"
    return 1
  fi

  # Verify checksum
  if ! verify_checksum "${TEMP_DIR}/${bundle_name}" "${bundle_name}"; then
    echo "⚠ Bundle checksum verification failed"
    return 1
  fi

  # Install bundle to lib directory
  sudo mkdir -p "${AWF_LIB_DIR}"
  sudo cp "${TEMP_DIR}/${bundle_name}" "${AWF_LIB_DIR}/${bundle_name}"

  # Create wrapper script using the absolute path to node.
  # Using an unquoted heredoc (<<WRAPPER) so that ${node_bin} is expanded
  # at wrapper-creation time, while \$@ is left as the literal $@ for
  # runtime argument forwarding.
  sudo tee "${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}" > /dev/null <<WRAPPER
#!/bin/bash
exec ${node_bin} /usr/local/lib/awf/awf-bundle.js "\$@"
WRAPPER
  sudo chmod +x "${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}"

  echo "✓ Installed awf bundle to ${AWF_LIB_DIR}/${bundle_name}"
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
  curl -fsSL --retry 6 --retry-delay 15 --retry-max-time 300 -o "${TEMP_DIR}/${awf_binary}" "${binary_url}"

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
  curl -fsSL --retry 6 --retry-delay 15 --retry-max-time 300 -o "${TEMP_DIR}/${awf_binary}" "${binary_url}"

  # Verify checksum
  verify_checksum "${TEMP_DIR}/${awf_binary}" "${awf_binary}"

  # Make binary executable and install
  chmod +x "${TEMP_DIR}/${awf_binary}"
  sudo mv "${TEMP_DIR}/${awf_binary}" "${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}"
}

install_platform_binary() {
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
}

# Try lightweight bundle first, fall back to platform binary
if has_node_20; then
  if ! install_bundle; then
    echo "⚠ Bundle install failed, falling back to platform binary..."
    install_platform_binary
  fi
else
  echo "Node.js >= 20 not available, falling back to platform binary..."
  install_platform_binary
fi

# Save installed binary to cache directory for future runs (skipped on cache hit)
save_to_cache

# Verify installation by running --version with sudo.
# Use sudo to match how awf is invoked in subsequent steps (sudo -E awf ...).
# On GPU runners (e.g. aw-gpu-runner-T4), /usr/local/bin may be inaccessible
# to the current non-root user due to filesystem or security policy restrictions,
# so running the version check without sudo would fail with "Permission denied".
# A successful run prints the version string (e.g. "0.25.13") to stdout.
# Also clear DIFC (Data Integrity and Filtering Controls) proxy env vars
# set by start_difc_proxy.sh. When the DIFC proxy is active, GITHUB_API_URL
# and GITHUB_GRAPHQL_URL point to localhost:18443 and GH_HOST is overridden.
# The AWF bundle may try to reach these endpoints on startup, causing the
# version check to fail with a connection error if the proxy rejects the request.
sudo env -u GITHUB_API_URL -u GITHUB_GRAPHQL_URL -u GH_HOST \
    "${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}" --version

echo "✓ AWF installation complete"
