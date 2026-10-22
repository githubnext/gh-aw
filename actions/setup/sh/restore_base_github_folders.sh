#!/usr/bin/env bash
set +o histexpand

#
# restore_base_github_folders.sh - Restore agent config folders/files from the base
#                                   branch snapshot after PR checkout
#
# After checkout_pr_branch runs the workspace contains PR-branch content,
# which may include attacker-controlled skill/instruction files for fork PRs.
# This script overwrites agent-specific folders and root instruction files with
# the trusted snapshot saved by save_base_github_folders.sh during the activation
# job. If HEAD is exactly the verified default-branch tip, the snapshot is instead
# materialized from that commit to avoid reverting base changes since activation.
# It also removes .mcp.json from the workspace root, which may contain
# untrusted MCP server configuration from the PR branch.
#
# For each item:
#   - If present in the base snapshot: restore it (overwrite PR-branch content)
#   - If absent from the base snapshot but present in the workspace: remove it
#     (prevent the PR branch from injecting files that the base branch doesn't have)
#
# The lists of folders and files MUST match those used in save_base_github_folders.sh
# and are passed via the same environment variables:
#
#   GH_AW_AGENT_FOLDERS  - space-separated list of directories to restore
#                          (e.g. ".agents .claude .codex .gemini .github")
#   GH_AW_AGENT_FILES    - space-separated list of root files to restore
#                          (e.g. "AGENTS.md CLAUDE.md GEMINI.md")
#   DEFAULT_BRANCH       - trusted repository default branch, when available
#
# Exit codes:
#   0 - Success
#   1 - Trusted configuration could not be materialized

set -euo pipefail

WORKSPACE="${GITHUB_WORKSPACE:-$(pwd)}"
SRC="/tmp/gh-aw/base"

# Parse the engine-registry-derived lists from environment variables.
# These must match the values used in save_base_github_folders.sh.
IFS=' ' read -ra FOLDERS <<< "${GH_AW_AGENT_FOLDERS:-}"
IFS=' ' read -ra ROOT_FILES <<< "${GH_AW_AGENT_FILES:-}"

# DEFAULT_BRANCH is set by safe outputs; other workflows can use the same
# trusted repository metadata directly from the runner's event payload.
if [ -z "${DEFAULT_BRANCH:-}" ] && [ -f "${GITHUB_EVENT_PATH:-}" ]; then
  DEFAULT_BRANCH=$(jq -r '.repository.default_branch // empty' "${GITHUB_EVENT_PATH}")
fi

# Only an exact remote-tracking ref at the workspace root can establish trust.
# In particular, PR branch names can resemble refs, and origin/pr-head is the
# checkout script's untrusted PR staging ref, not a trusted base branch.
if [ -n "${DEFAULT_BRANCH:-}" ] && [ "${DEFAULT_BRANCH}" != "pr-head" ] &&
  TOPLEVEL=$(git -C "${WORKSPACE}" rev-parse --show-toplevel 2>/dev/null) &&
  [ "${TOPLEVEL}" = "$(cd "${WORKSPACE}" && pwd -P)" ] &&
  BASE_SHA=$(git -C "${WORKSPACE}" show-ref --verify --hash "refs/remotes/origin/${DEFAULT_BRANCH}" 2>/dev/null) &&
  HEAD_SHA=$(git -C "${WORKSPACE}" rev-parse --verify HEAD 2>/dev/null) &&
  [ "${HEAD_SHA}" = "${BASE_SHA}" ]; then
  # Do not simply skip restoration: sparse checkouts and earlier ambient-folder
  # copies may be missing files or already contain the stale activation content.
  FRESH_SNAPSHOT=$(mktemp -d)
  trap 'rm -rf "${FRESH_SNAPSHOT}"' EXIT
  for ITEM in "${FOLDERS[@]+"${FOLDERS[@]}"}" "${ROOT_FILES[@]+"${ROOT_FILES[@]}"}"; do
    TREE_ENTRY=$(GIT_NO_LAZY_FETCH=1 git -C "${WORKSPACE}" ls-tree "${BASE_SHA}" -- "${ITEM}")
    if [ -n "${TREE_ENTRY}" ]; then
      if ! GIT_NO_LAZY_FETCH=1 git -C "${WORKSPACE}" --work-tree="${FRESH_SNAPSHOT}" restore \
        --source="${BASE_SHA}" --worktree --ignore-skip-worktree-bits -- "${ITEM}"; then
        echo "::error::Cannot materialize trusted config ${ITEM} at ${BASE_SHA}; include its objects in checkout."
        exit 1
      fi
    fi
  done
  SRC="${FRESH_SNAPSHOT}"
  echo "Restoring agent config from verified default-branch checkout ${BASE_SHA}"

  # A later ambient restore must not copy the activation version back again.
  for FOLDER in "${FOLDERS[@]+"${FOLDERS[@]}"}"; do
    AMBIENT="/tmp/gh-aw/ambient-folders/${FOLDER}"
    if [ -e "${AMBIENT}" ] || [ -L "${AMBIENT}" ]; then
      rm -rf "${AMBIENT}"
      if [ -d "${SRC}/${FOLDER}" ]; then
        mkdir -p "$(dirname "${AMBIENT}")"
        cp -r "${SRC}/${FOLDER}" "${AMBIENT}"
      fi
    fi
  done
else
  echo "Checkout is not a verified default-branch tip; restoring activation snapshot"
fi

for FOLDER in "${FOLDERS[@]+"${FOLDERS[@]}"}"; do
  SNAPSHOT="${SRC}/${FOLDER}"
  DEST="${WORKSPACE}/${FOLDER}"
  if [ -d "${SNAPSHOT}" ]; then
    rm -rf "${DEST}"
    cp -r "${SNAPSHOT}" "${DEST}"
    echo "Restored ${FOLDER} from base branch snapshot"
  elif [ -d "${DEST}" ]; then
    # PR branch injected this directory but base doesn't have it — remove it
    rm -rf "${DEST}"
    echo "Removed PR-injected ${FOLDER} (not present in base branch)"
  else
    echo "No base branch snapshot for ${FOLDER}, skipping"
  fi
done

for FILE in "${ROOT_FILES[@]+"${ROOT_FILES[@]}"}"; do
  SNAPSHOT="${SRC}/${FILE}"
  DEST="${WORKSPACE}/${FILE}"
  if [ -f "${SNAPSHOT}" ]; then
    cp "${SNAPSHOT}" "${DEST}"
    echo "Restored ${FILE} from base branch snapshot"
  elif [ -f "${DEST}" ]; then
    # PR branch injected this file but base doesn't have it — remove it
    rm -f "${DEST}"
    echo "Removed PR-injected ${FILE} (not present in base branch)"
  else
    echo "No base branch snapshot for ${FILE}, skipping"
  fi
done

# Remove .mcp.json — may contain untrusted MCP server config from the PR branch
if [ -f "${WORKSPACE}/.mcp.json" ]; then
  rm -f "${WORKSPACE}/.mcp.json"
  echo "Removed .mcp.json from workspace"
fi
