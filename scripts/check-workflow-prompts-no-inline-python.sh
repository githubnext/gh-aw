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

warning_count=0

echo "Checking workflow prompts for inline shell(python -c ...) patterns..."
echo ""

while IFS= read -r file; do
    matches=$(
        awk '
            BEGIN { in_code = 0 }
            /^```/ { in_code = !in_code; next }
            in_code {
                line = tolower($0)
                if (line ~ /shell\([[:space:]]*python3?[[:space:]]+-c/) {
                    printf "%d:%s\n", NR, $0
                }
            }
        ' "$file" 2>/dev/null || true
    )
    if [ -n "$matches" ]; then
        echo -e "${RED}WARNING${NC}: $file"
        while IFS= read -r match; do
            line_number="${match%%:*}"
            line_content="${match#*:}"
            echo "  $match"
            echo "::warning file=$file,line=$line_number,title=Inline Python in workflow prompt::$line_content"
        done <<< "$matches"
        echo ""
        warning_count=$((warning_count + 1))
    fi
done < <(find .github/workflows -name "*.md" -type f | sort)

echo "------------------------------------------------------------"

if [ "$warning_count" -eq 0 ]; then
    echo -e "${GREEN}No inline shell(python -c ...) patterns found in workflow prompts${NC}"
    exit 0
fi

echo -e "${RED}$warning_count workflow prompt file(s) contain inline shell(python -c ...) patterns${NC}"
echo ""
echo "Use structured workflow tools (view/grep/glob) or shell-native commands instead of inline Python one-liners."
echo "This is a warning and does not fail lint."
exit 0
