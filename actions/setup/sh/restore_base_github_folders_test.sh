#!/usr/bin/env bash
# Tests for restore_base_github_folders.sh
# Run: bash restore_base_github_folders_test.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESTORE_SCRIPT="${SCRIPT_DIR}/restore_base_github_folders.sh"

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

cleanup() {
  rm -rf "${TEST_WORKSPACE:-}" "/tmp/gh-aw/base"
}
trap cleanup EXIT

echo "Testing restore_base_github_folders.sh..."
echo ""

# ── Test 1: Restores .github and .agents, removes .mcp.json ──────────────────
echo "Test 1: Snapshot present → restores .github/.agents and removes .mcp.json"
TEST_WORKSPACE=$(mktemp -d)

# Simulate snapshot from activation job
mkdir -p /tmp/gh-aw/base/.github/skills
echo "trusted skill" >/tmp/gh-aw/base/.github/skills/SKILL.md
mkdir -p /tmp/gh-aw/base/.agents
echo "trusted agent" >/tmp/gh-aw/base/.agents/agent.md

# Simulate PR-branch workspace (untrusted content)
mkdir -p "${TEST_WORKSPACE}/.github/skills"
echo "evil skill" >"${TEST_WORKSPACE}/.github/skills/SKILL.md"
mkdir -p "${TEST_WORKSPACE}/.agents"
echo "evil agent" >"${TEST_WORKSPACE}/.agents/agent.md"
echo '{"mcpServers":{}}' >"${TEST_WORKSPACE}/.mcp.json"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${RESTORE_SCRIPT}" >/dev/null 2>&1

assert ".github restored" "[ -d '${TEST_WORKSPACE}/.github' ]"
assert ".github/skills/SKILL.md content is trusted" "grep -q 'trusted skill' '${TEST_WORKSPACE}/.github/skills/SKILL.md'"
assert ".agents restored" "[ -d '${TEST_WORKSPACE}/.agents' ]"
assert ".agents/agent.md content is trusted" "grep -q 'trusted agent' '${TEST_WORKSPACE}/.agents/agent.md'"
assert ".mcp.json removed" "[ ! -f '${TEST_WORKSPACE}/.mcp.json' ]"
rm -rf "${TEST_WORKSPACE}" /tmp/gh-aw/base
echo ""

# ── Test 2: No snapshot → workspace unchanged, exits 0 ───────────────────────
echo "Test 2: No snapshot in /tmp/gh-aw/base → workspace untouched, exits 0"
TEST_WORKSPACE=$(mktemp -d)
mkdir -p "${TEST_WORKSPACE}/.github"
echo "pr content" >"${TEST_WORKSPACE}/.github/README.md"
rm -rf /tmp/gh-aw/base

OUTPUT=$(GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${RESTORE_SCRIPT}" 2>&1)
EXIT_CODE=$?

assert "exits 0 when no snapshot" "[ ${EXIT_CODE} -eq 0 ]"
assert "workspace .github not removed" "[ -f '${TEST_WORKSPACE}/.github/README.md' ]"
rm -rf "${TEST_WORKSPACE}"
echo ""

# ── Test 3: Only .github snapshot → only .github restored ────────────────────
echo "Test 3: Only .github snapshot → only .github replaced"
TEST_WORKSPACE=$(mktemp -d)

mkdir -p /tmp/gh-aw/base/.github
echo "trusted" >/tmp/gh-aw/base/.github/trusted.md
mkdir -p "${TEST_WORKSPACE}/.github"
echo "evil" >"${TEST_WORKSPACE}/.github/evil.md"
mkdir -p "${TEST_WORKSPACE}/.agents"
echo "pr agent" >"${TEST_WORKSPACE}/.agents/agent.md"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${RESTORE_SCRIPT}" >/dev/null 2>&1

assert ".github replaced" "[ -f '${TEST_WORKSPACE}/.github/trusted.md' ]"
assert "evil file removed from .github" "[ ! -f '${TEST_WORKSPACE}/.github/evil.md' ]"
assert ".agents untouched when no snapshot" "[ -f '${TEST_WORKSPACE}/.agents/agent.md' ]"
rm -rf "${TEST_WORKSPACE}" /tmp/gh-aw/base
echo ""

# ── Test 4: .mcp.json absent → exits 0 without error ─────────────────────────
echo "Test 4: No .mcp.json in workspace → exits 0"
TEST_WORKSPACE=$(mktemp -d)
rm -rf /tmp/gh-aw/base

OUTPUT=$(GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${RESTORE_SCRIPT}" 2>&1)
EXIT_CODE=$?

assert "exits 0 when .mcp.json absent" "[ ${EXIT_CODE} -eq 0 ]"
rm -rf "${TEST_WORKSPACE}"
echo ""

# ── Summary ──────────────────────────────────────────────────────────────────
echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
