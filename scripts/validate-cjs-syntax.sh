#!/bin/bash
set +o histexpand

# validate-cjs-syntax.sh — Catch module-load SyntaxErrors in all runtime .cjs files
#
# Runs `node --check` on every non-test .cjs file under actions/setup/js/ so that
# mistakes such as `await` inside a non-async function (PR #43170 regression) are
# caught in CI before any workflow runs.
#
# `node --check` parses the file and reports SyntaxErrors without executing the
# module, so no @actions/core or other runtime dependency is needed.
#
# Exit codes:
#   0 - All files parsed cleanly
#   1 - One or more files contain a SyntaxError (or failed to parse)

set -euo pipefail

# Disable colors when not connected to a TTY, when NO_COLOR is set, or when
# TERM=dumb — keeps output readable in CI step summaries.
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != "dumb" ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    NC=''
fi

failure_count=0
checked_count=0

echo "Checking .cjs files for module-load SyntaxErrors (node --check)..."
echo ""

# Find all non-test .cjs files under actions/setup/js/, excluding node_modules.
while IFS= read -r file; do
    checked_count=$((checked_count + 1))
    if ! output=$(node --check "$file" 2>&1); then
        echo -e "${RED}SYNTAX ERROR${NC}: $file"
        while IFS= read -r line; do
            echo "  $line"
        done <<< "$output"
        echo ""
        failure_count=$((failure_count + 1))
    fi
done < <(find actions/setup/js -name "*.cjs" -not -name "*.test.cjs" -not -path "*/node_modules/*" -type f | sort)

echo "------------------------------------------------------------"

if [ "$failure_count" -eq 0 ]; then
    echo -e "${GREEN}All $checked_count .cjs files parsed cleanly${NC}"
    exit 0
fi

echo -e "${RED}$failure_count of $checked_count .cjs file(s) failed syntax check${NC}"
echo ""
echo "Fix the SyntaxErrors above before proceeding."
echo "Common cause: 'await' used outside an async function (e.g. in a sync helper"
echo "that was auto-fixed by the ESLint require-await-core-summary-write rule)."
exit 1
