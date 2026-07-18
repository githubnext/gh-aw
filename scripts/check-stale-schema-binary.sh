#!/bin/bash
set +o histexpand

# check-stale-schema-binary.sh - Guard against a stale binary when schema files change
#
# Detects whether any file under pkg/parser/schemas/ was modified (relative to a
# base ref or the current working tree) without a corresponding rebuild of the
# gh-aw binary.  When drift is found the script exits non-zero with a clear
# remediation message.
#
# Because schema files are embedded at compile time via //go:embed, changing them
# without rebuilding means the running binary still holds the old schema.  This
# script is the parallel of check-stale-lock-files.sh for the schema → binary
# relationship.
#
# Usage:
#   check-stale-schema-binary.sh [--binary <path>] [--schemas-dir <dir>] [--base-ref <git-ref>]
#
# Options:
#   --binary <path>      Path to the gh-aw binary (default: ./gh-aw).
#   --schemas-dir <dir>  Directory containing the JSON schema files
#                        (default: pkg/parser/schemas).
#   --base-ref <git-ref> Git base ref for detecting changed files via
#                        `git diff <base-ref>...HEAD`. Intended for CI use.
#
# Exit codes:
#   0 - No modified schema files, or all modified schemas have a binary newer
#       than the most recently changed schema file
#   1 - Schema files were modified but the binary has not been rebuilt

set -euo pipefail

# Disable colors when not connected to a TTY, when NO_COLOR is set, or when
# TERM=dumb — keeps output readable in CI step summaries.
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

BINARY="./gh-aw"
SCHEMAS_DIR="pkg/parser/schemas"
BASE_REF=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --binary)
            BINARY="${2:?--binary requires an argument}"
            shift 2
            ;;
        --schemas-dir)
            SCHEMAS_DIR="${2:?--schemas-dir requires an argument}"
            shift 2
            ;;
        --base-ref)
            BASE_REF="${2:?--base-ref requires an argument}"
            shift 2
            ;;
        *)
            echo -e "${RED}ERROR${NC}: unknown argument: $1" >&2
            echo "Usage: check-stale-schema-binary.sh [--binary <path>] [--schemas-dir <dir>] [--base-ref <git-ref>]" >&2
            exit 1
            ;;
    esac
done

if [ ! -d "$SCHEMAS_DIR" ]; then
    echo -e "${RED}ERROR${NC}: schemas directory not found: $SCHEMAS_DIR" >&2
    exit 1
fi

collect_modified_files() {
    if [ -n "$BASE_REF" ]; then
        if git rev-parse --verify "${BASE_REF}^{commit}" >/dev/null 2>&1; then
            git diff --name-only "${BASE_REF}...HEAD"
            return
        fi
        echo -e "${YELLOW}WARN${NC}: --base-ref not found (${BASE_REF}); falling back to working-tree check." >&2
    fi

    # Local path: staged/unstaged changes relative to HEAD plus untracked files.
    git diff --name-only HEAD 2>/dev/null || true
    git ls-files --others --exclude-standard "$SCHEMAS_DIR" 2>/dev/null || true
}

all_modified=$(collect_modified_files)

# Filter to JSON files within the schemas directory.
schemas_prefix="${SCHEMAS_DIR#./}"
modified_schemas=$(printf '%s\n' "$all_modified" \
    | grep "^${schemas_prefix}/.*\.json$" \
    || true)

if [ -z "$modified_schemas" ]; then
    echo -e "${GREEN}✓ No modified schema files detected.${NC}"
    exit 0
fi

echo "Modified schema file(s) detected:"
while IFS= read -r f; do
    [ -n "$f" ] || continue
    echo "  $f"
done <<< "$modified_schemas"
echo ""

# If the binary does not exist, it definitely hasn't been rebuilt.
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}ERROR${NC}: Binary not found at '$BINARY' — schema files were modified but the binary has not been built."
    echo ""
    echo -e "${YELLOW}Fix:${NC} Rebuild the binary so the updated schemas are embedded:"
    echo ""
    echo "  make build"
    echo ""
    exit 1
fi

# Compare modification times: the binary must be newer than every changed schema file.
# Note: git checkout does not preserve file timestamps, so this check is meaningful
# only in a local working tree where file edits set real mtimes.  CI workflows should
# always run `make build` before executing the binary regardless.
#
# For deleted schema files the individual path no longer exists; use the schemas
# directory mtime as a proxy (directory mtime updates on any entry add/remove).
stale_schemas=()
while IFS= read -r f; do
    [ -n "$f" ] || continue
    if [ -f "$f" ]; then
        ref_path="$f"
    else
        # Deleted file: fall back to the schemas directory mtime.
        ref_path="$SCHEMAS_DIR"
    fi
    if [ "$ref_path" -nt "$BINARY" ]; then
        stale_schemas+=("$f")
    fi
done <<< "$modified_schemas"

if [ ${#stale_schemas[@]} -eq 0 ]; then
    echo -e "${GREEN}✓ Binary is up to date with all schema changes.${NC}"
    exit 0
fi

echo -e "${RED}ERROR${NC}: The following schema change(s) are not yet reflected in the binary at '${BINARY}':"
echo ""
for f in "${stale_schemas[@]}"; do
    echo "  $f"
done
echo ""
echo -e "${YELLOW}Fix:${NC} Rebuild the binary so the updated schemas are embedded:"
echo ""
echo "  make build"
echo ""
echo "Schema files are embedded at compile time via //go:embed."
echo "Running \`go test ./pkg/parser/...\` validates schemas but does not rebuild the ./gh-aw binary."
exit 1
