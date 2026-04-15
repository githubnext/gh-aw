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

# ── Test 1: Core folders + root files restored, .mcp.json removed ────────────
echo "Test 1: Snapshot present → restores folders, root files, removes .mcp.json"
TEST_WORKSPACE=$(mktemp -d)

# Base branch snapshot
mkdir -p /tmp/gh-aw/base/.github/skills
echo "trusted skill" >/tmp/gh-aw/base/.github/skills/SKILL.md
mkdir -p /tmp/gh-aw/base/.agents
echo "trusted agent" >/tmp/gh-aw/base/.agents/agent.md
echo "trusted agents" >/tmp/gh-aw/base/AGENTS.md
echo "trusted claude" >/tmp/gh-aw/base/CLAUDE.md

# PR-branch workspace (untrusted content)
mkdir -p "${TEST_WORKSPACE}/.github/skills"
echo "evil skill" >"${TEST_WORKSPACE}/.github/skills/SKILL.md"
mkdir -p "${TEST_WORKSPACE}/.agents"
echo "evil agent" >"${TEST_WORKSPACE}/.agents/agent.md"
echo "evil agents" >"${TEST_WORKSPACE}/AGENTS.md"
echo "evil claude" >"${TEST_WORKSPACE}/CLAUDE.md"
echo '{"mcpServers":{}}' >"${TEST_WORKSPACE}/.mcp.json"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${RESTORE_SCRIPT}" >/dev/null 2>&1

assert ".github/skills/SKILL.md restored to trusted" "grep -q 'trusted skill' '${TEST_WORKSPACE}/.github/skills/SKILL.md'"
assert ".agents/agent.md restored to trusted" "grep -q 'trusted agent' '${TEST_WORKSPACE}/.agents/agent.md'"
assert "AGENTS.md restored to trusted" "grep -q 'trusted agents' '${TEST_WORKSPACE}/AGENTS.md'"
assert "CLAUDE.md restored to trusted" "grep -q 'trusted claude' '${TEST_WORKSPACE}/CLAUDE.md'"
assert ".mcp.json removed" "[ ! -f '${TEST_WORKSPACE}/.mcp.json' ]"
rm -rf "${TEST_WORKSPACE}" /tmp/gh-aw/base
echo ""

# ── Test 2: Engine-specific folders restored ─────────────────────────────────
echo "Test 2: Engine-specific folders (.claude, .gemini, .cursor, .windsurf, .codex) restored"
TEST_WORKSPACE=$(mktemp -d)

mkdir -p /tmp/gh-aw/base/.claude/commands
echo "trusted cmd" >/tmp/gh-aw/base/.claude/commands/cmd.md
mkdir -p /tmp/gh-aw/base/.gemini
echo '{"trusted":true}' >/tmp/gh-aw/base/.gemini/settings.json

# PR-branch: evil versions
mkdir -p "${TEST_WORKSPACE}/.claude/commands"
echo "evil cmd" >"${TEST_WORKSPACE}/.claude/commands/cmd.md"
mkdir -p "${TEST_WORKSPACE}/.gemini"
echo '{"evil":true}' >"${TEST_WORKSPACE}/.gemini/settings.json"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${RESTORE_SCRIPT}" >/dev/null 2>&1

assert ".claude/commands/cmd.md restored" "grep -q 'trusted cmd' '${TEST_WORKSPACE}/.claude/commands/cmd.md'"
assert ".gemini/settings.json restored" "grep -q 'trusted' '${TEST_WORKSPACE}/.gemini/settings.json'"
rm -rf "${TEST_WORKSPACE}" /tmp/gh-aw/base
echo ""

# ── Test 3: PR-injected folder not in base is removed ────────────────────────
echo "Test 3: PR-injected folder absent from base → removed from workspace"
TEST_WORKSPACE=$(mktemp -d)
rm -rf /tmp/gh-aw/base

# PR branch injected .claude but base has no .claude
mkdir -p "${TEST_WORKSPACE}/.claude"
echo "evil instructions" >"${TEST_WORKSPACE}/.claude/CLAUDE.md"
# PR branch also injected AGENTS.md
echo "evil agents" >"${TEST_WORKSPACE}/AGENTS.md"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${RESTORE_SCRIPT}" >/dev/null 2>&1
EXIT_CODE=$?

assert "exits 0" "[ ${EXIT_CODE} -eq 0 ]"
assert ".claude removed (not in base)" "[ ! -d '${TEST_WORKSPACE}/.claude' ]"
assert "AGENTS.md removed (not in base)" "[ ! -f '${TEST_WORKSPACE}/AGENTS.md' ]"
rm -rf "${TEST_WORKSPACE}"
echo ""

# ── Test 4: PR-injected items absent from base are removed ───────────────────
echo "Test 4: No /tmp/gh-aw/base → PR-branch injected items removed"
TEST_WORKSPACE=$(mktemp -d)
mkdir -p "${TEST_WORKSPACE}/.cursor/rules"
echo "evil cursor rule" >"${TEST_WORKSPACE}/.cursor/rules/rule.mdc"
echo "evil gemini" >"${TEST_WORKSPACE}/GEMINI.md"
rm -rf /tmp/gh-aw/base

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${RESTORE_SCRIPT}" >/dev/null 2>&1
EXIT_CODE=$?

assert "exits 0" "[ ${EXIT_CODE} -eq 0 ]"
assert ".cursor removed (no base snapshot)" "[ ! -d '${TEST_WORKSPACE}/.cursor' ]"
assert "GEMINI.md removed (no base snapshot)" "[ ! -f '${TEST_WORKSPACE}/GEMINI.md' ]"
rm -rf "${TEST_WORKSPACE}"
echo ""

# ── Test 5: .mcp.json absent → exits 0 without error ────────────────────────
echo "Test 5: No .mcp.json in workspace → exits 0"
TEST_WORKSPACE=$(mktemp -d)
rm -rf /tmp/gh-aw/base

EXIT_CODE=0
GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${RESTORE_SCRIPT}" >/dev/null 2>&1 || EXIT_CODE=$?

assert "exits 0 when .mcp.json absent" "[ ${EXIT_CODE} -eq 0 ]"
rm -rf "${TEST_WORKSPACE}"
echo ""

# ── Test 6: Partial snapshot — only some items present ────────────────────────
echo "Test 6: Partial snapshot → only present items restored; absent PR items removed"
TEST_WORKSPACE=$(mktemp -d)

# Base has .github but not .windsurf
mkdir -p /tmp/gh-aw/base/.github
echo "trusted" >/tmp/gh-aw/base/.github/trusted.md
# no .windsurf in base

# PR has both
mkdir -p "${TEST_WORKSPACE}/.github"
echo "evil" >"${TEST_WORKSPACE}/.github/evil.md"
mkdir -p "${TEST_WORKSPACE}/.windsurf/rules"
echo "evil rule" >"${TEST_WORKSPACE}/.windsurf/rules/rule.md"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${RESTORE_SCRIPT}" >/dev/null 2>&1

assert ".github restored from base" "[ -f '${TEST_WORKSPACE}/.github/trusted.md' ]"
assert "evil .github file removed" "[ ! -f '${TEST_WORKSPACE}/.github/evil.md' ]"
assert ".windsurf removed (not in base)" "[ ! -d '${TEST_WORKSPACE}/.windsurf' ]"
rm -rf "${TEST_WORKSPACE}" /tmp/gh-aw/base
echo ""

# ── Summary ──────────────────────────────────────────────────────────────────
echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
