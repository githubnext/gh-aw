#!/bin/bash

# check-workflow-prompts-no-inline-python.sh - Enforce no inline shell(python -c ...) snippets in workflow prompts
#
# Scans workflow markdown files under .github/workflows and reports prompt examples
# that embed `shell(python -c ...)` or `shell(python3 -c ...)` patterns.

set -euo pipefail

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
pattern='^[[:space:]]*([-*][[:space:]]+)?shell\([[:space:]]*python3?[[:space:]]+-c'

echo "Checking workflow prompts for inline shell(python -c ...) patterns..."
echo ""

while IFS= read -r file; do
    matches=$(grep -nEi "$pattern" "$file" 2>/dev/null || true)
    if [ -n "$matches" ]; then
        echo -e "${RED}VIOLATION${NC}: $file"
        while IFS= read -r match; do
            echo "  $match"
        done <<< "$matches"
        echo ""
        violation_count=$((violation_count + 1))
    fi
done < <(find .github/workflows -name "*.md" -type f | sort)

echo "------------------------------------------------------------"

if [ "$violation_count" -eq 0 ]; then
    echo -e "${GREEN}No inline shell(python -c ...) patterns found in workflow prompts${NC}"
    exit 0
fi

echo -e "${RED}$violation_count workflow prompt file(s) contain inline shell(python -c ...) patterns${NC}"
echo ""
echo "Use structured workflow tools (view/grep/glob) or shell-native commands instead of inline Python one-liners."
exit 1
