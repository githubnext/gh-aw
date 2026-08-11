#!/usr/bin/env bash
set +o histexpand

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/install_ripgrep.sh"

TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo "PASS: $1"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo "FAIL: $1"; echo "  $2"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

setup_test_env() {
  TEST_DIR=$(mktemp -d)
}

cleanup_test_env() {
  rm -rf "${TEST_DIR}"
}

echo "Running install_ripgrep.sh tests..."
echo

echo "Test 1: rg already exists — skips apt..."
setup_test_env
SUDO_CALLED="${TEST_DIR}/sudo_called"
cat > "${TEST_DIR}/rg" <<'EOF'
#!/bin/bash
echo "ripgrep 14.1.1"
EOF
cat > "${TEST_DIR}/sudo" <<EOF
#!/bin/bash
touch "${SUDO_CALLED}"
exit 1
EOF
chmod +x "${TEST_DIR}/rg" "${TEST_DIR}/sudo"

output=$(PATH="${TEST_DIR}" /bin/bash "${SCRIPT}" 2>&1)
exit_code=$?

if [ $exit_code -eq 0 ]; then
  pass "existing rg exits 0"
else
  fail "existing rg returned non-zero (exit=${exit_code})" "$output"
fi
if echo "$output" | grep -q "ripgrep 14.1.1"; then
  pass "prints existing rg version"
else
  fail "missing rg version output" "$output"
fi
if [ ! -f "${SUDO_CALLED}" ]; then
  pass "sudo was NOT called when rg exists"
else
  fail "sudo was called unexpectedly" "$output"
fi
cleanup_test_env

echo "Test 2: rg missing — installs with apt..."
setup_test_env
SUDO_CALLS="${TEST_DIR}/sudo_calls"
cat > "${TEST_DIR}/sudo" <<EOF
#!/bin/bash
printf '%s\n' "\$*" >> "${SUDO_CALLS}"
exit 0
EOF
chmod +x "${TEST_DIR}/sudo"

output=$(PATH="${TEST_DIR}" /bin/bash "${SCRIPT}" 2>&1)
exit_code=$?

if [ $exit_code -eq 0 ]; then
  pass "missing rg install path exits 0"
else
  fail "missing rg install path returned non-zero (exit=${exit_code})" "$output"
fi
if grep -qx "apt-get update -qq" "${SUDO_CALLS}" && grep -qx "apt-get install -y -qq ripgrep" "${SUDO_CALLS}"; then
  pass "apt-get update and install were called through sudo"
else
  fail "missing expected sudo apt-get calls" "$(cat "${SUDO_CALLS}" 2>/dev/null || true)"
fi
cleanup_test_env

echo
echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -ne 0 ]; then
  exit 1
fi
