#!/bin/bash
set +o histexpand

# check-action-sh-no-python.sh - Enforce that action shell scripts never invoke python or python3
#
# Scans all .sh files under actions/ (excluding node_modules) and reports any line
# that invokes the `python` or `python3` interpreter.  Action scripts must be
# self-contained and must not rely on Python being present in the runner environment.
#
# Exit codes:
#   0 - No violations found
#   1 - One or more violations found

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

violation_count=0

echo "Checking action shell scripts for python/python3 invocations..."
echo ""

# Find all .sh files under actions/, excluding node_modules subtrees.
while IFS= read -r file; do
    # grep -n returns "line_number:line_content" for each matching line.
    # The pattern \bpython3?\b matches the standalone words "python" and "python3"
    # but not compound identifiers like "mcp_handler_python".
    # Use grep -P for Perl-compatible word boundaries.
    matches=$(grep -nP '\bpython3?\b' "$file" 2>/dev/null || true)
    if [ -n "$matches" ]; then
        echo -e "${RED}VIOLATION${NC}: $file"
        while IFS= read -r match; do
            echo "  $match"
        done <<< "$matches"
        echo ""
        violation_count=$((violation_count + 1))
    fi
done < <(find actions -name "*.sh" -not -path "*/node_modules/*" -type f | sort)

echo "------------------------------------------------------------"

if [ "$violation_count" -eq 0 ]; then
    echo -e "${GREEN}All action shell scripts are free of python/python3 invocations${NC}"
    exit 0
fi

echo -e "${RED}$violation_count action shell script(s) invoke python or python3${NC}"
echo ""
echo "Action scripts must not rely on Python being available in the runner"
echo "environment.  Replace python/python3 calls with shell-native alternatives"
echo "(e.g. grep, awk, sed, node) or move the logic into a .js/.cjs helper."
exit 1
