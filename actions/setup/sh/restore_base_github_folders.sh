#!/usr/bin/env bash
#
# restore_base_github_folders.sh - Restore .github and .agents from the base
#                                   branch snapshot after PR checkout
#
# After checkout_pr_branch runs the workspace contains PR-branch content,
# which may include attacker-controlled skill/instruction files for fork PRs.
# This script overwrites .github and .agents with the trusted snapshot that
# was saved by save_base_github_folders.sh during the activation job.
# It also removes .mcp.json from the workspace root, which may contain
# untrusted MCP server configuration from the PR branch.
#
# Exit codes:
#   0 - Success

set -euo pipefail

WORKSPACE="${GITHUB_WORKSPACE:-$(pwd)}"
SRC="/tmp/gh-aw/base"

for FOLDER in .github .agents; do
  SNAPSHOT="${SRC}/${FOLDER}"
  DEST="${WORKSPACE}/${FOLDER}"
  if [ -d "${SNAPSHOT}" ]; then
    rm -rf "${DEST}"
    cp -r "${SNAPSHOT}" "${DEST}"
    echo "Restored ${FOLDER} from base branch snapshot"
  else
    echo "No base branch snapshot for ${FOLDER}, skipping"
  fi
done

# Remove .mcp.json — may contain untrusted MCP server config from the PR branch
if [ -f "${WORKSPACE}/.mcp.json" ]; then
  rm -f "${WORKSPACE}/.mcp.json"
  echo "Removed .mcp.json from workspace"
fi
