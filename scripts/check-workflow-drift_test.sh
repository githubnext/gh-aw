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

# Test 1: matching lock file exits 0.
echo "Test 1: matching lock file exits 0..."
TEST_REPO="$TMP_ROOT/stable"
mkdir -p "$TEST_REPO"
create_fixture_repo "$TEST_REPO"
create_fake_binary "$TEST_REPO/fake-gh-aw"
if (cd "$TEST_REPO" && FAKE_COMPILE_MODE=stable bash "$DRIFT_SCRIPT" "$TEST_REPO/fake-gh-aw" >/tmp/check-workflow-drift-test1.txt 2>&1); then
  pass "matching lock file exits 0"
else
  fail "matching lock file should exit 0" "$(cat /tmp/check-workflow-drift-test1.txt)"
fi

# Test 2: drift is reported and the original file is restored afterwards.
echo "Test 2: drift is reported without leaving the repo dirty..."
TEST_REPO="$TMP_ROOT/mutate"
mkdir -p "$TEST_REPO"
create_fixture_repo "$TEST_REPO"
create_fake_binary "$TEST_REPO/fake-gh-aw"
if (cd "$TEST_REPO" && FAKE_COMPILE_MODE=mutate bash "$DRIFT_SCRIPT" "$TEST_REPO/fake-gh-aw" >/tmp/check-workflow-drift-test2.txt 2>&1); then
  fail "drift should exit 1" "$(cat /tmp/check-workflow-drift-test2.txt)"
elif grep -q ".github/workflows/example.lock.yml" /tmp/check-workflow-drift-test2.txt \
  && grep -q "report_progress" /tmp/check-workflow-drift-test2.txt \
  && grep -q "^lock: original$" "$TEST_REPO/.github/workflows/example.lock.yml"; then
  pass "drift is reported and the original file is restored"
else
  fail "drift output or restoration was incorrect" "$(cat /tmp/check-workflow-drift-test2.txt; echo; cat "$TEST_REPO/.github/workflows/example.lock.yml")"
fi

# Test 3: missing binary gets a targeted error.
echo "Test 3: missing binary path gets a targeted error..."
TEST_REPO="$TMP_ROOT/missing-binary"
mkdir -p "$TEST_REPO"
create_fixture_repo "$TEST_REPO"
if (cd "$TEST_REPO" && bash "$DRIFT_SCRIPT" "$TEST_REPO/does-not-exist" >/tmp/check-workflow-drift-test3.txt 2>&1); then
  fail "missing binary should exit 1" "$(cat /tmp/check-workflow-drift-test3.txt)"
elif grep -q "binary not found" /tmp/check-workflow-drift-test3.txt; then
  pass "missing binary reports a targeted error"
else
  fail "missing binary error message was incorrect" "$(cat /tmp/check-workflow-drift-test3.txt)"
fi

echo
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"

if [ "$TESTS_FAILED" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
