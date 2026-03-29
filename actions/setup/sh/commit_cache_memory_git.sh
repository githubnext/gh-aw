#!/bin/bash
# commit_cache_memory_git.sh
# Post-agent git commit for integrity-aware cache-memory.
#
# This script is run AFTER the agent executes and BEFORE the cache is saved.
# It commits all agent-written changes to the current integrity branch so that
# the git history accurately reflects which run wrote which data.
#
# Required environment variables:
#   GH_AW_CACHE_DIR:   Path to the cache-memory directory (e.g. /tmp/gh-aw/cache-memory)
#   GITHUB_RUN_ID:     GitHub Actions run ID (used as commit message)

set -euo pipefail

CACHE_DIR="${GH_AW_CACHE_DIR:-/tmp/gh-aw/cache-memory}"
RUN_ID="${GITHUB_RUN_ID:-unknown}"

if [ ! -d "$CACHE_DIR/.git" ]; then
  echo "No git repository found at $CACHE_DIR — skipping git commit"
  exit 0
fi

cd "$CACHE_DIR"

git config user.email "gh-aw@github.com"
git config user.name "gh-aw"

# Stage all changes (new files, modifications, deletions)
git add -A

# Commit on the current integrity branch; allow empty commits in case
# the agent made no changes (idempotent).
git commit --allow-empty -m "run-${RUN_ID}" -q || true

# Keep the repo small: pack loose objects and prune unreachable ones.
git gc --auto -q 2>/dev/null || true

echo "Cache memory git commit complete (run: $RUN_ID)"
