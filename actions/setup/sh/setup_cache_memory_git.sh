#!/bin/bash
# setup_cache_memory_git.sh
# Pre-agent git setup for integrity-aware cache-memory.
#
# This script is run AFTER the cache is restored and BEFORE the agent executes.
# It ensures the cache directory contains a git repository with integrity branches
# and checks out the correct branch for the current run's integrity level.
#
# Required environment variables:
#   GH_AW_CACHE_DIR:       Path to the cache-memory directory (e.g. /tmp/gh-aw/cache-memory)
#   GH_AW_MIN_INTEGRITY:   Integrity level for this run (merged|approved|unapproved|none)

set -euo pipefail

CACHE_DIR="${GH_AW_CACHE_DIR:-/tmp/gh-aw/cache-memory}"
INTEGRITY="${GH_AW_MIN_INTEGRITY:-none}"

# All integrity levels in descending order (highest first)
LEVELS=("merged" "approved" "unapproved" "none")

mkdir -p "$CACHE_DIR"
cd "$CACHE_DIR"

# --- Security: clear git hooks before any git operations ---
# Git hook files under .git/hooks/ are preserved in the cache but are NOT tracked
# by git (git add -A ignores .git/). A compromised agent run could write executable
# hooks (e.g. post-checkout, post-merge) that would be restored from cache and
# executed on the host runner before the AWF sandbox is established. Remove all
# non-sample hook files immediately after cache restore to prevent this.
if [ -d .git/hooks ]; then
  find .git/hooks -type f ! -name '*.sample' -delete
fi

# --- Format detection & migration ---
if [ ! -d .git ]; then
  # No git repo yet — either a fresh cache or a legacy flat-file cache.
  # Initialize a git repository with an empty baseline commit on the highest-trust
  # branch, then create all other integrity branches from that empty state.
  # IMPORTANT: Legacy flat files (written at unknown/none integrity in a previous
  # version of gh-aw) are committed to the 'none' branch only to prevent trust
  # escalation — do NOT commit them to 'merged' or any higher-trust branch.
  git init -b merged -q
  git config user.email "gh-aw@github.com"
  git config user.name "gh-aw"
  # Disable hooks immediately after init so that no cached hook file can fire
  # during checkout or merge operations later in this script.
  git config core.hooksPath /dev/null
  # Create an empty initial commit as the trusted baseline for all branches
  git commit --allow-empty -m "initial" -q

  # Create all integrity branches from the empty baseline
  for level in "${LEVELS[@]}"; do
    if [ "$level" != "merged" ]; then
      git branch "$level" 2>/dev/null || true
    fi
  done

  # Migrate any pre-existing flat files to the 'none' branch only (lowest trust).
  # Switching to 'none' before staging ensures legacy data cannot be read by
  # higher-integrity runs via the merge-down step.
  git checkout -q none
  git add -A
  git commit --allow-empty -m "migrate-legacy-files" -q

  echo "Cache memory git repository initialized with branches: ${LEVELS[*]}"
else
  # Existing repo: disable hooks as belt-and-suspenders after the hook-file
  # deletion above, ensuring no residual configuration can re-enable hooks.
  git config core.hooksPath /dev/null
fi

# --- Checkout current integrity branch ---
# Use -q to suppress "Switched to branch" noise
git checkout -q "$INTEGRITY"

# --- Merge down from higher-integrity branches ---
# Read semantics: lower-integrity runs see higher-integrity data via merge,
# but higher-integrity runs never see lower-integrity data.
# -X theirs: higher-integrity branch wins conflicts.
for level in "${LEVELS[@]}"; do
  if [ "$level" = "$INTEGRITY" ]; then
    break
  fi
  # Merge higher-integrity branch into the current branch
  if git merge "$level" -X theirs --no-edit -m "merge-from-$level" -q 2>/tmp/gh-aw-merge-err; then
    echo "Merged integrity branch '$level' into '$INTEGRITY'"
  else
    merge_exit=$?
    # Abort the merge to restore a clean working tree, then hard-reset to the
    # pre-merge state so the agent always starts from a consistent, usable tree.
    git merge --abort 2>/dev/null || git reset --hard HEAD 2>/dev/null || true
    # Ignore "already up-to-date" and "nothing to merge" — fail fast on real errors
    if grep -qiE "already up.to.date|nothing to merge|nothing to commit" /tmp/gh-aw-merge-err 2>/dev/null; then
      echo "Nothing to merge from '$level' into '$INTEGRITY' (already up-to-date)"
    else
      echo "ERROR: merge from '$level' into '$INTEGRITY' failed (exit $merge_exit):" >&2
      cat /tmp/gh-aw-merge-err >&2
      exit "$merge_exit"
    fi
  fi
done

echo "Cache memory git setup complete (integrity: $INTEGRITY)"
