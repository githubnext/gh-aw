#!/usr/bin/env bash
set +o histexpand

# Tests for install_copilot_cli.sh flag-parsing and variable-override logic.
# Run: bash install_copilot_cli_test.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo "PASS: $1"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo "FAIL: $1"; echo "  $2"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

# Inline the flag-parsing and directory-override block from install_copilot_cli.sh
# in a subshell so we can vary the arguments without touching the real filesystem.
parse_and_override() {
  local args=("$@")
  bash -c '
    INSTALL_DIR="/usr/local/bin"
    VERSION=""
    ROOTLESS=false
    for arg in "$@"; do
      case "$arg" in
        --rootless) ROOTLESS=true ;;
        --*) ;;
        *)
          if [ -z "$VERSION" ]; then VERSION="$arg"; fi
          ;;
      esac
    done
    if [ "$ROOTLESS" = "true" ]; then
      INSTALL_DIR="${HOME}/.local/bin"
    fi
    echo "INSTALL_DIR=${INSTALL_DIR}"
    echo "ROOTLESS=${ROOTLESS}"
    echo "VERSION=${VERSION}"
  ' -- "${args[@]}"
}

echo "Running install_copilot_cli.sh tests..."
echo

# Test 1: --rootless sets INSTALL_DIR to $HOME/.local/bin
echo "Test 1: --rootless sets INSTALL_DIR to \$HOME/.local/bin..."
result=$(parse_and_override --rootless)
expected_install_dir="${HOME}/.local/bin"
if echo "$result" | grep -q "INSTALL_DIR=${expected_install_dir}"; then
  pass "INSTALL_DIR is ${expected_install_dir}"
else
  fail "INSTALL_DIR was not ${expected_install_dir}" "$result"
fi

# Test 2: --rootless sets ROOTLESS=true
echo "Test 2: --rootless sets ROOTLESS=true..."
if echo "$result" | grep -q "ROOTLESS=true"; then
  pass "ROOTLESS is true"
else
  fail "ROOTLESS was not true" "$result"
fi

# Test 3: without --rootless, INSTALL_DIR stays /usr/local/bin
echo "Test 3: without --rootless, INSTALL_DIR stays /usr/local/bin..."
result=$(parse_and_override)
if echo "$result" | grep -q "INSTALL_DIR=/usr/local/bin"; then
  pass "INSTALL_DIR is /usr/local/bin"
else
  fail "INSTALL_DIR was not /usr/local/bin" "$result"
fi

# Test 4: without --rootless, ROOTLESS stays false
echo "Test 4: without --rootless, ROOTLESS stays false..."
if echo "$result" | grep -q "ROOTLESS=false"; then
  pass "ROOTLESS is false"
else
  fail "ROOTLESS was not false" "$result"
fi

# Test 5: VERSION and --rootless can be passed in either order
echo "Test 5: VERSION followed by --rootless sets both correctly..."
result=$(parse_and_override 1.2.3 --rootless)
expected_install_dir="${HOME}/.local/bin"
if echo "$result" | grep -q "INSTALL_DIR=${expected_install_dir}" && echo "$result" | grep -q "VERSION=1.2.3"; then
  pass "INSTALL_DIR is ${expected_install_dir} and VERSION is 1.2.3 when passed in order"
else
  fail "Unexpected result when VERSION and --rootless are passed in order" "$result"
fi

# Test 5b: --rootless before VERSION also works
echo "Test 5b: --rootless before VERSION sets both correctly..."
result=$(parse_and_override --rootless 1.2.3)
if echo "$result" | grep -q "INSTALL_DIR=${expected_install_dir}" && echo "$result" | grep -q "VERSION=1.2.3"; then
  pass "INSTALL_DIR is ${expected_install_dir} and VERSION is 1.2.3 when --rootless precedes VERSION"
else
  fail "Unexpected result when --rootless precedes VERSION" "$result"
fi

