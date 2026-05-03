#!/usr/bin/env bash
set +o histexpand

# Test script for audit_pre_agent_workspace.sh
# Run: bash audit_pre_agent_workspace_test.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="${SCRIPT_DIR}/audit_pre_agent_workspace.sh"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Temporary workspace for tests
TEST_ROOT="$(mktemp -d)"

cleanup() {
  rm -rf "${TEST_ROOT}"
}
trap cleanup EXIT

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

echo "Testing audit_pre_agent_workspace.sh..."
echo ""

# ── Test 1: Script syntax is valid ──────────────────────────────────────────
echo "Test 1: Script syntax is valid"
if bash -n "${SCRIPT_PATH}" 2>/dev/null; then
  assert "Script syntax is valid" "true"
else
  assert "Script has syntax errors" "false"
fi
echo ""

# ── Test 2: Writes audit file to /tmp/gh-aw/pre-agent-audit.txt ─────────────
echo "Test 2: Audit file is written to the correct path"
WORKSPACE="${TEST_ROOT}/workspace-2"
mkdir -p "${WORKSPACE}/.github/agents"
echo "agent.md" > "${WORKSPACE}/.github/agents/my.md"
TMPDIR="${TEST_ROOT}/tmp2"
mkdir -p "${TMPDIR}/gh-aw"
GH_OUT="${TEST_ROOT}/github_output2"
touch "${GH_OUT}"
rm -f /tmp/gh-aw/pre-agent-audit.txt
OUTPUT="$(GITHUB_WORKSPACE="${WORKSPACE}" HOME="${TEST_ROOT}/home2" RUNNER_TEMP="${TMPDIR}" GITHUB_OUTPUT="${GH_OUT}" bash "${SCRIPT_PATH}" 2>&1)"
assert "Audit file created at /tmp/gh-aw/pre-agent-audit.txt" "[ -f /tmp/gh-aw/pre-agent-audit.txt ]"
assert "Audit file is non-empty" "[ -s /tmp/gh-aw/pre-agent-audit.txt ]"
echo ""

# ── Test 3: Audit file contains expected section headers ────────────────────
echo "Test 3: Section headers are present in audit output"
assert "Contains workspace agents header"  "grep -q 'Workspace agents' /tmp/gh-aw/pre-agent-audit.txt"
assert "Contains workspace skills header"  "grep -q 'Workspace skills' /tmp/gh-aw/pre-agent-audit.txt"
assert "Contains workspace copilot header" "grep -q 'Workspace copilot' /tmp/gh-aw/pre-agent-audit.txt"
assert "Contains workspace claude header"  "grep -q 'Workspace claude' /tmp/gh-aw/pre-agent-audit.txt"
assert "Contains workspace codex header"   "grep -q 'Workspace codex' /tmp/gh-aw/pre-agent-audit.txt"
assert "Contains workspace gemini header"  "grep -q 'Workspace gemini' /tmp/gh-aw/pre-agent-audit.txt"
assert "Contains workspace crush header"   "grep -q 'Workspace crush' /tmp/gh-aw/pre-agent-audit.txt"
assert "Contains workspace opencode header" "grep -q 'Workspace opencode' /tmp/gh-aw/pre-agent-audit.txt"
assert "Contains workspace pi header"      "grep -q 'Workspace pi' /tmp/gh-aw/pre-agent-audit.txt"
assert "Contains agent user home header"   "grep -q 'Agent user home' /tmp/gh-aw/pre-agent-audit.txt"
assert "Contains gh extensions header"     "grep -q 'gh extensions' /tmp/gh-aw/pre-agent-audit.txt"
assert "Contains gh-aw temp header"        "grep -q 'gh-aw temp directory' /tmp/gh-aw/pre-agent-audit.txt"
echo ""

# ── Test 4: Lists files that exist ──────────────────────────────────────────
echo "Test 4: Existing agent file appears in audit output"
assert "Agent file listed in audit" "grep -q 'my.md' /tmp/gh-aw/pre-agent-audit.txt"
echo ""

# ── Test 5: Missing directories are noted as (not found) ────────────────────
echo "Test 5: Missing directories show (not found)"
assert "Missing directory noted" "grep -q '(not found)' /tmp/gh-aw/pre-agent-audit.txt"
echo ""

# ── Test 6: GITHUB_OUTPUT contains pre-agent-audit-file entry ───────────────
echo "Test 6: GITHUB_OUTPUT is written"
assert "pre-agent-audit-file in GITHUB_OUTPUT" "grep -q 'pre-agent-audit-file=' '${GH_OUT}'"
assert "pre-agent-audit-line-count in GITHUB_OUTPUT" "grep -q 'pre-agent-audit-line-count=' '${GH_OUT}'"
echo ""

# ── Test 7: Excludes node_modules ───────────────────────────────────────────
echo "Test 7: node_modules directory is excluded"
WORKSPACE="${TEST_ROOT}/workspace-7"
mkdir -p "${WORKSPACE}/.github/agents/node_modules/some-pkg"
echo "index.js" > "${WORKSPACE}/.github/agents/node_modules/some-pkg/index.js"
echo "agent.md" > "${WORKSPACE}/.github/agents/agent.md"
TMPDIR="${TEST_ROOT}/tmp7"
mkdir -p "${TMPDIR}/gh-aw"
GH_OUT7="${TEST_ROOT}/github_output7"
touch "${GH_OUT7}"
rm -f /tmp/gh-aw/pre-agent-audit.txt
GITHUB_WORKSPACE="${WORKSPACE}" HOME="${TEST_ROOT}/home7" RUNNER_TEMP="${TMPDIR}" GITHUB_OUTPUT="${GH_OUT7}" bash "${SCRIPT_PATH}" >/dev/null 2>&1
assert "node_modules excluded from listing" "! grep -q 'node_modules/some-pkg' /tmp/gh-aw/pre-agent-audit.txt"
assert "agent.md still listed despite node_modules sibling" "grep -q 'agent.md' /tmp/gh-aw/pre-agent-audit.txt"
echo ""

# ── Summary ──────────────────────────────────────────────────────────────────
echo "Results: ${TESTS_PASSED} passed, ${TESTS_FAILED} failed"
if [ "${TESTS_FAILED}" -gt 0 ]; then
  exit 1
fi
exit 0
