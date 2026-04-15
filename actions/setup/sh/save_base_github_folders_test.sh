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

# ── Test 1: Core folders (.github, .agents) are saved ────────────────────────
echo "Test 1: .github and .agents are copied to /tmp/gh-aw/base"
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

# ── Test 2: Engine-specific folders are saved ─────────────────────────────────
echo "Test 2: Engine-specific folders (.claude, .gemini, .cursor, .windsurf, .codex) are saved"
TEST_WORKSPACE=$(mktemp -d)
mkdir -p "${TEST_WORKSPACE}/.claude/commands"
echo "claude cmd" >"${TEST_WORKSPACE}/.claude/commands/cmd.md"
mkdir -p "${TEST_WORKSPACE}/.gemini"
echo "{}" >"${TEST_WORKSPACE}/.gemini/settings.json"
mkdir -p "${TEST_WORKSPACE}/.cursor/rules"
echo "rule" >"${TEST_WORKSPACE}/.cursor/rules/rule.mdc"
mkdir -p "${TEST_WORKSPACE}/.windsurf/rules"
echo "rule" >"${TEST_WORKSPACE}/.windsurf/rules/rule.md"
mkdir -p "${TEST_WORKSPACE}/.codex"
echo "config" >"${TEST_WORKSPACE}/.codex/config"
rm -rf "${REAL_DEST}"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${SAVE_SCRIPT}" >/dev/null 2>&1

assert "saves .claude" "[ -d '${REAL_DEST}/.claude' ]"
assert "saves .claude/commands/cmd.md" "[ -f '${REAL_DEST}/.claude/commands/cmd.md' ]"
assert "saves .gemini" "[ -d '${REAL_DEST}/.gemini' ]"
assert "saves .gemini/settings.json" "[ -f '${REAL_DEST}/.gemini/settings.json' ]"
assert "saves .cursor" "[ -d '${REAL_DEST}/.cursor' ]"
assert "saves .windsurf" "[ -d '${REAL_DEST}/.windsurf' ]"
assert "saves .codex" "[ -d '${REAL_DEST}/.codex' ]"
rm -rf "${TEST_WORKSPACE}" "${REAL_DEST}"
echo ""

# ── Test 3: Root instruction files are saved ──────────────────────────────────
echo "Test 3: Root instruction files (AGENTS.md, CLAUDE.md, GEMINI.md) are saved"
TEST_WORKSPACE=$(mktemp -d)
echo "agents instructions" >"${TEST_WORKSPACE}/AGENTS.md"
echo "claude instructions" >"${TEST_WORKSPACE}/CLAUDE.md"
echo "gemini instructions" >"${TEST_WORKSPACE}/GEMINI.md"
rm -rf "${REAL_DEST}"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${SAVE_SCRIPT}" >/dev/null 2>&1

assert "saves AGENTS.md" "[ -f '${REAL_DEST}/AGENTS.md' ]"
assert "saves CLAUDE.md" "[ -f '${REAL_DEST}/CLAUDE.md' ]"
assert "saves GEMINI.md" "[ -f '${REAL_DEST}/GEMINI.md' ]"
assert "AGENTS.md content preserved" "grep -q 'agents instructions' '${REAL_DEST}/AGENTS.md'"
rm -rf "${TEST_WORKSPACE}" "${REAL_DEST}"
echo ""

# ── Test 4: Only .github present — other items skipped without error ──────────
echo "Test 4: Only .github present → only .github is copied, script exits 0"
TEST_WORKSPACE=$(mktemp -d)
mkdir -p "${TEST_WORKSPACE}/.github/instructions"
echo "instructions" >"${TEST_WORKSPACE}/.github/instructions/README.md"
rm -rf "${REAL_DEST}"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${SAVE_SCRIPT}" >/dev/null 2>&1
EXIT_CODE=$?

assert "exits 0" "[ ${EXIT_CODE} -eq 0 ]"
assert "saves .github" "[ -d '${REAL_DEST}/.github' ]"
assert ".agents not created when absent" "[ ! -d '${REAL_DEST}/.agents' ]"
assert ".claude not created when absent" "[ ! -d '${REAL_DEST}/.claude' ]"
assert "AGENTS.md not created when absent" "[ ! -f '${REAL_DEST}/AGENTS.md' ]"
rm -rf "${TEST_WORKSPACE}" "${REAL_DEST}"
echo ""

# ── Test 5: Nothing present → dest not created, exits 0 ──────────────────────
echo "Test 5: No watched items → /tmp/gh-aw/base not created"
TEST_WORKSPACE=$(mktemp -d)
rm -rf "${REAL_DEST}"

EXIT_CODE=0
GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${SAVE_SCRIPT}" >/dev/null 2>&1 || EXIT_CODE=$?

assert "exits 0 when nothing to save" "[ ${EXIT_CODE} -eq 0 ]"
assert "/tmp/gh-aw/base not created" "[ ! -d '${REAL_DEST}' ]"
rm -rf "${TEST_WORKSPACE}"
echo ""

# ── Test 6: Re-run clears stale snapshot (idempotent) ────────────────────────
echo "Test 6: Re-run overwrites stale snapshot (idempotent)"
TEST_WORKSPACE=$(mktemp -d)
mkdir -p "${TEST_WORKSPACE}/.github"
echo "new content" >"${TEST_WORKSPACE}/.github/new.md"
# Pre-create a stale snapshot with different content
mkdir -p "${REAL_DEST}/.github"
echo "stale content" >"${REAL_DEST}/.github/stale.md"

GITHUB_WORKSPACE="${TEST_WORKSPACE}" bash "${SAVE_SCRIPT}" >/dev/null 2>&1

assert "new file present after re-run" "[ -f '${REAL_DEST}/.github/new.md' ]"
assert "stale file removed on re-run" "[ ! -f '${REAL_DEST}/.github/stale.md' ]"
rm -rf "${TEST_WORKSPACE}" "${REAL_DEST}"
echo ""

# ── Summary ──────────────────────────────────────────────────────────────────
echo "Tests passed: ${TESTS_PASSED}"
echo "Tests failed: ${TESTS_FAILED}"

if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi

echo "✓ All tests passed!"
