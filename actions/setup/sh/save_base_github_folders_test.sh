#!/usr/bin/env bash
# Tests for save_base_github_folders.sh
# Run: bash save_base_github_folders_test.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SAVE_SCRIPT="${SCRIPT_DIR}/save_base_github_folders.sh"

TESTS_PASSED=0
TESTS_FAILED=0

assert() {
  local name="$1"
  local condition="$2"
  if eval "${condition}"; then
    echo "✓ ${name}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo "✗ ${name}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

# Use a temporary directory to isolate /tmp/gh-aw/base writes
REAL_DEST="/tmp/gh-aw/base"

cleanup() {
  rm -rf "${TEST_WORKSPACE:-}" "${REAL_DEST}"
}
trap cleanup EXIT

echo "Testing save_base_github_folders.sh..."
echo ""

# ── Test 1: Both .github and .agents present ─────────────────────────────────
echo "Test 1: Both .github and .agents present → both copied to /tmp/gh-aw/base"
TEST_WORKSPACE=$(mktemp -d)
mkdir -p "${TEST_WORKSPACE}/.github/skills"
echo "skill content" >"${TEST_WORKSPACE}/.github/skills/SKILL.md"
mkdir -p "${TEST_WORKSPACE}/.agents"
echo "agent content" >"${TEST_WORKSPACE}/.agents/agent.md"
rm -rf "${REAL_DEST}"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${SAVE_SCRIPT}" >/dev/null 2>&1

assert "saves .github to /tmp/gh-aw/base" "[ -d '${REAL_DEST}/.github' ]"
assert "saves SKILL.md" "[ -f '${REAL_DEST}/.github/skills/SKILL.md' ]"
assert "saves .agents to /tmp/gh-aw/base" "[ -d '${REAL_DEST}/.agents' ]"
assert "saves agent.md" "[ -f '${REAL_DEST}/.agents/agent.md' ]"
rm -rf "${TEST_WORKSPACE}" "${REAL_DEST}"
echo ""

# ── Test 2: Only .github present ─────────────────────────────────────────────
echo "Test 2: Only .github present → only .github is copied"
TEST_WORKSPACE=$(mktemp -d)
mkdir -p "${TEST_WORKSPACE}/.github/instructions"
echo "instructions" >"${TEST_WORKSPACE}/.github/instructions/README.md"
rm -rf "${REAL_DEST}"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${SAVE_SCRIPT}" >/dev/null 2>&1

assert "saves .github" "[ -d '${REAL_DEST}/.github' ]"
assert ".agents not created when absent" "[ ! -d '${REAL_DEST}/.agents' ]"
rm -rf "${TEST_WORKSPACE}" "${REAL_DEST}"
echo ""

# ── Test 3: Neither folder present → dest not created ────────────────────────
echo "Test 3: Neither .github nor .agents → /tmp/gh-aw/base not created"
TEST_WORKSPACE=$(mktemp -d)
rm -rf "${REAL_DEST}"

OUTPUT=$(GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${SAVE_SCRIPT}" 2>&1)
EXIT_CODE=$?

assert "exits 0 when nothing to save" "[ ${EXIT_CODE} -eq 0 ]"
assert "/tmp/gh-aw/base not created" "[ ! -d '${REAL_DEST}' ]"
rm -rf "${TEST_WORKSPACE}"
echo ""

# ── Test 4: Existing /tmp/gh-aw/base is reused ───────────────────────────────
echo "Test 4: Existing /tmp/gh-aw/base is reused without error"
TEST_WORKSPACE=$(mktemp -d)
mkdir -p "${TEST_WORKSPACE}/.github"
mkdir -p "${REAL_DEST}"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${SAVE_SCRIPT}" >/dev/null 2>&1
EXIT_CODE=$?

assert "exits 0 when base dir already exists" "[ ${EXIT_CODE} -eq 0 ]"
assert ".github saved into existing base dir" "[ -d '${REAL_DEST}/.github' ]"
rm -rf "${TEST_WORKSPACE}" "${REAL_DEST}"
echo ""

# ── Summary ──────────────────────────────────────────────────────────────────
echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
