#!/bin/bash
# Apply Astro to GFM conversion to all docs files
# Usage: ./scripts/apply-astro-conversion.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$REPO_ROOT"

# Find all markdown files in docs/src/content/docs that contain Astro adornments
FILES=$(grep -r "^:::" docs/src/content/docs --include="*.md" -l | sort)

if [ -z "$FILES" ]; then
    echo "No files with Astro adornments found."
    exit 0
fi

echo "Converting Astro adornments to GFM alerts in the following files:"
echo "$FILES"
echo ""

for file in $FILES; do
    echo "Processing: $file"
    "$SCRIPT_DIR/convert-astro-to-gfm.sh" "$file"
done

echo ""
echo "Conversion complete!"
echo ""
echo "Files converted: $(echo "$FILES" | wc -l)"
