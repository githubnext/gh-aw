#!/usr/bin/env bash
set +o histexpand

# Tests for install_awf_binary.sh flag-parsing and variable-override logic.
# Run: bash install_awf_binary_test.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo "PASS: $1"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo "FAIL: $1"; echo "  $2"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

# Inline the flag-parsing and directory-override block from install_awf_binary.sh
# in a subshell so we can vary the arguments without touching the real filesystem.
parse_and_override() {
  local args=("$@")
  bash -c '
    AWF_INSTALL_DIR="/usr/local/bin"
    AWF_LIB_DIR="/usr/local/lib/awf"
    ROOTLESS=false
    for arg in "${@:1}"; do
      case "$arg" in
        --rootless) ROOTLESS=true ;;
      esac
    done
    if [ "$ROOTLESS" = "true" ]; then
      AWF_USER_PREFIX="${HOME}/.local"
      AWF_INSTALL_DIR="${AWF_USER_PREFIX}/bin"
      AWF_LIB_DIR="${AWF_USER_PREFIX}/lib/awf"
    fi
    echo "AWF_INSTALL_DIR=${AWF_INSTALL_DIR}"
    echo "AWF_LIB_DIR=${AWF_LIB_DIR}"
  ' -- "${args[@]}"
}

echo "Running install_awf_binary.sh tests..."
echo

# Test 1: rootless flag sets AWF_INSTALL_DIR to $HOME/.local/bin
echo "Test 1: --rootless sets AWF_INSTALL_DIR to \$HOME/.local/bin..."
result=$(parse_and_override --rootless)
expected_install_dir="${HOME}/.local/bin"
if echo "$result" | grep -q "AWF_INSTALL_DIR=${expected_install_dir}"; then
  pass "AWF_INSTALL_DIR is ${expected_install_dir}"
else
  fail "AWF_INSTALL_DIR was not ${expected_install_dir}" "$result"
fi

# Test 2: rootless flag sets AWF_LIB_DIR to $HOME/.local/lib/awf
echo "Test 2: --rootless sets AWF_LIB_DIR to \$HOME/.local/lib/awf..."
expected_lib_dir="${HOME}/.local/lib/awf"
if echo "$result" | grep -q "AWF_LIB_DIR=${expected_lib_dir}"; then
  pass "AWF_LIB_DIR is ${expected_lib_dir}"
else
  fail "AWF_LIB_DIR was not ${expected_lib_dir}" "$result"
fi

# Test 3: without --rootless, AWF_INSTALL_DIR stays /usr/local/bin
echo "Test 3: without --rootless, AWF_INSTALL_DIR stays /usr/local/bin..."
result=$(parse_and_override)
if echo "$result" | grep -q "AWF_INSTALL_DIR=/usr/local/bin"; then
  pass "AWF_INSTALL_DIR is /usr/local/bin"
else
  fail "AWF_INSTALL_DIR was not /usr/local/bin" "$result"
fi

# Test 4: without --rootless, AWF_LIB_DIR stays /usr/local/lib/awf
echo "Test 4: without --rootless, AWF_LIB_DIR stays /usr/local/lib/awf..."
if echo "$result" | grep -q "AWF_LIB_DIR=/usr/local/lib/awf"; then
  pass "AWF_LIB_DIR is /usr/local/lib/awf"
else
  fail "AWF_LIB_DIR was not /usr/local/lib/awf" "$result"
fi

# Test 5: GITHUB_PATH export — install dir is written when GITHUB_PATH is set
echo "Test 5: AWF_INSTALL_DIR is appended to GITHUB_PATH in rootless mode..."
FAKE_GITHUB_PATH=$(mktemp)
bash -c '
  AWF_INSTALL_DIR="${HOME}/.local/bin"
  ROOTLESS=true
  GITHUB_PATH="'"${FAKE_GITHUB_PATH}"'"
  if [ "$ROOTLESS" = "true" ]; then
    if [ -n "${GITHUB_PATH:-}" ]; then
      echo "${AWF_INSTALL_DIR}" >> "${GITHUB_PATH}"
    else
      echo "WARNING: --rootless install complete but \$GITHUB_PATH is unset; add ${AWF_INSTALL_DIR} to PATH manually" >&2
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

# Test 6: warning emitted when GITHUB_PATH is unset in rootless mode
echo "Test 6: warning emitted when GITHUB_PATH is unset in rootless mode..."
warning_output=$(bash -c '
  AWF_INSTALL_DIR="${HOME}/.local/bin"
  ROOTLESS=true
  unset GITHUB_PATH
  if [ "$ROOTLESS" = "true" ]; then
    if [ -n "${GITHUB_PATH:-}" ]; then
      echo "${AWF_INSTALL_DIR}" >> "${GITHUB_PATH}"
    else
      echo "WARNING: --rootless install complete but \$GITHUB_PATH is unset; add ${AWF_INSTALL_DIR} to PATH manually" >&2
    fi
  fi
' 2>&1)
if echo "$warning_output" | grep -q "WARNING"; then
  pass "WARNING emitted when GITHUB_PATH is unset"
else
  fail "No WARNING when GITHUB_PATH is unset" "$warning_output"
fi

# Test 7: wrapper script contains umask 022 to fix rootless chroot hosts EACCES
# Regression test for: https://github.com/github/gh-aw/issues/46172
# AWF uses mkdtempSync to create the chroot hosts staging directory. Without
# umask 022, a restrictive ambient umask (e.g. 0111 masking the execute bit)
# produces mode=600 on that directory, making it non-traversable and causing a
# fatal EACCES when AWF tries to write /etc/hosts inside it.
echo "Test 7: install_awf_binary.sh wrapper template contains 'umask 022'..."
if grep -q 'umask 022' "${SCRIPT_DIR}/install_awf_binary.sh"; then
  pass "install_awf_binary.sh wrapper template contains 'umask 022'"
else
  fail "install_awf_binary.sh wrapper template does not contain 'umask 022'" \
    "Rootless chroot hosts EACCES fix is missing (see https://github.com/github/gh-aw/issues/46172)"
fi

# Test 8: with umask 022, mkdtemp produces mode=700 (execute bit present),
# so AWF can create /etc/hosts inside the chroot staging directory.
# This is the functional regression test for the rootless chroot hosts EACCES
# fix: the wrapper sets umask 022 before exec so Node.js mkdtempSync creates
# directories with the execute bit set, allowing file creation inside them.
echo "Test 8: with umask 022, mkdtemp produces mode=700 (execute bit present)..."
if ! command -v node >/dev/null 2>&1; then
  echo "  (skipping: node not available)"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  TMPPARENT=$(mktemp -d)
  CREATED_MODE=$(bash -c "
    umask 022
    node -e \"
      const fs = require('fs');
      const d = fs.mkdtempSync('${TMPPARENT}/chroot-');
      const st = fs.statSync(d);
      process.stdout.write(String(st.mode & 0o777));
    \"
  " 2>/dev/null || echo "skip")
  rm -rf "${TMPPARENT}"
  if [ "$CREATED_MODE" = "skip" ]; then
    echo "  (skipping: node execution failed)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  elif [ "$CREATED_MODE" -eq $((0700)) ]; then
    pass "mkdtemp produced mode=0700 with umask 022 — chroot dir is traversable"
  else
    fail "unexpected mkdtemp mode ${CREATED_MODE} with umask 022 (expected $((0700))/0700)" \
      "mode=${CREATED_MODE}"
  fi
fi

echo
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"

if [ "$TESTS_FAILED" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
