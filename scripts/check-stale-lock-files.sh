#!/bin/bash
set +o histexpand

# check-stale-lock-files.sh - Lightweight guard for stale .lock.yml files
#
# Detects workflow .md files that have been modified (in the git working tree or
# staging area) without their compiled .lock.yml being regenerated.
#
# Unlike check-workflow-drift.sh, this script does not require the gh-aw binary:
# it uses git to identify modified .md files and checks whether each has a
# corresponding .lock.yml that was also modified.  Use check-workflow-drift.sh
# for a thorough recompile-based verification; use this script as a fast early
# gate that catches the obvious case where a .md was edited and not recompiled.
#
# When there are no modified .md files in the working tree or staging area the
# script exits 0 immediately, so it is a no-op on a clean checkout.
#
# Usage:
#   check-stale-lock-files.sh [--dir <workflows-dir>]
#
# Options:
#   --dir <dir>   Workflows directory to scan (default: .github/workflows).
#                 The script only examines .md files under this directory.
#
# Exit codes:
#   0 - No modified .md files detected, or every modified .md has a
#       correspondingly modified .lock.yml
#   1 - One or more modified .md files lack an up-to-date .lock.yml

set -euo pipefail

# Disable colors when not connected to a TTY, when NO_COLOR is set, or when
# TERM=dumb — this keeps output readable when captured into CI step summaries.
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != "dumb" ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    NC=''
fi

WORKFLOWS_DIR=".github/workflows"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --dir)
            WORKFLOWS_DIR="${2:?--dir requires an argument}"
            shift 2
            ;;
        *)
            echo -e "${RED}ERROR${NC}: unknown argument: $1" >&2
            echo "Usage: check-stale-lock-files.sh [--dir <workflows-dir>]" >&2
            exit 1
            ;;
    esac
done

if [ ! -d "$WORKFLOWS_DIR" ]; then
    echo -e "${RED}ERROR${NC}: workflows directory not found: $WORKFLOWS_DIR" >&2
    exit 1
fi

# Collect all files that git sees as modified (staged or unstaged vs HEAD).
# On a clean checkout with no edits this set is empty, so the check is a no-op.
all_modified=$(git diff --name-only HEAD 2>/dev/null || true)

# Filter to .md files within the workflows directory.
# Strip a leading "./" from WORKFLOWS_DIR for consistent prefix matching.
# Exclude subdirectories whose files are compiled into parent workflow lock files
# rather than producing their own lock file (e.g. shared/, and skill directories).
workflows_prefix="${WORKFLOWS_DIR#./}"
modified_mds=$(printf '%s\n' "$all_modified" \
    | grep "^${workflows_prefix}.*\.md$" \
    | grep -v "^${workflows_prefix}/shared/" \
    | grep -v "^${workflows_prefix}/skills/" \
    || true)

if [ -z "$modified_mds" ]; then
    echo -e "${GREEN}✓ No modified workflow markdown files detected.${NC}"
    exit 0
fi

stale_files=()
missing_locks=()

while IFS= read -r md; do
    [ -n "$md" ] || continue
    lock="${md%.md}.lock.yml"

    if [ ! -f "$lock" ]; then
        missing_locks+=("$md")
    elif ! printf '%s\n' "$all_modified" | grep -Fxq "$lock"; then
        stale_files+=("$md")
    fi
done <<< "$modified_mds"

if [ ${#stale_files[@]} -eq 0 ] && [ ${#missing_locks[@]} -eq 0 ]; then
    echo -e "${GREEN}✓ All modified workflow lock files are up to date.${NC}"
    exit 0
fi

echo ""

if [ ${#missing_locks[@]} -gt 0 ]; then
    echo -e "${RED}ERROR${NC}: The following workflow .md files have no compiled .lock.yml:"
    echo ""
    for f in "${missing_locks[@]}"; do
        echo "  $f"
    done
    echo ""
fi

if [ ${#stale_files[@]} -gt 0 ]; then
    echo -e "${RED}ERROR${NC}: The following workflow .md files were modified but their .lock.yml was not regenerated:"
    echo ""
    for f in "${stale_files[@]}"; do
        echo "  $f"
    done
    echo ""
fi

echo -e "${YELLOW}Fix:${NC} Recompile the workflow lock files, then commit them together with their .md sources:"
echo ""
echo "  make recompile"
echo ""
exit 1
