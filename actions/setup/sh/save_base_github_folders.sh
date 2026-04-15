#!/usr/bin/env bash
#
# save_base_github_folders.sh - Snapshot agent config folders/files from the workspace
#
# Copies agent-specific folders and root instruction files from $GITHUB_WORKSPACE
# into /tmp/gh-aw/base/ so that they can be included in the activation artifact
# and later restored in the agent job after checkout_pr_branch runs.
#
# Covered items:
#   Directories: .github  .agents  .claude  .gemini  .cursor  .windsurf  .codex
#   Root files:  AGENTS.md  CLAUDE.md  GEMINI.md
#
# This prevents fork PRs from injecting malicious skill or instruction files
# into the agent's context: the activation job runs on the base branch, so the
# snapshot always reflects the trusted base-branch content.
#
# Exit codes:
#   0 - Success

set -euo pipefail

WORKSPACE="${GITHUB_WORKSPACE:-$(pwd)}"
DEST="/tmp/gh-aw/base"

# Engine-specific configuration directories to snapshot
FOLDERS=(.github .agents .claude .gemini .cursor .windsurf .codex)

# Root-level instruction/agent files to snapshot
# AGENTS.md  - cross-engine convention (Codex, Copilot, Windsurf, Gemini, Claude)
# CLAUDE.md  - Claude Code
# GEMINI.md  - Google Gemini CLI
ROOT_FILES=(AGENTS.md CLAUDE.md GEMINI.md)

for FOLDER in "${FOLDERS[@]}"; do
  SRC="${WORKSPACE}/${FOLDER}"
  if [ -d "${SRC}" ]; then
    mkdir -p "${DEST}"
    rm -rf "${DEST}/${FOLDER}"
    cp -r "${SRC}" "${DEST}/${FOLDER}"
    echo "Saved ${FOLDER} to ${DEST}/${FOLDER}"
  else
    echo "${FOLDER} not found in workspace, skipping"
  fi
done

for FILE in "${ROOT_FILES[@]}"; do
  SRC="${WORKSPACE}/${FILE}"
  if [ -f "${SRC}" ]; then
    mkdir -p "${DEST}"
    cp "${SRC}" "${DEST}/${FILE}"
    echo "Saved ${FILE} to ${DEST}/${FILE}"
  else
    echo "${FILE} not found in workspace, skipping"
  fi
done
