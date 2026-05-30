#!/usr/bin/env bash
set +o histexpand

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
REQUESTED_VERSION="${VERSION:-latest}"

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

# Normalize Copilot versions so toolcache lookups can match both v-prefixed and bare versions.
normalize_version() {
  local version="${1:-}"
  printf '%s\n' "${version#v}"
}

# Compare dotted numeric versions without relying on GNU-specific sort -V.
version_is_greater() {
  local left="${1:-0}"
  local right="${2:-0}"
  local left_parts=()
  local right_parts=()
  local max_parts=0
  local i=0
  local left_part=0
  local right_part=0

  IFS='.' read -r -a left_parts <<< "$left"
  IFS='.' read -r -a right_parts <<< "$right"

  if [ "${#left_parts[@]}" -gt "${#right_parts[@]}" ]; then
    max_parts="${#left_parts[@]}"
  else
    max_parts="${#right_parts[@]}"
  fi

  for ((i = 0; i < max_parts; i++)); do
    left_part="${left_parts[i]:-0}"
    right_part="${right_parts[i]:-0}"

    if ((10#$left_part > 10#$right_part)); then
      return 0
    fi
    if ((10#$left_part < 10#$right_part)); then
      return 1
    fi
  done

  return 1
}

# Look up a compatible Copilot CLI from the Actions toolcache before downloading a release tarball.
find_cached_copilot_bin() {
  local requested_version="${1:-latest}"
  local requested_version_normalized=""
  local tool_cache_root=""
  local candidate=""
  local candidate_dir=""
  local candidate_arch=""
  local candidate_version=""
  local candidate_version_normalized=""
  local best_candidate=""
  local best_version=""

  if [ "$requested_version" != "latest" ]; then
    requested_version_normalized="$(normalize_version "$requested_version")"
  fi

  for tool_cache_root in \
    "${RUNNER_TOOL_CACHE:-}" \
    /opt/hostedtoolcache \
    /home/runner/work/_tool
  do
    if [ -z "$tool_cache_root" ] || [ ! -d "${tool_cache_root}/copilot-cli" ]; then
      continue
    fi

    while IFS= read -r candidate; do
      candidate_dir="$(dirname "$candidate")"
      candidate_arch="$(basename "$(dirname "$candidate_dir")")"
      candidate_version="$(basename "$(dirname "$(dirname "$candidate_dir")")")"
      candidate_version_normalized="$(normalize_version "$candidate_version")"

      if [ "$candidate_arch" != "$ARCH_NAME" ]; then
        continue
      fi

      if [ -n "$requested_version_normalized" ]; then
        if [ "$candidate_version_normalized" = "$requested_version_normalized" ]; then
          printf '%s\n' "$candidate"
          return 0
        fi
        continue
      fi

      if [ -z "$best_candidate" ] || version_is_greater "$candidate_version_normalized" "$best_version"; then
        best_candidate="$candidate"
        best_version="$candidate_version_normalized"
      fi
    done < <(find "${tool_cache_root}/copilot-cli" -maxdepth 4 -type f -path '*/bin/copilot' 2>/dev/null | sort)
  done

  if [ -n "$best_candidate" ]; then
    printf '%s\n' "$best_candidate"
    return 0
  fi

  return 1
}

# Make a cached Copilot CLI available both to the current shell and later GitHub Actions steps.
activate_cached_copilot_bin() {
  local cached_copilot_bin="$1"
  local cached_copilot_dir=""
  local wrapper_path=""

  cached_copilot_dir="$(dirname "$cached_copilot_bin")"
  export PATH="${cached_copilot_dir}:$PATH"

  if [ -n "${GITHUB_PATH:-}" ]; then
    echo "$cached_copilot_dir" >> "${GITHUB_PATH}"
    return 0
  fi

  wrapper_path="${TEMP_DIR}/copilot"
  cat > "$wrapper_path" <<EOF
#!/usr/bin/env bash
exec "$cached_copilot_bin" "\$@"
EOF
  sudo install -m 0755 "$wrapper_path" "${INSTALL_DIR}/copilot"
}

# Create temp directory with cleanup on exit
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Prefer the runner toolcache when a compatible Copilot CLI is already available.
if CACHED_COPILOT_BIN="$(find_cached_copilot_bin "$REQUESTED_VERSION")"; then
  echo "Using cached GitHub Copilot CLI from ${CACHED_COPILOT_BIN}"
  activate_cached_copilot_bin "$CACHED_COPILOT_BIN"

  echo "Verifying cached Copilot CLI installation..."
  if command -v copilot >/dev/null 2>&1; then
    copilot --version
    echo "✓ Copilot CLI installation complete"
    exit 0
  fi

  echo "ERROR: Cached Copilot CLI activation failed - command not found"
  exit 1
fi

# Build download URLs
if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ]; then
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
