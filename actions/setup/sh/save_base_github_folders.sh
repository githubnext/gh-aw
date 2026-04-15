#!/usr/bin/env bash
#
# save_base_github_folders.sh - Snapshot .github and .agents from the workspace
#
# Copies .github and .agents from $GITHUB_WORKSPACE into /tmp/gh-aw/base/ so
# that they can be included in the activation artifact and later restored in
# the agent job after checkout_pr_branch runs.
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

for FOLDER in .github .agents; do
  SRC="${WORKSPACE}/${FOLDER}"
  if [ -d "${SRC}" ]; then
    mkdir -p "${DEST}"
    cp -r "${SRC}" "${DEST}/${FOLDER}"
    echo "Saved ${FOLDER} to ${DEST}/${FOLDER}"
  else
    echo "${FOLDER} not found in workspace, skipping"
  fi
done
