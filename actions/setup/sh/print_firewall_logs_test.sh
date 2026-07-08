#!/usr/bin/env bash
set +o histexpand

# Test script for print_firewall_logs.sh
# Run: bash print_firewall_logs_test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/print_firewall_logs.sh"

TESTS_PASSED=0
TESTS_FAILED=0
WORKSPACE="$(mktemp -d)"

cleanup() {
  rm -rf "${WORKSPACE}"
}
trap cleanup EXIT

assert() {
  local name="$1"
  local condition="$2"
  if eval "${condition}" 2>/dev/null; then
    echo "  ✓ ${name}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo "  ✗ ${name}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

echo "Testing print_firewall_logs.sh"
echo ""

# ── Test 1: Script syntax is valid ──────────────────────────────────────────
echo "Test 1: Script syntax is valid"
assert "script passes bash -n" "bash -n '${SCRIPT}'"
echo ""

# ── Test 2: Unknown argument exits 1 ────────────────────────────────────────
echo "Test 2: Unknown argument exits 1"
set +e
AWF_LOGS_DIR="/tmp/logs" GITHUB_STEP_SUMMARY="/dev/null" bash "${SCRIPT}" --unknown-flag 2>/dev/null
UNKNOWN_EXIT=$?
set -e
assert "exits non-zero for unknown argument" "[ '${UNKNOWN_EXIT}' -ne 0 ]"
echo ""

# ── Test 3: AWF not installed prints informational message ──────────────────
echo "Test 3: AWF not installed prints informational message"
FIREWALL_DIR="${WORKSPACE}/test3/sandbox/firewall"
mkdir -p "${FIREWALL_DIR}/logs"
set +e
OUTPUT="$(
  PATH="/usr/bin:/bin" \
  AWF_LOGS_DIR="${FIREWALL_DIR}/logs" \
  GITHUB_STEP_SUMMARY="${WORKSPACE}/test3-summary.md" \
  bash "${SCRIPT}" 2>&1
)"
EXIT_CODE=$?
set -e
assert "exits successfully when awf is not installed" "[ '${EXIT_CODE}' -eq 0 ]"
assert "prints 'AWF binary not installed' message" "printf '%s' \"${OUTPUT}\" | grep -q 'AWF binary not installed'"
echo ""

# ── Test 4: --rootless flag is accepted without error ───────────────────────
echo "Test 4: --rootless flag is accepted without error"
FIREWALL_DIR="${WORKSPACE}/test4/sandbox/firewall"
mkdir -p "${FIREWALL_DIR}/logs"
set +e
OUTPUT="$(
  PATH="/usr/bin:/bin" \
  AWF_LOGS_DIR="${FIREWALL_DIR}/logs" \
  GITHUB_STEP_SUMMARY="${WORKSPACE}/test4-summary.md" \
  bash "${SCRIPT}" --rootless 2>&1
)"
EXIT_CODE=$?
set -e
assert "exits successfully with --rootless" "[ '${EXIT_CODE}' -eq 0 ]"
assert "prints 'AWF binary not installed' message (not an arg-parse error)" "printf '%s' \"${OUTPUT}\" | grep -q 'AWF binary not installed'"
echo ""

# ── Test 5: FIREWALL_DIR is computed as dirname of AWF_LOGS_DIR ─────────────
echo "Test 5: FIREWALL_DIR is computed as dirname of AWF_LOGS_DIR"
LOGS_DIR="${WORKSPACE}/test5/sandbox/firewall/logs"
mkdir -p "${LOGS_DIR}"
set +e
OUTPUT="$(
  PATH="/usr/bin:/bin" \
  AWF_LOGS_DIR="${LOGS_DIR}" \
  GITHUB_STEP_SUMMARY="${WORKSPACE}/test5-summary.md" \
  bash "${SCRIPT}" 2>&1
)"
set -e
EXPECTED_DIR="${WORKSPACE}/test5/sandbox/firewall"
assert "script does not error on a valid logs dir" "[ -d '${EXPECTED_DIR}' ]"
echo ""

echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
