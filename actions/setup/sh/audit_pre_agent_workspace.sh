#!/usr/bin/env bash
set +o histexpand

#
# audit_pre_agent_workspace.sh - Capture a file listing of agent-related directories
# before the AI engine starts.
#
# This script runs after all pre-agent preparation (skills, agents, MCP servers) is
# complete and writes a complete file listing of agent-related directories to
# /tmp/gh-aw/pre-agent-audit.txt.  The listing is also surfaced via GITHUB_OUTPUT
# so downstream steps can reference it.
#
# Directories scanned:
#   $GITHUB_WORKSPACE/.github/agents/       - workspace agent files
#   $GITHUB_WORKSPACE/.github/skills/       - workspace skill files
#   $GITHUB_WORKSPACE/.github/copilot/      - workspace Copilot config
#   $HOME/.github/                          - agent user home .github
#   $HOME/.local/share/gh/extensions/       - installed gh extensions
#   $RUNNER_TEMP/gh-aw/                     - runner temp gh-aw directory
#
# Common cache directories (node_modules, __pycache__, .cache, vendor, .npm, .yarn,
# .pnpm-store, site-packages, .bundle) are excluded to keep the listing concise.
#
# Environment variables (set automatically by GitHub Actions):
#   GITHUB_WORKSPACE   - path to the checked-out repository
#   HOME               - agent user home directory
#   RUNNER_TEMP        - runner temporary directory
#   GITHUB_OUTPUT      - path to the GitHub Actions output file
#
# GitHub Actions outputs written:
#   pre-agent-audit-file        - path to the audit file
#   pre-agent-audit-line-count  - number of lines in the audit file
#
# Exit codes:
#   0 - always (uses continue-on-error in the workflow step)

set -euo pipefail

AUDIT_FILE="/tmp/gh-aw/pre-agent-audit.txt"
mkdir -p /tmp/gh-aw

# list_dir prints a section header and runs find on the given directory,
# excluding common cache folders.  Missing directories are silently noted.
list_dir() {
  local label="$1"
  local dir="$2"
  echo "--- ${label}: ${dir} ---"
  find "${dir}" \
    -not -path '*/node_modules/*' \
    -not -path '*/__pycache__/*' \
    -not -path '*/.cache/*' \
    -not -path '*/vendor/*' \
    -not -path '*/.npm/*' \
    -not -path '*/.yarn/*' \
    -not -path '*/.pnpm-store/*' \
    -not -path '*/site-packages/*' \
    -not -path '*/.bundle/*' \
    -print 2>/dev/null || echo "(not found)"
}

{
  echo "=== Pre-agent workspace audit ==="
  list_dir "Workspace agents"       "${GITHUB_WORKSPACE}/.github/agents"
  list_dir "Workspace skills"       "${GITHUB_WORKSPACE}/.github/skills"
  list_dir "Workspace copilot"      "${GITHUB_WORKSPACE}/.github/copilot"
  list_dir "Agent user home .github" "${HOME}/.github"
  list_dir "gh extensions"          "${HOME}/.local/share/gh/extensions"
  list_dir "gh-aw temp directory"   "${RUNNER_TEMP}/gh-aw"
} > "${AUDIT_FILE}"

LINE_COUNT="$(wc -l < "${AUDIT_FILE}" | tr -d ' ')"
echo "pre-agent-audit-file=${AUDIT_FILE}" >> "${GITHUB_OUTPUT}"
echo "pre-agent-audit-line-count=${LINE_COUNT}" >> "${GITHUB_OUTPUT}"
echo "Pre-agent audit written to ${AUDIT_FILE} (${LINE_COUNT} lines)"
