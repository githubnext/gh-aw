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
COPILOT_DIR="${HOME}/.copilot"
COPILOT_TOOLCACHE_MAX_DEPTH=4

# Fix directory ownership before installation
# This is needed because a previous AWF run on the same runner may have used
# `sudo -E awf --enable-chroot ...`, which creates the .copilot directory with
# root ownership. The Copilot CLI (running as the runner user) then fails when
# trying to create subdirectories. See: https://github.com/github/gh-aw/issues/12066
echo "Ensuring correct ownership of $COPILOT_DIR..."
mkdir -p "$COPILOT_DIR"
sudo chown -R "$(id -u):$(id -g)" "$COPILOT_DIR"

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

# Check if a version string contains only numeric parts separated by dots.
# Returns success (0) if version is purely numeric (e.g., "1.2.3"), failure (1) otherwise.
version_is_numeric() {
  local version="${1:-}"
  local parts=()
  local part=""
  
  IFS='.' read -r -a parts <<< "$version"
  
  for part in "${parts[@]}"; do
    # Check if part is empty or contains non-digits
    if [[ ! "$part" =~ ^[0-9]+$ ]]; then
      return 1
    fi
  done
  
  return 0
}

# Compare dotted numeric versions without relying on GNU-specific sort -V.
# Returns success only when the left version is strictly greater than the right version.
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

    # Use base-10 parsing so values like 08 are never treated as invalid octal literals.
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

  echo "Searching toolcache for GitHub Copilot CLI (requested: ${requested_version}, arch: ${ARCH_NAME})..." >&2

  if [ "$requested_version" != "latest" ]; then
    requested_version_normalized="$(normalize_version "$requested_version")"
  fi

  for tool_cache_root in \
    "${RUNNER_TOOL_CACHE:-}" \
    /opt/hostedtoolcache \
    /home/runner/work/_tool
  do
    if [ -z "$tool_cache_root" ]; then
      continue
    fi
    if [ ! -d "${tool_cache_root}/copilot-cli" ]; then
      echo "  Toolcache root ${tool_cache_root}/copilot-cli not found, skipping" >&2
      continue
    fi

    echo "  Scanning toolcache root: ${tool_cache_root}/copilot-cli" >&2

    while IFS= read -r candidate; do
      candidate_dir="$(dirname "$candidate")"
      candidate_arch="$(basename "$(dirname "$candidate_dir")")"
      candidate_version="$(basename "$(dirname "$(dirname "$candidate_dir")")")"
      candidate_version_normalized="$(normalize_version "$candidate_version")"

      echo "  Found candidate: ${candidate} (version: ${candidate_version_normalized}, arch: ${candidate_arch})" >&2

      # Skip non-numeric versions (e.g., "1.2.3-beta.1") to prevent arithmetic expansion errors
      if ! version_is_numeric "$candidate_version_normalized"; then
        echo "  Skipping candidate (non-numeric version: ${candidate_version_normalized})" >&2
        continue
      fi

      if [ "$candidate_arch" != "$ARCH_NAME" ]; then
        echo "  Skipping candidate (arch mismatch: want ${ARCH_NAME}, got ${candidate_arch})" >&2
        continue
      fi

      if [ -n "$requested_version_normalized" ]; then
        if [ "$candidate_version_normalized" = "$requested_version_normalized" ]; then
          echo "  Exact version match found: ${candidate}" >&2
          printf '%s\n' "$candidate"
          return 0
        fi
        echo "  Skipping candidate (version mismatch: want ${requested_version_normalized}, got ${candidate_version_normalized})" >&2
        continue
      fi

      if [ -z "$best_candidate" ] || version_is_greater "$candidate_version_normalized" "$best_version"; then
        echo "  New best candidate: ${candidate} (${candidate_version_normalized} > ${best_version:-none})" >&2
        best_candidate="$candidate"
        best_version="$candidate_version_normalized"
      fi
    done < <(find "${tool_cache_root}/copilot-cli" -maxdepth "${COPILOT_TOOLCACHE_MAX_DEPTH}" -type f -path '*/bin/copilot' 2>/dev/null)
  done

  if [ -n "$best_candidate" ]; then
    echo "  Selected best cached version: ${best_version} at ${best_candidate}" >&2
    printf '%s\n' "$best_candidate"
    return 0
  fi

  echo "  No compatible toolcache entry found" >&2
  return 1
}

# Make a cached Copilot CLI available both to the current shell and later GitHub Actions steps.
activate_cached_copilot_bin() {
  local cached_copilot_bin="$1"
  local cached_copilot_dir=""
  local wrapper_path=""

  cached_copilot_dir="$(dirname "$cached_copilot_bin")"
  echo "Activating cached Copilot CLI from ${cached_copilot_bin}..."
  export PATH="${cached_copilot_dir}:$PATH"
  echo "  Prepended ${cached_copilot_dir} to PATH"

  if [ -n "${GITHUB_PATH:-}" ]; then
    echo "  Exporting ${cached_copilot_dir} to GITHUB_PATH (${GITHUB_PATH})"
    echo "$cached_copilot_dir" >> "${GITHUB_PATH}"
    return 0
  fi

  # Outside GitHub Actions there is no GITHUB_PATH file, so install a small wrapper
  # instead of symlinking or copying the cached script and risking broken relative paths.
  echo "  GITHUB_PATH not set — installing wrapper at ${INSTALL_DIR}/copilot"
  wrapper_path="${TEMP_DIR}/copilot"
  cat > "$wrapper_path" <<EOF
#!/usr/bin/env bash
exec "$cached_copilot_bin" "\$@"
EOF
  sudo install -m 0755 "$wrapper_path" "${INSTALL_DIR}/copilot"
  echo "  Wrapper installed at ${INSTALL_DIR}/copilot"
}

# Create temp directory with cleanup on exit
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Prefer the runner toolcache when a compatible Copilot CLI is already available.
if CACHED_COPILOT_BIN="$(find_cached_copilot_bin "$REQUESTED_VERSION")"; then
  echo "Using cached GitHub Copilot CLI from ${CACHED_COPILOT_BIN}"
  activate_cached_copilot_bin "$CACHED_COPILOT_BIN"

  echo "Verifying cached Copilot CLI installation..."
  RESOLVED_COPILOT="$(command -v copilot 2>/dev/null || true)"
  if [ -n "$RESOLVED_COPILOT" ]; then
    echo "  Resolved copilot binary: ${RESOLVED_COPILOT}"
    "$RESOLVED_COPILOT" --version
    echo "✓ Copilot CLI installation complete (cached)"
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
