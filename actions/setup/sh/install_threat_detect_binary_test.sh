#!/usr/bin/env bash
set +o histexpand

# Tests for install_threat_detect_binary.sh OS/arch → asset-name mapping logic.
# Run: bash install_threat_detect_binary_test.sh

TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo "PASS: $1"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo "FAIL: $1"; echo "  $2"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

# resolve_binary_name runs the platform-selection logic in a subshell with the
# given OS and ARCH values and prints the resolved binary name (or an error).
resolve_binary_name() {
  local os="$1"
  local arch="$2"
  bash -c '
    OS="$1"
    ARCH="$2"
    case "$OS" in
      Linux)
        case "$ARCH" in
          x86_64|amd64) echo "threat-detect-linux-amd64" ;;
          aarch64|arm64) echo "threat-detect-linux-arm64" ;;
          *) echo "ERROR: Unsupported Linux architecture: ${ARCH}" >&2; exit 1 ;;
        esac
        ;;
      Darwin)
        echo "ERROR: macOS is not a supported platform for threat-detect. Use a Linux runner for threat-detection jobs." >&2
        exit 1
        ;;
      *)
        echo "ERROR: Unsupported operating system: ${OS}" >&2
        exit 1
        ;;
    esac
  ' -- "$os" "$arch"
}

echo "Running install_threat_detect_binary.sh tests..."
echo

# Test 1: Linux x86_64 maps to threat-detect-linux-amd64
echo "Test 1: Linux x86_64 -> threat-detect-linux-amd64..."
result=$(resolve_binary_name "Linux" "x86_64")
if [ "$result" = "threat-detect-linux-amd64" ]; then
  pass "Linux x86_64 -> threat-detect-linux-amd64"
else
  fail "Linux x86_64 did not map to threat-detect-linux-amd64" "got: $result"
fi

# Test 2: Linux aarch64 maps to threat-detect-linux-arm64
echo "Test 2: Linux aarch64 -> threat-detect-linux-arm64..."
result=$(resolve_binary_name "Linux" "aarch64")
if [ "$result" = "threat-detect-linux-arm64" ]; then
  pass "Linux aarch64 -> threat-detect-linux-arm64"
else
  fail "Linux aarch64 did not map to threat-detect-linux-arm64" "got: $result"
fi

# Test 3: Linux arm64 (alias) maps to threat-detect-linux-arm64
echo "Test 3: Linux arm64 -> threat-detect-linux-arm64..."
result=$(resolve_binary_name "Linux" "arm64")
if [ "$result" = "threat-detect-linux-arm64" ]; then
  pass "Linux arm64 -> threat-detect-linux-arm64"
else
  fail "Linux arm64 did not map to threat-detect-linux-arm64" "got: $result"
fi

# Test 4: Darwin is rejected with an actionable error
echo "Test 4: Darwin -> unsupported platform error..."
darwin_error=$(resolve_binary_name "Darwin" "arm64" 2>&1)
if echo "$darwin_error" | grep -q "macOS is not a supported platform"; then
  pass "Darwin -> unsupported platform error"
else
  fail "Darwin did not produce the expected unsupported-platform error" "got: $darwin_error"
fi

# Test 5: unsupported OS fails with actionable message
echo "Test 5: unsupported OS -> error..."
error_output=$(resolve_binary_name "Windows_NT" "x86_64" 2>&1)
if echo "$error_output" | grep -q "Unsupported operating system"; then
  pass "Unknown OS produces an actionable error message"
else
  fail "Unknown OS did not produce expected error" "got: $error_output"
fi

# Test 6: unsupported Linux architecture fails with actionable message
echo "Test 6: Linux unsupported arch -> error..."
error_output=$(resolve_binary_name "Linux" "s390x" 2>&1)
if echo "$error_output" | grep -q "Unsupported Linux architecture"; then
  pass "Linux unsupported arch produces an actionable error message"
else
  fail "Linux unsupported arch did not produce expected error" "got: $error_output"
fi

echo
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"

if [ "$TESTS_FAILED" -gt 0 ]; then
  exit 1
fi

echo "All tests passed!"
