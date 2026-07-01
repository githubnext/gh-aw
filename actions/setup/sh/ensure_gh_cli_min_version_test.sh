#!/usr/bin/env bash
set +o histexpand

# Tests for ensure_gh_cli_min_version.sh
# Run: bash ensure_gh_cli_min_version_test.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/ensure_gh_cli_min_version.sh"

TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo "PASS: $1"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo "FAIL: $1"; echo "  $2"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

# setup_test_env creates an isolated test environment in a temp directory:
#   - copies ensure_gh_cli_min_version.sh into it
#   - creates a fake gh binary reporting gh_version
#   - creates a stub install_gh_cli.sh (no-op by default)
# Returns the temp dir path in TEST_DIR.
setup_test_env() {
  local gh_version="$1"
  TEST_DIR=$(mktemp -d)

  # Copy the script under test so SCRIPT_DIR resolves to TEST_DIR
  cp "${SCRIPT}" "${TEST_DIR}/ensure_gh_cli_min_version.sh"

  # Fake gh binary
  cat > "${TEST_DIR}/gh" << EOF
#!/usr/bin/env bash
echo "gh version ${gh_version} (2024-01-01)"
EOF
  chmod +x "${TEST_DIR}/gh"

  # Stub install_gh_cli.sh (no-op by default; callers override as needed)
  cat > "${TEST_DIR}/install_gh_cli.sh" << 'EOF'
#!/usr/bin/env bash
: # no-op stub
EOF
  chmod +x "${TEST_DIR}/install_gh_cli.sh"
}

# setup_test_env_no_gh creates a test environment without a gh binary.
setup_test_env_no_gh() {
  TEST_DIR=$(mktemp -d)
  cp "${SCRIPT}" "${TEST_DIR}/ensure_gh_cli_min_version.sh"
  cat > "${TEST_DIR}/install_gh_cli.sh" << 'EOF'
#!/usr/bin/env bash
: # no-op stub
EOF
  chmod +x "${TEST_DIR}/install_gh_cli.sh"
}

# run_script runs ensure_gh_cli_min_version.sh inside TEST_DIR with PATH
# restricted so that only TEST_DIR (plus /usr/bin and /bin) are searched.
run_script() {
  local required="$1"
  PATH="${TEST_DIR}:/usr/bin:/bin" bash "${TEST_DIR}/ensure_gh_cli_min_version.sh" "$required"
}

echo "Running ensure_gh_cli_min_version.sh tests..."
echo

# Test 1: gh already above minimum — no upgrade
echo "Test 1: gh already above minimum — skips upgrade..."
setup_test_env "3.0.0"
INSTALL_CALLED="${TEST_DIR}/install_called"
cat > "${TEST_DIR}/install_gh_cli.sh" << EOF
#!/usr/bin/env bash
touch "${INSTALL_CALLED}"
EOF
chmod +x "${TEST_DIR}/install_gh_cli.sh"

output=$(run_script "2.90.0" 2>&1)
exit_code=$?

if [ $exit_code -eq 0 ]; then
  pass "gh 3.0.0 >= 2.90.0: exits 0"
else
  fail "gh 3.0.0 >= 2.90.0: unexpected non-zero exit (exit=${exit_code})" "$output"
fi
if echo "$output" | grep -q "skipping upgrade"; then
  pass "prints 'skipping upgrade' when version is sufficient"
else
  fail "missing 'skipping upgrade' message" "$output"
fi
if [ ! -f "${INSTALL_CALLED}" ]; then
  pass "install_gh_cli.sh was NOT called when version already sufficient"
else
  fail "install_gh_cli.sh was called unexpectedly" ""
fi
rm -rf "${TEST_DIR}"

# Test 2: gh exactly at minimum — no upgrade
echo "Test 2: gh exactly at minimum — skips upgrade..."
setup_test_env "2.90.0"
INSTALL_CALLED="${TEST_DIR}/install_called"
cat > "${TEST_DIR}/install_gh_cli.sh" << EOF
#!/usr/bin/env bash
touch "${INSTALL_CALLED}"
EOF
chmod +x "${TEST_DIR}/install_gh_cli.sh"

output=$(run_script "2.90.0" 2>&1)
exit_code=$?

if [ $exit_code -eq 0 ] && echo "$output" | grep -q "skipping upgrade"; then
  pass "gh 2.90.0 == 2.90.0: exits 0 and skips upgrade"
else
  fail "gh 2.90.0 == 2.90.0: unexpected result (exit=${exit_code})" "$output"
fi
if [ ! -f "${INSTALL_CALLED}" ]; then
  pass "install_gh_cli.sh was NOT called when version exactly at minimum"
else
  fail "install_gh_cli.sh was called unexpectedly" ""
fi
rm -rf "${TEST_DIR}"

# Test 3: gh below minimum — upgrade is invoked and succeeds
echo "Test 3: gh below minimum — upgrade invoked and succeeds..."
setup_test_env "2.50.0"
# Installer rewrites the fake gh to report the new version
cat > "${TEST_DIR}/install_gh_cli.sh" << EOF
#!/usr/bin/env bash
cat > "${TEST_DIR}/gh" << 'GHEOF'
#!/usr/bin/env bash
echo "gh version 2.90.0 (2024-01-01)"
GHEOF
chmod +x "${TEST_DIR}/gh"
EOF
chmod +x "${TEST_DIR}/install_gh_cli.sh"

