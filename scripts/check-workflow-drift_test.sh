#!/bin/bash
set +o histexpand

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DRIFT_SCRIPT="$SCRIPT_DIR/check-workflow-drift.sh"

TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo "PASS: $1"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo "FAIL: $1"; echo "  $2"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

create_fixture_repo() {
  local repo_dir="$1"

  mkdir -p "$repo_dir/.github/workflows"
  cat > "$repo_dir/.github/workflows/example.md" <<'EOF'
# Example workflow
EOF
  cat > "$repo_dir/.github/workflows/example.lock.yml" <<'EOF'
lock: original
EOF
}

create_fake_binary() {
  local path="$1"
  cat > "$path" <<'EOF'
#!/bin/bash
set -euo pipefail

if [ "${1:-}" != "compile" ]; then
  echo "unexpected command: ${1:-}" >&2
  exit 1
fi

case "${FAKE_COMPILE_MODE:-stable}" in
  stable)
    ;;
  mutate)
    cat > .github/workflows/example.lock.yml <<'OUT'
lock: mutated
OUT
    ;;
  fail)
    echo "compile failed" >&2
    exit 1
    ;;
  *)
    echo "unknown FAKE_COMPILE_MODE: ${FAKE_COMPILE_MODE:-}" >&2
    exit 1
    ;;
esac
EOF
  chmod +x "$path"
}

echo "Running check-workflow-drift.sh tests..."
echo

TMP_ROOT=$(mktemp -d)
trap 'rm -rf "$TMP_ROOT"' EXIT
TEST1_OUTPUT="$TMP_ROOT/test1-output.txt"
TEST2_OUTPUT="$TMP_ROOT/test2-output.txt"
TEST3_OUTPUT="$TMP_ROOT/test3-output.txt"

# Test 1: matching lock file exits 0.
echo "Test 1: matching lock file exits 0..."
TEST_REPO="$TMP_ROOT/stable"
mkdir -p "$TEST_REPO"
create_fixture_repo "$TEST_REPO"
create_fake_binary "$TEST_REPO/fake-gh-aw"
if (cd "$TEST_REPO" && FAKE_COMPILE_MODE=stable bash "$DRIFT_SCRIPT" "$TEST_REPO/fake-gh-aw" >"$TEST1_OUTPUT" 2>&1); then
  pass "matching lock file exits 0"
else
  fail "matching lock file should exit 0" "$(cat "$TEST1_OUTPUT")"
fi

# Test 2: drift is reported and the original file is restored afterwards.
echo "Test 2: drift is reported without leaving the repo dirty..."
TEST_REPO="$TMP_ROOT/mutate"
mkdir -p "$TEST_REPO"
create_fixture_repo "$TEST_REPO"
create_fake_binary "$TEST_REPO/fake-gh-aw"
if (cd "$TEST_REPO" && FAKE_COMPILE_MODE=mutate bash "$DRIFT_SCRIPT" "$TEST_REPO/fake-gh-aw" >"$TEST2_OUTPUT" 2>&1); then
  fail "drift should exit 1" "$(cat "$TEST2_OUTPUT")"
elif grep -q ".github/workflows/example.lock.yml" "$TEST2_OUTPUT" \
  && grep -q "report_progress" "$TEST2_OUTPUT" \
  && grep -q "^lock: original$" "$TEST_REPO/.github/workflows/example.lock.yml"; then
  pass "drift is reported and the original file is restored"
else
  fail "drift output or restoration was incorrect" "$(cat "$TEST2_OUTPUT"; echo; cat "$TEST_REPO/.github/workflows/example.lock.yml")"
fi

# Test 3: missing binary gets a targeted error.
echo "Test 3: missing binary path gets a targeted error..."
TEST_REPO="$TMP_ROOT/missing-binary"
mkdir -p "$TEST_REPO"
create_fixture_repo "$TEST_REPO"
if (cd "$TEST_REPO" && bash "$DRIFT_SCRIPT" "$TEST_REPO/does-not-exist" >"$TEST3_OUTPUT" 2>&1); then
  fail "missing binary should exit 1" "$(cat "$TEST3_OUTPUT")"
elif grep -q "binary not found" "$TEST3_OUTPUT"; then
  pass "missing binary reports a targeted error"
else
  fail "missing binary error message was incorrect" "$(cat "$TEST3_OUTPUT")"
fi

echo
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"

if [ "$TESTS_FAILED" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
