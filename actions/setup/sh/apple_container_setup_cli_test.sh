#!/bin/bash
set +o histexpand

# Test script for apple_container_setup_cli.sh
#
# The CLI version window is the gate that keeps AWF's init-image contract honest:
# a `container` release outside [MIN, MAX) may relocate the real vminitd and boot
# a guest with no capability relay. These tests exercise the comparison directly
# rather than through an installation, so they run anywhere.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="$SCRIPT_DIR/apple_container_setup_cli.sh"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

print_result() {
  local test_name="$1"
  local result="$2"
  local message="${3:-}"

  TESTS_RUN=$((TESTS_RUN + 1))

  if [ "$result" = "PASS" ]; then
    echo -e "${GREEN}✓ PASS${NC}: $test_name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo -e "${RED}✗ FAIL${NC}: $test_name"
    if [ -n "$message" ]; then
      echo -e "  ${YELLOW}Message:${NC} $message"
    fi
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

# Extract the pure version-comparison helpers from the script and evaluate them
# in isolation. Sourcing the whole script would run the resolution logic and try
# to touch the host, so only the two functions under test are lifted out.
eval "$(sed -n '/^version_key()/,/^}/p' "$SCRIPT_PATH")"
eval "$(sed -n '/^version_in_window()/,/^}/p' "$SCRIPT_PATH")"

MIN_CLI="0.4.0"
MAX_CLI="1.0.0"

assert_in_window() {
  local version="$1"
  if version_in_window "$version"; then
    print_result "version ${version} is accepted" "PASS"
  else
    print_result "version ${version} is accepted" "FAIL" "expected ${version} to be inside [${MIN_CLI}, ${MAX_CLI})"
  fi
}

assert_out_of_window() {
  local version="$1" why="$2"
  if version_in_window "$version"; then
    print_result "version ${version} is rejected (${why})" "FAIL" "expected ${version} to be outside [${MIN_CLI}, ${MAX_CLI})"
  else
    print_result "version ${version} is rejected (${why})" "PASS"
  fi
}

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "Test 1: Script syntax is valid"
echo "═══════════════════════════════════════════════════════════"
if bash -n "$SCRIPT_PATH" 2>/dev/null; then
  print_result "Script syntax is valid" "PASS"
else
  print_result "Script syntax is valid" "FAIL" "bash -n reported an error"
fi

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "Test 2: Versions inside the supported window are accepted"
echo "═══════════════════════════════════════════════════════════"
assert_in_window "0.4.0"    # inclusive lower bound
assert_in_window "0.4.1"
assert_in_window "0.9.0"
assert_in_window "0.12.3"   # the pinned release
assert_in_window "0.99.99"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "Test 3: Versions outside the supported window are rejected"
echo "═══════════════════════════════════════════════════════════"
assert_out_of_window "0.3.0" "below the minimum"
assert_out_of_window "0.1.0" "below the minimum"
assert_out_of_window "1.0.0" "exclusive upper bound"
assert_out_of_window "1.2.2" "1.x is outside AWF's contract range"
assert_out_of_window "1.3.0" "1.x is outside AWF's contract range"
assert_out_of_window "2.0.0" "far above the maximum"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "Test 4: Numeric comparison, not lexicographic"
echo "═══════════════════════════════════════════════════════════"
# A naive string compare would place "0.12.3" below "0.4.0" and reject the
# pinned release outright, so this ordering is asserted explicitly.
assert_in_window "0.12.0"
assert_in_window "0.10.0"
if [ "$(version_key 0.12.3)" \> "$(version_key 0.4.0)" ]; then
  print_result "0.12.3 sorts above 0.4.0" "PASS"
else
  print_result "0.12.3 sorts above 0.4.0" "FAIL" "double-digit minor versions must compare numerically"
fi

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "Test 5: Pre-release suffixes do not sneak past the upper bound"
echo "═══════════════════════════════════════════════════════════"
# 1.0.0-beta must not be treated as "less than 1.0.0": the init image layout is
# already allowed to have changed by then.
assert_out_of_window "1.0.0-beta" "pre-release of an unsupported major"
assert_in_window "0.12.3-rc1"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "Test 6: Malformed versions are rejected rather than assumed"
echo "═══════════════════════════════════════════════════════════"
assert_out_of_window "" "empty"
assert_out_of_window "latest" "non-numeric"
assert_out_of_window "v0.12.3" "unstripped v prefix is not parsed as numeric"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "Test 7: Two-component versions are tolerated"
echo "═══════════════════════════════════════════════════════════"
assert_in_window "0.5"
assert_out_of_window "1.0" "exclusive upper bound without a patch component"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "Summary"
echo "═══════════════════════════════════════════════════════════"
echo "Tests run:    $TESTS_RUN"
echo -e "Tests passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests failed: ${RED}${TESTS_FAILED}${NC}"

if [ "$TESTS_FAILED" -gt 0 ]; then
  exit 1
fi
echo ""
echo -e "${GREEN}All apple_container_setup_cli.sh tests passed${NC}"
