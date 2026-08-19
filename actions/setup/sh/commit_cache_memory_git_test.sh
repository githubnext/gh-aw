#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/commit_cache_memory_git.sh"

TESTS_PASSED=0
TESTS_FAILED=0
WORKSPACE="$(mktemp -d)"

cleanup() {
  rm -rf "${WORKSPACE}"
}
trap cleanup EXIT

assert() {
  local name="$1"
  shift
  if "$@" 2>/dev/null; then
    echo "  ✓ ${name}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo "  ✗ ${name}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

run_script() {
  GH_AW_CACHE_DIR="$1" GITHUB_RUN_ID="test-run" bash "${SCRIPT}" 2>&1
}

echo "Testing commit_cache_memory_git.sh"
echo ""

echo "Test 1: Script syntax is valid"
assert "script passes bash -n" bash -n "${SCRIPT}"
echo ""

echo "Test 2: Agent-controlled hooks and filters cannot execute"
D="${WORKSPACE}/test2"
SENTINEL_HOOK="${WORKSPACE}/hook-executed"
SENTINEL_FILTER="${WORKSPACE}/filter-executed"
SENTINEL_SIGNING="${WORKSPACE}/signing-executed"
mkdir -p "${D}/evil-hooks"
git -C "${D}" init -q
git -C "${D}" config user.email "test@example.com"
git -C "${D}" config user.name "Test"
touch "${D}/initial"
git -C "${D}" add initial
git -C "${D}" commit -qm initial
git -C "${D}" config extensions.worktreeConfig true
git -C "${D}" config --worktree core.hooksPath "${D}/evil-hooks"
git -C "${D}" config --worktree filter.evil.clean "touch ${SENTINEL_FILTER}"
git -C "${D}" config core.worktree "${WORKSPACE}"
git -C "${D}" config core.gitProxy "touch ${WORKSPACE}/proxy-executed"
git -C "${D}" config commit.gpgSign true
git -C "${D}" config gpg.program "${D}/evil-gpg"
cat > "${D}/evil-hooks/post-commit" <<EOF
#!/usr/bin/env bash
touch "${SENTINEL_HOOK}"
EOF
cat > "${D}/evil-gpg" <<EOF
#!/usr/bin/env bash
touch "${SENTINEL_SIGNING}"
exit 1
EOF
chmod +x "${D}/evil-hooks/post-commit" "${D}/evil-gpg"
printf 'content\n' > "${D}/agent-data"
printf 'agent-data filter=evil\n' > "${D}/.gitattributes"
run_script "${D}" >/dev/null
assert "post-commit hook was not executed" test ! -e "${SENTINEL_HOOK}"
assert "clean filter was not executed" test ! -e "${SENTINEL_FILTER}"
assert "signing program was not executed" test ! -e "${SENTINEL_SIGNING}"
assert "hooks path hardened" test "$(git -C "${D}" config --default '' core.hooksPath)" = "/dev/null"
assert "worktree override removed" test -z "$(git -C "${D}" config --local --get core.worktree || true)"
assert "git proxy removed" test -z "$(git -C "${D}" config --local --get core.gitProxy || true)"
assert "worktree config removed" test ! -e "${D}/.git/config.worktree"
assert "worktree config extension removed" test -z "$(git -C "${D}" config --local --get extensions.worktreeConfig || true)"
assert "agent changes committed" test "$(git -C "${D}" log -1 --format=%s)" = "run-test-run"
echo ""

echo "Test 3: Symlinked git metadata is rejected without losing history"
D="${WORKSPACE}/test3"
REAL_GIT="${WORKSPACE}/test3-git"
mkdir -p "${D}"
git -C "${D}" init -q
git -C "${D}" config user.email "test@example.com"
git -C "${D}" config user.name "Test"
touch "${D}/initial"
git -C "${D}" add initial
git -C "${D}" commit -qm initial
INITIAL_COMMIT="$(git -C "${D}" rev-parse HEAD)"
mv "${D}/.git" "${REAL_GIT}"
ln -s "${REAL_GIT}" "${D}/.git"
if run_script "${D}" >/dev/null; then
  SYMLINK_REJECTED=false
else
  SYMLINK_REJECTED=true
fi
assert "symlinked metadata was rejected" test "${SYMLINK_REJECTED}" = true
assert "symlinked metadata was not replaced" test -L "${D}/.git"
assert "existing history was preserved" test "$(git -C "${D}" rev-parse HEAD)" = "${INITIAL_COMMIT}"
echo ""

echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
