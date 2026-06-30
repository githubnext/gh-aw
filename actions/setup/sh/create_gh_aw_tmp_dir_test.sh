#!/usr/bin/env bash
set +o histexpand

# Test script for create_gh_aw_tmp_dir.sh
# Run: bash create_gh_aw_tmp_dir_test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/create_gh_aw_tmp_dir.sh"

TESTS_PASSED=0
TESTS_FAILED=0

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

echo "Testing create_gh_aw_tmp_dir.sh"
echo ""

# ── Test 1: Script syntax is valid ──────────────────────────────────────────
echo "Test 1: Script syntax is valid"
assert "script passes bash -n" "bash -n '${SCRIPT}'"
echo ""

# ── Test 2: Creates expected directories when they don't exist ───────────────
echo "Test 2: Creates expected directories from scratch"
set +e
OUTPUT="$(bash "${SCRIPT}" 2>&1)"
EXIT_CODE=$?
set -e
assert "script exits 0" "[ '${EXIT_CODE}' = '0' ]"
assert "/tmp/gh-aw/agent directory created" "[ -d /tmp/gh-aw/agent ]"
assert "/tmp/gh-aw/sandbox/agent/logs directory created" "[ -d /tmp/gh-aw/sandbox/agent/logs ]"
assert "output mentions created directory" "echo '${OUTPUT}' | grep -q 'Created /tmp/gh-aw/agent directory'"
echo ""

# ── Test 3: Preserves sandbox owned by current user ──────────────────────────
echo "Test 3: Preserves sandbox directory when owned by current user"
mkdir -p /tmp/gh-aw/sandbox
MARKER_FILE="/tmp/gh-aw/sandbox/.owner-check-marker"
touch "${MARKER_FILE}"
set +e
OUTPUT="$(bash "${SCRIPT}" 2>&1)"
EXIT_CODE=$?
set -e
assert "script exits 0 with existing user-owned sandbox" "[ '${EXIT_CODE}' = '0' ]"
assert "/tmp/gh-aw/sandbox/agent/logs created" "[ -d /tmp/gh-aw/sandbox/agent/logs ]"
# Marker should still be present — sandbox was NOT removed since we own it and it is writable
assert "user-owned sandbox is preserved (marker still present)" "[ -f '${MARKER_FILE}' ]"
assert "no WARN about reclaiming in output" "! echo '${OUTPUT}' | grep -q 'reclaiming'"
rm -f "${MARKER_FILE}"
echo ""

# ── Test 4: Reclaims sandbox that is not writable by the current user ─────────
# We simulate a non-writable sandbox by creating a read-only directory.  A fake sudo records
# its arguments and performs the actual removal (rm -rf works here because the *parent*
# /tmp/gh-aw is writable by the current user even when the child has mode 555).
echo "Test 4: Reclaims sandbox that is not writable (simulated)"
mkdir -p /tmp/gh-aw/sandbox
MARKER_FILE="/tmp/gh-aw/sandbox/.non-writable-marker"
touch "${MARKER_FILE}"
# Make the sandbox not writable by the current user
chmod 555 /tmp/gh-aw/sandbox

FAKE_BIN="$(mktemp -d)"
SUDO_ARGS_FILE="${FAKE_BIN}/sudo_args"

# Fake sudo: record the full argument list, then execute the real command.
# Before running rm -rf, fix permissions on mode-555 directories so the
# removal succeeds (mimicking what root privilege provides in production).
cat > "${FAKE_BIN}/sudo" <<EOF
#!/usr/bin/env bash
echo "\$*" >> "${SUDO_ARGS_FILE}"
# Fix permissions on any non-writable subdirectories before removal
# (root bypasses DAC checks; simulate that by chmod-ing first)
for arg in "\$@"; do
  if [ -d "\${arg}" ]; then
    find "\${arg}" -type d -not -perm -u+w -exec chmod u+w {} + 2>/dev/null || true
  fi
done
exec "\$@"
EOF
chmod +x "${FAKE_BIN}/sudo"

set +e
OUTPUT="$(PATH="${FAKE_BIN}:${PATH}" bash "${SCRIPT}" 2>&1)"
EXIT_CODE=$?
set -e

assert "script exits 0 with non-writable sandbox" "[ '${EXIT_CODE}' = '0' ]"
assert "sudo was invoked" "[ -f '${SUDO_ARGS_FILE}' ]"
assert "sudo was called with rm -rf and the sandbox path" "grep -q 'rm -rf.*sandbox' '${SUDO_ARGS_FILE}'"
assert "WARN about reclaiming appears in output" "echo '${OUTPUT}' | grep -q 'reclaiming'"
assert "/tmp/gh-aw/sandbox/agent/logs recreated after removal" "[ -d /tmp/gh-aw/sandbox/agent/logs ]"
assert "non-writable sandbox was removed (marker gone)" "[ ! -f '${MARKER_FILE}' ]"

rm -rf "${FAKE_BIN}"
echo ""

# ── Summary ──────────────────────────────────────────────────────────────────
echo "Results: ${TESTS_PASSED} passed, ${TESTS_FAILED} failed"
if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

