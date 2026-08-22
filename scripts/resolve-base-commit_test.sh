#!/bin/bash
set +o histexpand

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOLVE_SCRIPT="$SCRIPT_DIR/resolve-base-commit.sh"

TESTS_PASSED=0
TESTS_FAILED=0

pass() {
    echo "PASS: $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}
fail() {
    echo "FAIL: $1"
    echo "  $2"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

git_init() {
    local dir="$1"
    git -C "$dir" init -q -b main
    git -C "$dir" config user.email "test@test.com"
    git -C "$dir" config user.name "Test"
}

commit_file() {
    local dir="$1" name="$2"
    printf '%s\n' "$name" > "$dir/$name"
    git -C "$dir" add "$name"
    git -C "$dir" commit -q -m "add $name"
}

echo "Running resolve-base-commit.sh tests..."
echo

TMP_ROOT=$(mktemp -d)
trap 'rm -rf "$TMP_ROOT"' EXIT

# ---------------------------------------------------------------------------
# Fixture: an "upstream" repository with a few commits, used as the remote.
# ---------------------------------------------------------------------------
UPSTREAM="$TMP_ROOT/upstream"
mkdir -p "$UPSTREAM"
git_init "$UPSTREAM"
commit_file "$UPSTREAM" "one.txt"
commit_file "$UPSTREAM" "two.txt"
commit_file "$UPSTREAM" "three.txt"
UPSTREAM_HEAD=$(git -C "$UPSTREAM" rev-parse HEAD)

# ---------------------------------------------------------------------------
# Test 1: full clone with origin/main present resolves the true merge-base.
# ---------------------------------------------------------------------------
FULL="$TMP_ROOT/full"
git clone -q "$UPSTREAM" "$FULL"
git -C "$FULL" config user.email "test@test.com"
git -C "$FULL" config user.name "Test"
commit_file "$FULL" "feature.txt"
if output=$(cd "$FULL" && bash "$RESOLVE_SCRIPT" --base-ref origin/main 2>/dev/null); then
    if [ "$output" = "$UPSTREAM_HEAD" ]; then
        pass "full clone resolves merge-base with origin/main"
    else
        fail "full clone resolves merge-base with origin/main" "expected $UPSTREAM_HEAD, got $output"
    fi
else
    fail "full clone resolves merge-base with origin/main" "script exited non-zero"
fi

# ---------------------------------------------------------------------------
# Test 2: shallow clone (fetch-depth: 1 equivalent) still resolves a base
# commit by fetching/deepening the base branch.
# ---------------------------------------------------------------------------
SHALLOW="$TMP_ROOT/shallow"
git clone -q --depth=1 "file://$UPSTREAM" "$SHALLOW"
git -C "$SHALLOW" config user.email "test@test.com"
git -C "$SHALLOW" config user.name "Test"
# Emulate the CI checkout: no remote-tracking branch for main is usable yet.
git -C "$SHALLOW" update-ref -d refs/remotes/origin/main 2>/dev/null || true
commit_file "$SHALLOW" "feature.txt"
if output=$(cd "$SHALLOW" && bash "$RESOLVE_SCRIPT" --base-ref origin/main 2>/dev/null); then
    if git -C "$SHALLOW" rev-parse --verify --quiet "${output}^{commit}" >/dev/null; then
        pass "shallow clone resolves a base commit"
    else
        fail "shallow clone resolves a base commit" "output is not a commit: $output"
    fi
else
    fail "shallow clone resolves a base commit" "script exited non-zero"
fi

# ---------------------------------------------------------------------------
# Test 3: --no-fetch with a missing base ref falls back to HEAD~1.
# ---------------------------------------------------------------------------
LOCAL="$TMP_ROOT/local"
mkdir -p "$LOCAL"
git_init "$LOCAL"
commit_file "$LOCAL" "one.txt"
commit_file "$LOCAL" "two.txt"
EXPECTED_PARENT=$(git -C "$LOCAL" rev-parse HEAD~1)
if output=$(cd "$LOCAL" && bash "$RESOLVE_SCRIPT" --base-ref origin/main --no-fetch 2>/dev/null); then
    if [ "$output" = "$EXPECTED_PARENT" ]; then
        pass "missing base ref falls back to HEAD~1"
    else
        fail "missing base ref falls back to HEAD~1" "expected $EXPECTED_PARENT, got $output"
    fi
else
    fail "missing base ref falls back to HEAD~1" "script exited non-zero"
fi

# ---------------------------------------------------------------------------
# Test 4: single-commit repository without a base ref fails with exit code 1.
# ---------------------------------------------------------------------------
SINGLE="$TMP_ROOT/single"
mkdir -p "$SINGLE"
git_init "$SINGLE"
commit_file "$SINGLE" "one.txt"
if (cd "$SINGLE" && bash "$RESOLVE_SCRIPT" --base-ref origin/main --no-fetch >/dev/null 2>&1); then
    fail "unresolvable base ref exits non-zero" "script exited 0"
else
    pass "unresolvable base ref exits non-zero"
fi

# ---------------------------------------------------------------------------
# Test 5: shallow clone whose base branch has advanced upstream requires
# fetching the base branch to find the shared ancestor.
# ---------------------------------------------------------------------------
DIVERGED="$TMP_ROOT/diverged"
git clone -q --depth=1 "file://$UPSTREAM" "$DIVERGED"
git -C "$DIVERGED" config user.email "test@test.com"
git -C "$DIVERGED" config user.name "Test"
git -C "$DIVERGED" update-ref -d refs/remotes/origin/main 2>/dev/null || true
commit_file "$DIVERGED" "feature.txt"
commit_file "$UPSTREAM" "four.txt"
if output=$(cd "$DIVERGED" && bash "$RESOLVE_SCRIPT" --base-ref origin/main 2>/dev/null); then
    if [ "$output" = "$UPSTREAM_HEAD" ]; then
        pass "shallow clone with advanced base branch finds shared ancestor"
    else
        fail "shallow clone with advanced base branch finds shared ancestor" "expected $UPSTREAM_HEAD, got $output"
    fi
else
    fail "shallow clone with advanced base branch finds shared ancestor" "script exited non-zero"
fi

echo
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"
[ "$TESTS_FAILED" -eq 0 ]