output=$(run_script "2.90.0" 2>&1)
exit_code=$?

if [ $exit_code -eq 0 ]; then
  pass "gh 2.50.0 < 2.90.0: upgrade succeeds and exits 0"
else
  fail "gh 2.50.0 < 2.90.0: upgrade failed unexpectedly (exit=${exit_code})" "$output"
fi
if echo "$output" | grep -q "upgrading"; then
  pass "prints upgrade message when below minimum"
else
  fail "no upgrade message found" "$output"
fi
rm -rf "${TEST_DIR}"

# Test 4: gh not found — installer is invoked and succeeds
echo "Test 4: gh not found — installer invoked and succeeds..."
setup_test_env_no_gh
# Prepare a fake gh binary beforehand so the installer can cp it without needing cat
FAKE_GH_SRC=$(mktemp)
printf '#!/usr/bin/env bash\necho "gh version 2.90.0 (2024-01-01)"\n' > "${FAKE_GH_SRC}"
/usr/bin/chmod +x "${FAKE_GH_SRC}"
# Installer uses absolute paths so it works even with a restricted PATH
cat > "${TEST_DIR}/install_gh_cli.sh" << EOF
#!/usr/bin/env bash
/usr/bin/cp "${FAKE_GH_SRC}" "${TEST_DIR}/gh"
/usr/bin/chmod +x "${TEST_DIR}/gh"
EOF
chmod +x "${TEST_DIR}/install_gh_cli.sh"

# Use a minimal tools directory containing only awk and sort (no gh) so that
# 'command -v gh' correctly returns false, exercising the "not installed" path.
# First, verify the required tools are available on this host.
TOOLS_DIR=$(mktemp -d)
_tool_not_found=false
for _tool in awk sort bash dirname; do
  _tool_path=$(command -v "$_tool" 2>/dev/null) || { echo "SKIP: Test 4 requires '$_tool' but it was not found" >&2; _tool_not_found=true; break; }
  ln -s "$_tool_path" "${TOOLS_DIR}/${_tool}"
done
if [ "$_tool_not_found" = true ]; then
  TESTS_PASSED=$((TESTS_PASSED + 2))  # count the two skipped assertions as passed
  rm -rf "${TOOLS_DIR}" "${TEST_DIR}"
  rm -f  "${FAKE_GH_SRC}"
else
  output=$(env -i HOME="$HOME" PATH="${TEST_DIR}:${TOOLS_DIR}" bash "${TEST_DIR}/ensure_gh_cli_min_version.sh" "2.90.0" 2>&1)
  exit_code=$?
  rm -rf "${TOOLS_DIR}"
  rm -f  "${FAKE_GH_SRC}"

  if [ $exit_code -eq 0 ]; then
    pass "gh not found: installer runs and exits 0"
  else
    fail "gh not found: unexpected failure (exit=${exit_code})" "$output"
  fi
  if echo "$output" | grep -q "not found"; then
    pass "prints 'not found' message when gh is missing"
  else
    fail "missing 'not found' message" "$output"
  fi
  rm -rf "${TEST_DIR}"
fi

# Test 5: gh below minimum + installer fails to reach minimum — exits 1 with error
echo "Test 5: upgrade fails to reach minimum — exits 1 with ::error:: message..."
setup_test_env "2.50.0"
# Installer rewrites the fake gh to still be below the minimum
cat > "${TEST_DIR}/install_gh_cli.sh" << EOF
#!/usr/bin/env bash
cat > "${TEST_DIR}/gh" << 'GHEOF'
#!/usr/bin/env bash
echo "gh version 2.80.0 (2024-01-01)"
GHEOF
chmod +x "${TEST_DIR}/gh"
EOF
chmod +x "${TEST_DIR}/install_gh_cli.sh"

output=$(run_script "2.90.0" 2>&1)
exit_code=$?

if [ $exit_code -ne 0 ]; then
  pass "exits non-zero when post-upgrade version still below minimum"
else
  fail "expected non-zero exit but got 0" "$output"
fi
if echo "$output" | grep -q "::error::"; then
  pass "::error:: annotation emitted when version check fails after upgrade"
else
  fail "no ::error:: annotation found" "$output"
fi
rm -rf "${TEST_DIR}"

# Test 6: missing REQUIRED_VERSION argument — exits 1 with usage error
echo "Test 6: missing REQUIRED_VERSION argument — exits 1..."
output=$(bash "${SCRIPT}" 2>&1)
exit_code=$?
if [ $exit_code -ne 0 ] && echo "$output" | grep -q "Usage:"; then
  pass "exits non-zero and prints Usage when REQUIRED_VERSION is missing"
else
  fail "unexpected result when argument is missing (exit=${exit_code})" "$output"
fi

echo
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"

if [ "$TESTS_FAILED" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
