#!/usr/bin/env bash
set +o histexpand

# Tests for install_threat_detect_binary.sh platform resolver.
# Run: bash actions/setup/sh/install_threat_detect_binary_test.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=actions/setup/sh/install_threat_detect_binary.sh
source "${SCRIPT_DIR}/install_threat_detect_binary.sh"
set +e

TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo "PASS: $1"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo "FAIL: $1"; echo "  $2"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

assert_maps_to() {
  local os="$1"
  local arch="$2"
  local expected="$3"
  local result

  if result="$(resolve_binary_name "$os" "$arch" 2>&1)"; then
    if [ "$result" = "$expected" ]; then
      pass "${os} ${arch} -> ${expected}"
    else
      fail "${os} ${arch} did not map to ${expected}" "got: ${result}"
    fi
  else
    fail "${os} ${arch} unexpectedly failed" "got: ${result}"
  fi
}

assert_fails_with() {
  local os="$1"
  local arch="$2"
  local expected_msg="$3"
  local result
  local exit_code

  result="$(resolve_binary_name "$os" "$arch" 2>&1)"
  exit_code=$?

  if [ "$exit_code" -ne 0 ] && echo "$result" | grep -q "$expected_msg"; then
    pass "${os} ${arch} -> expected error"
  else
    fail "${os} ${arch} did not fail as expected" "exit=${exit_code}, output=${result}"
  fi
}

echo "Running install_threat_detect_binary.sh tests..."
echo

assert_maps_to "Linux" "x86_64" "threat-detect-linux-amd64"
assert_maps_to "Linux" "amd64" "threat-detect-linux-amd64"
assert_maps_to "Linux" "aarch64" "threat-detect-linux-arm64"
assert_maps_to "Linux" "arm64" "threat-detect-linux-arm64"
assert_maps_to "Darwin" "x86_64" "threat-detect-darwin-x64"
assert_maps_to "Darwin" "arm64" "threat-detect-darwin-arm64"

assert_fails_with "Darwin" "aarch64" "Unsupported macOS architecture"
assert_fails_with "Linux" "s390x" "Unsupported Linux architecture"
assert_fails_with "Windows_NT" "x86_64" "Unsupported operating system"

echo
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"

if [ "$TESTS_FAILED" -gt 0 ]; then
  exit 1
fi

echo "All tests passed!"
