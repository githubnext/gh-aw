#!/usr/bin/env bash
set -euo pipefail

# Tests for install_copilot_cli.sh toolcache selection logic.
# Run: bash actions/setup/sh/install_copilot_cli_test.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_SCRIPT="${SCRIPT_DIR}/install_copilot_cli.sh"
TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "$TEST_ROOT"' EXIT

# Load the production helpers without executing the install script's top-level code.
# Each function ends with an unindented closing brace.
# shellcheck source=/dev/null
source <(sed -n \
  -e '/^normalize_version()/,/^}/p' \
  -e '/^version_is_numeric()/,/^}/p' \
  -e '/^version_is_greater()/,/^}/p' \
  -e '/^is_cache_expired()/,/^}/p' \
  -e '/^find_cached_copilot_bin()/,/^}/p' \
  "$INSTALL_SCRIPT")

export ARCH_NAME="x64"
export COPILOT_TOOLCACHE_MAX_DEPTH=4
export SECONDS_PER_DAY=86400
RUNNER_TOOL_CACHE="${TEST_ROOT}/toolcache"

make_cached_copilot() {
  local version="$1"
  local binary="${RUNNER_TOOL_CACHE}/copilot-cli/${version}/${ARCH_NAME}/bin/copilot"
  mkdir -p "$(dirname "$binary")"
  printf '#!/usr/bin/env bash\nprintf "copilot %s\\n"\n' "$version" > "$binary"
  chmod +x "$binary"
  printf '%s\n' "$binary"
}

assert_found() {
  local description="$1"
  shift
  if ! find_cached_copilot_bin "$@" >/dev/null; then
    echo "FAIL: ${description}" >&2
    exit 1
  fi
  echo "PASS: ${description}"
}

assert_not_found() {
  local description="$1"
  shift
  if find_cached_copilot_bin "$@" >/dev/null; then
    echo "FAIL: ${description}" >&2
    exit 1
  fi
  echo "PASS: ${description}"
}

cached_binary="$(make_cached_copilot "1.0.56")"

assert_not_found \
  "explicit pin equal to the compiler default rejects a mismatched in-range cache entry" \
  "1.0.75" "1.0.21" "1.0.75" "14"

assert_found \
  "compiler default accepts an in-range cache entry" \
  "latest" "1.0.21" "1.0.75" ""

assert_found \
  "compiler default accepts an entry in an open-ended range" \
  "latest" "1.0.21" "*" ""

touch -d "20 days ago" "$cached_binary"
assert_not_found \
  "compiler-default range fallback rejects an expired cache entry" \
  "latest" "1.0.21" "1.0.75" "14"
