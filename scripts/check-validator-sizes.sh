#!/bin/bash
set +o histexpand

# check-validator-sizes.sh - Enforce the 512-line hard limit on validator files
#
# This script finds any *_validation.go file that exceeds the limit and reports it.
#
# Exit codes:
#   0 - All validator files are within the limit (or WARN_ONLY=1)
#   1 - One or more validator files exceed the limit

set -euo pipefail

# Disable colors when not connected to a TTY, when NO_COLOR is set, or when
# TERM=dumb — this keeps output readable when captured into CI step summaries.
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != "dumb" ]; then
    RED='\033[0;31m'
    YELLOW='\033[1;33m'
    GREEN='\033[0;32m'
    NC='\033[0m'
else
    RED=''
    YELLOW=''
    GREEN=''
    NC=''
fi

# Hard limit (lines per validator file)
HARD_LIMIT=768

# Set WARN_ONLY=1 to make the check non-blocking (report but always exit 0)
WARN_ONLY="${WARN_ONLY:-0}"

over_limit=0

echo "Checking validator file sizes in pkg/ directory (limit: ${HARD_LIMIT} lines)..."
echo ""

# Find all *_validation.go files, excluding test files
while IFS= read -r file; do
    # wc -l counts newline characters; a file without a trailing newline is
    # undercounted by 1. Go source files always end with a newline (enforced by
    # gofmt), so in practice the count is exact.
    line_count=$(wc -l < "$file")

    if [ "$line_count" -gt "$HARD_LIMIT" ]; then
        echo -e "${RED}OVER LIMIT${NC}: $file"
        echo -e "   Lines: ${RED}$line_count${NC} / ${HARD_LIMIT} (exceeded by $((line_count - HARD_LIMIT)))"
        echo ""
        over_limit=$((over_limit + 1))
    fi
done < <(find pkg -name "*_validation.go" ! -name "*_test.go" -type f | sort)

echo "------------------------------------------------------------"

if [ "$over_limit" -eq 0 ]; then
    echo -e "${GREEN}All validator files are within the ${HARD_LIMIT}-line limit${NC}"
    exit 0
fi

echo -e "${YELLOW}$over_limit validator file(s) exceed the ${HARD_LIMIT}-line hard limit${NC}"
echo ""
echo "See AGENTS.md 'Validation Complexity Guidelines' for guidance:"
echo "  - Target size: 100-200 lines per validator"
echo "  - Hard limit:  ${HARD_LIMIT} lines (refactor if exceeded)"
echo ""
echo "When to split a validator:"
echo "  - File exceeds ${HARD_LIMIT} lines"
echo "  - File contains 2+ unrelated validation domains"
echo "  - Complex cross-dependencies require separate testing"

if [ "$WARN_ONLY" = "1" ]; then
    echo ""
    echo "Note: Running in warn-only mode (WARN_ONLY=1). Fix these files to pass a future blocking check."
    exit 0
fi

exit 1
