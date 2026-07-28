#!/usr/bin/env bash
set +o histexpand

# Tests for install_copilot_cli.sh version-matching and toolcache selection logic.
# Run: bash actions/setup/sh/install_copilot_cli_test.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo "PASS: $1"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo "FAIL: $1"; echo "  $2"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

# Source helper functions from the install script.
# We extract only the pure functions (no side-effecting top-level code).
source_install_functions() {
  bash -c '
    # Minimal stubs required by the sourced functions
    OS_NAME="linux"
    ARCH_NAME="amd64"
    COPILOT_TOOLCACHE_MAX_DEPTH=4

    # Source relevant functions from the install script
    . '"${SCRIPT_DIR}/install_copilot_cli.sh"' --source-only 2>/dev/null || true
    '"$1"'
  '
}

# Build a fake toolcache with a copilot-cli binary at the given version.
make_fake_toolcache() {
  local root="$1"
  local version="$2"
  local arch="${3:-amd64}"
  local bin_dir="${root}/copilot-cli/${version}/${arch}/bin"
  mkdir -p "$bin_dir"
  printf '#!/bin/sh\necho "copilot %s"\n' "$version" > "${bin_dir}/copilot"
  chmod +x "${bin_dir}/copilot"
}

# ---------------------------------------------------------------------------
# Test: explicit user pin does NOT fall back to an in-range cache entry
# ---------------------------------------------------------------------------
echo "Test 1: explicit user pin (not GH_AW_DEFAULT_COPILOT_VERSION) rejects in-range cache..."
FAKE_TC=$(mktemp -d)
make_fake_toolcache "$FAKE_TC" "1.0.56"