# Test 6: GITHUB_PATH export — install dir is written when GITHUB_PATH is set
echo "Test 6: INSTALL_DIR is appended to GITHUB_PATH in rootless mode..."
FAKE_GITHUB_PATH=$(mktemp)
bash -c '
  INSTALL_DIR="${HOME}/.local/bin"
  ROOTLESS=true
  GITHUB_PATH="'"${FAKE_GITHUB_PATH}"'"
  if [ "$ROOTLESS" = "true" ]; then
    if [ -n "${GITHUB_PATH:-}" ]; then
      echo "${INSTALL_DIR}" >> "${GITHUB_PATH}"
    else
      echo "WARNING: --rootless install complete but \$GITHUB_PATH is unset; add ${INSTALL_DIR} to PATH manually" >&2
    fi
  fi
'
expected_path_entry="${HOME}/.local/bin"
if grep -qF "${expected_path_entry}" "${FAKE_GITHUB_PATH}"; then
  pass "${expected_path_entry} written to GITHUB_PATH"
else
  fail "${expected_path_entry} not found in GITHUB_PATH" "$(cat "${FAKE_GITHUB_PATH}")"
fi
rm -f "${FAKE_GITHUB_PATH}"

# Test 7: warning emitted when GITHUB_PATH is unset in rootless mode
echo "Test 7: warning emitted when GITHUB_PATH is unset in rootless mode..."
warning_output=$(bash -c '
  INSTALL_DIR="${HOME}/.local/bin"
  ROOTLESS=true
  unset GITHUB_PATH
  if [ "$ROOTLESS" = "true" ]; then
    if [ -n "${GITHUB_PATH:-}" ]; then
      echo "${INSTALL_DIR}" >> "${GITHUB_PATH}"
    else
      echo "WARNING: --rootless install complete but \$GITHUB_PATH is unset; add ${INSTALL_DIR} to PATH manually" >&2
    fi
  fi
' 2>&1)
if echo "$warning_output" | grep -q "WARNING"; then
  pass "WARNING emitted when GITHUB_PATH is unset"
else
  fail "No WARNING when GITHUB_PATH is unset" "$warning_output"
fi

# Test 8: maybe_sudo skips sudo in rootless mode
echo "Test 8: maybe_sudo runs command directly in rootless mode..."
maybe_sudo_output=$(bash -c '
  ROOTLESS=true
  maybe_sudo() {
    if [ "$ROOTLESS" = "true" ]; then
      "$@"
    else
      sudo "$@"
    fi
  }
  maybe_sudo echo "direct"
')
if [ "$maybe_sudo_output" = "direct" ]; then
  pass "maybe_sudo runs command directly without sudo in rootless mode"
else
  fail "maybe_sudo did not run command directly in rootless mode" "$maybe_sudo_output"
fi

# Test 9: maybe_sudo invokes sudo in non-rootless mode
echo "Test 9: maybe_sudo invokes sudo in non-rootless mode..."
FAKE_BIN_DIR=$(mktemp -d)
FAKE_SUDO="${FAKE_BIN_DIR}/sudo"
FAKE_SUDO_LOG="${FAKE_BIN_DIR}/sudo.log"
export FAKE_SUDO_LOG
cat > "${FAKE_SUDO}" <<'SUDOSCRIPT'
#!/usr/bin/env bash
echo "sudo-invoked: $*" >> "${FAKE_SUDO_LOG}"
exec "$@"
SUDOSCRIPT
chmod +x "${FAKE_SUDO}"
PATH="${FAKE_BIN_DIR}:${PATH}" bash -c '
  ROOTLESS=false
  maybe_sudo() {
    if [ "$ROOTLESS" = "true" ]; then
      "$@"
    else
      sudo "$@"
    fi
  }
  maybe_sudo echo "via-sudo"
'
if [ -f "${FAKE_SUDO_LOG}" ] && grep -q "sudo-invoked" "${FAKE_SUDO_LOG}"; then
  pass "maybe_sudo invokes sudo in non-rootless mode"
else
  fail "maybe_sudo did not invoke sudo in non-rootless mode" "sudo log: $(cat "${FAKE_SUDO_LOG}" 2>/dev/null || echo '(empty)')"
fi
unset FAKE_SUDO_LOG
rm -rf "${FAKE_BIN_DIR}"

echo
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"

if [ "$TESTS_FAILED" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
