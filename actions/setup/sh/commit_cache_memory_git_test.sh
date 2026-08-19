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
  local condition="$2"
  if eval "${condition}" 2>/dev/null; then
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
assert "script passes bash -n" "bash -n '${SCRIPT}'"
echo ""

echo "Test 2: Agent-controlled hooks and filters cannot execute"
D="${WORKSPACE}/test2"
SENTINEL_HOOK="${WORKSPACE}/hook-executed"
SENTINEL_FILTER="${WORKSPACE}/filter-executed"
mkdir -p "${D}/evil-hooks"
git -C "${D}" init -q
git -C "${D}" config user.email "test@example.com"
git -C "${D}" config user.name "Test"
touch "${D}/initial"
git -C "${D}" add initial
git -C "${D}" commit -qm initial
git -C "${D}" config core.hooksPath "${D}/evil-hooks"
git -C "${D}" config filter.evil.clean "touch ${SENTINEL_FILTER}"
cat > "${D}/evil-hooks/pre-commit" <<EOF
#!/usr/bin/env bash
touch "${SENTINEL_HOOK}"
EOF
chmod +x "${D}/evil-hooks/pre-commit"
printf 'content\n' > "${D}/agent-data"
printf 'agent-data filter=evil\n' > "${D}/.gitattributes"
run_script "${D}" >/dev/null
assert "pre-commit hook was not executed" "[ ! -e '${SENTINEL_HOOK}' ]"
assert "clean filter was not executed" "[ ! -e '${SENTINEL_FILTER}' ]"
assert "hooks path hardened" "[ \"\$(git -C '${D}' config --default '' core.hooksPath)\" = '/dev/null' ]"
assert "filter configuration removed" "! git -C '${D}' config --local --name-only --get-regexp '^filter\\.' >/dev/null 2>&1"
assert "agent changes committed" "git -C '${D}' log -1 --format=%s | grep -qx 'run-test-run'"
echo ""

echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