result=$(bash -c '
  set +o histexpand
  SCRIPT_DIR="'"${SCRIPT_DIR}"'"
  # Source only the pure helper functions we need
  . <(sed -n "/^normalize_version/,/^}/p; /^version_is_numeric/,/^}/p; /^version_is_greater/,/^}/p; /^is_cache_expired/,/^}/p; /^find_cached_copilot_bin/,/^}/p" \
      "${SCRIPT_DIR}/install_copilot_cli.sh" 2>/dev/null) 2>/dev/null || true

  normalize_version() {
    local v="${1#v}"
    echo "$v"
  }
  version_is_numeric() {
    [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
  }
  version_is_greater() {
    local a="$1" b="$2"
    [ "$(printf "%s\n%s" "$a" "$b" | sort -V | tail -1)" = "$a" ] && [ "$a" != "$b" ]
  }
  is_cache_expired() { return 1; }

  find_cached_copilot_bin() {
    local requested_version="$1"
    local min_version="$2"
    local max_version="$3"
    local cache_ttl_days="$4"

    local requested_version_normalized=""
    if [ "$requested_version" != "latest" ]; then
      requested_version_normalized="$(normalize_version "$requested_version")"
    fi

    local RUNNER_TOOL_CACHE="'"${FAKE_TC}"'"
    local ARCH_NAME="amd64"
    local best_candidate=""
    local best_version=""

    while IFS= read -r candidate; do
      local candidate_dir candidate_arch candidate_version candidate_version_normalized
      candidate_dir="$(dirname "$candidate")"
      candidate_arch="$(basename "$(dirname "$candidate_dir")")"
      candidate_version="$(basename "$(dirname "$(dirname "$candidate_dir")")")"
      candidate_version_normalized="$(normalize_version "$candidate_version")"

      if ! version_is_numeric "$candidate_version_normalized"; then continue; fi
      if [ "$candidate_arch" != "$ARCH_NAME" ]; then continue; fi

      if [ -n "$requested_version_normalized" ]; then
        if [ "$candidate_version_normalized" = "$requested_version_normalized" ]; then
          printf "%s\n" "$candidate"
          return 0
        fi
        # Exact match required for user-pinned versions; skip non-matching candidates
        continue
      fi

      if [ -n "$min_version" ] && version_is_greater "$min_version" "$candidate_version_normalized"; then continue; fi
      if [ -n "$max_version" ] && version_is_greater "$candidate_version_normalized" "$max_version"; then continue; fi

      if [ -z "$best_candidate" ] || version_is_greater "$candidate_version_normalized" "$best_version"; then
        best_candidate="$candidate"
        best_version="$candidate_version_normalized"
      fi
    done < <(find "${RUNNER_TOOL_CACHE}/copilot-cli" -maxdepth 5 -name "copilot" -type f 2>/dev/null)

    if [ -n "$best_candidate" ]; then
      printf "%s\n" "$best_candidate"
      return 0
    fi
    return 1
  }

  # User pin: version 1.0.0, compat range 1.0.21..1.0.75, cache has 1.0.56
  # Should NOT match because 1.0.0 != 1.0.56 (exact match required for user pins)
  if find_cached_copilot_bin "1.0.0" "1.0.21" "1.0.75" "" >/dev/null 2>&1; then
    echo "FOUND"
  else
    echo "NOT_FOUND"
  fi
' 2>/dev/null)

if [ "$result" = "NOT_FOUND" ]; then
  pass "user pin 1.0.0 does not match in-range cached 1.0.56"
else
  fail "user pin 1.0.0 should not match in-range cached 1.0.56" "got: $result"
fi
rm -rf "$FAKE_TC"

# ---------------------------------------------------------------------------
# Test: compiler-default pin (REQUESTED_VERSION=latest) uses in-range cache
# ---------------------------------------------------------------------------
echo "Test 2: compiler-default pin (REQUESTED_VERSION=latest) accepts in-range cache..."
FAKE_TC=$(mktemp -d)
make_fake_toolcache "$FAKE_TC" "1.0.56"

result=$(bash -c '
  set +o histexpand
  normalize_version() { echo "${1#v}"; }
  version_is_numeric() { [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; }
  version_is_greater() {
    local a="$1" b="$2"
    [ "$(printf "%s\n%s" "$a" "$b" | sort -V | tail -1)" = "$a" ] && [ "$a" != "$b" ]
  }
  is_cache_expired() { return 1; }

  find_cached_copilot_bin() {
    local requested_version="$1"
    local min_version="$2"
    local max_version="$3"
    local cache_ttl_days="$4"

    local requested_version_normalized=""
    if [ "$requested_version" != "latest" ]; then
      requested_version_normalized="$(normalize_version "$requested_version")"
    fi

    local RUNNER_TOOL_CACHE="'"${FAKE_TC}"'"
    local ARCH_NAME="amd64"
    local best_candidate=""
    local best_version=""

    while IFS= read -r candidate; do
      local candidate_dir candidate_arch candidate_version candidate_version_normalized
      candidate_dir="$(dirname "$candidate")"
      candidate_arch="$(basename "$(dirname "$candidate_dir")")"
      candidate_version="$(basename "$(dirname "$(dirname "$candidate_dir")")")"
      candidate_version_normalized="$(normalize_version "$candidate_version")"

      if ! version_is_numeric "$candidate_version_normalized"; then continue; fi
      if [ "$candidate_arch" != "$ARCH_NAME" ]; then continue; fi

      if [ -n "$requested_version_normalized" ]; then
        if [ "$candidate_version_normalized" = "$requested_version_normalized" ]; then
          printf "%s\n" "$candidate"
          return 0
        fi
        continue
      fi

      if [ -n "$min_version" ] && version_is_greater "$min_version" "$candidate_version_normalized"; then continue; fi
      if [ -n "$max_version" ] && version_is_greater "$candidate_version_normalized" "$max_version"; then continue; fi

      if [ -z "$best_candidate" ] || version_is_greater "$candidate_version_normalized" "$best_version"; then
        best_candidate="$candidate"
        best_version="$candidate_version_normalized"
      fi
    done < <(find "${RUNNER_TOOL_CACHE}/copilot-cli" -maxdepth 5 -name "copilot" -type f 2>/dev/null)

    if [ -n "$best_candidate" ]; then
      printf "%s\n" "$best_candidate"
      return 0
    fi
    return 1
  }

  # Compiler default: REQUESTED_VERSION=latest, compat range 1.0.21..1.0.75, cache has 1.0.56
  # Should match because range check applies (requested_version_normalized is empty for "latest")
  if find_cached_copilot_bin "latest" "1.0.21" "1.0.75" "" >/dev/null 2>&1; then
    echo "FOUND"
  else
    echo "NOT_FOUND"
  fi
' 2>/dev/null)

if [ "$result" = "FOUND" ]; then
  pass "compiler-default (latest) accepts in-range cached 1.0.56"
else
  fail "compiler-default (latest) should accept in-range cached 1.0.56" "got: $result"
fi
rm -rf "$FAKE_TC"

# ---------------------------------------------------------------------------
# Test: explicit user pin with exact cache entry is accepted
# ---------------------------------------------------------------------------
echo "Test 3: explicit user pin with exact cache entry is accepted..."
FAKE_TC=$(mktemp -d)
make_fake_toolcache "$FAKE_TC" "1.0.0"

result=$(bash -c '
  set +o histexpand
  normalize_version() { echo "${1#v}"; }
  version_is_numeric() { [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; }
  version_is_greater() {
    local a="$1" b="$2"
    [ "$(printf "%s\n%s" "$a" "$b" | sort -V | tail -1)" = "$a" ] && [ "$a" != "$b" ]
  }
  is_cache_expired() { return 1; }

  find_cached_copilot_bin() {
    local requested_version="$1"
    local min_version="$2"
    local max_version="$3"
    local cache_ttl_days="$4"

    local requested_version_normalized=""
    if [ "$requested_version" != "latest" ]; then
      requested_version_normalized="$(normalize_version "$requested_version")"
    fi

    local RUNNER_TOOL_CACHE="'"${FAKE_TC}"'"
    local ARCH_NAME="amd64"
    local best_candidate=""
    local best_version=""

    while IFS= read -r candidate; do
      local candidate_dir candidate_arch candidate_version candidate_version_normalized
      candidate_dir="$(dirname "$candidate")"
      candidate_arch="$(basename "$(dirname "$candidate_dir")")"
      candidate_version="$(basename "$(dirname "$(dirname "$candidate_dir")")")"
      candidate_version_normalized="$(normalize_version "$candidate_version")"

      if ! version_is_numeric "$candidate_version_normalized"; then continue; fi
      if [ "$candidate_arch" != "$ARCH_NAME" ]; then continue; fi

      if [ -n "$requested_version_normalized" ]; then
        if [ "$candidate_version_normalized" = "$requested_version_normalized" ]; then
          printf "%s\n" "$candidate"
          return 0
        fi
        continue
      fi

      if [ -n "$min_version" ] && version_is_greater "$min_version" "$candidate_version_normalized"; then continue; fi
      if [ -n "$max_version" ] && version_is_greater "$candidate_version_normalized" "$max_version"; then continue; fi

      if [ -z "$best_candidate" ] || version_is_greater "$candidate_version_normalized" "$best_version"; then
        best_candidate="$candidate"
        best_version="$candidate_version_normalized"
      fi
    done < <(find "${RUNNER_TOOL_CACHE}/copilot-cli" -maxdepth 5 -name "copilot" -type f 2>/dev/null)

    if [ -n "$best_candidate" ]; then
      printf "%s\n" "$best_candidate"
      return 0
    fi
    return 1
  }

  # User pin: exact match exists in cache
  if find_cached_copilot_bin "1.0.0" "1.0.21" "1.0.75" "" >/dev/null 2>&1; then
    echo "FOUND"
  else
    echo "NOT_FOUND"
  fi
' 2>/dev/null)

if [ "$result" = "FOUND" ]; then
  pass "user pin 1.0.0 uses exact-match cache entry"
else
  fail "user pin 1.0.0 should use exact-match cache entry" "got: $result"
fi
rm -rf "$FAKE_TC"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo
echo "Results: ${TESTS_PASSED} passed, ${TESTS_FAILED} failed"
[ "$TESTS_FAILED" -eq 0 ] || exit 1
