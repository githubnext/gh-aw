#!/bin/bash
set +o histexpand

# resolve-base-commit.sh - Resolve the base commit used for impacted-change detection
#
# Prints a single commit-ish to stdout that callers can diff against
# (`git diff --name-only <base-commit>`). Diagnostics go to stderr so the
# stdout value stays machine readable.
#
# CI checks out the repository with `fetch-depth: 1`, so `origin/main` is often
# missing (or shares no history with HEAD) and `git merge-base origin/main HEAD`
# fails. This script incrementally fetches and deepens the base ref until a
# merge-base can be computed, and degrades gracefully when fetching is not
# possible (offline, no remote, detached fixture repositories).
#
# Resolution order:
#   1. merge-base of BASE_REF and HEAD, if it can be computed already
#   2. after fetching the base branch and deepening history (increasing depths)
#   3. after a full unshallow of the repository
#   4. the base ref tip itself, if it exists but shares no reachable history
#   5. HEAD~1, when HEAD has a parent and no base ref is available
#
# Usage:
#   resolve-base-commit.sh [--base-ref <git-ref>] [--no-fetch]
#
# Options:
#   --base-ref <git-ref>  Base ref to resolve against (default: $BASE_REF or origin/main)
#   --no-fetch            Never contact the remote; only use local history
#
# Exit codes:
#   0 - A base commit was resolved and printed to stdout
#   1 - No base commit could be resolved

set -euo pipefail

BASE_REF="${BASE_REF:-origin/main}"
ALLOW_FETCH=1
# Depths tried when deepening a shallow clone before giving up and unshallowing.
FETCH_DEPTHS="${RESOLVE_BASE_COMMIT_DEPTHS:-50 500}"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --base-ref)
            BASE_REF="${2:?--base-ref requires an argument}"
            shift 2
            ;;
        --no-fetch)
            ALLOW_FETCH=0
            shift
            ;;
        -h | --help)
            sed -n '/^# resolve-base-commit\.sh/,/^# *1 - No base commit/p' "${BASH_SOURCE[0]}"
            exit 0
            ;;
        *)
            echo "Error: unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

log() { echo "$*" >&2; }

merge_base() {
    git merge-base "$BASE_REF" HEAD 2>/dev/null || true
}

is_shallow() {
    [ "$(git rev-parse --is-shallow-repository 2>/dev/null || echo false)" = "true" ]
}

# Split "origin/main" into remote ("origin") and branch ("main"). Refs without a
# known remote prefix are treated as local and are not fetched.
REMOTE=""
BRANCH=""
if [[ "$BASE_REF" == */* ]]; then
    candidate_remote="${BASE_REF%%/*}"
    if git remote 2>/dev/null | grep -qx "$candidate_remote"; then
        REMOTE="$candidate_remote"
        BRANCH="${BASE_REF#*/}"
    fi
fi

fetch_base_ref() {
    local depth_arg="$1"
    [ -n "$REMOTE" ] || return 1
    # shellcheck disable=SC2086 # depth_arg is intentionally word-split (may be empty)
    git fetch --no-tags --quiet $depth_arg "$REMOTE" \
        "+refs/heads/${BRANCH}:refs/remotes/${REMOTE}/${BRANCH}" 2>/dev/null || return 1
}

# Deepen the shallow history of the checked-out branch itself: without it, HEAD
# still looks parentless and no merge-base can be computed. Restrict the fetch
# to the current branch when it is known so unrelated branches are not pulled.
deepen_head() {
    local depth="$1"
    local current_branch
    current_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)
    if [ -n "$current_branch" ] && [ "$current_branch" != "HEAD" ] &&
        git fetch --no-tags --quiet "--deepen=$depth" "$REMOTE" \
            "+refs/heads/${current_branch}:refs/remotes/${REMOTE}/${current_branch}" 2>/dev/null; then
        return 0
    fi
    git fetch --no-tags --quiet "--deepen=$depth" "$REMOTE" 2>/dev/null || return 1
}

BASE_COMMIT=$(merge_base)

if [ -z "$BASE_COMMIT" ] && [ "$ALLOW_FETCH" = "1" ] && [ -n "$REMOTE" ]; then
    for depth in $FETCH_DEPTHS; do
        log "resolve-base-commit: fetching $BASE_REF (depth $depth) to resolve merge-base..."
        fetch_base_ref "--depth=$depth" || true
        if is_shallow; then
            deepen_head "$depth" || true
        fi
        BASE_COMMIT=$(merge_base)
        [ -n "$BASE_COMMIT" ] && break
    done

    if [ -z "$BASE_COMMIT" ]; then
        log "resolve-base-commit: deepening did not connect histories; unshallowing repository..."
        if is_shallow; then
            git fetch --no-tags --quiet --unshallow "$REMOTE" 2>/dev/null || true
        fi
        fetch_base_ref "" || true
        BASE_COMMIT=$(merge_base)
    fi
fi

if [ -z "$BASE_COMMIT" ] && git rev-parse --verify --quiet "${BASE_REF}^{commit}" >/dev/null 2>&1; then
    # The base ref exists but shares no reachable history with HEAD (typical for
    # grafted shallow clones). Diffing against its tip still yields a usable,
    # if slightly wider, set of changed files.
    BASE_COMMIT=$(git rev-parse "${BASE_REF}^{commit}")
    log "resolve-base-commit: no merge-base with $BASE_REF; falling back to its tip $BASE_COMMIT."
fi

if [ -z "$BASE_COMMIT" ] && git rev-parse --verify --quiet "HEAD~1^{commit}" >/dev/null 2>&1; then
    BASE_COMMIT=$(git rev-parse "HEAD~1^{commit}")
    log "resolve-base-commit: $BASE_REF unavailable; falling back to HEAD~1 ($BASE_COMMIT)."
fi

if [ -z "$BASE_COMMIT" ]; then
    log "Error: unable to determine merge-base from BASE_REF=$BASE_REF."
    log "Set BASE_REF explicitly, for example: BASE_REF=origin/main"
    exit 1
fi

printf '%s\n' "$BASE_COMMIT"
